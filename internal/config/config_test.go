package config

import (
	"os"
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
	if cfg.AI.Provider != "anthropic" {
		t.Errorf("expected provider 'anthropic', got '%s'", cfg.AI.Provider)
	}
}
