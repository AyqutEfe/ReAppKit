package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStarterCatalogHasUniqueExactIDs(t *testing.T) {
	apps, err := LoadApps("")
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) == 0 {
		t.Fatal("starter app catalog is empty")
	}
	var chromeFound, weztermFound bool
	for _, app := range apps {
		if app.ID == "wez.wezterm" {
			weztermFound = true
			if app.Scope != "auto" || !app.RequiresAdmin {
				t.Fatalf("WezTerm must specify auto scope and requires_admin, got scope=%q requires_admin=%v", app.Scope, app.RequiresAdmin)
			}
		}
		if app.Title == "Google Chrome" {
			chromeFound = app.ID == "Google.Chrome.EXE"
		}
	}
	if !chromeFound {
		t.Fatal("starter Chrome entry must use user-scope Google.Chrome.EXE")
	}
	if !weztermFound {
		t.Fatal("starter catalog must contain wez.wezterm")
	}
	for _, app := range apps {
		if invalidID(app.ID) {
			t.Fatalf("invalid WinGet ID %q", app.ID)
		}
	}
}

func TestCustomCatalogRejectsDuplicateIDs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apps.json")
	data := []byte(`[ {"id":"Git.Git","title":"Git"}, {"id":"git.git","title":"Duplicate"} ]`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadApps(path); err == nil {
		t.Fatal("duplicate package ID accepted")
	}
}

func TestCatalogScopeDefaultsAndAdminWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "apps.json")
	data := []byte(`[ {"id":"Git.Git","title":"Git"}, {"id":"wez.wezterm","title":"WezTerm","scope":"auto"}, {"id":"Test.Machine","title":"Machine","scope":"machine"} ]`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	apps, err := LoadApps(path)
	if err != nil {
		t.Fatal(err)
	}
	if apps[0].Scope != "user" || apps[0].RequiresAdmin {
		t.Fatalf("default=%+v", apps[0])
	}
	if !apps[1].RequiresAdmin || !apps[2].RequiresAdmin {
		t.Fatalf("admin warning missing: %+v", apps)
	}
	if err := os.WriteFile(path, []byte(`[ {"id":"Git.Git","title":"Git","scope":"invalid"} ]`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadApps(path); err == nil {
		t.Fatal("invalid scope accepted")
	}
}

func TestGitUsesMachineScopeInElevatedWorker(t *testing.T) {
	apps, err := LoadApps("")
	if err != nil {
		t.Fatal(err)
	}
	for _, app := range apps {
		if app.ID == "Git.Git" {
			if !app.RequiresAdmin || app.Scope != "machine" {
				t.Fatalf("Git=%+v", app)
			}
			return
		}
	}
	t.Fatal("Git missing")
}
