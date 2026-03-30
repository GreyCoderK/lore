// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !windows

package notify

import (
	"os"
	"syscall"
)

// isProcessAlive checks if a process with the given PID is still running.
// On Unix, Signal(0) checks existence without sending a real signal.
func isProcessAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}
