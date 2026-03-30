// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build windows

package notify

// isProcessAlive checks if a process with the given PID is still running.
// On Windows, there is no reliable, side-effect-free way to probe a process.
// We always return false (treat lock as stale). The worst case is two
// concurrent notifications, which is acceptable for best-effort notification.
func isProcessAlive(_ int) bool {
	return false
}
