//go:build !windows

package main

import (
	"errors"
	"log/slog"
	"time"
)

// Running as a LaunchDaemon (macOS) or systemd service (Linux) comes in the
// Mac and Linux phase (docs/plan.md). Until then the agent runs in a console.

var errNotWindows = errors.New("installing as a service is supported only on Windows for now")

func isService() bool { return false }

func runService(string, time.Duration) {}

func installService(*slog.Logger, string, string, string) error { return errNotWindows }

func uninstallService(*slog.Logger, string) error { return errNotWindows }
