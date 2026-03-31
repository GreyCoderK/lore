// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"fmt"
	"runtime"
)

// NotifyOSSimple sends a simple OS notification (no dialog/form).
// This is the last-resort notification before silent mode.
func NotifyOSSimple(commitMsg string, opts DialogOpts) error {
	opts.defaults()

	// Sanitize commit message to prevent injection in all platforms.
	safe := sanitizeForShell(commitMsg)

	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(
			`display notification "%s" with title "Lore"`,
			escapeAppleScript(safe),
		)
		return opts.StartCommand("osascript", []string{"-e", script}, nil)

	case "linux":
		if _, err := opts.LookPath("notify-send"); err == nil {
			// Pass commit message as a separate argument (not interpolated in shell).
			// Use --icon flag instead of -a for broader compatibility.
			return opts.StartCommand("notify-send",
				[]string{"--app-name=Lore", "Lore", safe}, nil)
		}
		return errUnsupportedOS

	case "windows":
		// Use single-quoted strings with proper PowerShell escaping.
		msg := escapePowerShell(safe)
		script := fmt.Sprintf(
			`[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null; `+
				`$n = New-Object System.Windows.Forms.NotifyIcon; `+
				`$n.Icon = [System.Drawing.SystemIcons]::Information; `+
				`$n.Visible = $true; `+
				`$n.ShowBalloonTip(5000, 'Lore', '%s', 'Info')`,
			msg,
		)
		return opts.StartCommand("powershell", []string{"-NoProfile", "-Command", script}, nil)

	default:
		return errUnsupportedOS
	}
}
