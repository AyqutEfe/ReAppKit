package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Choice is a display-only option. It contains no executable action.
type Choice struct {
	ID          string
	Title       string
	Description string
	Category    string
}

// Selection is the user's chosen input for the two selection screens.
type Selection struct {
	Apps     []Choice
	Settings []Choice
}

// Result is one row shown on the results screen.
type Result struct {
	ID      string
	Title   string
	Status  string
	Message string
}

// ConfirmFunc is called only when the user presses Enter on the summary screen.
// It should return a Bubble Tea command that performs the work and eventually
// returns either ResultsMsg or ExecutionErrorMsg.
type ConfirmFunc func(Selection) tea.Cmd

// ResultsMsg carries the final rows to display after the confirmed operation.
type ResultsMsg struct{ Results []Result }

// ProgressMsg reports the latest completed job while execution is running.
type ProgressMsg struct{ Result Result }

// ExecutionErrorMsg reports an execution-level error while keeping the UI alive.
type ExecutionErrorMsg struct{ Err error }

type screen uint8

const (
	appsScreen screen = iota
	settingsScreen
	summaryScreen
	resultsScreen
)

// Model owns the UI state. It never performs system changes itself.
type Model struct {
	apps       []Choice
	settings   []Choice
	selected   map[string]bool
	confirm    ConfirmFunc
	screen     screen
	cursor     int
	width      int
	busy       bool
	results    []Result
	message    string
}

// NewModel creates the UI with caller-provided catalog data. Passing a nil
// ConfirmFunc keeps the flow in demo mode; Enter on the summary shows that no
// system changes were made.
func NewModel(apps, settings []Choice, confirm ConfirmFunc) Model {
	return Model{
		apps:     cloneChoices(apps),
		settings: cloneChoices(settings),
		selected: make(map[string]bool),
		confirm:  confirm,
		screen:   appsScreen,
		width:    80,
	}
}

// DemoChoices returns visibly labeled sample choices for UI-only runs.
func DemoChoices() (apps, settings []Choice) {
	return []Choice{
		{ID: "demo-vscode", Title: "Visual Studio Code", Description: "Örnek uygulama; hiçbir şey kurulmaz.", Category: "Geliştirme"},
		{ID: "demo-git", Title: "Git", Description: "Örnek uygulama; hiçbir şey kurulmaz.", Category: "Geliştirme"},
		{ID: "demo-wezterm", Title: "WezTerm", Description: "Örnek uygulama; hiçbir şey kurulmaz.", Category: "Terminal"},
	}, []Choice{
		{ID: "demo-wallpaper", Title: "Duvar kâğıdı", Description: "Örnek ayar; kullanıcı dosyası bekleniyor.", Category: "Windows"},
		{ID: "demo-terminal-profile", Title: "Terminal profili", Description: "Örnek ayar; hiçbir dosya değiştirilmez.", Category: "Uygulama"},
	}
}

func cloneChoices(src []Choice) []Choice { return append([]Choice(nil), src...) }

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		if m.width < 40 {
			m.width = 40
		}
	case tea.MouseClickMsg:
		return m.updateMouse(msg)
	case ResultsMsg:
		m.results = append([]Result(nil), msg.Results...)
		m.screen = resultsScreen
		m.busy = false
		m.message = ""
	case ProgressMsg:
		if m.busy {
			m.message = fmt.Sprintf("%s: %s", msg.Result.Title, msg.Result.Status)
		}
	case ExecutionErrorMsg:
		m.busy = false
		if msg.Err != nil {
			m.message = "İşlem hatası: " + msg.Err.Error()
		}
	case tea.KeyPressMsg:
		return m.updateKey(msg.String())
	}
	return m, nil
}

func (m Model) updateKey(key string) (tea.Model, tea.Cmd) {
	if key == "ctrl+c" || key == "q" {
		return m, tea.Quit
	}
	if m.busy { return m, nil }
	switch m.screen {
	case appsScreen, settingsScreen:
		items := m.currentChoices()
		switch key {
		case "up", "k":
			if m.cursor > 0 { m.cursor-- }
		case "down", "j":
			if m.cursor < len(items)-1 { m.cursor++ }
		case " ", "space", "enter":
			m.toggleCursor()
		case "right", "tab", "n":
			m.nextScreen()
		case "left", "backspace", "b":
			if m.screen == settingsScreen { m.screen = appsScreen; m.cursor = clampCursor(m.cursor, len(m.apps)) }
		}
	case summaryScreen:
		switch key {
		case "left", "backspace", "b":
			m.screen = settingsScreen
			m.cursor = clampCursor(m.cursor, len(m.settings))
		case "enter", " ", "space":
			return m.confirmSelection()
		}
	case resultsScreen:
		if key == "r" && m.confirm != nil {
			m.screen = summaryScreen
			m.message = "Yeniden onaylarsan tamamlanmış işler durum kontrolüyle atlanır."
			return m, nil
		}
		if key == "enter" || key == "esc" { return m, tea.Quit }
	}
	return m, nil
}

func (m Model) confirmSelection() (tea.Model, tea.Cmd) {
	if m.busy { return m, nil }
	m.busy = true
	m.message = ""
	if m.confirm == nil {
		m.results = demoResults(m.selection())
		m.screen = resultsScreen
		m.busy = false
		return m, nil
	}
	cmd := m.confirm(m.selection())
	if cmd == nil {
		m.busy = false
		m.message = "İşlem başlatılamadı. Tekrar denemek için Enter'a basın."
	}
	return m, cmd
}

func demoResults(sel Selection) []Result {
	rows := make([]Result, 0, len(sel.Apps)+len(sel.Settings))
	for _, c := range sel.Apps { rows = append(rows, Result{ID:c.ID, Title:c.Title, Status:"Demo", Message:"Bu örnek akışta kurulmadı."}) }
	for _, c := range sel.Settings { rows = append(rows, Result{ID:c.ID, Title:c.Title, Status:"Demo", Message:"Bu örnek akışta değiştirilmedi."}) }
	if len(rows) == 0 { rows = append(rows, Result{Title:"Seçim yok", Status:"Bilgi", Message:"Herhangi bir işlem seçilmedi."}) }
	return rows
}

func (m Model) updateMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.busy { return m, nil }
	if msg.Button != tea.MouseLeft { return m, nil }
	y := msg.Y
	switch m.screen {
	case appsScreen, settingsScreen:
		items := m.currentChoices()
		const firstChoiceLine = 6
		if y >= firstChoiceLine && y < firstChoiceLine+len(items) {
			m.cursor = y - firstChoiceLine
			m.toggleCursor()
			return m, nil
		}
		displayedRows := len(items)
		if displayedRows == 0 {
			displayedRows = 1 // the empty-state message occupies a row
		}
		navLine := firstChoiceLine + displayedRows + 2
		if y == navLine {
			if msg.X < m.width/2 && m.screen == settingsScreen { m.screen = appsScreen; m.cursor = clampCursor(m.cursor, len(m.apps)); return m, nil }
			m.nextScreen()
		}
	case summaryScreen:
		navLine := m.summaryActionLine()
		if y == navLine {
			if msg.X < m.width/2 { m.screen = settingsScreen; m.cursor = clampCursor(m.cursor, len(m.settings)); return m, nil }
			return m.confirmSelection()
		}
	case resultsScreen:
		if y == 4 { return m, tea.Quit }
	}
	return m, nil
}

func (m *Model) nextScreen() {
	if m.screen == appsScreen { m.screen = settingsScreen; m.cursor = clampCursor(m.cursor, len(m.settings)); return }
	if m.screen == settingsScreen { m.screen = summaryScreen; m.cursor = 0 }
}

func (m Model) currentChoices() []Choice {
	if m.screen == settingsScreen { return m.settings }
	return m.apps
}

func (m *Model) toggleCursor() {
	items := m.currentChoices()
	if m.cursor < 0 || m.cursor >= len(items) { return }
	id := selectionKey(m.screen, items[m.cursor].ID)
	if id == "" { return }
	m.selected[id] = !m.selected[id]
}

func (m Model) selection() Selection {
	var s Selection
	for _, c := range m.apps { if m.selected[selectionKey(appsScreen, c.ID)] { s.Apps = append(s.Apps, c) } }
	for _, c := range m.settings { if m.selected[selectionKey(settingsScreen, c.ID)] { s.Settings = append(s.Settings, c) } }
	return s
}

func selectionKey(which screen, id string) string {
	if which == settingsScreen { return "setting:" + id }
	return "app:" + id
}

func (m Model) summaryActionLine() int {
	sel := m.selection()
	selected := len(sel.Apps) + len(sel.Settings)
	line := 7 + selected
	if selected == 0 { line++ } // empty-selection explanation
	if m.confirm == nil { line++ } // demo-mode explanation
	if m.busy { line += 2 }
	if m.message != "" { line += 2 }
	return line
}

// Selected returns a snapshot of the current selection.
func (m Model) Selected() Selection { return m.selection() }

func clampCursor(cursor, length int) int {
	if length == 0 { return 0 }
	if cursor >= length { return length-1 }
	if cursor < 0 { return 0 }
	return cursor
}

// View implements tea.Model.
func (m Model) View() tea.View {
	var b strings.Builder
	switch m.screen {
	case appsScreen:
		m.writeChoices(&b, "Uygulamalar", "1/2 · Kurmak istediklerini seç", m.apps, "Uygulama listesinden seçim yap. Space/Enter seçer; Tab veya → sonraki ekrana geçer.", false)
	case settingsScreen:
		m.writeChoices(&b, "Ayarlar", "2/2 · Uygulamak istediklerini seç", m.settings, "Ayarlar yalnızca açıkça seçilip özet onaylandığında çalışır.", true)
	case summaryScreen:
		m.writeSummary(&b)
	case resultsScreen:
		m.writeResults(&b)
	}
	v := tea.NewView(b.String())
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m Model) writeChoices(b *strings.Builder, title, step string, items []Choice, help string, canBack bool) {
	fmt.Fprintf(b, "ReAppKit · %s\n\n%s\n\n%s\n\n", title, step, help)
	if len(items) == 0 { b.WriteString("  Bu bölümde örnek bulunmuyor.\n") }
	for i, c := range items {
		cursor := "  "
		if i == m.cursor { cursor = "› " }
		check := "[ ]"
		which := appsScreen
		if m.screen == settingsScreen { which = settingsScreen }
		if m.selected[selectionKey(which, c.ID)] { check = "[x]" }
		label := c.Title
		if c.Category != "" { label += "  ·  " + c.Category }
		fmt.Fprintf(b, "%s%s %s\n", cursor, check, fit(label, m.width-10))
	}
	fmt.Fprintf(b, "\nSeçili: %d\n", len(m.currentSelection()))
	if canBack { b.WriteString("[← Geri]   ") }
	if m.screen == appsScreen { b.WriteString("[Devam →]") } else { b.WriteString("[Özeti göster →]") }
	b.WriteString("\n\n↑/↓ gezin · Space seç · q çık\n")
}

func (m Model) writeSummary(b *strings.Builder) {
	sel := m.selection()
	fmt.Fprintf(b, "ReAppKit · Özet\n\nSeçilen işlemler: %d uygulama, %d ayar\n\n", len(sel.Apps), len(sel.Settings))
	if len(sel.Apps) == 0 && len(sel.Settings) == 0 { b.WriteString("  Hiçbir işlem seçilmedi.\n") }
	for _, c := range sel.Apps { fmt.Fprintf(b, "  • Uygulama: %s\n", fit(c.Title, m.width-16)) }
	for _, c := range sel.Settings { fmt.Fprintf(b, "  • Ayar: %s\n", fit(c.Title, m.width-16)) }
	b.WriteString("\nEnter ile açıkça onaylayınca işlem başlatılır.\n")
	if m.confirm == nil { b.WriteString("Demo modu: sistemde değişiklik yapılmaz.\n") }
	if m.busy { b.WriteString("\nİşlem sürüyor…\n") }
	if m.message != "" { fmt.Fprintf(b, "\n%s\n", fit(m.message, m.width-1)) }
	b.WriteString("\n[← Geri]   [Enter ile onayla ve başlat]\n\n←/Backspace geri · Enter onay · q çık\n")
}

func (m Model) writeResults(b *strings.Builder) {
	b.WriteString("ReAppKit · Sonuçlar\n\n")
	showSourceHint := false
	for _, r := range m.results {
		title := r.Title
		if title == "" { title = r.ID }
		fmt.Fprintf(b, "  [%s] %s\n", r.Status, fit(title, m.width-14))
		if r.Message != "" {
			for _, line := range wrap(r.Message, m.width-8) {
				fmt.Fprintf(b, "      %s\n", line)
			}
			lower := strings.ToLower(r.Message)
			if strings.Contains(lower, "msstore") && (strings.Contains(lower, "8a150046") || strings.Contains(lower, "agreement") || strings.Contains(lower, "view the following")) {
				showSourceHint = true
			}
		}
	}
	if len(m.results) == 0 { b.WriteString("  İşlem sonucu bulunmuyor.\n") }
	if showSourceHint {
		for i, line := range wrap("Microsoft Store kaynak koşullarını incelemek için PowerShell'de `winget list --id Google.Chrome --exact` çalıştırıp etkileşimli istemi yanıtla; ardından r ile tekrar dene.", m.width-1) {
			if i == 0 { b.WriteString("\n") }
			b.WriteString(line + "\n")
		}
	}
	if m.confirm == nil { b.WriteString("\nDemo modu: bu oturum sistemde değişiklik yapmadı.\n") }
	if m.confirm != nil { b.WriteString("\nr: yeniden denemek için özete dön.\n") }
	b.WriteString("\nEnter veya q ile çık.\n")
}

func (m Model) currentSelection() []Choice {
	if m.screen == settingsScreen { return m.selection().Settings }
	return m.selection().Apps
}

func fit(s string, width int) string {
	if width < 4 { return s }
	if len([]rune(s)) <= width { return s }
	r := []rune(s)
	return string(r[:width-1]) + "…"
}

func wrap(s string, width int) []string {
	if width < 4 { return []string{s} }
	var lines []string
	for _, paragraph := range strings.Split(s, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 { lines = append(lines, ""); continue }
		line := words[0]
		for _, word := range words[1:] {
			if len([]rune(line))+1+len([]rune(word)) > width {
				lines = append(lines, line)
				line = word
			} else {
				line += " " + word
			}
		}
		lines = append(lines, line)
	}
	return lines
}
