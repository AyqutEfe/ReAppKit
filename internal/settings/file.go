// Package settings creates opt-in, user-specified file configuration jobs.
package settings

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/AyqutEfe/ReAppKit/internal/core"
)

// FileJob copies a user-provided regular file to an explicit target. An
// existing target gets a unique sibling backup before replacement.
func FileJob(id, title, source, target string, dependsOn []string) (core.Job, error) {
	if id == "" || title == "" || !filepath.IsAbs(source) || !filepath.IsAbs(target) {
		return core.Job{}, errors.New("file job requires ID, title and absolute source/target paths")
	}
	cleanSource, cleanTarget := filepath.Clean(source), filepath.Clean(target)
	if strings.EqualFold(cleanSource, cleanTarget) { return core.Job{}, errors.New("source and target must differ") }
	if _, err := readRegular(cleanSource); err != nil { return core.Job{}, fmt.Errorf("source: %w", err) }
	check := func(context.Context) (bool, error) {
		src, err := readRegular(cleanSource)
		if err != nil { return false, err }
		dst, err := readRegular(cleanTarget)
		if errors.Is(err, os.ErrNotExist) { return false, nil }
		if err != nil { return false, err }
		return bytes.Equal(src, dst), nil
	}
	return core.Job{
		ID: id, Title: title, DependsOn: append([]string(nil), dependsOn...),
		Check: check,
		Apply: func(ctx context.Context) error { return copyWithBackup(ctx, cleanSource, cleanTarget) },
		Verify: check,
	}, nil
}

func readRegular(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil { return nil, err }
	if !info.Mode().IsRegular() { return nil, fmt.Errorf("not a regular file: %s", path) }
	return os.ReadFile(path)
}

func copyWithBackup(ctx context.Context, source, target string) error {
	if err := ctx.Err(); err != nil { return err }
	src, err := readRegular(source)
	if err != nil { return fmt.Errorf("source: %w", err) }
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0700); err != nil { return err }
	mode := os.FileMode(0600)
	info, err := os.Lstat(target)
	if err == nil {
		if !info.Mode().IsRegular() { return fmt.Errorf("target is not a regular file: %s", target) }
		mode = info.Mode().Perm()
		if err := backup(target, parent, mode); err != nil { return err }
	} else if !errors.Is(err, os.ErrNotExist) { return err }
	if err := ctx.Err(); err != nil { return err }
	tmp, err := os.CreateTemp(parent, "."+filepath.Base(target)+".reappkit-*")
	if err != nil { return err }
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(mode); err != nil { tmp.Close(); return err }
	if _, err := tmp.Write(src); err != nil { tmp.Close(); return err }
	if err := tmp.Sync(); err != nil { tmp.Close(); return err }
	if err := tmp.Close(); err != nil { return err }
	if err := ctx.Err(); err != nil { return err }
	return os.Rename(tmpName, target)
}

func backup(target, parent string, mode os.FileMode) error {
	in, err := os.Open(target)
	if err != nil { return err }
	defer in.Close()
	out, err := os.CreateTemp(parent, filepath.Base(target)+".reappkit-backup-*")
	if err != nil { return err }
	name := out.Name()
	remove := true
	defer func() { if remove { os.Remove(name) } }()
	if err := out.Chmod(mode); err != nil { out.Close(); return err }
	if _, err := io.Copy(out, in); err != nil { out.Close(); return err }
	if err := out.Sync(); err != nil { out.Close(); return err }
	if err := out.Close(); err != nil { return err }
	remove = false
	return nil
}
