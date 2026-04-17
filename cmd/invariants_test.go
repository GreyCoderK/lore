// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariants I1–I3 enforcement at the command layer.
//
// These tests exist so the matrix in
// _bmad-output/test-artifacts/invariants-coverage-matrix.md can point at
// named `TestI[1-3]_*` functions. They are deliberately thin: the heavy
// lifting is in the dedicated per-feature test suites (e.g. the preview
// countingTransport machinery). The goal here is a fast, focused runtime
// gate that catches accidental regressions at the command dispatch layer.
// ═══════════════════════════════════════════════════════════════════════════

// TestI1_DraftMakesZeroHTTPCalls — I1 runtime layer, command dispatch path.
//
// Wraps `http.DefaultTransport` with a counting RoundTripper and runs the
// draft cobra command through its normal flag-parse + RunE cycle against a
// tiny stand-alone corpus. Any HTTP call during draft is an I1 violation.
//
// Complements:
//   - internal/config: compile-time `ValidateDraftOfflineInvariant()` via
//     reflection on DraftConfig (rejects any field with AI/Provider/Endpoint
//     markers).
//   - cmd/angela_review_preview_test.go: countingTransport already proves
//     --preview is HTTP-free. This test proves DRAFT is too (separate code
//     path, could have its own regressions).
func TestI1_DraftMakesZeroHTTPCalls(t *testing.T) {
	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()
	counter := &countingTransport{}
	http.DefaultTransport = counter

	docsDir := writeFiveDocs(t)
	streams, _, _ := streamsForPreview()

	cfg := &config.Config{}
	path := docsDir
	cmd := newAngelaDraftCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--all", "--quiet"})
	// Draft's RunE may return an error for corpus-level findings; that's fine.
	// What matters for I1 is that no HTTP call fires, regardless of exit code.
	_ = cmd.Execute()

	if got := atomic.LoadInt64(&counter.count); got != 0 {
		t.Errorf("I1 violation: draft made %d HTTP call(s); draft must stay offline", got)
	}
}

// TestI2_DualMode_DetectsAllThreeModes — I2 runtime layer.
//
// Uses real temp directories to exercise DetectMode over the three supported
// repository shapes: standalone (no .lore/, no .git/), hybrid (.git/ only),
// and lore-native (.lore/ + .git/). Uses config.DetectMode directly — the
// command layer should never second-guess this; if DetectMode returns the
// right mode, the command dispatch downstream is a plain switch.
//
// Complements internal/config/mode_test.go which already covers the three
// cases at the unit layer. This test doubles as the matrix anchor so
// invariants-coverage-matrix.md can cite `TestI2_*`.
func TestI2_DualMode_DetectsAllThreeModes(t *testing.T) {
	cases := []struct {
		name      string
		setup     func(t *testing.T) string
		wantMode  config.Mode
	}{
		{
			name: "standalone (no .lore, no .git)",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Just some markdown so corpus reader would find something.
				writeFile(t, filepath.Join(dir, "README.md"), "# standalone\n")
				return dir
			},
			wantMode: config.ModeStandalone,
		},
		{
			name: "hybrid (.git only)",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
					t.Fatalf("mkdir .git: %v", err)
				}
				return dir
			},
			wantMode: config.ModeHybrid,
		},
		{
			name: "lore-native (.lore + .git)",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
					t.Fatalf("mkdir .git: %v", err)
				}
				if err := os.MkdirAll(filepath.Join(dir, ".lore"), 0o755); err != nil {
					t.Fatalf("mkdir .lore: %v", err)
				}
				return dir
			},
			wantMode: config.ModeLoreNative,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := tc.setup(t)
			got := config.DetectMode(dir, &config.Config{})
			if got != tc.wantMode {
				t.Errorf("DetectMode(%s) = %v, want %v", tc.name, got, tc.wantMode)
			}
		})
	}
}

// TestI3_ZeroConfig_DraftPreviewBaseline — I3 runtime layer, command dispatch.
//
// Runs the two commands most likely to touch config defaults (`angela draft`
// and `angela review --preview`) with a completely empty config.Config{}
// and zero `.lorerc`. The commands must either succeed or fail cleanly —
// never panic on a nil map access or missing default. Panics in test
// helpers surface as test failures, so this doubles as a smoke test for
// the zero-config contract.
//
// Complements internal/config/angela_mvp_test.go:
// TestLoadFromDir_AngelaDefaults_ZeroConfig (loads defaults) and
// cmd/angela_review_preview_test.go:TestReviewPreview_NoPanicOnZeroConfig
// (preview acceptance). This version exercises the full cobra dispatch
// path, not just the inner function.
func TestI3_ZeroConfig_DraftPreviewBaseline(t *testing.T) {
	t.Run("draft --all with empty config", func(t *testing.T) {
		docsDir := writeFiveDocs(t)
		streams, _, _ := streamsForPreview()
		cfg := &config.Config{} // entirely empty — exercises defaults
		path := docsDir
		cmd := newAngelaDraftCmd(cfg, streams, &path)
		cmd.SetArgs([]string{"--all", "--quiet"})
		// Error tolerated; panic = I3 violation (caught by test harness).
		_ = cmd.Execute()
	})

	t.Run("review --preview with empty config", func(t *testing.T) {
		docsDir := writeFiveDocs(t)
		streams, _, _ := streamsForPreview()
		cfg := &config.Config{}
		path := docsDir
		cmd := newAngelaReviewCmd(cfg, streams, &path)
		cmd.SetArgs([]string{"--preview"})
		if err := cmd.Execute(); err != nil {
			t.Errorf("review --preview must succeed on zero config, got: %v", err)
		}
	})
}

// writeFile is a tiny helper so the I2 setup stays declarative. The broader
// test suite already has writeFiveDocs; this one writes a single file.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// suppress unused-import warning when the test file compiles without the
// streams.IOStreams being referenced directly. The existing test harness
// re-exports streamsForPreview which uses domain.IOStreams.
var _ = domain.IOStreams{}
