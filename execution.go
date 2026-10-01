package main

import (
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"fmt"
	"github.com/AyqutEfe/ReAppKit/catalog"
	"github.com/AyqutEfe/ReAppKit/internal/core"
	"github.com/AyqutEfe/ReAppKit/internal/ui"
	"github.com/AyqutEfe/ReAppKit/internal/winget"
	"io"
)

type adminSession interface {
	Install(context.Context, catalog.App) error
	Close() error
}
type executionCommand struct {
	ctx        context.Context
	selection  ui.Selection
	apps       []catalog.App
	settings   []catalog.Setting
	client     *winget.Client
	resultsDir string
	stdout     io.Writer
	result     tea.Msg
	startAdmin func(context.Context, []catalog.App) (adminSession, error)
	checkNetwork func(context.Context) error
}

var _ tea.ExecCommand = (*executionCommand)(nil)

func (c *executionCommand) SetStdin(io.Reader)    {}
func (c *executionCommand) SetStdout(w io.Writer) { c.stdout = w }
func (c *executionCommand) SetStderr(io.Writer)   {}

func (c *executionCommand) Run() error {
	if c.stdout == nil {
		c.stdout = io.Discard
	}
	// Validate all jobs before UAC or changes.
	if _, err := selectedJobs(c.selection, c.apps, c.settings, c.client); err != nil {
		c.result = ui.ExecutionErrorMsg{Err: err}
		return nil
	}
	pending, err := c.preflight()
	if err != nil {
		c.result = ui.ExecutionErrorMsg{Err: fmt.Errorf("%w: %w", errPreflightStopped, err)}
		return nil
	}
	var admin adminSession
	if len(pending) > 0 {
		fmt.Fprintln(c.stdout, "\nWindows yönetici onayını bir kez verin; ardından kurulumlar sırayla devam edecek.")
		for _, app := range pending {
			fmt.Fprintf(c.stdout, "  • %s\n", app.Title)
		}
		start := c.startAdmin
		if start == nil {
			start = startAdminSession
		}
		var err error
		admin, err = start(c.ctx, pending)
		if err != nil {
			c.result = ui.ExecutionErrorMsg{Err: fmt.Errorf("başlangıç yönetici onayı tamamlanamadı; hiçbir iş başlatılmadı: %w", err)}
			return nil
		}
		defer admin.Close()
	}
	fmt.Fprintln(c.stdout, "\nReAppKit · Kurulum\nUygulamalar ve seçili ayarlar sırayla uygulanıyor…")
	c.result = executeSelection(c.ctx, c.selection, c.apps, c.settings, c.client, c.resultsDir, func(item core.Item) {
		fmt.Fprintf(c.stdout, "[%s] %s\n", item.State, item.Title)
		if item.Error != "" {
			fmt.Fprintln(c.stdout, item.Error)
		}
	}, func(ctx context.Context, app catalog.App) error {
		if admin == nil {
			return errors.New("bu iş başlangıç yönetici onayına dahil değil; r ile yeniden dene")
		}
		return admin.Install(ctx, app)
	})
	return nil
}
