package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/core"
	"github.com/AyqutEfe/ReAppKit/internal/settings"
	"github.com/AyqutEfe/ReAppKit/internal/ui"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
)

func main() {
	var appCatalog, settingsCatalog, resultsDir string
	var apply bool
	var workerPlan string
	flag.StringVar(&workerPlan, "admin-worker", "", "internal elevated installation worker")
	flag.StringVar(&appCatalog, "apps-catalog", "", "optional JSON app catalog (default: built-in starter list)")
	flag.StringVar(&settingsCatalog, "settings-catalog", "", "optional JSON file settings catalog")
	flag.StringVar(&resultsDir, "results-dir", "", "directory for machine-readable run results")
	flag.BoolVar(&apply, "apply", false, "allow selected jobs to run after summary confirmation")
	flag.Parse()
	if workerPlan != "" {
		if err := runAdminWorker(workerPlan); err != nil {
			exitErr(err)
		}
		return
	}

	if apply && (runtime.GOOS != "windows" || runtime.GOARCH != "amd64") {
		fmt.Fprintln(os.Stderr, "ReAppKit uygulama modu şu anda yalnız Windows x64 için destekleniyor.")
		os.Exit(1)
	}
	apps, err := catalog.LoadApps(appCatalog)
	if err != nil {
		exitErr(err)
	}
	settingsList, err := catalog.LoadSettings(settingsCatalog)
	if err != nil {
		exitErr(err)
	}

	appChoices := make([]ui.Choice, 0, len(apps))
	for _, app := range apps {
		appChoices = append(appChoices, ui.Choice{
			ID:            app.ID,
			Title:         app.Title,
			Description:   app.Description,
			Category:      app.Category,
			RequiresAdmin: app.RequiresAdmin,
		})
	}
	settingChoices := make([]ui.Choice, 0, len(settingsList))
	for _, setting := range settingsList {
		settingChoices = append(settingChoices, ui.Choice{ID: "setting:" + setting.ID, Title: setting.Title, Description: setting.Description, Category: setting.Category})
	}

	var confirm ui.ConfirmFunc
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if apply {
		client := winget.New(nil)
		confirm = func(selection ui.Selection) tea.Cmd {
			command := &executionCommand{ctx: ctx, selection: selection, apps: apps, settings: settingsList, client: client, resultsDir: resultsDir}
			// Release terminal rendering while obtaining the initial UAC permission.
			return tea.Exec(command, func(err error) tea.Msg {
				if err != nil {
					return ui.ExecutionErrorMsg{Err: err}
				}
				return command.result
			})
		}
	}
	model := ui.NewModel(appChoices, settingChoices, confirm)
	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		exitErr(err)
	}
}

func executeSelection(ctx context.Context, selection ui.Selection, apps []catalog.App, settingsList []catalog.Setting, client *winget.Client, resultsDir string, onEvent func(core.Item), adminInstall ...func(context.Context, catalog.App) error) tea.Msg {
	jobs, err := selectedJobs(selection, apps, settingsList, client, adminInstall...)
	if err != nil {
		return ui.ExecutionErrorMsg{Err: err}
	}
	result := core.Execute(ctx, jobs, true, onEvent)
	rows := make([]ui.Result, 0, len(result.Items)+2)
	for _, item := range result.Items {
		rows = append(rows, ui.Result{ID: item.ID, Title: item.Title, Status: string(item.State), Message: item.Error})
	}
	if result.Error != "" {
		rows = append(rows, ui.Result{Title: "İşlem hatası", Status: "failed", Message: result.Error})
	}
	if len(jobs) == 0 {
		rows = append(rows, ui.Result{Title: "Seçim yok", Status: "skipped", Message: "Herhangi bir işlem seçilmedi."})
	}
	if len(jobs) != 0 {
		if _, err := writeResult(resultsDir, selection, result); err != nil {
			rows = append(rows, ui.Result{Title: "Sonuç kaydı", Status: "failed", Message: err.Error()})
		}
	}
	return ui.ResultsMsg{Results: rows}
}

func selectedJobs(selection ui.Selection, apps []catalog.App, settingsList []catalog.Setting, client *winget.Client, adminInstall ...func(context.Context, catalog.App) error) ([]core.Job, error) {
	appByID := make(map[string]catalog.App, len(apps))
	for _, app := range apps {
		appByID[app.ID] = app
	}
	settingByID := make(map[string]catalog.Setting, len(settingsList))
	for _, setting := range settingsList {
		settingByID["setting:"+setting.ID] = setting
	}
	selectedApps := make(map[string]bool, len(selection.Apps))
	jobs := make([]core.Job, 0, len(selection.Apps)+len(selection.Settings))
	// Stable phases: ordinary apps, admin apps, then dependent file settings.
	orderedChoices := make([]ui.Choice, 0, len(selection.Apps))
	for _, admin := range []bool{false, true} {
		for _, choice := range selection.Apps {
			app, found := appByID[choice.ID]
			if !found {
				return nil, fmt.Errorf("unknown selected app %q", choice.ID)
			}
			if app.RequiresAdmin == admin {
				orderedChoices = append(orderedChoices, choice)
			}
		}
	}
	for _, choice := range orderedChoices {
		app, found := appByID[choice.ID]
		if !found {
			return nil, fmt.Errorf("unknown selected app %q", choice.ID)
		}
		if selectedApps[app.ID] {
			return nil, fmt.Errorf("duplicate selected app %q", app.ID)
		}
		selectedApps[app.ID] = true
		job := client.Job(app.ID, app.Title, app.Scope)
		if app.RequiresAdmin {
			job.Apply = func(ctx context.Context) error {
				if err := ctx.Err(); err != nil {
					return err
				}
				if len(adminInstall) == 0 || adminInstall[0] == nil {
					return errors.New("yönetici kurulum süreci başlangıçta yetkilendirilmedi")
				}
				return adminInstall[0](ctx, app)
			}
		}
		jobs = append(jobs, job)
	}
	for _, choice := range selection.Settings {
		setting, found := settingByID[choice.ID]
		if !found {
			return nil, fmt.Errorf("unknown selected setting %q", choice.ID)
		}
		var dependencies []string
		if setting.RequiresApp != "" && selectedApps[setting.RequiresApp] {
			dependencies = []string{"app:" + setting.RequiresApp}
		}
		job, err := settings.FileJob("setting:"+setting.ID, setting.Title, setting.Source, setting.Target, dependencies)
		if err != nil {
			return nil, fmt.Errorf("setting %s: %w", setting.ID, err)
		}
		if setting.RequiresApp != "" && !selectedApps[setting.RequiresApp] {
			check := job.Check
			requiredID := setting.RequiresApp
			job.Check = func(ctx context.Context) (bool, error) {
				installed, err := client.Installed(ctx, requiredID)
				if err != nil {
					return false, err
				}
				if !installed {
					return false, fmt.Errorf("requires installed application %s", requiredID)
				}
				return check(ctx)
			}
		}
		jobs = append(jobs, job)
	}
	if _, err := core.Validate(jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func writeResult(dir string, selection ui.Selection, result core.Result) (string, error) {
	if dir == "" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("LOCALAPPDATA is not available; pass --results-dir")
		}
		dir = filepath.Join(base, "ReAppKit", "runs")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	selected := make([]string, 0, len(selection.Apps)+len(selection.Settings))
	for _, app := range selection.Apps {
		selected = append(selected, app.ID)
	}
	for _, setting := range selection.Settings {
		selected = append(selected, setting.ID)
	}
	payload := struct {
		At       time.Time   `json:"at"`
		Selected []string    `json:"selected"`
		Result   core.Result `json:"result"`
	}{At: time.Now().UTC(), Selected: selected, Result: result}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	name := "run-" + time.Now().UTC().Format("20060102-150405.000000000") + ".json"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		return "", err
	}
	return path, nil
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "ReAppKit:", strings.TrimSpace(err.Error()))
	os.Exit(1)
}
