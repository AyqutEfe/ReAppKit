// Package winget adapts exact WinGet package IDs to core jobs.
package winget

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AyqutEfe/ReAppKit/internal/core"
)

// Runner permits a deterministic fake; production uses exec.CommandContext.
type Runner interface {
	Run(context.Context, ...string) ([]byte, error)
}

type Client struct {
	runner Runner
	source string
	input  io.Reader
	output io.Writer
}

// WithSource leaves the original client unchanged so mixed plans remain isolated.
func (c *Client) WithSource(source string) *Client {
	clone := *c
	clone.source = source
	return &clone
}

// WithInteraction connects WinGet's own agreement prompt to the released
// terminal. No agreement is accepted by ReAppKit on the user's behalf.
func (c *Client) WithInteraction(input io.Reader, output io.Writer) *Client {
	clone := *c
	clone.input, clone.output = input, output
	return &clone
}

type interactiveRunner interface {
	RunInteractive(context.Context, io.Reader, io.Writer, ...string) ([]byte, error)
}

func (c *Client) sourceName() string {
	if c.source == "" {
		return "winget"
	}
	return c.source
}

func (c *Client) validateSource() error {
	if source := c.sourceName(); source != "winget" && source != "msstore" {
		return fmt.Errorf("unsupported WinGet source %q", source)
	}
	return nil
}

func New(runner Runner) *Client {
	if runner == nil {
		runner = osRunner{}
	}
	return &Client{runner: runner}
}

type osRunner struct{}

func wingetCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	if runtime.GOOS != "windows" {
		return nil, errors.New("WinGet is supported only on Windows")
	}
	if runtime.GOARCH != "amd64" {
		return nil, errors.New("ReAppKit requires Windows 11 x64")
	}
	version, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Environment]::OSVersion.Version.Build").Output()
	if err != nil {
		return nil, fmt.Errorf("cannot determine Windows build: %w", err)
	}
	build, err := strconv.Atoi(strings.TrimSpace(string(version)))
	if err != nil || build < 22000 {
		return nil, errors.New("ReAppKit requires Windows 11 build 22000 or later")
	}
	return exec.CommandContext(ctx, "winget", args...), nil
}

func (osRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	command, err := wingetCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	return command.CombinedOutput()
}

func (osRunner) RunInteractive(ctx context.Context, input io.Reader, output io.Writer, args ...string) ([]byte, error) {
	command, err := wingetCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	var captured bytes.Buffer
	stream := io.MultiWriter(output, &captured)
	command.Stdin, command.Stdout, command.Stderr = input, stream, stream
	err = command.Run()
	return captured.Bytes(), err
}

func (c *Client) Available(ctx context.Context) error {
	output, err := c.runner.Run(ctx, "--version")
	if err != nil {
		return fmt.Errorf("WinGet unavailable: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if strings.TrimSpace(string(output)) == "" {
		return errors.New("WinGet version output is empty")
	}
	return nil
}

var packageID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func (c *Client) Job(id, title string, scopes ...string) core.Job {
	scope := "user"
	if len(scopes) > 0 && scopes[0] != "" {
		scope = scopes[0]
	}
	check := func(ctx context.Context) (bool, error) { return c.Installed(ctx, id) }
	return core.Job{
		ID: "app:" + id, Title: title,
		Check:  check,
		Apply:  func(ctx context.Context) error { return c.Install(ctx, id, scope) },
		Verify: check,
	}
}

func validateID(id string) error {
	if !packageID.MatchString(id) {
		return fmt.Errorf("invalid WinGet package ID %q", id)
	}
	return nil
}

func (c *Client) Installed(ctx context.Context, id string) (bool, error) {
	if err := c.validateSource(); err != nil {
		return false, err
	}
	if err := validateID(id); err != nil {
		return false, err
	}
	if err := c.Available(ctx); err != nil {
		return false, err
	}
	// Restrict lookup to the same catalog used for installation. Without a
	// source, WinGet also opens msstore, whose first-use agreement can fail a
	// non-interactive check before it can report installed applications.
	args := []string{"list", "--id", id, "--exact", "--source", c.sourceName(), "--disable-interactivity"}
	output, err := c.runner.Run(ctx, args...)
	// A transient WinGet cache failure must not cause an already-installed
	// package to be installed again. Retry only this read-only query once.
	if hasExitCode(err, 0x80071130) {
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timer.C:
			output, err = c.runner.Run(ctx, args...)
		}
	}
	if err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) && uint32(exitErr.ExitCode()) == 0x8A150014 {
			return false, nil
		}
		if sourceAgreementRequired(err) {
			return false, sourceAgreementError(id, c.sourceName())
		}
		if hasExitCode(err, 0x80071130) {
			return false, fmt.Errorf("%s kurulum durumu doğrulanamadı: WinGet kaynak önbelleği açılamadı (0x80071130). PowerShell'de `winget source update --name %s` çalıştırıp yeniden deneyin; uygulama kurulu olabilir: %w", id, c.sourceName(), err)
		}
		return false, fmt.Errorf("WinGet list %s: %w: %s", id, err, strings.TrimSpace(string(output)))
	}
	// WinGet list output is localized, but package IDs remain stable. The exact
	// query limits results before we inspect its table rows.
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		for _, field := range fields {
			if field == id {
				return true, nil
			}
		}
	}
	return false, nil
}

func (c *Client) Install(ctx context.Context, id string, scopes ...string) error {
	if err := c.validateSource(); err != nil {
		return err
	}
	scope := "user"
	if len(scopes) > 0 && scopes[0] != "" {
		scope = scopes[0]
	}
	if err := validateID(id); err != nil {
		return err
	}
	if scope != "user" && scope != "machine" && scope != "auto" {
		return fmt.Errorf("invalid WinGet install scope %q", scope)
	}
	if c.sourceName() == "msstore" && scope != "user" {
		return errors.New("Microsoft Store apps require the original user process and user scope")
	}
	if err := c.Available(ctx); err != nil {
		return err
	}
	args := []string{"install", "--id", id, "--exact", "--source", c.sourceName()}
	// Some manifests (including WezTerm) omit Scope. An explicit scope then
	// filters out their installer; auto lets WinGet select without this filter.
	if scope != "auto" && c.sourceName() != "msstore" {
		args = append(args, "--scope", scope)
	}
	args = append(args, "--silent", "--disable-interactivity")
	output, err := c.runner.Run(ctx, args...)
	if (packageAgreementRequired(err) || sourceAgreementRequired(err)) && c.input != nil && c.output != nil {
		if runner, ok := c.runner.(interactiveRunner); ok {
			fmt.Fprintf(c.output, "\n%s: WinGet koşulları gösterecek. İnceleyip kabul ediyorsanız istemi onaylayın; reddederseniz bu uygulama kurulmaz.\n", id)
			// Keep the installer silent, but allow WinGet's agreement prompt.
			output, err = runner.RunInteractive(ctx, c.input, c.output, args[:len(args)-1]...)
		}
	}
	if err != nil {
		if installerCancelled(err, output) {
			return fmt.Errorf("%s kurulumu iptal edildi veya Windows izin isteği tamamlanamadı; yeniden denemede UAC penceresini kontrol edip onaylayın: %w: %s", id, err, strings.TrimSpace(string(output)))
		}
		if sourceAgreementRequired(err) {
			return sourceAgreementError(id, c.sourceName())
		}
		if packageAgreementRequired(err) {
			return fmt.Errorf("%s paket koşulları onaylanmadı; bu uygulama kurulmadı (0x8a150041). PowerShell'de `winget install --id %s --exact --source %s` ile koşulları inceleyebilirsiniz: %w", id, id, c.sourceName(), err)
		}
		if hasExitCode(err, 0x80190194) {
			return fmt.Errorf("%s kurucu indirme adresi HTTP 404 döndürdü; paket kaynağındaki bağlantı mevcut değil. `winget source update --name %s` sonrası yeniden deneyin; sürerse yayıncının resmi dağıtımını kontrol edin: %w: %s", id, c.sourceName(), err, strings.TrimSpace(string(output)))
		}
		if noApplicableInstaller(err) && scope == "auto" {
			return fmt.Errorf("WinGet has no applicable installer for %s without a scope filter; check architecture, Windows version and WinGet logs: %w: %s", id, err, strings.TrimSpace(string(output)))
		}
		if noApplicableInstaller(err) {
			return fmt.Errorf("WinGet has no applicable %s-scope installer for %s; check this package's supported scope and architecture: %w", scope, id, err)
		}
		return fmt.Errorf("WinGet install %s: %w: %s", id, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func noApplicableInstaller(err error) bool {
	var exitErr interface{ ExitCode() int }
	return errors.As(err, &exitErr) && uint32(exitErr.ExitCode()) == 0x8A150010
}

func sourceAgreementRequired(err error) bool {
	return hasExitCode(err, 0x8A150046)
}

func packageAgreementRequired(err error) bool {
	return hasExitCode(err, 0x8A150041)
}

func hasExitCode(err error, code uint32) bool {
	var exitErr interface{ ExitCode() int }
	return errors.As(err, &exitErr) && uint32(exitErr.ExitCode()) == code
}

func sourceAgreementError(id, source string) error {
	return fmt.Errorf("WinGet source agreements require your review: run `winget list --id %s --exact --source %s` in PowerShell, review and accept the terms, then retry ReAppKit", id, source)
}

// A generic installer failure alone does not prove an elevation cancellation.
func installerCancelled(err error, output []byte) bool {
	var exitErr interface{ ExitCode() int }
	if !errors.As(err, &exitErr) {
		return false
	}
	code := uint32(exitErr.ExitCode())
	if code == 0x800704C7 {
		return true
	}
	text := strings.ToLower(string(output))
	return code == 0x8A15010C && (strings.Contains(text, "cancelled the installation") || strings.Contains(text, "canceled the installation"))
}
