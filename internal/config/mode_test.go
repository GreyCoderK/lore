// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"
)

// ═══════════════════════════════════════════════════════════════
// Story 8.4 tests: mode detection + ApplyModeOverrides
// ═══════════════════════════════════════════════════════════════

// makeDir creates a subdirectory inside t.TempDir(). Fails the test on
// error so callers can treat it as infallible.
func makeDir(t *testing.T, parent, name string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(parent, name), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", name, err)
	}
}

// TestDetectMode_LoreNative verifies that a directory containing .lore/
// is classified as lore-native, even when .git/ is also present (which
// is the normal state of a lore-initialized project).
func TestDetectMode_LoreNative(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, ".lore")
	makeDir(t, dir, ".git") // realistic: lore projects usually have git too

	cfg := &Config{} // ModeDetection empty → auto
	got := DetectMode(dir, cfg)
	if got != ModeLoreNative {
		t.Errorf("DetectMode = %s, want lore-native", got)
	}
}

// TestDetectMode_Hybrid verifies that a directory with only .git/ (no
// .lore/) is classified as hybrid. This is the common case for a mkdocs
// site versioned with git but never initialized with lore.
func TestDetectMode_Hybrid(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, ".git")

	cfg := &Config{}
	got := DetectMode(dir, cfg)
	if got != ModeHybrid {
		t.Errorf("DetectMode = %s, want hybrid", got)
	}
}

// TestDetectMode_Standalone verifies that an empty directory (no .lore/,
// no .git/) is classified as standalone. This is the CI container case:
// the user checked out source code and runs lore angela draft directly.
func TestDetectMode_Standalone(t *testing.T) {
	dir := t.TempDir()

	cfg := &Config{}
	got := DetectMode(dir, cfg)
	if got != ModeStandalone {
		t.Errorf("DetectMode = %s, want standalone", got)
	}
}

// TestDetectMode_ExplicitOverride verifies that a non-"auto" value in
// cfg.Angela.ModeDetection is respected. Uses the counter-intuitive
// combination of "standalone override in a .lore/ directory" to prove
// the override fires before filesystem probing.
func TestDetectMode_ExplicitOverride(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, ".lore") // auto-detect would return lore-native

	tests := []struct {
		value string
		want  Mode
	}{
		{"lore-native", ModeLoreNative},
		{"hybrid", ModeHybrid},
		{"standalone", ModeStandalone},
		{"LORE-NATIVE", ModeLoreNative}, // case-insensitive
		{"  hybrid  ", ModeHybrid},      // whitespace tolerated
	}

	for _, tc := range tests {
		t.Run(tc.value, func(t *testing.T) {
			cfg := &Config{}
			cfg.Angela.ModeDetection = tc.value
			got := DetectMode(dir, cfg)
			if got != tc.want {
				t.Errorf("DetectMode with %q = %s, want %s", tc.value, got, tc.want)
			}
		})
	}
}

// TestDetectMode_UnknownOverrideFallsBackToAuto verifies that a typo in
// mode_detection doesn't silently force standalone — we fall through to
// filesystem probing and log via validate.go's unknown-field detection.
func TestDetectMode_UnknownOverrideFallsBackToAuto(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, ".lore")

	cfg := &Config{}
	cfg.Angela.ModeDetection = "lore-natif" // typo
	got := DetectMode(dir, cfg)
	if got != ModeLoreNative {
		t.Errorf("typo in mode_detection should fall through to auto-detect, got %s", got)
	}
}

// TestDetectMode_AutoString verifies that "auto" and "" both trigger
// filesystem probing (equivalent behavior).
func TestDetectMode_AutoString(t *testing.T) {
	dir := t.TempDir()
	makeDir(t, dir, ".git")

	for _, val := range []string{"", "auto", "AUTO", "  auto  "} {
		t.Run(val, func(t *testing.T) {
			cfg := &Config{}
			cfg.Angela.ModeDetection = val
			got := DetectMode(dir, cfg)
			if got != ModeHybrid {
				t.Errorf("mode_detection=%q should auto-probe, got %s", val, got)
			}
		})
	}
}

// TestResolveStateDir_AllModes verifies the state directory resolution
// rules for each mode with the default (empty) StateDir config.
func TestResolveStateDir_AllModes(t *testing.T) {
	workDir := "/fake/work"
	cfg := &Config{} // StateDir empty

	tests := []struct {
		mode Mode
		want string
	}{
		{ModeLoreNative, filepath.Join(workDir, ".lore", "angela")},
		{ModeHybrid, filepath.Join(workDir, ".angela-state")},
		{ModeStandalone, filepath.Join(workDir, ".angela-state")},
	}

	for _, tc := range tests {
		t.Run(tc.mode.String(), func(t *testing.T) {
			got := ResolveStateDir(workDir, cfg, tc.mode)
			if got != tc.want {
				t.Errorf("ResolveStateDir(%s) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}

// TestResolveStateDir_ExplicitRelative verifies that an explicit relative
// StateDir value is joined with workDir.
func TestResolveStateDir_ExplicitRelative(t *testing.T) {
	cfg := &Config{}
	cfg.Angela.StateDir = "custom-state"
	got := ResolveStateDir("/fake/work", cfg, ModeLoreNative)
	want := filepath.Join("/fake/work", "custom-state")
	if got != want {
		t.Errorf("ResolveStateDir = %q, want %q", got, want)
	}
}

// TestResolveStateDir_ExplicitAbsolute verifies that an absolute
// StateDir value is returned as-is (not joined with workDir).
func TestResolveStateDir_ExplicitAbsolute(t *testing.T) {
	cfg := &Config{}
	cfg.Angela.StateDir = "/tmp/my-angela-state"
	got := ResolveStateDir("/fake/work", cfg, ModeHybrid)
	if got != "/tmp/my-angela-state" {
		t.Errorf("ResolveStateDir = %q, want absolute path unchanged", got)
	}
}

// TestApplyModeOverrides_StandaloneNonTTYPromotesToJSON verifies the
// core CI UX improvement: in standalone + non-TTY mode, draft output
// defaults to json so pipelines can parse it without a flag.
func TestApplyModeOverrides_StandaloneNonTTYPromotesToJSON(t *testing.T) {
	cfg := &Config{}
	cfg.Angela.Draft.Output.Format = "human" // at default
	ApplyModeOverrides(cfg, ModeStandalone, false /* !stdoutIsTTY */)
	if cfg.Angela.Draft.Output.Format != "json" {
		t.Errorf("expected auto-promotion to json, got %q", cfg.Angela.Draft.Output.Format)
	}
}

// TestApplyModeOverrides_StandaloneTTYStaysHuman verifies that a
// terminal user in standalone mode still gets the human format — the
// auto-promotion only fires on non-TTY.
func TestApplyModeOverrides_StandaloneTTYStaysHuman(t *testing.T) {
	cfg := &Config{}
	cfg.Angela.Draft.Output.Format = "human"
	ApplyModeOverrides(cfg, ModeStandalone, true /* stdoutIsTTY */)
	if cfg.Angela.Draft.Output.Format != "human" {
		t.Errorf("expected format to stay 'human' on TTY, got %q", cfg.Angela.Draft.Output.Format)
	}
}

// TestApplyModeOverrides_RespectsExplicitJSON verifies that a user who
// already configured format: json (or any non-default value) is not
// re-affected by the override. The rule is "auto-promote only from
// the zero-config default".
func TestApplyModeOverrides_RespectsExplicitJSON(t *testing.T) {
	cfg := &Config{}
	cfg.Angela.Draft.Output.Format = "json" // user explicitly set it
	ApplyModeOverrides(cfg, ModeStandalone, false)
	if cfg.Angela.Draft.Output.Format != "json" {
		t.Errorf("should stay json, got %q", cfg.Angela.Draft.Output.Format)
	}
}

// TestApplyModeOverrides_LoreNativeNoChanges verifies that lore-native
// mode does not mutate the config — the whole point of mode detection
// is that lore-native is the canonical path.
func TestApplyModeOverrides_LoreNativeNoChanges(t *testing.T) {
	cfg := &Config{}
	cfg.Angela.Draft.Output.Format = "human"
	ApplyModeOverrides(cfg, ModeLoreNative, false)
	if cfg.Angela.Draft.Output.Format != "human" {
		t.Errorf("lore-native should not override format, got %q", cfg.Angela.Draft.Output.Format)
	}
}

// TestParseModeString_AllVariants exercises the normalization rules
// (case-insensitive, whitespace, known aliases).
func TestParseModeString_AllVariants(t *testing.T) {
	tests := []struct {
		in   string
		want Mode
		ok   bool
	}{
		{"lore-native", ModeLoreNative, true},
		{"LORE-NATIVE", ModeLoreNative, true},
		{"lore_native", ModeLoreNative, true},
		{"lorenative", ModeLoreNative, true},
		{"hybrid", ModeHybrid, true},
		{"  Hybrid  ", ModeHybrid, true},
		{"standalone", ModeStandalone, true},
		{"banana", ModeStandalone, false}, // unknown → safest fallback
		{"", ModeStandalone, false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := parseModeString(tc.in)
			if got != tc.want || ok != tc.ok {
				t.Errorf("parseModeString(%q) = (%s, %v), want (%s, %v)",
					tc.in, got, ok, tc.want, tc.ok)
			}
		})
	}
}

// TestMode_String verifies the canonical string mapping used by log
// messages and test assertions.
func TestMode_String(t *testing.T) {
	tests := []struct {
		m    Mode
		want string
	}{
		{ModeLoreNative, "lore-native"},
		{ModeHybrid, "hybrid"},
		{ModeStandalone, "standalone"},
		{Mode(99), "unknown"},
	}
	for _, tc := range tests {
		if got := tc.m.String(); got != tc.want {
			t.Errorf("Mode(%d).String() = %q, want %q", tc.m, got, tc.want)
		}
	}
}
