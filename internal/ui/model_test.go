package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func press(m Model, key tea.KeyPressMsg) (Model, tea.Cmd) {
	next, cmd := m.Update(key)
	return next.(Model), cmd
}

func TestSelectionSurvivesBackAndNext(t *testing.T) {
	apps := []Choice{{ID:"git", Title:"Git"}}
	settings := []Choice{{ID:"wallpaper", Title:"Wallpaper"}}
	m := NewModel(apps, settings, nil)
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeySpace}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeySpace}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyBackspace}))
	if got := m.Selected(); len(got.Apps) != 1 || got.Apps[0].ID != "git" { t.Fatalf("app selection lost: %#v", got.Apps) }
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	if got := m.Selected(); len(got.Settings) != 1 || got.Settings[0].ID != "wallpaper" { t.Fatalf("setting selection lost: %#v", got.Settings) }
}

func TestConfirmCallbackRequiresSummaryEnterAndRunsOnce(t *testing.T) {
	calls := 0
	m := NewModel([]Choice{{ID:"git", Title:"Git"}}, nil, func(Selection) tea.Cmd { calls++; return func() tea.Msg { return ResultsMsg{} } })
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeySpace}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	if calls != 0 { t.Fatal("callback ran before summary") }
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	if calls != 0 { t.Fatal("callback ran while navigating to summary") }
	m, cmd := press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyEnter}))
	if calls != 1 || cmd == nil { t.Fatalf("callback calls=%d, cmd=%v", calls, cmd) }
	_, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyEnter}))
	if calls != 1 { t.Fatalf("callback ran %d times", calls) }
}

func TestDemoConfirmationShowsResultsWithoutSystemCallback(t *testing.T) {
	m := NewModel([]Choice{{ID:"git", Title:"Git"}}, nil, nil)
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeySpace}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyEnter}))
	if m.screen != resultsScreen { t.Fatalf("screen=%v, want results", m.screen) }
	if len(m.results) != 1 || m.results[0].Status != "Demo" { t.Fatalf("unexpected results: %#v", m.results) }
}

func TestMouseTogglesVisibleChoiceAndEmptySectionCanContinue(t *testing.T) {
	m := NewModel([]Choice{{ID:"shared", Title:"App"}}, []Choice{{ID:"shared", Title:"Setting"}}, nil)
	next, _ := m.Update(tea.MouseClickMsg(tea.Mouse{X:4, Y:6, Button:tea.MouseLeft}))
	m = next.(Model)
	if got := m.Selected(); len(got.Apps) != 1 { t.Fatalf("mouse did not select app: %#v", got) }
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	if m.screen != summaryScreen { t.Fatalf("screen=%v, want summary", m.screen) }
	if got := m.Selected(); len(got.Apps) != 1 || len(got.Settings) != 0 { t.Fatalf("same IDs collided across screens: %#v", got) }

	empty := NewModel(nil, nil, nil)
	empty, _ = press(empty, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	next, _ = empty.Update(tea.MouseClickMsg(tea.Mouse{X:70, Y:9, Button:tea.MouseLeft}))
	if next.(Model).screen != summaryScreen { t.Fatalf("empty settings continue click did not open summary") }
}

func TestSummaryMouseConfirmLineMatchesRenderedLayout(t *testing.T) {
	m := NewModel([]Choice{{ID:"a", Title:"A"}}, nil, nil)
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code:tea.KeyTab}))
	if got := m.summaryActionLine(); got != 9 { t.Fatalf("demo summary button line=%d, want 9", got) }
	m.selected[selectionKey(appsScreen, "a")] = true
	if got := m.summaryActionLine(); got != 9 { t.Fatalf("selected demo summary button line=%d, want 9", got) }
}

func TestResultsWrapWinGetAgreementErrorAndShowInteractiveHint(t *testing.T) {
	m := NewModel(nil, nil, func(Selection) tea.Cmd { return nil })
	m.screen = resultsScreen
	m.width = 56
	m.results = []Result{{Title: "Google Chrome", Status: "failed", Message: "check: WinGet list Google.Chrome: exit status 0x8a150046: The msstore source requires that you view the following agreements before using the source."}}
	view := m.View().Content
	if !strings.Contains(view, "following agreements before using the") || !strings.Contains(view, "source.") {
		t.Fatalf("long result message was truncated: %s", view)
	}
	if !strings.Contains(view, "winget list --id Google.Chrome --exact") {
		t.Fatalf("interactive agreement hint missing: %s", view)
	}
}

func TestSummaryIncludesAdminWarningWhenMachineAppSelected(t *testing.T) {
	m := NewModel([]Choice{{ID: "wez.wezterm", Title: "WezTerm", RequiresAdmin: true}}, nil, nil)
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code: tea.KeySpace}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	m, _ = press(m, tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if m.screen != summaryScreen {
		t.Fatalf("screen = %v, want summary", m.screen)
	}
	view := m.View().Content
	if !strings.Contains(view, "Yönetici") && !strings.Contains(view, "UAC") {
		t.Fatalf("summary view missing UAC warning: %s", view)
	}
	if got := m.summaryActionLine(); got != 11 {
		t.Fatalf("summaryActionLine = %d, want 11", got)
	}
}

func TestSummaryPreservesLongPreflightErrorAndMouseRetry(t *testing.T) {
	calls := 0
	m := NewModel(nil, nil, func(Selection) tea.Cmd {
		calls++
		return func() tea.Msg { return ResultsMsg{} }
	})
	m.screen = summaryScreen
	m.width = 48
	m.message = "Ön kontrol tamamlanamadı; hiçbir değişiklik başlatılmadı. App Installer kurulumunu kontrol edin ve PowerShell'de winget --version çalıştırın."
	lines := strings.Split(m.View().Content, "\n")
	if !strings.Contains(strings.Join(lines, " "), "winget --version çalıştırın.") {
		t.Fatalf("error guidance truncated: %v", lines)
	}
	buttonLine := -1
	for i, line := range lines {
		if strings.Contains(line, "[Enter ile onayla ve başlat]") {
			buttonLine = i
			break
		}
	}
	if buttonLine < 0 || m.summaryActionLine() != buttonLine {
		t.Fatalf("mouse line=%d rendered line=%d", m.summaryActionLine(), buttonLine)
	}
	_, cmd := m.Update(tea.MouseClickMsg(tea.Mouse{X: 40, Y: buttonLine, Button: tea.MouseLeft}))
	if calls != 1 || cmd == nil {
		t.Fatalf("retry calls=%d command=%v", calls, cmd)
	}
}
