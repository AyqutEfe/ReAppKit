package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/ui"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
)

func TestSelectedJobsOnlyBuildsChosenWork(t *testing.T) {
	source := filepath.Join(t.TempDir(), "profile.ps1")
	if err := os.WriteFile(source, []byte("prompt"), 0600); err != nil { t.Fatal(err) }
	target := filepath.Join(t.TempDir(), "profile.ps1")
	apps := []catalog.App{{ID: "Git.Git", Title: "Git"}, {ID: "Google.Chrome", Title: "Chrome"}}
	settings := []catalog.Setting{{ID: "powershell-profile", Title: "PowerShell profile", Source: source, Target: target, RequiresApp: "Git.Git"}}
	selection := ui.Selection{
		Apps: []ui.Choice{{ID: "Git.Git"}},
		Settings: []ui.Choice{{ID: "setting:powershell-profile"}},
	}
	jobs, err := selectedJobs(selection, apps, settings, winget.New(nil))
	if err != nil { t.Fatal(err) }
	if len(jobs) != 2 { t.Fatalf("got %d jobs, want 2", len(jobs)) }
	if jobs[0].ID != "app:Git.Git" || jobs[1].ID != "setting:powershell-profile" {
		t.Fatalf("unexpected jobs: %s, %s", jobs[0].ID, jobs[1].ID)
	}
	if len(jobs[1].DependsOn) != 1 || jobs[1].DependsOn[0] != jobs[0].ID {
		t.Fatalf("setting dependency = %v", jobs[1].DependsOn)
	}
}

func TestSelectedJobsRejectsUnknownChoice(t *testing.T) {
	_, err := selectedJobs(ui.Selection{Apps: []ui.Choice{{ID: "Not.InCatalog"}}}, nil, nil, winget.New(nil))
	if err == nil { t.Fatal("unknown app was accepted") }
}
