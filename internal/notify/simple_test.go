// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
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
