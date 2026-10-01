package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AyqutEfe/ReAppKit/catalog"
)

// This only tests HTTPS connectivity to the default WinGet CDN. Vendor
// downloads and source agreements are still checked by WinGet itself.
const wingetConnectivityURL = "https://cdn.winget.microsoft.com/"

func checkWinGetConnection(ctx context.Context) error {
	return probeConnection(ctx, &http.Client{Timeout: 10 * time.Second}, wingetConnectivityURL)
}

func probeConnection(ctx context.Context, client *http.Client, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("WinGet sunucusuna HTTPS bağlantısı kurulamadı; internet, proxy ve güvenlik duvarını kontrol edip yeniden deneyin: %w", err)
	}
	defer resp.Body.Close()
	// A CDN root can return 403/404 even though DNS, TLS and HTTP work.
	if resp.StatusCode == http.StatusProxyAuthRequired || resp.StatusCode >= 500 {
		return fmt.Errorf("WinGet bağlantı kontrolü HTTP %d döndürdü; proxy veya sunucu erişimini kontrol edip yeniden deneyin", resp.StatusCode)
	}
	return nil
}

func (c *executionCommand) preflight() ([]catalog.App, error) {
	fmt.Fprintln(c.stdout, "\nReAppKit · Ön kontrol")
	byID := make(map[string]catalog.App, len(c.apps))
	for _, app := range c.apps {
		byID[app.ID] = app
	}
	needsWinGet := len(c.selection.Apps) > 0
	for _, choice := range c.selection.Settings {
		for _, setting := range c.settings {
			if choice.ID == "setting:"+setting.ID && setting.RequiresApp != "" {
				needsWinGet = true
			}
		}
	}
	if !needsWinGet {
		fmt.Fprintln(c.stdout, "[skipped] WinGet ve ağ: yalnız yerel dosya işleri seçildi.")
		return nil, c.ctx.Err()
	}
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	err := c.client.Available(ctx)
	cancel()
	if err != nil {
		return nil, fmt.Errorf("WinGet/Windows ön kontrolü: %w. Windows 11 x64 ve App Installer kurulumunu kontrol edin; PowerShell'de winget --version çalıştırın", err)
	}
	fmt.Fprintln(c.stdout, "[ok] Windows ve WinGet erişimi")
	var pending []catalog.App
	missing := false
	for _, choice := range c.selection.Apps {
		app := byID[choice.ID]
		ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
		installed, err := c.client.Installed(ctx, app.ID)
		cancel()
		if err != nil {
			return nil, fmt.Errorf("uygulama ön kontrolü %s: %w", app.Title, err)
		}
		if installed {
			fmt.Fprintf(c.stdout, "[skipped] %s zaten kurulu\n", app.Title)
			continue
		}
		missing = true
		if app.RequiresAdmin {
			pending = append(pending, app)
		}
	}
	if missing {
		check := c.checkNetwork
		if check == nil {
			check = checkWinGetConnection
		}
		ctx, cancel := context.WithTimeout(c.ctx, 10*time.Second)
		err := check(ctx)
		cancel()
		if err != nil {
			return nil, err
		}
		fmt.Fprintln(c.stdout, "[ok] WinGet CDN HTTPS bağlantısı (kurucu indirmeleri ayrıca kontrol edilir)")
	} else {
		fmt.Fprintln(c.stdout, "[skipped] Ağ: kurulum gerektiren uygulama yok.")
	}
	if err := c.ctx.Err(); err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		fmt.Fprintln(c.stdout, "[skipped] Yönetici izni: eksik yönetici uygulaması yok.")
	}
	return pending, nil
}

var errPreflightStopped = errors.New("ön kontrol tamamlanamadı; kurulum ve dosya değişikliği başlatılmadı")
