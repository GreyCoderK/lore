// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"net"
	"os"
	"strings"
	"time"
)

// EnvironmentSource identifies the IDE or context that launched the git commit.
type EnvironmentSource = string

const (
	EnvTerminal  EnvironmentSource = "terminal"
	EnvVSCode    EnvironmentSource = "vscode"
	EnvCursor    EnvironmentSource = "cursor"
	EnvWindsurf  EnvironmentSource = "windsurf"
	EnvCodium    EnvironmentSource = "codium"
	EnvJetBrains EnvironmentSource = "jetbrains"
	EnvNeovim    EnvironmentSource = "neovim"
	EnvVim       EnvironmentSource = "vim"
	EnvEmacs     EnvironmentSource = "emacs"
	EnvCI        EnvironmentSource = "ci"
	EnvGitGUI    EnvironmentSource = "git-gui"
	EnvUnknown   EnvironmentSource = "unknown"
)

// EnvOpts holds injectable dependencies for environment detection (testability).
type EnvOpts struct {
	// GetEnv reads an environment variable. Defaults to os.Getenv.
	GetEnv func(string) string

	// IsTTY reports whether the current environment is an interactive TTY.
	// When nil, defaults to false (caller should check TTY separately).
	IsTTY func() bool

	// DialTimeout tests socket connectivity. Defaults to net.DialTimeout.
	DialTimeout func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (o *EnvOpts) getEnv(key string) string {
	if o.GetEnv != nil {
		return o.GetEnv(key)
	}
	return os.Getenv(key)
}

func (o *EnvOpts) dialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	if o.DialTimeout != nil {
		return o.DialTimeout(network, address, timeout)
	}
	return net.DialTimeout(network, address, timeout)
}

// DetectEnvironment identifies the IDE or context that launched the git process.
// Detection order (first match wins):
//  1. CI (priority absolute — never notify in CI)
//  2. Emacs (INSIDE_EMACS is reliable)
//  3. Neovim (NVIM env var)
//  4. Vim (VIM without NVIM)
//  5. VS Code family (distinguish forks via GIT_ASKPASS path)
//  6. JetBrains (IDEA_INITIAL_DIRECTORY or JediTerm)
//  7. TTY interactive → terminal
//  8. Unknown non-TTY
func DetectEnvironment(opts EnvOpts) EnvironmentSource {
	// 1. CI — absolute priority, never notify.
	if isCI(&opts) {
		return EnvCI
	}

	// 2. Emacs — INSIDE_EMACS is reliable.
	if opts.getEnv("INSIDE_EMACS") != "" {
		return EnvEmacs
	}

	// 3. Neovim — subprocess of nvim.
	if opts.getEnv("NVIM") != "" {
		return EnvNeovim
	}

	// 4. Vim — VIM set but not Neovim.
	if opts.getEnv("VIM") != "" {
		return EnvVim
	}

	// 5. VS Code family — distinguish forks via GIT_ASKPASS path.
	if env := detectVSCodeFamily(&opts); env != "" {
		return env
	}

	// 6. JetBrains — IDEA_INITIAL_DIRECTORY or JediTerm terminal.
	if opts.getEnv("IDEA_INITIAL_DIRECTORY") != "" ||
		opts.getEnv("TERMINAL_EMULATOR") == "JetBrains-JediTerm" {
		return EnvJetBrains
	}

	// 7. Fleet — treat as JetBrains for MVP.
	if opts.getEnv("FLEET_PROPERTIES_FILE") != "" {
		return EnvJetBrains
	}

	// 8. TTY interactive → normal terminal.
	if opts.IsTTY != nil && opts.IsTTY() {
		return EnvTerminal
	}

	// 9. Unknown non-TTY (Git GUI probable).
	return EnvUnknown
}

// IsVSCodeFamily returns true if the environment is VS Code or one of its forks.
func IsVSCodeFamily(env EnvironmentSource) bool {
	switch env {
	case EnvVSCode, EnvCursor, EnvWindsurf, EnvCodium:
		return true
	}
	return false
}

// IsRemoteEnvironment returns true if running in a remote context where
// local IPC and display are unavailable.
func IsRemoteEnvironment(opts EnvOpts) bool {
	if opts.getEnv("REMOTE_CONTAINERS") != "" ||
		opts.getEnv("CODESPACES") != "" ||
		opts.getEnv("VSCODE_REMOTE_CONTAINERS_SESSION") != "" {
		return true
	}
	return isWSLWithoutDisplay(&opts)
}

// DetectIPCSocket finds the IPC socket for the VS Code family instance.
// Tries fork-specific env vars first, then falls back to VSCODE_IPC_HOOK_CLI.
// Returns empty string if no live socket is found.
func DetectIPCSocket(env EnvironmentSource, opts EnvOpts) string {
	var candidates []string

	switch env {
	case EnvCursor:
		candidates = append(candidates,
			opts.getEnv("CURSOR_IPC_HOOK_CLI"),
			opts.getEnv("VSCODE_IPC_HOOK_CLI"),
		)
	case EnvWindsurf:
		candidates = append(candidates,
			opts.getEnv("WINDSURF_IPC_HOOK_CLI"),
			opts.getEnv("VSCODE_IPC_HOOK_CLI"),
		)
	case EnvCodium:
		candidates = append(candidates,
			opts.getEnv("VSCODIUM_IPC_HOOK_CLI"),
			opts.getEnv("VSCODE_IPC_HOOK_CLI"),
		)
	default:
		candidates = append(candidates,
			opts.getEnv("VSCODE_IPC_HOOK_CLI"),
		)
	}

	for _, sock := range candidates {
		if sock != "" && isSocketAlive(sock, &opts) {
			return sock
		}
	}
	return ""
}

// isCI checks for CI/CD environment signals.
func isCI(opts *EnvOpts) bool {
	if opts.getEnv("CI") != "" {
		return true
	}
	ciVars := []string{
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL",
		"CIRCLECI", "BUILDKITE", "TRAVIS",
	}
	for _, v := range ciVars {
		if opts.getEnv(v) != "" {
			return true
		}
	}
	return false
}

// detectVSCodeFamily distinguishes VS Code forks via GIT_ASKPASS path.
func detectVSCodeFamily(opts *EnvOpts) EnvironmentSource {
	askpass := opts.getEnv("GIT_ASKPASS")
	if askpass != "" {
		lower := strings.ToLower(askpass)
		switch {
		case strings.Contains(lower, "cursor"):
			return EnvCursor
		case strings.Contains(lower, "windsurf"):
			return EnvWindsurf
		case strings.Contains(lower, "codium"):
			return EnvCodium
		case strings.Contains(lower, "code"):
			return EnvVSCode
		}
	}
	// Secondary signal: VSCODE_GIT_ASKPASS_NODE without identifiable askpass path.
	if opts.getEnv("VSCODE_GIT_ASKPASS_NODE") != "" {
		return EnvVSCode
	}
	return ""
}

// isWSLWithoutDisplay detects WSL environments without X11/Wayland forwarding.
func isWSLWithoutDisplay(opts *EnvOpts) bool {
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err != nil {
		return false
	}
	return opts.getEnv("DISPLAY") == "" && opts.getEnv("WAYLAND_DISPLAY") == ""
}

// isSocketAlive checks if a Unix socket is responsive.
// Uses a short timeout (100ms) since local sockets respond in < 1ms.
func isSocketAlive(socketPath string, opts *EnvOpts) bool {
	conn, err := opts.dialTimeout("unix", socketPath, 100*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
