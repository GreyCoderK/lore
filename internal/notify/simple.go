// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"fmt"
	"os"
	"path/filepath"
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
		// Prefer terminal-notifier if available (supports custom icons).
		if tnPath, err := opts.LookPath("terminal-notifier"); err == nil {
			return opts.StartCommand(tnPath, []string{
				"-title", "Lore",
				"-message", safe,
				"-appIcon", findLogoPNG(),
			}, nil)
		}
		// Fallback to osascript (no custom icon — uses Script Editor icon).
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

// findLogoPNG locates the Lore logo PNG by walking up from the current
// working directory looking for a git repo with assets/logo.png.
func findLogoPNG() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidates := []string{
		filepath.Join(wd, "assets", "logo.png"),
		filepath.Join(wd, "docs", "assets", "logo.png"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
