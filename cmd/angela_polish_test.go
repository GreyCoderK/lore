// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/testutil"
)

func TestAngelaPolishCmd_NoArgs(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}

	cmd := newAngelaPolishCmd(cfg, streams)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing filename arg")
	}
}

func TestAngelaPolishCmd_Flags(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}

	cmd := newAngelaPolishCmd(cfg, streams)

	if cmd.Use != "polish <filename>" {
		t.Errorf("Use = %q", cmd.Use)
	}

	dryRunFlag := cmd.Flag("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag")
	}

	yesFlag := cmd.Flag("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag")
	}
}

// Test --dry-run flag default value
func TestAngelaPolishCmd_DryRunFlagDefault(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newAngelaPolishCmd(cfg, streams)

	f := cmd.Flag("dry-run")
	if f == nil {
		t.Fatal("expected --dry-run flag to exist")
	}
	if f.DefValue != "false" {
		t.Errorf("--dry-run default = %q, want 'false'", f.DefValue)
	}
}

// Test --yes flag default value
func TestAngelaPolishCmd_YesFlagDefault(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newAngelaPolishCmd(cfg, streams)

	f := cmd.Flag("yes")
	if f == nil {
		t.Fatal("expected --yes flag to exist")
	}
	if f.DefValue != "false" {
		t.Errorf("--yes default = %q, want 'false'", f.DefValue)
	}
}

// .lore dir exists but file doesn't → not found error
func TestAngelaPolishCmd_FileNotFound(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "nonexistent-doc.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want 'not found'", err)
	}
}

// Provider configured with bad endpoint → connection error (not "no provider" error)
func TestAngelaPolishCmd_BadEndpoint(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-20\"\n---\n## Why\nSome reason."
	if err := os.WriteFile(filepath.Join(docsDir, "test-doc.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	cfg.AI.Provider = "ollama"
	cfg.AI.Endpoint = "http://127.0.0.1:1" // connection refused
	cfg.AI.Model = "test-model"

	streams, _, _ := testStreams()
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "test-doc.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from bad endpoint")
	}
	// Should NOT be the "no provider" error — the provider was created
	if strings.Contains(err.Error(), "no AI provider configured") {
		t.Errorf("expected connection error, not 'no AI provider configured': %q", err)
	}
}

// Polish with --dry-run flag parses correctly (no cobra error)
func TestAngelaPolishCmd_DryRunFlagParsed(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-20\"\n---\n## Why\nSome reason."
	if err := os.WriteFile(filepath.Join(docsDir, "test-doc.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	// No provider → will fail, but NOT because of --dry-run flag being unknown
	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--dry-run", "test-doc.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error (no provider)")
	}
	if strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("--dry-run should be recognized, got: %q", err)
	}
}

// Polish with --yes flag parses correctly (no cobra error)
func TestAngelaPolishCmd_YesFlagParsed(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-03-20\"\n---\n## Why\nSome reason."
	if err := os.WriteFile(filepath.Join(docsDir, "test-doc.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newAngelaCmd(cfg, streams)
	cmd.SetArgs([]string{"polish", "--yes", "test-doc.md"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error (no provider)")
	}
	if strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("--yes should be recognized, got: %q", err)
	}
}

func TestSanitizeAudience(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"CTO", "cto"},
		{"équipe commerciale", "équipe-commerciale"},
		{"", "audience"},
		{strings.Repeat("a", 100), strings.Repeat("a", 50)},
		{"  spaces  ", "spaces"},
		{"foo--bar", "foo-bar"},
	}
	for _, tt := range tests {
		got := sanitizeAudience(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeAudience(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{5 * time.Second, "5.0s"},
		{30 * time.Second, "30.0s"},
		{65 * time.Second, "1m5s"},
		{120 * time.Second, "2m0s"},
	}
	for _, tt := range tests {
		got := formatElapsed(tt.input)
		if got != tt.want {
			t.Errorf("formatElapsed(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"context deadline", fmt.Errorf("context deadline exceeded"), true},
		{"client timeout", fmt.Errorf("Client.Timeout exceeded"), true},
		{"normal error", fmt.Errorf("something went wrong"), false},
		{"nil error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeoutError(tt.err)
			if got != tt.want {
				t.Errorf("isTimeoutError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
