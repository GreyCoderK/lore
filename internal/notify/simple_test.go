// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifyOSSimple_Detached(t *testing.T) {
	var captured struct {
		name string
		args []string
	}

	err := NotifyOSSimple("fix(auth): token refresh", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			captured.name = name
			captured.args = args
			return nil
		},
		// Force osascript path on darwin by hiding terminal-notifier.
		LookPath: func(file string) (string, error) {
			if file == "terminal-notifier" {
				return "", fmt.Errorf("not found")
			}
			return exec.LookPath(file)
		},
	})

	switch runtime.GOOS {
	case "darwin":
		require.NoError(t, err)
		assert.Equal(t, "osascript", captured.name)
		assert.Contains(t, captured.args[1], "fix(auth): token refresh")
	case "linux":
		// May return errUnsupportedOS if notify-send is not installed.
		if err == nil {
			assert.Equal(t, "notify-send", captured.name)
		}
	case "windows":
		require.NoError(t, err)
		assert.Equal(t, "powershell", captured.name)
	}
}

func TestNotifyOSSimple_SanitizesInput(t *testing.T) {
	var capturedArgs []string

	_ = NotifyOSSimple(`msg"; malicious "injection`, DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			capturedArgs = args
			return nil
		},
	})

	if runtime.GOOS == "darwin" && len(capturedArgs) > 1 {
		// The script should contain escaped quotes, not raw ones.
		assert.NotContains(t, capturedArgs[1], `"; malicious "`)
	}
}

func TestNotifyOSSimple_TerminalNotifier(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("terminal-notifier path only runs on darwin")
	}

	var captured struct {
		name string
		args []string
	}

	err := NotifyOSSimple("feat: new widget", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			captured.name = name
			captured.args = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			if file == "terminal-notifier" {
				return "/usr/local/bin/terminal-notifier", nil
			}
			return "", fmt.Errorf("not found")
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/terminal-notifier", captured.name)
	assert.Contains(t, captured.args, "-title")
	assert.Contains(t, captured.args, "Lore")
	assert.Contains(t, captured.args, "feat: new widget")
}

func TestNotifyOSSimple_TerminalNotifier_HasIconArg(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("terminal-notifier path only runs on darwin")
	}

	var capturedArgs []string
	err := NotifyOSSimple("test", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			capturedArgs = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			if file == "terminal-notifier" {
				return "/usr/local/bin/terminal-notifier", nil
			}
			return "", fmt.Errorf("not found")
		},
	})

	require.NoError(t, err)
	assert.Contains(t, capturedArgs, "-appIcon")
	// The arg after -appIcon should be a non-empty path ending in .png.
	for i, arg := range capturedArgs {
		if arg == "-appIcon" && i+1 < len(capturedArgs) {
			assert.NotEmpty(t, capturedArgs[i+1], "appIcon path should not be empty")
			assert.Contains(t, capturedArgs[i+1], ".png")
		}
	}
}

func TestNotifyOSSimple_LinuxWithNotifySend_HasIconArg(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}

	var capturedArgs []string
	err := NotifyOSSimple("test", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			capturedArgs = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			if file == "notify-send" {
				return "/usr/bin/notify-send", nil
			}
			return "", fmt.Errorf("not found")
		},
	})

	require.NoError(t, err)
	hasIcon := false
	for _, arg := range capturedArgs {
		if len(arg) > 7 && arg[:7] == "--icon=" {
			hasIcon = true
			assert.Contains(t, arg, ".png")
		}
	}
	assert.True(t, hasIcon, "notify-send args should contain --icon flag, got: %v", capturedArgs)
}

func TestNotifyOSSimple_Windows_HasIconInScript(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}

	var capturedArgs []string
	err := NotifyOSSimple("test", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			capturedArgs = args
			return nil
		},
	})

	require.NoError(t, err)
	// The PowerShell script should reference Bitmap (custom icon), not SystemIcons.
	script := capturedArgs[len(capturedArgs)-1]
	assert.Contains(t, script, "Bitmap", "Windows script should load icon from Bitmap, not SystemIcons")
}

func TestNotifyOSSimple_EmptyMessage(t *testing.T) {
	var capturedArgs []string

	err := NotifyOSSimple("", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			capturedArgs = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	})

	switch runtime.GOOS {
	case "darwin":
		// osascript fallback should still work with empty message.
		require.NoError(t, err)
		assert.Equal(t, 2, len(capturedArgs)) // ["-e", script]
	case "linux":
		// errUnsupportedOS if notify-send not found.
		assert.Error(t, err)
	case "windows":
		require.NoError(t, err)
	}
}

func TestNotifyOSSimple_StartCommandError(t *testing.T) {
	cmdErr := fmt.Errorf("command failed")

	err := NotifyOSSimple("test msg", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			return cmdErr
		},
		LookPath: func(file string) (string, error) {
			return "", fmt.Errorf("not found")
		},
	})

	switch runtime.GOOS {
	case "darwin", "windows":
		assert.ErrorIs(t, err, cmdErr)
	case "linux":
		// notify-send not found -> errUnsupportedOS (LookPath fails).
		assert.Error(t, err)
	}
}

func TestNotifyOSSimple_LinuxWithNotifySend(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}

	var captured struct {
		name string
		args []string
	}

	err := NotifyOSSimple("fix: memory leak", DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			captured.name = name
			captured.args = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			if file == "notify-send" {
				return "/usr/bin/notify-send", nil
			}
			return "", fmt.Errorf("not found")
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "notify-send", captured.name)
	assert.Contains(t, captured.args, "--app-name=Lore")
	assert.Contains(t, captured.args, "fix: memory leak")
}
