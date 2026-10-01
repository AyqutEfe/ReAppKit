//go:build windows

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"golang.org/x/sys/windows"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf16"
)

func processElevated() bool                 { return windows.GetCurrentProcessToken().IsElevated() }
func powershellLiteral(value string) string { return "'" + strings.ReplaceAll(value, "'", "''") + "'" }
func workerLaunchScript(executable, encodedPlan string) string {
	return "$ErrorActionPreference='Stop'; try { $worker = Start-Process -FilePath " + powershellLiteral(executable) +
		" -ArgumentList " + powershellLiteral("--admin-worker "+encodedPlan) +
		" -Verb RunAs -WindowStyle Hidden -PassThru; $worker.WaitForExit(); exit $worker.ExitCode } catch { Write-Error $_; exit 1 }"
}
func encodePowerShell(script string) string {
	units := utf16.Encode([]rune(script))
	bytes := make([]byte, len(units)*2)
	for i, unit := range units {
		bytes[i*2], bytes[i*2+1] = byte(unit), byte(unit>>8)
	}
	return base64.StdEncoding.EncodeToString(bytes)
}
func launchAdminWorker(ctx context.Context, executable, encodedPlan string) error {
	command := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShell(workerLaunchScript(executable, encodedPlan)))
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
