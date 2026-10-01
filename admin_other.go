//go:build !windows

package main

import (
	"context"
	"errors"
)

func processElevated() bool { return false }
func launchAdminWorker(context.Context, string, string) error {
	return errors.New("admin worker is supported only on Windows")
}
