package settings

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/AyqutEfe/ReAppKit/internal/core"
)

func TestFileJobBacksUpAndSkipsRerun(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	target := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(source, []byte("new settings"), 0600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(target, []byte("old settings"), 0600); err != nil { t.Fatal(err) }
	job, err := FileJob("settings:test", "Test", source, target, nil)
	if err != nil { t.Fatal(err) }
	if result := core.Execute(context.Background(), []core.Job{job}, false, nil); result.Error == "" { t.Fatal("no confirmation accepted") }
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "old settings" { t.Fatalf("target changed before confirmation: %q, %v", data, err) }
	result := core.Execute(context.Background(), []core.Job{job}, true, nil)
	if result.Error != "" || result.Items[0].State != core.Succeeded { t.Fatalf("result: %+v", result) }
	data, err = os.ReadFile(target)
	if err != nil || string(data) != "new settings" { t.Fatalf("target: %q, %v", data, err) }
	backups, err := filepath.Glob(target + ".reappkit-backup-*")
	if err != nil || len(backups) != 1 { t.Fatalf("backup paths: %v, %v", backups, err) }
	data, err = os.ReadFile(backups[0])
	if err != nil || string(data) != "old settings" { t.Fatalf("backup: %q, %v", data, err) }
	result = core.Execute(context.Background(), []core.Job{job}, true, nil)
	if result.Items[0].State != core.Skipped { t.Fatalf("rerun: %+v", result) }
	backups, _ = filepath.Glob(target + ".reappkit-backup-*")
	if len(backups) != 1 { t.Fatalf("rerun created another backup: %v", backups) }
}

func TestFileJobRejectsUnsafePaths(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	if err := os.WriteFile(source, []byte("x"), 0600); err != nil { t.Fatal(err) }
	if _, err := FileJob("x", "X", source, source, nil); err == nil { t.Fatal("same path accepted") }
	if _, err := FileJob("x", "X", "relative", filepath.Join(dir, "target"), nil); err == nil { t.Fatal("relative source accepted") }
}
