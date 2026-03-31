// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".lorerc.yaml", `
ai:
  provider: anthropic
  model: claude-sonnet-4-20250514
hooks:
  post_commit: true
`)

	report := ValidateConfig(dir)

	assert.True(t, report.Valid)
	assert.True(t, report.OK())
	assert.Empty(t, report.Warnings)
	assert.Empty(t, report.Errors)
	assert.Equal(t, "anthropic", report.Active["ai.provider"])
	assert.Equal(t, "claude-sonnet-4-20250514", report.Active["ai.model"])
}

func TestValidateConfig_UnknownFieldWithSuggestion(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".lorerc.yaml", `
ai:
  povider: anthropic
`)

	report := ValidateConfig(dir)

	assert.True(t, report.Valid) // warnings don't make it invalid
	assert.False(t, report.OK())
	require.Len(t, report.Warnings, 1)
	assert.Contains(t, report.Warnings[0], "ai.povider")
	assert.Contains(t, report.Warnings[0], "did you mean")
	assert.Contains(t, report.Warnings[0], "ai.provider")
}

func TestValidateConfig_UnknownFieldNoSuggestion(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".lorerc.yaml", `
ai:
  zzzzz: something
`)

	report := ValidateConfig(dir)

	assert.True(t, report.Valid)
	assert.False(t, report.OK())
	require.Len(t, report.Warnings, 1)
	assert.Contains(t, report.Warnings[0], "ai.zzzzz")
	assert.NotContains(t, report.Warnings[0], "did you mean")
}

func TestValidateConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".lorerc.yaml", `
ai:
  provider: [invalid yaml
`)

	report := ValidateConfig(dir)

	assert.False(t, report.Valid)
	require.NotEmpty(t, report.Errors)
	assert.Contains(t, report.Errors[0], "YAML parse error")
}

func TestValidateConfig_NoConfigFiles(t *testing.T) {
	dir := t.TempDir()

	report := ValidateConfig(dir)

	assert.True(t, report.Valid)
	assert.True(t, report.OK())
	// Active values should have defaults
	assert.Equal(t, "draft", report.Active["angela.mode"])
}

func TestValidateConfig_LocalOverridesShared(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".lorerc.yaml", `
ai:
  provider: ollama
`)
	writeYAML(t, dir, ".lorerc.local.yaml", `
ai:
  provider: anthropic
  api_key: sk-test-123
`)
	// Ensure .lorerc.local is owner-only to avoid permission warning.
	os.Chmod(filepath.Join(dir, ".lorerc.local.yaml"), 0600)

	report := ValidateConfig(dir)

	assert.True(t, report.Valid)
	assert.Equal(t, "anthropic", report.Active["ai.provider"])
}

func TestValidateConfig_TypoInLocalFile(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, ".lorerc.local.yaml", `
ai:
  tiemout: 60s
`)
	os.Chmod(filepath.Join(dir, ".lorerc.local.yaml"), 0600)

	report := ValidateConfig(dir)

	require.Len(t, report.Warnings, 1)
	assert.Contains(t, report.Warnings[0], "ai.tiemout")
	assert.Contains(t, report.Warnings[0], "ai.timeout")
	assert.Contains(t, report.Warnings[0], ".lorerc.local.yaml")
}

func TestValidateConfig_ActiveValuesShowDefaults(t *testing.T) {
	dir := t.TempDir()

	report := ValidateConfig(dir)

	assert.Equal(t, "30s", report.Active["ai.timeout"])
	assert.Equal(t, "2000", report.Active["angela.max_tokens"])
	assert.Equal(t, "markdown", report.Active["output.format"])
	assert.Equal(t, filepath.Join(".lore", "docs"), report.Active["output.dir"])
	assert.Equal(t, "true", report.Active["hooks.post_commit"])
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"povider", "provider", 1},
		{"tiemout", "timeout", 2},
		{"zzzzz", "provider", 8},
		{"model", "mode", 1},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_vs_%s", tt.a, tt.b), func(t *testing.T) {
			assert.Equal(t, tt.want, levenshtein(tt.a, tt.b))
		})
	}
}

func TestSuggestField(t *testing.T) {
	tests := []struct {
		unknown string
		want    string
	}{
		{"ai.povider", "ai.provider"},
		{"ai.tiemout", "ai.timeout"},
		{"ai.zzzzz", ""},
		{"angela.mod", "angela.mode"},
	}

	for _, tt := range tests {
		t.Run(tt.unknown, func(t *testing.T) {
			assert.Equal(t, tt.want, suggestField(tt.unknown))
		})
	}
}

func TestValidFields_MatchesConfigStruct(t *testing.T) {
	// Verify that validFields contains all fields from setDefaults.
	// If this test fails, a new field was added to Config but not to validFields.
	expectedLeafFields := []string{
		"ai.provider", "ai.model", "ai.api_key", "ai.endpoint", "ai.timeout",
		"angela.mode", "angela.max_tokens", "angela.style_guide",
		"templates.dir",
		"hooks.post_commit",
		"output.dir", "output.format",
	}

	for _, field := range expectedLeafFields {
		assert.True(t, validFields[field], "validFields missing %q — add it to stay in sync with Config struct", field)
	}
}

func TestMaskSecret(t *testing.T) {
	assert.Equal(t, "****", maskSecret("sk-abc123"))
	assert.Equal(t, "(not set)", maskSecret(""))
}

// writeYAML is a test helper that writes a YAML file.
func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	require.NoError(t, err)
}
