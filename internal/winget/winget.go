// Package winget adapts exact WinGet package IDs to core jobs.
package winget

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/AyqutEfe/ReAppKit/internal/core"
)

// Runner permits a deterministic fake; production uses exec.CommandContext.
type Runner interface { Run(context.Context, ...string) ([]byte, error) }

type Client struct { runner Runner }

func New(runner Runner) *Client {
	if runner == nil { runner = osRunner{} }
	return &Client{runner: runner}
}

type osRunner struct{}

func (osRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if runtime.GOOS != "windows" { return nil, errors.New("WinGet is supported only on Windows") }
	if runtime.GOARCH != "amd64" { return nil, errors.New("ReAppKit requires Windows 11 x64") }
	version, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Environment]::OSVersion.Version.Build").Output()
	if err != nil { return nil, fmt.Errorf("cannot determine Windows build: %w", err) }
	build, err := strconv.Atoi(strings.TrimSpace(string(version)))
	if err != nil || build < 22000 { return nil, errors.New("ReAppKit requires Windows 11 build 22000 or later") }
	return exec.CommandContext(ctx, "winget", args...).CombinedOutput()
}

func (c *Client) Available(ctx context.Context) error {
	output, err := c.runner.Run(ctx, "--version")
	if err != nil { return fmt.Errorf("WinGet unavailable: %w: %s", err, strings.TrimSpace(string(output))) }
	if strings.TrimSpace(string(output)) == "" { return errors.New("WinGet version output is empty") }
	return nil
}

var packageID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func (c *Client) Job(id, title string) core.Job {
	check := func(ctx context.Context) (bool, error) { return c.Installed(ctx, id) }
	return core.Job{
		ID: "app:" + id, Title: title,
		Check: check,
		Apply: func(ctx context.Context) error { return c.Install(ctx, id) },
		Verify: check,
	}
}

func validateID(id string) error {
	if !packageID.MatchString(id) { return fmt.Errorf("invalid WinGet package ID %q", id) }
	return nil
}

func (c *Client) Installed(ctx context.Context, id string) (bool, error) {
	if err := validateID(id); err != nil { return false, err }
	if err := c.Available(ctx); err != nil { return false, err }
	// Restrict lookup to the same catalog used for installation. Without a
	// source, WinGet also opens msstore, whose first-use agreement can fail a
	// non-interactive check before it can report installed applications.
	output, err := c.runner.Run(ctx, "list", "--id", id, "--exact", "--source", "winget", "--disable-interactivity")
	if err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) && uint32(exitErr.ExitCode()) == 0x8A150014 { return false, nil }
		if sourceAgreementRequired(err) { return false, sourceAgreementError(id) }
		return false, fmt.Errorf("WinGet list %s: %w: %s", id, err, strings.TrimSpace(string(output)))
	}
	// WinGet list output is localized, but package IDs remain stable. The exact
	// query limits results before we inspect its table rows.
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 { continue }
		for _, field := range fields {
			if field == id { return true, nil }
		}
	}
	return false, nil
}

func (c *Client) Install(ctx context.Context, id string) error {
	if err := validateID(id); err != nil { return err }
	if err := c.Available(ctx); err != nil { return err }
	output, err := c.runner.Run(ctx, "install", "--id", id, "--exact", "--source", "winget", "--scope", "user", "--silent", "--disable-interactivity")
	if err != nil {
		if sourceAgreementRequired(err) { return sourceAgreementError(id) }
		if noApplicableInstaller(err) {
			return fmt.Errorf("WinGet has no applicable user-scope installer for %s; check this package's supported scope and architecture: %w", id, err)
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
	var exitErr interface{ ExitCode() int }
	return errors.As(err, &exitErr) && uint32(exitErr.ExitCode()) == 0x8A150046
}

func sourceAgreementError(id string) error {
	return fmt.Errorf("WinGet source agreements require your review: run `winget list --id %s --exact --source winget` in PowerShell, review and accept the terms, then retry ReAppKit", id)
}
