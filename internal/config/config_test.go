package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load defaults: %v", err)
	}
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got '%s'", cfg.AI.Provider)
	}
	if cfg.Output.Format != "markdown" {
		t.Errorf("expected format 'markdown', got '%s'", cfg.Output.Format)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("LORE_AI_PROVIDER", "openai")
	defer os.Unsetenv("LORE_AI_PROVIDER")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load with env: %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", cfg.AI.Provider)
	}
}

func TestEnvOverrideNestedKey(t *testing.T) {
	os.Setenv("LORE_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("LORE_OUTPUT_FORMAT")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load with env: %v", err)
	}
	if cfg.Output.Format != "json" {
		t.Errorf("expected format 'json', got '%s'", cfg.Output.Format)
	}
}

func TestLoadDefaultsAllSections(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load defaults: %v", err)
	}
	if cfg.AI.Model != "claude-sonnet-4-20250514" {
		t.Errorf("expected model 'claude-sonnet-4-20250514', got '%s'", cfg.AI.Model)
	}
	if !cfg.Angela.Enabled {
		t.Error("expected angela.enabled=true by default")
	}
	if cfg.Angela.Mode != "draft" {
		t.Errorf("expected angela.mode 'draft', got '%s'", cfg.Angela.Mode)
	}
	if cfg.Templates.Dir != ".lore/templates" {
		t.Errorf("expected templates.dir '.lore/templates', got '%s'", cfg.Templates.Dir)
	}
	if !cfg.Hooks.PostCommit {
		t.Error("expected hooks.post_commit=true by default")
	}
	if cfg.Output.Dir != ".lore/docs" {
		t.Errorf("expected output.dir '.lore/docs', got '%s'", cfg.Output.Dir)
	}
}

func TestEnvOverridePriority(t *testing.T) {
	// Set multiple env vars simultaneously to verify cascade
	os.Setenv("LORE_AI_PROVIDER", "openai")
	os.Setenv("LORE_AI_MODEL", "gpt-4")
	os.Setenv("LORE_ANGELA_MODE", "review")
	defer func() {
		os.Unsetenv("LORE_AI_PROVIDER")
		os.Unsetenv("LORE_AI_MODEL")
		os.Unsetenv("LORE_ANGELA_MODE")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load with env: %v", err)
	}
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", cfg.AI.Provider)
	}
	if cfg.AI.Model != "gpt-4" {
		t.Errorf("expected model 'gpt-4', got '%s'", cfg.AI.Model)
	}
	if cfg.Angela.Mode != "review" {
		t.Errorf("expected angela.mode 'review', got '%s'", cfg.Angela.Mode)
	}
	// Non-overridden defaults should survive
	if cfg.Output.Format != "markdown" {
		t.Errorf("expected format 'markdown' (default), got '%s'", cfg.Output.Format)
	}
}

func TestLoadFromLorercFile(t *testing.T) {
	tmpDir := t.TempDir()
	lorercPath := filepath.Join(tmpDir, ".lorerc")
	os.WriteFile(lorercPath, []byte("ai:\n  provider: ollama\noutput:\n  format: html\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load from .lorerc: %v", err)
	}
	if cfg.AI.Provider != "ollama" {
		t.Errorf("expected provider 'ollama' from .lorerc, got '%s'", cfg.AI.Provider)
	}
	if cfg.Output.Format != "html" {
		t.Errorf("expected format 'html' from .lorerc, got '%s'", cfg.Output.Format)
	}
	// Defaults for unspecified keys should survive
	if cfg.Angela.Mode != "draft" {
		t.Errorf("expected angela.mode 'draft' (default), got '%s'", cfg.Angela.Mode)
	}
}

func TestLorercLocalOverridesLorerc(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".lorerc"), []byte("ai:\n  provider: ollama\n  model: llama3\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, ".lorerc.local"), []byte("ai:\n  provider: openai\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load with local override: %v", err)
	}
	// .lorerc.local overrides .lorerc
	if cfg.AI.Provider != "openai" {
		t.Errorf("expected provider 'openai' from .lorerc.local, got '%s'", cfg.AI.Provider)
	}
}

func TestEnvOverridesFiles(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, ".lorerc"), []byte("ai:\n  provider: ollama\n"), 0644)

	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.Setenv("LORE_AI_PROVIDER", "azure")
	defer os.Unsetenv("LORE_AI_PROVIDER")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("config: load with env over file: %v", err)
	}
	// Env vars override file values
	if cfg.AI.Provider != "azure" {
		t.Errorf("expected provider 'azure' from env, got '%s'", cfg.AI.Provider)
	}
}
