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
	var chromeFound bool
	for _, app := range apps {
		if app.ID == "wez.wezterm" {
			t.Fatal("starter catalog must defer WezTerm until the explicit admin phase")
		}
		if app.Title == "Google Chrome" {
			chromeFound = app.ID == "Google.Chrome.EXE"
		}
	}
	if !chromeFound {
		t.Fatal("starter Chrome entry must use user-scope Google.Chrome.EXE")
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
