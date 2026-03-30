// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func envMap(m map[string]string) func(string) string {
	return func(key string) string {
		return m[key]
	}
}

func noTTY() func() bool  { return func() bool { return false } }
func yesTTY() func() bool { return func() bool { return true } }
func noDial() func(string, string, time.Duration) (net.Conn, error) {
	return func(_, _ string, _ time.Duration) (net.Conn, error) {
		return nil, net.UnknownNetworkError("mock")
	}
}

func TestDetectEnvironment_CI(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{"CI=true", map[string]string{"CI": "true"}},
		{"CI=1", map[string]string{"CI": "1"}},
		{"CI=yes", map[string]string{"CI": "yes"}},
		{"GITHUB_ACTIONS", map[string]string{"GITHUB_ACTIONS": "true"}},
		{"GITLAB_CI", map[string]string{"GITLAB_CI": "true"}},
		{"JENKINS_URL", map[string]string{"JENKINS_URL": "http://jenkins"}},
		{"CIRCLECI", map[string]string{"CIRCLECI": "true"}},
		{"BUILDKITE", map[string]string{"BUILDKITE": "true"}},
		{"TRAVIS", map[string]string{"TRAVIS": "true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectEnvironment(EnvOpts{GetEnv: envMap(tt.env)})
			assert.Equal(t, EnvCI, got)
		})
	}
}

func TestDetectEnvironment_Emacs(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"INSIDE_EMACS": "29.1"}),
	})
	assert.Equal(t, EnvEmacs, got)
}

func TestDetectEnvironment_Neovim(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"NVIM": "/tmp/nvimsocket"}),
	})
	assert.Equal(t, EnvNeovim, got)
}

func TestDetectEnvironment_Vim(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"VIM": "/usr/share/vim"}),
	})
	assert.Equal(t, EnvVim, got)
}

func TestDetectEnvironment_VimNotNeovim(t *testing.T) {
	// When both VIM and NVIM are set, Neovim takes priority.
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"VIM": "/usr/share/vim", "NVIM": "/tmp/nvimsocket"}),
	})
	assert.Equal(t, EnvNeovim, got)
}

func TestDetectEnvironment_VSCode(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{
			"GIT_ASKPASS":             "/app/code/resources/app/extensions/git/dist/askpass.sh",
			"VSCODE_GIT_ASKPASS_NODE": "/app/code/node",
		}),
	})
	assert.Equal(t, EnvVSCode, got)
}

func TestDetectEnvironment_Cursor(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{
			"GIT_ASKPASS": "/Applications/Cursor.app/Contents/Resources/app/extensions/git/dist/askpass.sh",
		}),
	})
	assert.Equal(t, EnvCursor, got)
}

func TestDetectEnvironment_Windsurf(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{
			"GIT_ASKPASS": "/usr/share/windsurf/resources/app/extensions/git/dist/askpass.sh",
		}),
	})
	assert.Equal(t, EnvWindsurf, got)
}

func TestDetectEnvironment_Codium(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{
			"GIT_ASKPASS": "/usr/share/codium/resources/app/extensions/git/dist/askpass.sh",
		}),
	})
	assert.Equal(t, EnvCodium, got)
}

func TestDetectEnvironment_VSCodeFallbackAskpassNode(t *testing.T) {
	// GIT_ASKPASS doesn't contain "code" but VSCODE_GIT_ASKPASS_NODE is set.
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{
			"GIT_ASKPASS":             "/some/weird/path/askpass.sh",
			"VSCODE_GIT_ASKPASS_NODE": "/usr/bin/node",
		}),
	})
	assert.Equal(t, EnvVSCode, got)
}

func TestDetectEnvironment_JetBrains(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"IDEA_INITIAL_DIRECTORY": "/Users/dev/project"}),
	})
	assert.Equal(t, EnvJetBrains, got)
}

func TestDetectEnvironment_JetBrainsJediTerm(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"TERMINAL_EMULATOR": "JetBrains-JediTerm"}),
	})
	assert.Equal(t, EnvJetBrains, got)
}

func TestDetectEnvironment_Fleet(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"FLEET_PROPERTIES_FILE": "/tmp/fleet.properties"}),
	})
	assert.Equal(t, EnvJetBrains, got) // Fleet treated as JetBrains for MVP
}

func TestDetectEnvironment_Terminal(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{}),
		IsTTY:  yesTTY(),
	})
	assert.Equal(t, EnvTerminal, got)
}

func TestDetectEnvironment_Unknown(t *testing.T) {
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{}),
		IsTTY:  noTTY(),
	})
	assert.Equal(t, EnvUnknown, got)
}

func TestDetectEnvironment_CIPriority(t *testing.T) {
	// CI takes priority even when VS Code signals are present.
	got := DetectEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{
			"CI":                      "true",
			"VSCODE_GIT_ASKPASS_NODE": "/usr/bin/node",
		}),
	})
	assert.Equal(t, EnvCI, got)
}

func TestIsVSCodeFamily(t *testing.T) {
	assert.True(t, IsVSCodeFamily(EnvVSCode))
	assert.True(t, IsVSCodeFamily(EnvCursor))
	assert.True(t, IsVSCodeFamily(EnvWindsurf))
	assert.True(t, IsVSCodeFamily(EnvCodium))
	assert.False(t, IsVSCodeFamily(EnvJetBrains))
	assert.False(t, IsVSCodeFamily(EnvTerminal))
	assert.False(t, IsVSCodeFamily(EnvCI))
}

func TestIsRemoteEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"REMOTE_CONTAINERS", map[string]string{"REMOTE_CONTAINERS": "true"}, true},
		{"CODESPACES", map[string]string{"CODESPACES": "true"}, true},
		{"VSCODE_REMOTE", map[string]string{"VSCODE_REMOTE_CONTAINERS_SESSION": "abc"}, true},
		{"normal", map[string]string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRemoteEnvironment(EnvOpts{GetEnv: envMap(tt.env)})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectIPCSocket_VSCode(t *testing.T) {
	// No live socket → empty string.
	got := DetectIPCSocket(EnvVSCode, EnvOpts{
		GetEnv:      envMap(map[string]string{"VSCODE_IPC_HOOK_CLI": "/tmp/fake.sock"}),
		DialTimeout: noDial(),
	})
	assert.Equal(t, "", got)
}

func TestDetectIPCSocket_CursorFallback(t *testing.T) {
	// Cursor-specific var empty, falls back to VSCODE_IPC_HOOK_CLI.
	// But socket is dead → empty.
	got := DetectIPCSocket(EnvCursor, EnvOpts{
		GetEnv: envMap(map[string]string{
			"CURSOR_IPC_HOOK_CLI":  "",
			"VSCODE_IPC_HOOK_CLI": "/tmp/fake.sock",
		}),
		DialTimeout: noDial(),
	})
	assert.Equal(t, "", got)
}
