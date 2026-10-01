package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/ui"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type packageConsentError struct{}

func (packageConsentError) Error() string { return "package agreement pending" }
func (packageConsentError) ExitCode() int { return -1978335167 }

type terminalConsentRunner struct {
	consentRunner
	prompts int
}

func (r *terminalConsentRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if args[0] == "install" {
		return nil, packageConsentError{}
	}
	return r.consentRunner.Run(ctx, args...)
}

func (r *terminalConsentRunner) RunInteractive(ctx context.Context, input io.Reader, output io.Writer, args ...string) ([]byte, error) {
	r.prompts++
	answer, _ := io.ReadAll(input)
	if string(answer) != "y\n" {
		return nil, packageConsentError{}
	}
	fmt.Fprintln(output, "WinGet agreement prompt")
	return r.consentRunner.Run(ctx, args...)
}

func TestExecutionHandsReleasedTerminalToStoreAgreementPrompt(t *testing.T) {
	r := &terminalConsentRunner{consentRunner: consentRunner{installed: map[string]bool{}}}
	var output strings.Builder
	c := &executionCommand{ctx: context.Background(), apps: []catalog.App{{ID: "9NKSQGP7F2NH", Title: "WhatsApp", Source: "msstore", Scope: "user"}}, client: winget.New(r), resultsDir: t.TempDir(), selection: ui.Selection{Apps: []ui.Choice{{ID: "9NKSQGP7F2NH"}}}, checkStoreNetwork: func(context.Context) error { return nil }}
	c.SetStdin(strings.NewReader("y\n"))
	c.SetStdout(&output)
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
		t.Fatal("Store app requested admin session")
		return nil, nil
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	rows := c.result.(ui.ResultsMsg).Results
	if len(rows) != 1 || rows[0].Status != "succeeded" || r.prompts != 1 || !strings.Contains(output.String(), "WinGet agreement prompt") {
		t.Fatalf("results=%+v prompts=%d output=%s", rows, r.prompts, output.String())
	}
}

type consentRunner struct {
	installed map[string]bool
	installs  []string
}

func (r *consentRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	if args[0] == "--version" {
		return []byte("v1.9"), nil
	}
	id := args[2]
	if args[0] == "list" {
		if r.installed[id] {
			return []byte(fmt.Sprintf("Name %s 1 winget\n", id)), nil
		}
		return nil, nil
	}
	if args[0] != "install" {
		return nil, fmt.Errorf("unexpected command: %v", args)
	}
	r.installs = append(r.installs, id)
	r.installed[id] = true
	return nil, nil
}

type fakeAdmin struct {
	runner *consentRunner
	closed bool
	failID string
}

func (a *fakeAdmin) Install(ctx context.Context, app catalog.App) error {
	if app.ID == a.failID {
		return errors.New("installer failed")
	}
	return winget.New(a.runner).Install(ctx, app.ID, app.Scope)
}
func (a *fakeAdmin) Close() error { a.closed = true; return nil }
func sampleCommand(t *testing.T) (*executionCommand, *consentRunner) {
	t.Helper()
	r := &consentRunner{installed: map[string]bool{}}
	apps := []catalog.App{{ID: "Git.Git", Title: "Git", Scope: "machine", RequiresAdmin: true}, {ID: "Obsidian.Obsidian", Title: "Obsidian", Scope: "user"}, {ID: "wez.wezterm", Title: "WezTerm", Scope: "auto", RequiresAdmin: true}}
	c := &executionCommand{ctx: context.Background(), apps: apps, client: winget.New(r), resultsDir: t.TempDir(), stdout: io.Discard, selection: ui.Selection{Apps: []ui.Choice{{ID: "Git.Git"}, {ID: "Obsidian.Obsidian"}, {ID: "wez.wezterm"}}}}
	c.checkNetwork = func(context.Context) error { return nil }
	return c, r
}
func TestOneInitialApprovalThenSerialInstallationWithoutInput(t *testing.T) {
	c, r := sampleCommand(t)
	starts := 0
	a := &fakeAdmin{runner: r}
	c.startAdmin = func(_ context.Context, apps []catalog.App) (adminSession, error) {
		starts++
		if len(r.installs) != 0 {
			t.Fatal("changes occurred before initial approval")
		}
		if len(apps) != 2 || apps[0].ID != "Git.Git" || apps[1].ID != "wez.wezterm" {
			t.Fatalf("plan=%+v", apps)
		}
		return a, nil
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if starts != 1 || !a.closed {
		t.Fatalf("starts=%d closed=%v", starts, a.closed)
	}
	if !reflect.DeepEqual(r.installs, []string{"Obsidian.Obsidian", "Git.Git", "wez.wezterm"}) {
		t.Fatalf("order=%v", r.installs)
	}
	for _, row := range c.result.(ui.ResultsMsg).Results {
		if row.Status != "succeeded" {
			t.Fatalf("result=%+v", c.result)
		}
	}
	starts = 0
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	if starts != 0 {
		t.Fatal("already installed apps requested UAC")
	}
	for _, row := range c.result.(ui.ResultsMsg).Results {
		if row.Status != "skipped" {
			t.Fatalf("rerun=%+v", c.result)
		}
	}
}
func TestInitialUACDenialPreventsAllInstallsAndFileChanges(t *testing.T) {
	c, r := sampleCommand(t)
	source, target := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	c.settings = []catalog.Setting{{ID: "profile", Title: "Profile", Source: source, Target: target}}
	c.selection.Settings = []ui.Choice{{ID: "setting:profile"}}
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) { return nil, errors.New("UAC cancelled") }
	c.Run()
	if _, ok := c.result.(ui.ExecutionErrorMsg); !ok {
		t.Fatalf("result=%+v", c.result)
	}
	if len(r.installs) != 0 {
		t.Fatalf("installs=%v", r.installs)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old" {
		t.Fatalf("file=%s err=%v", data, err)
	}
}
func TestWorkerPlanOnlyIncludesMissingSelectedAdminApps(t *testing.T) {
	c, r := sampleCommand(t)
	r.installed["Git.Git"] = true
	c.startAdmin = func(_ context.Context, apps []catalog.App) (adminSession, error) {
		if len(apps) != 1 || apps[0].ID != "wez.wezterm" {
			t.Fatalf("plan=%+v", apps)
		}
		return &fakeAdmin{runner: r}, nil
	}
	c.Run()
	if !reflect.DeepEqual(r.installs, []string{"Obsidian.Obsidian", "wez.wezterm"}) {
		t.Fatalf("installs=%v", r.installs)
	}
}
func TestIndependentJobsContinueAfterAdminFailure(t *testing.T) {
	c, r := sampleCommand(t)
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
		return &fakeAdmin{runner: r, failID: "Git.Git"}, nil
	}
	c.Run()
	rows := c.result.(ui.ResultsMsg).Results
	if rows[0].Status != "succeeded" || rows[1].Status != "failed" || rows[2].Status != "succeeded" {
		t.Fatalf("result=%+v", rows)
	}
}
func TestInvalidSelectionFailsBeforeUAC(t *testing.T) {
	c, _ := sampleCommand(t)
	c.selection.Apps = append(c.selection.Apps, ui.Choice{ID: "Unknown.App"})
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
		t.Fatal("invalid selection requested UAC")
		return nil, nil
	}
	c.Run()
	if _, ok := c.result.(ui.ExecutionErrorMsg); !ok {
		t.Fatalf("result=%+v", c.result)
	}
}
