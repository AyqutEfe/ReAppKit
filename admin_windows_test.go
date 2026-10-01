//go:build windows

package main

import (
	"encoding/base64"
	"os/exec"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestWorkerLaunchQuotesExecutableAndRequestsSingleUAC(t *testing.T) {
	script := workerLaunchScript(`C:\Users\O'Brien\ReAppKit.exe`, "YWJj")
	if !strings.Contains(script, `'C:\Users\O''Brien\ReAppKit.exe'`) || strings.Count(script, "-Verb RunAs") != 1 || !strings.Contains(script, "-WindowStyle Hidden") || !strings.Contains(script, "--admin-worker YWJj") {
		t.Fatal(script)
	}
	data, err := base64.StdEncoding.DecodeString(encodePowerShell(script))
	if err != nil {
		t.Fatal(err)
	}
	var units []uint16
	for i := 0; i < len(data); i += 2 {
		units = append(units, uint16(data[i])|uint16(data[i+1])<<8)
	}
	if string(utf16.Decode(units)) != script {
		t.Fatal("PowerShell encoding changed script")
	}
}

func TestWorkerLaunchScriptParsesInWindowsPowerShell(t *testing.T) {
	script := workerLaunchScript(`C:\Users\O'Brien\ReAppKit.exe`, "YWJj")
	// Parse only: this does not launch a process or ask for elevation.
	check := "$text=[Text.Encoding]::Unicode.GetString([Convert]::FromBase64String('" + encodePowerShell(script) + "')); $tokens=$null; $errors=$null; [void][System.Management.Automation.Language.Parser]::ParseInput($text,[ref]$tokens,[ref]$errors); if($errors.Count -ne 0){$errors | Out-String | Write-Output; exit 1}"
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-EncodedCommand", encodePowerShell(check)).CombinedOutput()
	if err != nil {
		t.Fatalf("launcher PowerShell syntax: %v: %s", err, output)
	}
}
