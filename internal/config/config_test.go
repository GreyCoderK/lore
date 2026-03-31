// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestLoadFromDir_DefaultsOnly(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load defaults: %v", err)
	}
	if cfg.AI.Provider != "" {
		t.Errorf("expected provider '', got %q", cfg.AI.Provider)
	}
	if cfg.AI.Model != "" {
		t.Errorf("expected model '', got %q", cfg.AI.Model)
	}
	if cfg.Angela.Mode != "draft" {
		t.Errorf("expected angela.mode 'draft', got %q", cfg.Angela.Mode)
	}
	if cfg.Angela.MaxTokens != 2000 {
		t.Errorf("expected angela.max_tokens 2000, got %d", cfg.Angela.MaxTokens)
	}
	expectedTplDir := filepath.Join(".lore", "templates")
	if cfg.Templates.Dir != expectedTplDir {
		t.Errorf("expected templates.dir %q, got %q", expectedTplDir, cfg.Templates.Dir)
	}
	if !cfg.Hooks.PostCommit {
		t.Error("expected hooks.post_commit=true by default")
	}
	if cfg.Output.Format != "markdown" {
		t.Errorf("expected format 'markdown', got %q", cfg.Output.Format)
	}
	expectedOutDir := filepath.Join(".lore", "docs")
	if cfg.Output.Dir != expectedOutDir {
		t.Errorf("expected output.dir %q, got %q", expectedOutDir, cfg.Output.Dir)
	}
	if cfg.AI.Endpoint != "http://localhost:11434" {
		t.Errorf("expected ai.endpoint 'http://localhost:11434', got %q", cfg.AI.Endpoint)
	}
	if cfg.AI.Timeout != 30*time.Second {
		t.Errorf("expected ai.timeout 30s, got %v", cfg.AI.Timeout)
	}
}

func TestLoadFromDir_LorercOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: anthropic\noutput:\n  format: html\n"), 0644); err != nil {
		t.Fatalf("WriteFile .lorerc: %v", err)
	}

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load from .lorerc: %v", err)
	}
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic' from .lorerc, got %q", cfg.AI.Provider)
	}
	if cfg.Output.Format != "html" {
		t.Errorf("expected format 'html' from .lorerc, got %q", cfg.Output.Format)
	}
	if cfg.Angela.Mode != "draft" {
		t.Errorf("expected angela.mode 'draft' (default), got %q", cfg.Angela.Mode)
	}
}

func TestLoadFromDir_LocalOverridesLorerc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: ollama\n  model: llama3\n"), 0644); err != nil {
		t.Fatalf("WriteFile .lorerc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".lorerc.local"), []byte("ai:\n  provider: openai\n"), 0644); err != nil {
		t.Fatalf("WriteFile .lorerc.local: %v", err)
	}

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load with local override: %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected provider 'openai' from .lorerc.local, got %q", cfg.AI.Provider)
	}
}

func TestLoadFromDir_EnvOverridesLocal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: ollama\n"), 0644); err != nil {
		t.Fatalf("WriteFile .lorerc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".lorerc.local"), []byte("ai:\n  provider: openai\n"), 0644); err != nil {
		t.Fatalf("WriteFile .lorerc.local: %v", err)
	}

	t.Setenv("LORE_AI_PROVIDER", "azure")

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load with env override: %v", err)
	}
	if cfg.AI.Provider != "azure" {
		t.Errorf("expected provider 'azure' from env, got %q", cfg.AI.Provider)
	}
}

func TestLoadFromDir_EnvOverrideNestedKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LORE_OUTPUT_FORMAT", "json")

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load with env: %v", err)
	}
	if cfg.Output.Format != "json" {
		t.Errorf("expected format 'json', got %q", cfg.Output.Format)
	}
}

func TestLoadFromDir_EnvMultipleOverrides(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LORE_AI_PROVIDER", "openai")
	t.Setenv("LORE_AI_MODEL", "gpt-4")
	t.Setenv("LORE_ANGELA_MODE", "review")

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load with env: %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got %q", cfg.AI.Model)
	}
	if cfg.Angela.Mode != "review" {
		t.Errorf("expected angela.mode 'review', got %q", cfg.Angela.Mode)
	}
	if cfg.Output.Format != "markdown" {
		t.Errorf("expected format 'markdown' (default), got %q", cfg.Output.Format)
	}
}

func TestLoadFromDir_MissingFilesGraceful(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load with no files: %v", err)
	}
	if cfg.Angela.Mode != "draft" {
		t.Errorf("expected defaults, got angela.mode=%q", cfg.Angela.Mode)
	}
}

func TestLoadFromDir_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("invalid:\n  yaml: [broken"), 0644); err != nil {
		t.Fatalf("WriteFile .lorerc: %v", err)
	}

	_, err := LoadFromDir(dir)
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "config: read .lorerc") {
		t.Errorf("expected error to contain 'config: read .lorerc', got %q", errMsg)
	}
}

func TestLoadFromDirWithFlags_FlagOverridesEnv(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: ollama\n"), 0644)
	t.Setenv("LORE_AI_PROVIDER", "anthropic")

	cmd := &cobra.Command{}
	RegisterFlags(cmd)
	cmd.SetArgs([]string{"--ai-provider", "openai"})
	_ = cmd.Execute()

	cfg, err := LoadFromDirWithFlags(dir, cmd)
	if err != nil {
		t.Fatalf("config: load with flags: %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected provider 'openai' from flag, got %q", cfg.AI.Provider)
	}
}

func TestLoadFromDirWithFlags_FullCascade(t *testing.T) {
	dir := t.TempDir()
	// Level 1: defaults (angela.mode=draft, output.format=markdown)
	// Level 2: .lorerc overrides ai.provider
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("ai:\n  provider: ollama\n  model: llama3\nangela:\n  mode: review\n"), 0644)
	// Level 3: .lorerc.local overrides ai.provider
	os.WriteFile(filepath.Join(dir, ".lorerc.local"), []byte("ai:\n  provider: anthropic\n  api_key: sk-test\n"), 0644)
	// Level 4: env var overrides ai.model
	t.Setenv("LORE_AI_MODEL", "gpt-4")
	// Level 5: CLI flag overrides ai.provider
	cmd := &cobra.Command{}
	RegisterFlags(cmd)
	cmd.SetArgs([]string{"--ai-provider", "openai"})
	_ = cmd.Execute()

	cfg, err := LoadFromDirWithFlags(dir, cmd)
	if err != nil {
		t.Fatalf("config: full cascade: %v", err)
	}

	// Flag wins over env+local+lorerc for ai.provider
	if cfg.AI.Provider != "openai" {
		t.Errorf("AI.Provider: expected 'openai' (flag), got %q", cfg.AI.Provider)
	}
	// Env wins for ai.model (over .lorerc llama3)
	if cfg.AI.Model != "gpt-4" {
		t.Errorf("AI.Model: expected 'gpt-4' (env), got %q", cfg.AI.Model)
	}
	// .lorerc.local wins for api_key
	if cfg.AI.APIKey != "sk-test" {
		t.Errorf("AI.APIKey: expected 'sk-test' (local), got %q", cfg.AI.APIKey)
	}
	// .lorerc wins for angela.mode (over default draft)
	if cfg.Angela.Mode != "review" {
		t.Errorf("Angela.Mode: expected 'review' (lorerc), got %q", cfg.Angela.Mode)
	}
	// Default wins for output.format (nothing overrides)
	if cfg.Output.Format != "markdown" {
		t.Errorf("Output.Format: expected 'markdown' (default), got %q", cfg.Output.Format)
	}
}

func TestLoadFromDir_ConfigStructFullyPopulated(t *testing.T) {
	dir := t.TempDir()
	lorercContent := `ai:
  provider: anthropic
  model: claude-sonnet-4-20250514
  api_key: ""
angela:
  mode: full
  max_tokens: 4000
templates:
  dir: custom/templates
hooks:
  post_commit: false
output:
  format: html
  dir: custom/docs
`
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte(lorercContent), 0644)

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("config: load: %v", err)
	}
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("AI.Provider: got %q", cfg.AI.Provider)
	}
	if cfg.Angela.Mode != "full" {
		t.Errorf("Angela.Mode: got %q", cfg.Angela.Mode)
	}
	if cfg.Angela.MaxTokens != 4000 {
		t.Errorf("Angela.MaxTokens: got %d", cfg.Angela.MaxTokens)
	}
	if cfg.Templates.Dir != "custom/templates" {
		t.Errorf("Templates.Dir: got %q", cfg.Templates.Dir)
	}
	if cfg.Hooks.PostCommit {
		t.Error("Hooks.PostCommit: expected false")
	}
	if cfg.Output.Format != "html" {
		t.Errorf("Output.Format: got %q", cfg.Output.Format)
	}
	if cfg.Output.Dir != "custom/docs" {
		t.Errorf("Output.Dir: got %q", cfg.Output.Dir)
	}
}

func TestLoadFromDir_UnknownFieldWarnsToStderr(t *testing.T) {
	dir := t.TempDir()
	lorercContent := `ai:
  provider: anthropic
  providerr: typo-value
`
	os.WriteFile(filepath.Join(dir, ".lorerc"), []byte(lorercContent), 0644)

	// Capture warnings via WarnWriter instead of redirecting os.Stderr.
	var buf bytes.Buffer
	origWriter := WarnWriter
	WarnWriter = &buf
	t.Cleanup(func() { WarnWriter = origWriter })

	cfg, err := LoadFromDir(dir)

	// Config should still load successfully (non-fatal).
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %q", cfg.AI.Provider)
	}

	// A warning about the unknown field should have been printed.
	warning := buf.String()
	if !strings.Contains(warning, "Warning") {
		t.Errorf("expected warning on stderr, got %q", warning)
	}
	if !strings.Contains(warning, "providerr") {
		t.Errorf("expected warning to mention 'providerr', got %q", warning)
	}
}

func TestLoadFromDir_LanguageDefault(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("Language = %q, want 'en' (default)", cfg.Language)
	}
}

func TestLoadFromDir_LanguageFromLorerc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("language: fr\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if cfg.Language != "fr" {
		t.Errorf("Language = %q, want 'fr' from .lorerc", cfg.Language)
	}
}

func TestLoadFromDir_LanguageFromEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LORE_LANGUAGE", "de")
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if cfg.Language != "de" {
		t.Errorf("Language = %q, want 'de' from env LORE_LANGUAGE", cfg.Language)
	}
}

func TestLoadFromDir_LanguageEnvOverridesLorerc(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte("language: fr\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("LORE_LANGUAGE", "es")
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("LoadFromDir: %v", err)
	}
	if cfg.Language != "es" {
		t.Errorf("Language = %q, want 'es' (env overrides .lorerc)", cfg.Language)
	}
}
