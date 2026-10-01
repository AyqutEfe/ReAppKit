package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/ui"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
)

func TestSelectedJobsOnlyBuildsChosenWork(t *testing.T) {
	source := filepath.Join(t.TempDir(), "profile.ps1")
	if err := os.WriteFile(source, []byte("prompt"), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "profile.ps1")
	apps := []catalog.App{{ID: "Git.Git", Title: "Git", Scope: "user"}, {ID: "wez.wezterm", Title: "WezTerm", Scope: "machine", RequiresAdmin: true}}
	settings := []catalog.Setting{{ID: "powershell-profile", Title: "PowerShell profile", Source: source, Target: target, RequiresApp: "Git.Git"}}
	selection := ui.Selection{
		Apps:     []ui.Choice{{ID: "Git.Git"}, {ID: "wez.wezterm"}},
		Settings: []ui.Choice{{ID: "setting:powershell-profile"}},
	}
	jobs, err := selectedJobs(selection, apps, settings, winget.New(nil))
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 3 {
		t.Fatalf("got %d jobs, want 3", len(jobs))
	}
	if jobs[0].ID != "app:Git.Git" || jobs[1].ID != "app:wez.wezterm" || jobs[2].ID != "setting:powershell-profile" {
		t.Fatalf("unexpected jobs: %s, %s, %s", jobs[0].ID, jobs[1].ID, jobs[2].ID)
	}
	if len(jobs[2].DependsOn) != 1 || jobs[2].DependsOn[0] != jobs[0].ID {
		t.Fatalf("setting dependency = %v", jobs[2].DependsOn)
	}
}

type sourceRecordingRunner struct{ calls [][]string }

func (r *sourceRecordingRunner) Run(_ context.Context, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string(nil), args...))
	if args[0] == "--version" {
		return []byte("v1.9"), nil
	}
	if args[0] == "list" {
		return []byte(fmt.Sprintf("Name %s 1 %s\n", args[2], args[5])), nil
	}
	return nil, nil
}

func TestSelectedJobsRouteEachSourceAndSettingPrerequisite(t *testing.T) {
	r := &sourceRecordingRunner{}
	apps := []catalog.App{{ID: "9NT1R1C2HH7J", Title: "ChatGPT", Scope: "user", Source: "msstore"}, {ID: "Brave.Brave", Title: "Brave", Scope: "user", Source: "winget"}}
	jobs, err := selectedJobs(ui.Selection{Apps: []ui.Choice{{ID: apps[0].ID}, {ID: apps[1].ID}}}, apps, nil, winget.New(r))
	if err != nil {
		t.Fatal(err)
	}
	for i, job := range jobs {
		before := len(r.calls)
		if _, err := job.Check(context.Background()); err != nil {
			t.Fatal(err)
		}
		if err := job.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
		if _, err := job.Verify(context.Background()); err != nil {
			t.Fatal(err)
		}
		for _, call := range r.calls[before:] {
			if call[0] != "--version" && call[5] != apps[i].Source {
				t.Fatalf("wrong source for %s: %v", apps[i].ID, call)
			}
		}
	}
	source, target := filepath.Join(t.TempDir(), "source"), filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(source, []byte("profile"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := []catalog.Setting{{ID: "profile", Title: "Profile", Source: source, Target: target, RequiresApp: apps[0].ID}}
	r.calls = nil
	jobs, err = selectedJobs(ui.Selection{Settings: []ui.Choice{{ID: "setting:profile"}}}, apps, settings, winget.New(r))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jobs[0].Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 2 || r.calls[1][5] != "msstore" {
		t.Fatalf("prerequisite source=%v", r.calls)
	}
}

func TestSelectedJobsRejectsUnknownChoice(t *testing.T) {
	_, err := selectedJobs(ui.Selection{Apps: []ui.Choice{{ID: "Not.InCatalog"}}}, nil, nil, winget.New(nil))
	if err == nil {
		t.Fatal("unknown app was accepted")
	}
}
