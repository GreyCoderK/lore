// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- isWSLWithoutDisplay tests ---

func TestIsWSLWithoutDisplay_NotWSL(t *testing.T) {
	// On non-WSL systems (e.g. macOS, native Linux without WSL), the
	// /proc/sys/fs/binfmt_misc/WSLInterop file does not exist, so the
	// function should return false regardless of env vars.
	opts := EnvOpts{GetEnv: envMap(map[string]string{})}
	got := isWSLWithoutDisplay(&opts)
	assert.False(t, got, "should be false when WSLInterop file does not exist")
}

func TestIsWSLWithoutDisplay_NotWSL_WithDisplay(t *testing.T) {
	// Even with DISPLAY set, if WSLInterop doesn't exist, still false.
	opts := EnvOpts{GetEnv: envMap(map[string]string{"DISPLAY": ":0"})}
	got := isWSLWithoutDisplay(&opts)
	assert.False(t, got, "should be false when WSLInterop file does not exist even with DISPLAY")
}

func TestIsWSLWithoutDisplay_NotWSL_WithWayland(t *testing.T) {
	opts := EnvOpts{GetEnv: envMap(map[string]string{"WAYLAND_DISPLAY": "wayland-0"})}
	got := isWSLWithoutDisplay(&opts)
	assert.False(t, got, "should be false when WSLInterop file does not exist even with WAYLAND_DISPLAY")
}

// TestIsWSLWithoutDisplay_SimulatedWSL tests the WSL path by creating a fake
// WSLInterop file (only works on Linux where /proc is writable, skips otherwise).
func TestIsWSLWithoutDisplay_SimulatedWSL(t *testing.T) {
	// This test only works if we can create the WSLInterop sentinel file.
	// On macOS and most CI, /proc doesn't exist or isn't writable.
	wslPath := "/proc/sys/fs/binfmt_misc/WSLInterop"
	if _, err := os.Stat("/proc/sys/fs/binfmt_misc"); os.IsNotExist(err) {
		t.Skip("skipping WSL simulation: /proc/sys/fs/binfmt_misc does not exist")
	}

	// Try to create the file; if it fails (permissions), skip.
	f, err := os.Create(wslPath)
	if err != nil {
		t.Skip("skipping WSL simulation: cannot create WSLInterop file")
	}
	f.Close()
	defer os.Remove(wslPath)

	// No DISPLAY and no WAYLAND_DISPLAY -> true
	opts := EnvOpts{GetEnv: envMap(map[string]string{})}
	got := isWSLWithoutDisplay(&opts)
	assert.True(t, got, "should be true in simulated WSL without display")

	// With DISPLAY -> false
	opts2 := EnvOpts{GetEnv: envMap(map[string]string{"DISPLAY": ":0"})}
	got2 := isWSLWithoutDisplay(&opts2)
	assert.False(t, got2, "should be false in WSL with DISPLAY set")

	// With WAYLAND_DISPLAY -> false
	opts3 := EnvOpts{GetEnv: envMap(map[string]string{"WAYLAND_DISPLAY": "wayland-0"})}
	got3 := isWSLWithoutDisplay(&opts3)
	assert.False(t, got3, "should be false in WSL with WAYLAND_DISPLAY set")
}

// --- IsRemoteEnvironment additional coverage ---

func TestIsRemoteEnvironment_WSLWithoutDisplay(t *testing.T) {
	// On non-WSL: even with no DISPLAY, IsRemoteEnvironment returns false
	// because isWSLWithoutDisplay checks the WSLInterop file first.
	got := IsRemoteEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{}),
	})
	assert.False(t, got, "should be false on non-WSL without remote container env vars")
}

func TestIsRemoteEnvironment_RemoteContainersAndWSL(t *testing.T) {
	// REMOTE_CONTAINERS takes priority over WSL check
	got := IsRemoteEnvironment(EnvOpts{
		GetEnv: envMap(map[string]string{"REMOTE_CONTAINERS": "true"}),
	})
	assert.True(t, got)
}

// --- DetectIPCSocket additional coverage ---

func TestDetectIPCSocket_Windsurf(t *testing.T) {
	got := DetectIPCSocket(EnvWindsurf, EnvOpts{
		GetEnv: envMap(map[string]string{
			"WINDSURF_IPC_HOOK_CLI": "/tmp/windsurf.sock",
			"VSCODE_IPC_HOOK_CLI":  "/tmp/vscode.sock",
		}),
		DialTimeout: noDial(),
	})
	assert.Equal(t, "", got, "dead sockets should return empty")
}

func TestDetectIPCSocket_Codium(t *testing.T) {
	got := DetectIPCSocket(EnvCodium, EnvOpts{
		GetEnv: envMap(map[string]string{
			"VSCODIUM_IPC_HOOK_CLI": "/tmp/codium.sock",
		}),
		DialTimeout: noDial(),
	})
	assert.Equal(t, "", got, "dead sockets should return empty")
}

func TestDetectIPCSocket_NoEnvVars(t *testing.T) {
	got := DetectIPCSocket(EnvVSCode, EnvOpts{
		GetEnv:      envMap(map[string]string{}),
		DialTimeout: noDial(),
	})
	assert.Equal(t, "", got, "no env vars should return empty")
}

// --- isCI additional coverage ---

func TestIsCI_NoVars(t *testing.T) {
	opts := EnvOpts{GetEnv: envMap(map[string]string{})}
	assert.False(t, isCI(&opts))
}

func TestIsCI_BuildkiteOnly(t *testing.T) {
	opts := EnvOpts{GetEnv: envMap(map[string]string{"BUILDKITE": "true"})}
	assert.True(t, isCI(&opts))
}

// --- EnvOpts default methods ---

func TestEnvOpts_GetEnv_DefaultsToOsGetenv(t *testing.T) {
	opts := EnvOpts{}
	// Should not panic and return the actual env value (likely empty for a random key)
	_ = opts.getEnv("LORE_TEST_NONEXISTENT_VAR_12345")
}

func TestEnvOpts_DialTimeout_Default(t *testing.T) {
	opts := EnvOpts{}
	// Attempting to dial a non-existent socket should return an error, not panic
	_, err := opts.dialTimeout("unix", "/tmp/nonexistent-lore-test-sock-12345.sock", 1)
	assert.Error(t, err)
}
