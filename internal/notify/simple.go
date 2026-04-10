// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"fmt"
	"runtime"

	"github.com/greycoderk/lore/internal/brand"
)

// NotifyOSSimple sends a simple OS notification (no dialog/form).
// This is the last-resort notification before silent mode.
func NotifyOSSimple(commitMsg string, opts DialogOpts) error {
	opts.defaults()

	// Sanitize commit message to prevent injection in all platforms.
	safe := sanitizeForShell(commitMsg)

	switch runtime.GOOS {
	case "darwin":
		// Prefer terminal-notifier if available (supports custom icons + click actions).
		tnPath, tnErr := opts.LookPath("terminal-notifier")
		if tnErr != nil {
			// Try auto-install via Homebrew (silent, best-effort).
			if brewPath, brewErr := opts.LookPath("brew"); brewErr == nil {
				_ = opts.StartCommand(brewPath, []string{"install", "--quiet", "terminal-notifier"}, nil)
				tnPath, tnErr = opts.LookPath("terminal-notifier")
			}
		}
		if tnErr == nil {
			return opts.StartCommand(tnPath, []string{
				"-title", "Lore",
				"-message", safe,
				"-appIcon", brand.LogoPNGPath(),
			}, nil)
		}
		// Fallback to osascript display notification (no custom icon possible).
		script := fmt.Sprintf(
			`display notification "%s" with title "Lore"`,
			escapeAppleScript(safe),
		)
		return opts.StartCommand("osascript", []string{"-e", script}, nil)

	case "linux":
		if _, err := opts.LookPath("notify-send"); err == nil {
			args := []string{"--app-name=Lore"}
			if icon := brand.LogoPNGPath(); icon != "" {
				args = append(args, "--icon="+icon)
			}
			args = append(args, "Lore", safe)
			return opts.StartCommand("notify-send", args, nil)
		}
		return errUnsupportedOS

	case "windows":
		msg := escapePowerShell(safe)
		iconExpr := `[System.Drawing.SystemIcons]::Information`
		if icon := brand.LogoPNGPath(); icon != "" {
			iconExpr = fmt.Sprintf(
				`[System.Drawing.Icon]::FromHandle(([System.Drawing.Bitmap]::new('%s')).GetHicon())`,
				escapePowerShell(icon),
			)
		}
		script := fmt.Sprintf(
			`[System.Reflection.Assembly]::LoadWithPartialName('System.Windows.Forms') | Out-Null; `+
				`[System.Reflection.Assembly]::LoadWithPartialName('System.Drawing') | Out-Null; `+
				`$n = New-Object System.Windows.Forms.NotifyIcon; `+
				`$n.Icon = %s; `+
				`$n.Visible = $true; `+
				`$n.ShowBalloonTip(5000, 'Lore', '%s', 'Info')`,
			iconExpr, msg,
		)
		return opts.StartCommand("powershell", []string{"-NoProfile", "-Command", script}, nil)

	default:
		return errUnsupportedOS
	}
}
