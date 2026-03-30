// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotifyVSCodeTerminal_Success(t *testing.T) {
	var calls []string

	err := NotifyVSCodeTerminal("abc1234", EnvVSCode, "/tmp/ipc.sock",
		"/usr/local/bin/lore", "/Users/dev/project", VSCodeOpts{
			LookPath: func(name string) (string, error) {
				return "/usr/bin/" + name, nil
			},
			RunCommand: func(name string, args, env []string) error {
				calls = append(calls, "run:"+args[1])
				return nil
			},
			StartCommand: func(name string, args, env []string) error {
				calls = append(calls, "start:"+args[1])
				return nil
			},
			Sleep: func(d time.Duration) {}, // no-op
		})

	require.NoError(t, err)
	assert.Equal(t, []string{
		"run:workbench.action.terminal.new",
		"start:workbench.action.terminal.sendSequence",
	}, calls)
}

func TestNotifyVSCodeTerminal_CLINotFound(t *testing.T) {
	err := NotifyVSCodeTerminal("abc1234", EnvVSCode, "/tmp/ipc.sock",
		"/usr/local/bin/lore", "/Users/dev/project", VSCodeOpts{
			LookPath: func(string) (string, error) {
				return "", errors.New("not found")
			},
		})

	assert.ErrorIs(t, err, errCLINotFound)
}

func TestNotifyVSCodeTerminal_TerminalNewFails(t *testing.T) {
	err := NotifyVSCodeTerminal("abc1234", EnvVSCode, "/tmp/ipc.sock",
		"/usr/local/bin/lore", "/Users/dev/project", VSCodeOpts{
			LookPath: func(name string) (string, error) {
				return "/usr/bin/" + name, nil
			},
			RunCommand: func(string, []string, []string) error {
				return errors.New("ipc failed")
			},
		})

	assert.ErrorIs(t, err, errFallbackDialog)
}

func TestNotifyVSCodeTerminal_SendSequenceFails(t *testing.T) {
	err := NotifyVSCodeTerminal("abc1234", EnvVSCode, "/tmp/ipc.sock",
		"/usr/local/bin/lore", "/Users/dev/project", VSCodeOpts{
			LookPath: func(name string) (string, error) {
				return "/usr/bin/" + name, nil
			},
			RunCommand: func(string, []string, []string) error { return nil },
			StartCommand: func(string, []string, []string) error {
				return errors.New("sendSequence failed")
			},
			Sleep: func(d time.Duration) {},
		})

	assert.ErrorIs(t, err, errFallbackDialog)
}

func TestNotifyVSCodeTerminal_CursorCLI(t *testing.T) {
	var cliUsed string
	err := NotifyVSCodeTerminal("abc1234", EnvCursor, "/tmp/ipc.sock",
		"/usr/local/bin/lore", "/Users/dev/project", VSCodeOpts{
			LookPath: func(name string) (string, error) {
				cliUsed = name
				return "/usr/bin/" + name, nil
			},
			RunCommand:   func(string, []string, []string) error { return nil },
			StartCommand: func(string, []string, []string) error { return nil },
			Sleep:        func(d time.Duration) {},
		})

	require.NoError(t, err)
	assert.Equal(t, "cursor", cliUsed)
}

func TestNotifyVSCodeTerminal_IPCInjected(t *testing.T) {
	var capturedEnv []string

	_ = NotifyVSCodeTerminal("abc1234", EnvVSCode, "/tmp/my-instance.sock",
		"/usr/local/bin/lore", "/Users/dev/project", VSCodeOpts{
			LookPath: func(name string) (string, error) {
				return "/usr/bin/" + name, nil
			},
			RunCommand: func(_ string, _ []string, env []string) error {
				capturedEnv = env
				return nil
			},
			StartCommand: func(string, []string, []string) error { return nil },
			Sleep:        func(d time.Duration) {},
		})

	// Verify IPC socket is in the env.
	found := false
	for _, v := range capturedEnv {
		if v == "VSCODE_IPC_HOOK_CLI=/tmp/my-instance.sock" {
			found = true
			break
		}
	}
	assert.True(t, found, "VSCODE_IPC_HOOK_CLI should be injected into command env")
}



func TestAppendIPC(t *testing.T) {
	env := []string{"HOME=/home/user", "PATH=/usr/bin"}
	result := appendIPC(env, "/tmp/sock")
	assert.Contains(t, result, "VSCODE_IPC_HOOK_CLI=/tmp/sock")
}

func TestAppendIPC_ReplacesExisting(t *testing.T) {
	env := []string{"HOME=/home/user", "VSCODE_IPC_HOOK_CLI=/old/sock", "PATH=/usr/bin"}
	result := appendIPC(env, "/new/sock")
	found := 0
	for _, v := range result {
		if v == "VSCODE_IPC_HOOK_CLI=/new/sock" {
			found++
		}
	}
	assert.Equal(t, 1, found, "should replace, not duplicate")
}

func TestAppendIPC_Empty(t *testing.T) {
	env := []string{"HOME=/home/user"}
	result := appendIPC(env, "")
	assert.Equal(t, env, result, "empty socket should not modify env")
}
