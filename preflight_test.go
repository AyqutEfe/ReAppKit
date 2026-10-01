package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/ui"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
)

type preflightTransport func(*http.Request) (*http.Response, error)

func (f preflightTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConnectivityOnlyUsesHEADAndHandlesCDNRootResponses(t *testing.T) {
	for _, code := range []int{200, 403, 404, 407, 503} {
		client := &http.Client{Transport: preflightTransport(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodHead || r.URL.String() != wingetConnectivityURL {
				t.Fatalf("unexpected probe: %s %s", r.Method, r.URL)
			}
			return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
		})}
		err := probeConnection(context.Background(), client, wingetConnectivityURL)
		wantError := code == 407 || code >= 500
		if (err != nil) != wantError {
			t.Fatalf("status=%d error=%v", code, err)
		}
	}
}

func TestConnectivityTransportErrorPreservesCause(t *testing.T) {
	failure := errors.New("DNS unavailable")
	client := &http.Client{Transport: preflightTransport(func(*http.Request) (*http.Response, error) { return nil, failure })}
	err := probeConnection(context.Background(), client, wingetConnectivityURL)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "proxy") {
		t.Fatalf("error=%v", err)
	}
}

func TestNetworkFailureBeforeUACInstallsAndFileChanges(t *testing.T) {
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
	failure := errors.New("offline")
	c.checkNetwork = func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("network probe has no deadline")
		}
		return failure
	}
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
		t.Fatal("UAC before network check")
		return nil, nil
	}
	if err := c.Run(); err != nil {
		t.Fatal(err)
	}
	msg, ok := c.result.(ui.ExecutionErrorMsg)
	if !ok || !errors.Is(msg.Err, failure) || len(r.installs) != 0 {
		t.Fatalf("result=%+v installs=%v", c.result, r.installs)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old" {
		t.Fatalf("target=%q error=%v", data, err)
	}
	entries, err := os.ReadDir(c.resultsDir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected results: %v %v", entries, err)
	}
}

func TestInstalledAppsDoNotRequireConnectivity(t *testing.T) {
	c, r := sampleCommand(t)
	for _, app := range c.apps {
		r.installed[app.ID] = true
	}
	c.checkNetwork = func(context.Context) error { t.Fatal("network probe for installed apps"); return nil }
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
		t.Fatal("UAC for installed apps")
		return nil, nil
	}
	c.Run()
	for _, row := range c.result.(ui.ResultsMsg).Results {
		if row.Status != "skipped" {
			t.Fatalf("result=%+v", c.result)
		}
	}
}

type unavailableRunner struct{ calls int }

type sourceConsentExit int

func (e sourceConsentExit) Error() string { return "source agreement pending" }
func (e sourceConsentExit) ExitCode() int { return int(e) }

type sourceConsentRunner struct {
	consentRunner
	accepted       bool
	refuse         bool
	prompts        int
	queryDeadline  bool
	promptDeadline bool
}

func (r *sourceConsentRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	if args[0] == "list" && args[5] == "msstore" && !r.accepted {
		_, r.queryDeadline = ctx.Deadline()
		return nil, sourceConsentExit(-1978335162)
	}
	return r.consentRunner.Run(ctx, args...)
}

func (r *sourceConsentRunner) RunInteractive(ctx context.Context, input io.Reader, output io.Writer, args ...string) ([]byte, error) {
	r.prompts++
	_, r.promptDeadline = ctx.Deadline()
	if args[0] != "list" || len(r.installs) != 0 {
		return nil, errors.New("changes before source review")
	}
	fmtOutput := "Store source terms shown\n"
	io.WriteString(output, fmtOutput)
	if r.refuse {
		return nil, sourceConsentExit(-1978335162)
	}
	r.accepted = true
	return nil, sourceConsentExit(-1978335212)
}

func TestPreflightReviewsSourceBeforeUACAndChanges(t *testing.T) {
	for _, refuse := range []bool{false, true} {
		c, _ := sampleCommand(t)
		r := &sourceConsentRunner{consentRunner: consentRunner{installed: map[string]bool{}}, refuse: refuse}
		c.client = winget.New(r)
		c.apps = append(c.apps, catalog.App{ID: "9NKSQGP7F2NH", Title: "WhatsApp", Source: "msstore", Scope: "user"})
		c.selection.Apps = append(c.selection.Apps, ui.Choice{ID: "9NKSQGP7F2NH"})
		c.checkStoreNetwork = func(context.Context) error { return nil }
		var output strings.Builder
		c.SetStdin(strings.NewReader("y\n"))
		c.SetStdout(&output)
		source, target := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
		if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
			t.Fatal(err)
		}
		c.settings = []catalog.Setting{{ID: "profile", Title: "Profile", Source: source, Target: target}}
		c.selection.Settings = []ui.Choice{{ID: "setting:profile"}}
		starts := 0
		c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
			starts++
			if !r.accepted || len(r.installs) != 0 {
				t.Fatal("UAC or changes before source approval")
			}
			return &fakeAdmin{runner: &r.consentRunner}, nil
		}
		if err := c.Run(); err != nil {
			t.Fatal(err)
		}
		if r.prompts != 1 || !r.queryDeadline || r.promptDeadline || !strings.Contains(output.String(), "Store source terms shown") {
			t.Fatalf("prompts=%d query deadline=%v prompt deadline=%v output=%s", r.prompts, r.queryDeadline, r.promptDeadline, output.String())
		}
		data, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if refuse {
			msg, ok := c.result.(ui.ExecutionErrorMsg)
			if !ok || !errors.Is(msg.Err, errPreflightStopped) || starts != 0 || len(r.installs) != 0 || string(data) != "old" {
				t.Fatalf("refusal result=%+v starts=%d installs=%v target=%s", c.result, starts, r.installs, data)
			}
		} else {
			if starts != 1 || string(data) != "new" {
				t.Fatalf("starts=%d target=%s", starts, data)
			}
			for _, row := range c.result.(ui.ResultsMsg).Results {
				if row.Status != "succeeded" {
					t.Fatalf("row=%+v", row)
				}
			}
		}
	}
}

func (r *unavailableRunner) Run(context.Context, ...string) ([]byte, error) {
	r.calls++
	return nil, errors.New("WinGet missing")
}

func TestFileOnlyRunDoesNotRequireWinGetOrNetwork(t *testing.T) {
	c, _ := sampleCommand(t)
	r := &unavailableRunner{}
	c.client = winget.New(r)
	c.selection.Apps = nil
	source, target := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("profile"), 0600); err != nil {
		t.Fatal(err)
	}
	c.settings = []catalog.Setting{{ID: "profile", Title: "Profile", Source: source, Target: target}}
	c.selection.Settings = []ui.Choice{{ID: "setting:profile"}}
	c.checkNetwork = func(context.Context) error { t.Fatal("network probe for local files"); return nil }
	c.Run()
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "profile" || r.calls != 0 {
		t.Fatalf("target=%q error=%v WinGet calls=%d", data, err, r.calls)
	}
}

func TestMissingWinGetBlocksUserAppBeforeAnyChanges(t *testing.T) {
	c, _ := sampleCommand(t)
	r := &unavailableRunner{}
	c.client = winget.New(r)
	c.selection.Apps = []ui.Choice{{ID: "Obsidian.Obsidian"}}
	c.checkNetwork = func(context.Context) error { t.Fatal("network before WinGet availability"); return nil }
	c.Run()
	msg, ok := c.result.(ui.ExecutionErrorMsg)
	if !ok || !strings.Contains(msg.Err.Error(), "App Installer") || r.calls != 1 {
		t.Fatalf("result=%+v calls=%d", c.result, r.calls)
	}
}

func TestCancelledFileOnlyRunStopsBeforeCopy(t *testing.T) {
	c, _ := sampleCommand(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c.ctx = ctx
	c.selection.Apps = nil
	source, target := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	c.settings = []catalog.Setting{{ID: "profile", Title: "Profile", Source: source, Target: target}}
	c.selection.Settings = []ui.Choice{{ID: "setting:profile"}}
	c.Run()
	msg, ok := c.result.(ui.ExecutionErrorMsg)
	if !ok || !errors.Is(msg.Err, context.Canceled) {
		t.Fatalf("result=%+v", c.result)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old" {
		t.Fatalf("target=%q error=%v", data, err)
	}
}

func TestSettingRequiringAppStillChecksWinGet(t *testing.T) {
	c, _ := sampleCommand(t)
	r := &unavailableRunner{}
	c.client = winget.New(r)
	c.selection.Apps = nil
	source, target := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("profile"), 0600); err != nil {
		t.Fatal(err)
	}
	c.settings = []catalog.Setting{{ID: "profile", Title: "Profile", Source: source, Target: target, RequiresApp: "wez.wezterm"}}
	c.selection.Settings = []ui.Choice{{ID: "setting:profile"}}
	c.Run()
	if _, ok := c.result.(ui.ExecutionErrorMsg); !ok || r.calls != 1 {
		t.Fatalf("result=%+v calls=%d", c.result, r.calls)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("target created before prerequisite check: %v", err)
	}
}

func TestStoreOnlyPlanProbesStoreAndStaysInUserProcess(t *testing.T) {
	c, r := sampleCommand(t)
	c.apps = []catalog.App{{ID: "9NT1R1C2HH7J", Title: "ChatGPT", Source: "msstore", Scope: "user"}}
	c.selection.Apps = []ui.Choice{{ID: "9NT1R1C2HH7J"}}
	c.checkNetwork = func(context.Context) error { t.Fatal("community CDN probed for Store-only plan"); return nil }
	probes := 0
	c.checkStoreNetwork = func(context.Context) error { probes++; return nil }
	c.startAdmin = func(context.Context, []catalog.App) (adminSession, error) {
		t.Fatal("Store app requested UAC worker")
		return nil, nil
	}
	c.Run()
	if probes != 1 || len(r.installs) != 1 {
		t.Fatalf("probes=%d installs=%v result=%+v", probes, r.installs, c.result)
	}
	if c.result.(ui.ResultsMsg).Results[0].Status != "succeeded" {
		t.Fatalf("result=%+v", c.result)
	}
}
