package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/AyqutEfe/ReAppKit/catalog"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

func validPlan() adminPlan {
	return adminPlan{Address: "127.0.0.1:12345", Token: strings.Repeat("a", 64), Apps: []catalog.App{{ID: "Git.Git", Title: "Git", Scope: "machine", RequiresAdmin: true}, {ID: "wez.wezterm", Title: "WezTerm", Scope: "auto", RequiresAdmin: true}}}
}
func TestAdminPlanRejectsUnsafeScopeAddressAndIDs(t *testing.T) {
	plan := validPlan()
	encode := func(p adminPlan) string { data, _ := json.Marshal(p); return base64.StdEncoding.EncodeToString(data) }
	if _, err := decodeAdminPlan(encode(plan)); err != nil {
		t.Fatal(err)
	}
	bad := []adminPlan{}
	p := validPlan()
	p.Address = "10.0.0.1:1234"
	bad = append(bad, p)
	p = validPlan()
	p.Token = "abc"
	bad = append(bad, p)
	p = validPlan()
	p.Apps[0].Scope = "user"
	bad = append(bad, p)
	p = validPlan()
	p.Apps[0].ID = "Git.Git;whoami"
	bad = append(bad, p)
	p = validPlan()
	p.Apps[0].RequiresAdmin = false
	bad = append(bad, p)
	p = validPlan()
	p.Apps = append(p.Apps, p.Apps[0])
	bad = append(bad, p)
	p = validPlan()
	p.Apps[0].Source = "msstore"
	bad = append(bad, p)
	for _, p := range bad {
		if _, err := decodeAdminPlan(encode(p)); err == nil {
			t.Fatalf("accepted=%+v", p)
		}
	}
}
func TestWorkerOnlyInstallsApprovedPackagesAndReturnsFailures(t *testing.T) {
	parent, worker := net.Pipe()
	plan := validPlan()
	var calls []string
	done := make(chan error, 1)
	go func() {
		done <- serveAdminRequests(context.Background(), worker, plan, func(_ context.Context, id string, scope ...string) error {
			calls = append(calls, id+":"+scope[0])
			if id == "Git.Git" {
				return errors.New("installer failed")
			}
			return nil
		})
	}()
	decoder, encoder := json.NewDecoder(parent), json.NewEncoder(parent)
	var hello adminHello
	if err := decoder.Decode(&hello); err != nil || hello.Token != plan.Token {
		t.Fatalf("hello=%+v error=%v", hello, err)
	}
	for _, request := range []adminRequest{{"Not.App", "machine"}, {"Git.Git", "user"}, {"Git.Git", "machine"}, {"wez.wezterm", "auto"}} {
		if err := encoder.Encode(request); err != nil {
			t.Fatal(err)
		}
		var reply adminReply
		if err := decoder.Decode(&reply); err != nil {
			t.Fatal(err)
		}
		if request.ID != "wez.wezterm" && reply.Error == "" {
			t.Fatalf("failure not returned: %+v", request)
		}
		if request.ID == "wez.wezterm" && reply.Error != "" {
			t.Fatal(reply.Error)
		}
	}
	parent.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not exit after parent closed")
	}
	if !reflect.DeepEqual(calls, []string{"Git.Git:machine", "wez.wezterm:auto"}) {
		t.Fatalf("calls=%v", calls)
	}
}
func TestParentDisconnectCancelsRunningWorkerInstall(t *testing.T) {
	parent, worker := net.Pipe()
	started, cancelled := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serveAdminRequests(context.Background(), worker, validPlan(), func(ctx context.Context, _ string, _ ...string) error {
			close(started)
			<-ctx.Done()
			close(cancelled)
			return ctx.Err()
		})
	}()
	var hello adminHello
	if err := json.NewDecoder(parent).Decode(&hello); err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(parent).Encode(adminRequest{"Git.Git", "machine"}); err != nil {
		t.Fatal(err)
	}
	<-started
	parent.Close()
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("install not cancelled")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker leaked")
	}
}
