// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// VSCodeOpts holds injectable dependencies for VS Code terminal notification.
type VSCodeOpts struct {
	// LookPath searches for a binary in PATH. Defaults to exec.LookPath.
	LookPath func(file string) (string, error)

	// RunCommand executes a command synchronously. Defaults to cmd.Run().
	RunCommand func(name string, args []string, env []string) error

	// StartCommand launches a detached command. Defaults to cmd.Start().
	StartCommand func(name string, args []string, env []string) error

	// Sleep pauses execution. Defaults to time.Sleep.
	Sleep func(d time.Duration)
}

func (o *VSCodeOpts) defaults() {
	if o.LookPath == nil {
		o.LookPath = exec.LookPath
	}
	if o.RunCommand == nil {
		o.RunCommand = defaultRunCommand
	}
	if o.StartCommand == nil {
		o.StartCommand = defaultStartCommand
	}
	if o.Sleep == nil {
		o.Sleep = time.Sleep
	}
}

// NotifyVSCodeTerminal opens a terminal in the correct VS Code instance
// and sends a lore pending resolve command.
//
// Uses VSCODE_IPC_HOOK_CLI (inherited by the hook from the parent VS Code
// process) to target the correct instance even when multiple are open.
//
// This is a best-effort approach: code --command workbench.action.terminal.sendSequence
// is an undocumented internal API. If any step fails, the caller should fall
// back to OS dialog notification.
func NotifyVSCodeTerminal(hash string, env EnvironmentSource, ipcSocket, lorePath, repoRoot string, opts VSCodeOpts) error {
	opts.defaults()

	// 1. Find the CLI binary for the correct fork.
	cli, err := detectVSCodeCLI(env, opts.LookPath)
	if err != nil {
		return err
	}

	// 2. Build the environment with IPC socket targeting.
	cmdEnv := appendIPC(os.Environ(), ipcSocket)

	// 3. Build the one-liner with absolute paths and sanitized hash.
	command := fmt.Sprintf("cd %s && %s pending resolve --commit %s",
		shellQuote(repoRoot),
		shellQuote(lorePath),
		sanitizeCommitHash(hash),
	)

	// 4. Open a new terminal in the correct instance.
	if err := opts.RunCommand(cli,
		[]string{"--command", "workbench.action.terminal.new"},
		cmdEnv); err != nil {
		return fmt.Errorf("%w: terminal.new: %v", errFallbackDialog, err)
	}

	// 5. Wait for terminal to initialize.
	opts.Sleep(500 * time.Millisecond)

	// 6. Send the command to the terminal (undocumented API, best-effort).
	sequence := fmt.Sprintf(`{"text":"%s\n"}`, escapeForJSON(command))
	if err := opts.StartCommand(cli,
		[]string{"--command", "workbench.action.terminal.sendSequence", "--args", sequence},
		cmdEnv); err != nil {
		return fmt.Errorf("%w: sendSequence: %v", errFallbackDialog, err)
	}

	return nil
}

// detectVSCodeCLI finds the CLI binary name for the given VS Code fork.
func detectVSCodeCLI(env EnvironmentSource, lookPath func(string) (string, error)) (string, error) {
	var name string
	switch env {
	case EnvCursor:
		name = "cursor"
	case EnvWindsurf:
		name = "windsurf"
	case EnvCodium:
		name = "codium"
	default:
		name = "code"
	}
	path, err := lookPath(name)
	if err != nil {
		return "", fmt.Errorf("%w: %s", errCLINotFound, name)
	}
	return path, nil
}

// appendIPC returns a copy of environ with VSCODE_IPC_HOOK_CLI set.
// Does not mutate the input slice.
func appendIPC(environ []string, ipcSocket string) []string {
	if ipcSocket == "" {
		return environ
	}
	// Copy to avoid mutating the caller's slice.
	result := make([]string, len(environ))
	copy(result, environ)

	key := "VSCODE_IPC_HOOK_CLI="
	for i, v := range result {
		if strings.HasPrefix(v, key) {
			result[i] = key + ipcSocket
			return result
		}
	}
	return append(result, key+ipcSocket)
}

func defaultRunCommand(name string, args, env []string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	return cmd.Run()
}

func defaultStartCommand(name string, args, env []string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		return err
	}
	// Reap the child process in the background to prevent zombies.
	go cmd.Wait() //nolint:errcheck
	return nil
}
