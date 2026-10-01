package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type adminPlan struct {
	Address string        `json:"address"`
	Token   string        `json:"token"`
	Apps    []catalog.App `json:"apps"`
}
type adminHello struct {
	Token string `json:"token"`
	Error string `json:"error,omitempty"`
}
type adminReply struct {
	Error string `json:"error,omitempty"`
}
type adminRequest struct {
	ID    string `json:"id"`
	Scope string `json:"scope"`
}

var adminPackageID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func validateAdminApps(apps []catalog.App) (map[string]string, error) {
	if len(apps) == 0 || len(apps) > 256 {
		return nil, errors.New("invalid admin package count")
	}
	allowed := make(map[string]string)
	for _, app := range apps {
		if !adminPackageID.MatchString(app.ID) || !app.RequiresAdmin || (app.Scope != "machine" && app.Scope != "auto") || (app.Source != "" && app.Source != "winget") {
			return nil, fmt.Errorf("%s: yönetici işçisi yalnız machine/auto kapsamlı yönetici paketlerini kurabilir", app.ID)
		}
		if _, exists := allowed[app.ID]; exists {
			return nil, fmt.Errorf("duplicate admin package %s", app.ID)
		}
		allowed[app.ID] = app.Scope
	}
	return allowed, nil
}
func decodeAdminPlan(encoded string) (adminPlan, error) {
	var plan adminPlan
	if len(encoded) > 256*1024 {
		return plan, errors.New("admin plan too large")
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(raw, &plan); err != nil {
		return plan, err
	}
	host, port, err := net.SplitHostPort(plan.Address)
	if err != nil || host != "127.0.0.1" {
		return plan, errors.New("admin channel must use loopback")
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return plan, errors.New("invalid admin channel port")
	}
	token, err := hex.DecodeString(plan.Token)
	if err != nil || len(token) != 32 {
		return plan, errors.New("invalid admin channel token")
	}
	_, err = validateAdminApps(plan.Apps)
	return plan, err
}

type remoteAdminSession struct {
	conn    net.Conn
	encoder *json.Encoder
	decoder *json.Decoder
	allowed map[string]string
	mu      sync.Mutex
	cancel  context.CancelFunc
}

func (s *remoteAdminSession) Install(ctx context.Context, app catalog.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if scope, ok := s.allowed[app.ID]; !ok || scope != app.Scope {
		return errors.New("package is not in the approved admin plan")
	}
	stop := context.AfterFunc(ctx, func() { s.conn.Close() })
	defer stop()
	if err := s.encoder.Encode(adminRequest{ID: app.ID, Scope: app.Scope}); err != nil {
		return fmt.Errorf("yönetici bağlantısı: %w", err)
	}
	var reply adminReply
	if err := s.decoder.Decode(&reply); err != nil {
		return fmt.Errorf("yönetici işçisi sonucu alınamadı: %w", err)
	}
	if reply.Error != "" {
		return errors.New(reply.Error)
	}
	return nil
}
func (s *remoteAdminSession) Close() error { err := s.conn.Close(); s.cancel(); return err }

type directAdminSession struct {
	allowed map[string]string
	client  *winget.Client
}

func (s *directAdminSession) Close() error { return nil }
func (s *directAdminSession) Install(ctx context.Context, app catalog.App) error {
	if scope, ok := s.allowed[app.ID]; !ok || scope != app.Scope {
		return errors.New("package is not in the approved admin plan")
	}
	return s.client.Install(ctx, app.ID, app.Scope)
}

func startAdminSession(ctx context.Context, apps []catalog.App) (adminSession, error) {
	allowed, err := validateAdminApps(apps)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if processElevated() {
		return &directAdminSession{allowed: allowed, client: winget.New(nil)}, nil
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	defer listener.Close()
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, err
	}
	plan := adminPlan{Address: listener.Addr().String(), Token: hex.EncodeToString(token[:]), Apps: apps}
	payload, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	workerCtx, cancel := context.WithCancel(ctx)
	success := false
	defer func() {
		if !success {
			cancel()
		}
	}()
	launched := make(chan error, 1)
	go func() {
		launched <- launchAdminWorker(workerCtx, executable, base64.StdEncoding.EncodeToString(payload))
	}()
	type connection struct {
		conn net.Conn
		err  error
	}
	accepted := make(chan connection)
	go func() {
		conn, err := listener.Accept()
		select {
		case accepted <- connection{conn, err}:
		case <-workerCtx.Done():
			if conn != nil {
				conn.Close()
			}
		}
	}()
	var conn net.Conn
	select {
	case value := <-accepted:
		if value.err != nil {
			return nil, value.err
		}
		conn = value.conn
	case err := <-launched:
		return nil, fmt.Errorf("Windows yönetici onayı iptal edildi veya işçi başlatılamadı: %v", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Minute):
		return nil, errors.New("Windows yönetici onayı bekleme süresi doldu")
	}
	// Bound startup input and authenticate the one-time local connection.
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	var hello adminHello
	if err := json.NewDecoder(io.LimitReader(conn, 1024)).Decode(&hello); err != nil {
		conn.Close()
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(hello.Token), []byte(plan.Token)) != 1 {
		conn.Close()
		return nil, errors.New("admin worker authentication failed")
	}
	if hello.Error != "" {
		conn.Close()
		return nil, errors.New(hello.Error)
	}
	conn.SetDeadline(time.Time{})
	session := &remoteAdminSession{conn: conn, encoder: json.NewEncoder(conn), decoder: json.NewDecoder(conn), allowed: allowed, cancel: cancel}
	go func() { <-workerCtx.Done(); conn.Close() }()
	success = true
	return session, nil
}
func runAdminWorker(encoded string) error {
	plan, err := decodeAdminPlan(encoded)
	if err != nil {
		return err
	}
	if !processElevated() {
		return errors.New("admin worker requires Windows administrator privileges")
	}
	conn, err := net.DialTimeout("tcp4", plan.Address, 20*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	client := winget.New(nil)
	if err := client.Available(context.Background()); err != nil {
		_ = json.NewEncoder(conn).Encode(adminHello{Token: plan.Token, Error: err.Error()})
		return err
	}
	return serveAdminRequests(context.Background(), conn, plan, client.Install)
}
func serveAdminRequests(ctx context.Context, conn net.Conn, plan adminPlan, install func(context.Context, string, ...string) error) error {
	defer conn.Close()
	allowed, err := validateAdminApps(plan.Apps)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(adminHello{Token: plan.Token}); err != nil {
		return err
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	requests := make(chan adminRequest)
	go func() {
		defer cancel()
		decoder := json.NewDecoder(conn)
		for {
			var request adminRequest
			if err := decoder.Decode(&request); err != nil {
				return
			}
			select {
			case requests <- request:
			case <-workerCtx.Done():
				return
			}
		}
	}()
	for {
		select {
		case <-workerCtx.Done():
			return nil
		case request := <-requests:
			var err error
			if scope, ok := allowed[request.ID]; !ok || scope != request.Scope {
				err = errors.New("package is not in the approved admin plan")
			} else if workerCtx.Err() != nil {
				return nil
			} else {
				err = install(workerCtx, request.ID, request.Scope)
			}
			reply := adminReply{}
			if err != nil {
				reply.Error = err.Error()
			}
			if err := encoder.Encode(reply); err != nil {
				return err
			}
		}
	}
}
