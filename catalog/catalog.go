package catalog

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

//go:embed apps.json
var starterApps []byte

// App is one exact WinGet package offered on the first selection screen.
type App struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Category      string `json:"category"`
	Scope         string `json:"scope,omitempty"`
	RequiresAdmin bool   `json:"requires_admin,omitempty"`
}

// Setting describes a user-supplied file configuration. ReAppKit does not
// provide source or target paths on behalf of the user.
type Setting struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	RequiresApp string `json:"requires_app,omitempty"`
}

func LoadApps(path string) ([]App, error) {
	data := starterApps
	if path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read app catalog: %w", err)
		}
	}
	var apps []App
	if err := decode(data, &apps); err != nil {
		return nil, fmt.Errorf("parse app catalog: %w", err)
	}
	seen := make(map[string]bool, len(apps))
	for i := range apps {
		app := &apps[i]
		if invalidID(app.ID) || strings.TrimSpace(app.Title) == "" {
			return nil, fmt.Errorf("app %d needs a valid exact package ID and title", i+1)
		}
		if app.Scope == "" {
			app.Scope = "user"
		}
		if app.Scope != "user" && app.Scope != "machine" && app.Scope != "auto" {
			return nil, fmt.Errorf("app %d has invalid scope %q (must be user, machine or auto)", i+1, app.Scope)
		}
		// Automatic scope may select an admin installer; require the UI warning.
		if app.Scope == "machine" || app.Scope == "auto" {
			app.RequiresAdmin = true
		}
		if seen[strings.ToLower(app.ID)] {
			return nil, fmt.Errorf("duplicate app ID %q", app.ID)
		}
		seen[strings.ToLower(app.ID)] = true
	}
	return apps, nil
}

func LoadSettings(path string) ([]Setting, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read settings catalog: %w", err)
	}
	var settings []Setting
	if err := decode(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings catalog: %w", err)
	}
	seen := make(map[string]bool, len(settings))
	for i, setting := range settings {
		if invalidID(setting.ID) || strings.TrimSpace(setting.Title) == "" || strings.TrimSpace(setting.Source) == "" || strings.TrimSpace(setting.Target) == "" {
			return nil, fmt.Errorf("setting %d needs an ID, title, source and target", i+1)
		}
		if setting.RequiresApp != "" && invalidID(setting.RequiresApp) {
			return nil, fmt.Errorf("setting %q has an invalid requires_app ID", setting.ID)
		}
		if seen[strings.ToLower(setting.ID)] {
			return nil, fmt.Errorf("duplicate setting ID %q", setting.ID)
		}
		seen[strings.ToLower(setting.ID)] = true
	}
	return settings, nil
}

func decode(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("unexpected content after JSON array")
	}
	return nil
}

var validID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func invalidID(id string) bool { return !validID.MatchString(id) }
