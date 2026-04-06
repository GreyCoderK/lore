// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockOpts(t *testing.T) NotifyOpts {
	t.Helper()
	tmp := t.TempDir()
	return NotifyOpts{
		EnvOpts: EnvOpts{
			GetEnv:      envMap(map[string]string{}),
			DialTimeout: noDial(),
		},
		PathOpts: PathOpts{
			Executable: func() (string, error) { return "/usr/local/bin/lore", nil },
			GitCommand: func(args ...string) (string, error) { return tmp, nil },
			Getwd:      func() (string, error) { return tmp, nil },
		},
		DialogOpts: DialogOpts{
			StartCommand: func(string, []string, []string) error { return nil },
			LookPath:     func(string) (string, error) { return "/usr/bin/mock", nil },
		},
		VSCodeOpts: VSCodeOpts{
			LookPath:     func(string) (string, error) { return "/usr/bin/code", nil },
			RunCommand:   func(string, []string, []string) error { return nil },
			StartCommand: func(string, []string, []string) error { return nil },
			Sleep:        func(d time.Duration) {},
		},
		Config: NotifyConfig{Mode: ModeAuto},
	}
}

func TestNotifyNonTTY_Silent(t *testing.T) {
	called := false
	opts := mockOpts(t)
	opts.Config.Mode = ModeSilent
	opts.DialogOpts.StartCommand = func(string, []string, []string) error {
		called = true
		return nil
	}

	NotifyNonTTY("abc", EnvVSCode, "msg", "+1-0", "bugfix", "what", "why", opts)
	assert.False(t, called, "silent mode should not launch anything")
}

func TestNotifyNonTTY_DisabledEnv(t *testing.T) {
	called := false
	opts := mockOpts(t)
	opts.Config.DisabledEnvs = []string{"vim"}
	opts.DialogOpts.StartCommand = func(string, []string, []string) error {
		called = true
		return nil
	}

	NotifyNonTTY("abc", EnvVim, "msg", "+1-0", "note", "what", "why", opts)
	assert.False(t, called, "disabled env should not notify")
}

func TestNotifyNonTTY_VSCodeTerminal(t *testing.T) {
	var terminalCalled bool
	opts := mockOpts(t)
	opts.EnvOpts.GetEnv = envMap(map[string]string{
		"VSCODE_IPC_HOOK_CLI": "/tmp/test.sock",
	})
	// Make socket "alive".
	opts.EnvOpts.DialTimeout = func(_, _ string, _ time.Duration) (net.Conn, error) {
		return &mockConn{}, nil
	}
	opts.VSCodeOpts.RunCommand = func(_ string, args []string, _ []string) error {
		if len(args) > 1 && args[1] == "workbench.action.terminal.new" {
			terminalCalled = true
		}
		return nil
	}

	NotifyNonTTY("abc", EnvVSCode, "msg", "+1-0", "bugfix", "what", "why", opts)
	assert.True(t, terminalCalled, "should attempt VS Code terminal")
}

func TestNotifyNonTTY_FallbackToDialog(t *testing.T) {
	var dialogCalled bool
	opts := mockOpts(t)
	// No IPC socket → VS Code terminal fails → should fall to dialog.
	opts.EnvOpts.GetEnv = envMap(map[string]string{})
	opts.DialogOpts.StartCommand = func(name string, _ []string, _ []string) error {
		dialogCalled = true
		return nil
	}

	NotifyNonTTY("abc", EnvVSCode, "msg", "+1-0", "bugfix", "what", "why", opts)
	assert.True(t, dialogCalled, "should fall back to OS dialog")
}

func TestNotifyNonTTY_JetBrainsGoesDirectlyToDialog(t *testing.T) {
	var dialogCalled bool
	opts := mockOpts(t)
	opts.DialogOpts.StartCommand = func(name string, _ []string, _ []string) error {
		dialogCalled = true
		return nil
	}

	NotifyNonTTY("abc", EnvJetBrains, "msg", "+1-0", "bugfix", "what", "why", opts)
	assert.True(t, dialogCalled, "JetBrains should go directly to dialog")
}

func TestAcquireReleaseLock(t *testing.T) {
	tmp := t.TempDir()
	lockPath := filepath.Join(tmp, ".lore", "notification.lock")

	// First acquire should succeed.
	ok := acquireLock(lockPath)
	require.True(t, ok)

	// Second acquire should fail (we're the same process but lock exists).
	// On Unix, isProcessAlive(our PID) returns true → lock held → false.
	// On Windows, isProcessAlive always returns false → stale → re-acquired → true.
	ok2 := acquireLock(lockPath)
	if runtime.GOOS == "windows" {
		assert.True(t, ok2, "Windows treats all locks as stale")
	} else {
		assert.False(t, ok2)
	}

	// Release should allow re-acquire.
	releaseLock(lockPath)
	ok3 := acquireLock(lockPath)
	assert.True(t, ok3)
	releaseLock(lockPath)
}

func TestAcquireLock_StaleLock(t *testing.T) {
	tmp := t.TempDir()
	lockPath := filepath.Join(tmp, ".lore", "notification.lock")
	_ = os.MkdirAll(filepath.Dir(lockPath), 0o755)

	// Write a stale lock with a non-existent PID.
	_ = os.WriteFile(lockPath, []byte("999999999"), 0o644)

	// Should succeed (stale lock cleaned up).
	ok := acquireLock(lockPath)
	assert.True(t, ok)
	releaseLock(lockPath)
}

func TestDefaultNotifyConfig(t *testing.T) {
	cfg := DefaultNotifyConfig()
	assert.Equal(t, ModeAuto, cfg.Mode)
	assert.Nil(t, cfg.DisabledEnvs)
}

func TestNotifyNonTTY_TerminalMode(t *testing.T) {
	var vscodeCalled bool
	opts := mockOpts(t)
	opts.Config.Mode = ModeTerminal
	opts.VSCodeOpts.RunCommand = func(_ string, args []string, _ []string) error {
		vscodeCalled = true
		return nil
	}

	// Non-VSCode env → terminal mode silent
	NotifyNonTTY("abc", EnvUnknown, "msg", "+1-0", "bugfix", "what", "why", opts)
	assert.False(t, vscodeCalled, "terminal mode with non-vscode env should be silent")
}

func TestNotifyNonTTY_DialogMode(t *testing.T) {
	var dialogCalled bool
	opts := mockOpts(t)
	opts.Config.Mode = ModeDialog
	opts.DialogOpts.StartCommand = func(name string, _ []string, _ []string) error {
		dialogCalled = true
		return nil
	}

	NotifyNonTTY("abc", EnvUnknown, "msg", "+1-0", "bugfix", "what", "why", opts)
	assert.True(t, dialogCalled, "dialog mode should launch dialog")
}

func TestNotifyNonTTY_NotifyMode(t *testing.T) {
	var simpleCalled bool
	opts := mockOpts(t)
	opts.Config.Mode = ModeNotify
	opts.DialogOpts.StartCommand = func(name string, args []string, _ []string) error {
		simpleCalled = true
		return nil
	}

	NotifyNonTTY("abc", EnvUnknown, "msg", "+1-0", "bugfix", "what", "why", opts)
	// ModeNotify should try simple notification path
	assert.True(t, simpleCalled, "notify mode should try OS notification")
}

// mockConn implements net.Conn for testing socket aliveness.
type mockConn struct{}

func (m *mockConn) Read([]byte) (int, error)         { return 0, nil }
func (m *mockConn) Write([]byte) (int, error)        { return 0, nil }
func (m *mockConn) Close() error                     { return nil }
func (m *mockConn) LocalAddr() net.Addr              { return nil }
func (m *mockConn) RemoteAddr() net.Addr             { return nil }
func (m *mockConn) SetDeadline(time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(time.Time) error { return nil }
