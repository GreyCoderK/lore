// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I10 — Config cascade per-field.
//
// Contract: for every config field, the effective value resolves via a
// STRICT 5-level precedence chain (highest to lowest wins):
//
//   1. CLI flag (e.g. --ai-provider, --language) — highest, explicit per run
//   2. Environment variable (LORE_AI_PROVIDER, etc.) — user-runtime override
//   3. .lorerc.local — personal, gitignored (API keys, host-specific)
//   4. .lorerc — team-shared, version-controlled
//   5. Defaults — baked into setDefaults(v)
//
// A regression in cascade semantics would be invisible to users until they
// hit the exact scenario (flag + env, or env + local, etc.), at which
// point behavior would diverge unpredictably across machines. These tests
// assert the precedence holds for the 2 bound flags (--ai-provider,
// --language) and for a sample of non-flag fields via a table-driven
// cascade (defaults → lorerc → local → env).
//
// Existing companion tests (config_test.go):
//   - TestLoadFromDir_DefaultsOnly, TestLoadFromDir_LorercOverridesDefaults,
//     TestLoadFromDir_LocalOverridesLorerc, TestLoadFromDir_EnvOverridesLocal
//     cover each pairwise transition.
//   - TestLoadFromDirWithFlags_FullCascade covers the 5-level end-to-end
//     for ai.provider/model/api_key/mode.
//
// These TestI10_* anchors give invariants-coverage-matrix.md a stable
// pointer AND extend the coverage with a table-driven sweep across more
// fields and an anti-regression reflection check.
// ═══════════════════════════════════════════════════════════════════════════

// TestI10_FullCascade_BoundFlagWinsOverAll is the explicit-named anchor
// for the 5-level cascade on ai.provider (the canonical bound flag). If
// the flag path regresses, the test names this path directly.
func TestI10_FullCascade_BoundFlagWinsOverAll(t *testing.T) {
	dir := t.TempDir()
	// Level 2 (.lorerc): provider = ollama
	mustWrite(t, filepath.Join(dir, ".lorerc"),
		"ai:\n  provider: ollama\n")
	// Level 3 (.lorerc.local): provider = anthropic
	mustWrite(t, filepath.Join(dir, ".lorerc.local"),
		"ai:\n  provider: anthropic\n")
	// Level 4 (env): provider = azure
	t.Setenv("LORE_AI_PROVIDER", "azure")

	// Level 5 (flag): provider = openai
	cmd := &cobra.Command{}
	RegisterFlags(cmd)
	cmd.SetArgs([]string{"--ai-provider", "openai"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	cfg, err := LoadFromDirWithFlags(dir, cmd)
	if err != nil {
		t.Fatalf("LoadFromDirWithFlags: %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("I10 violation: flag should win, got provider=%q (expected openai)", cfg.AI.Provider)
	}
}

// TestI10_FullCascade_LanguageFlagWins — same cascade on the second bound
// flag. Distinct test so a regression on --language doesn't get masked by
// a passing --ai-provider test.
func TestI10_FullCascade_LanguageFlagWins(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".lorerc"), "language: en\n")
	mustWrite(t, filepath.Join(dir, ".lorerc.local"), "language: en\n")
	t.Setenv("LORE_LANGUAGE", "en")

	cmd := &cobra.Command{}
	RegisterFlags(cmd)
	cmd.SetArgs([]string{"--language", "fr"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute: %v", err)
	}

	cfg, err := LoadFromDirWithFlags(dir, cmd)
	if err != nil {
		t.Fatalf("LoadFromDirWithFlags: %v", err)
	}
	if cfg.Language != "fr" {
		t.Errorf("I10 violation: --language flag should win, got %q (expected fr)", cfg.Language)
	}
}

// TestI10_CascadeWithoutFlag_EnvWinsOverLocalAndLorerc — table-driven
// proof that for fields without a bound flag, the cascade still holds
// across the other 4 levels. Tests a mix of nested keys (ai.model,
// output.format, angela.mode) so the env-var underscore translation
// (LORE_AI_MODEL → ai.model) gets exercised on each.
func TestI10_CascadeWithoutFlag_EnvWinsOverLocalAndLorerc(t *testing.T) {
	cases := []struct {
		name        string
		yamlLorerc  string
		yamlLocal   string
		envKey      string
		envValue    string
		getActual   func(*Config) string
		wantFromEnv string
	}{
		{
			name:        "ai.model",
			yamlLorerc:  "ai:\n  model: claude-opus\n",
			yamlLocal:   "ai:\n  model: claude-sonnet\n",
			envKey:      "LORE_AI_MODEL",
			envValue:    "gpt-4",
			getActual:   func(c *Config) string { return c.AI.Model },
			wantFromEnv: "gpt-4",
		},
		{
			name:        "output.format",
			yamlLorerc:  "output:\n  format: text\n",
			yamlLocal:   "output:\n  format: markdown\n",
			envKey:      "LORE_OUTPUT_FORMAT",
			envValue:    "json",
			getActual:   func(c *Config) string { return c.Output.Format },
			wantFromEnv: "json",
		},
		{
			name:        "angela.mode",
			yamlLorerc:  "angela:\n  mode: draft\n",
			yamlLocal:   "angela:\n  mode: polish\n",
			envKey:      "LORE_ANGELA_MODE",
			envValue:    "review",
			getActual:   func(c *Config) string { return c.Angela.Mode },
			wantFromEnv: "review",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mustWrite(t, filepath.Join(dir, ".lorerc"), tc.yamlLorerc)
			mustWrite(t, filepath.Join(dir, ".lorerc.local"), tc.yamlLocal)
			t.Setenv(tc.envKey, tc.envValue)

			cfg, err := LoadFromDir(dir)
			if err != nil {
				t.Fatalf("LoadFromDir: %v", err)
			}
			if got := tc.getActual(cfg); got != tc.wantFromEnv {
				t.Errorf("I10 violation on %s: got %q, want %q (env level must beat local and lorerc)",
					tc.name, got, tc.wantFromEnv)
			}
		})
	}
}

// TestI10_Cascade_LocalOverridesLorerc_MultipleFields — same table-driven
// shape, one level down. Critical because this is the split between
// "shared" (.lorerc, checked in) and "personal" (.lorerc.local, gitignored
// for API keys): a regression here means team-members see API keys from
// their teammate's local config, or worse, share them unintentionally.
func TestI10_Cascade_LocalOverridesLorerc_MultipleFields(t *testing.T) {
	cases := []struct {
		name      string
		yamlRC    string
		yamlLocal string
		getActual func(*Config) string
		want      string
	}{
		{
			name:      "ai.api_key (the canonical .lorerc.local secret)",
			yamlRC:    "ai:\n  api_key: team-key\n",
			yamlLocal: "ai:\n  api_key: my-personal-key\n",
			getActual: func(c *Config) string { return c.AI.APIKey },
			want:      "my-personal-key",
		},
		{
			name:      "ai.endpoint (personal proxy / localhost override)",
			yamlRC:    "ai:\n  endpoint: https://api.example.com\n",
			yamlLocal: "ai:\n  endpoint: http://localhost:8080\n",
			getActual: func(c *Config) string { return c.AI.Endpoint },
			want:      "http://localhost:8080",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			mustWrite(t, filepath.Join(dir, ".lorerc"), tc.yamlRC)
			mustWrite(t, filepath.Join(dir, ".lorerc.local"), tc.yamlLocal)

			cfg, err := LoadFromDir(dir)
			if err != nil {
				t.Fatalf("LoadFromDir: %v", err)
			}
			if got := tc.getActual(cfg); got != tc.want {
				t.Errorf("I10 violation on %s: got %q, want %q (local must beat lorerc)",
					tc.name, got, tc.want)
			}
		})
	}
}

// TestI10_NoConfigFiles_DefaultsUsed — bottom of the cascade. A fresh dir
// with no .lorerc / no .lorerc.local / no env vars must yield defaults
// across every non-empty field. This is both an I3 (zero-config) assertion
// AND an I10 bottom-of-stack assertion.
func TestI10_NoConfigFiles_DefaultsUsed(t *testing.T) {
	dir := t.TempDir()
	// Explicitly unset env vars a prior test might have leaked.
	t.Setenv("LORE_AI_PROVIDER", "")
	t.Setenv("LORE_AI_MODEL", "")
	t.Setenv("LORE_LANGUAGE", "")

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}

	// Defaults are set in setDefaults(). We pick a few that have known
	// non-empty defaults per defaults.go:
	//   - language: "en"
	//   - output.format: "markdown"
	//   - ai.endpoint: "http://localhost:11434"
	// Avoid angela.mode: it's a deprecated field with no default anymore
	// (CHANGELOG [Unreleased] notes the removal).
	if cfg.Language != "en" {
		t.Errorf("I10 bottom: Language = %q, want %q (default)", cfg.Language, "en")
	}
	if cfg.Output.Format != "markdown" {
		t.Errorf("I10 bottom: Output.Format = %q, want %q (default)", cfg.Output.Format, "markdown")
	}
	if cfg.AI.Endpoint != "http://localhost:11434" {
		t.Errorf("I10 bottom: AI.Endpoint = %q, want default", cfg.AI.Endpoint)
	}
	// Bool defaults: hooks.post_commit is true by default.
	if !cfg.Hooks.PostCommit {
		t.Error("I10 bottom: Hooks.PostCommit = false, want true (default)")
	}
}

// mustWrite is a tiny fatal-on-error file writer so the table-driven tests
// stay readable.
func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
