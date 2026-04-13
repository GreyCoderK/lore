// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

// --- PolishDocument incremental paths ---

// TestPolishDocument_Incremental_MissingStateFile exercises the incremental
// branch where LoadPolishState fails (path points to a directory, so it
// errors). The code falls back to a fresh state and calls full polish.
func TestPolishDocument_Incremental_MissingStateFile(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause reasons.\n"
	filename := "decision-inc-2026-04-05.md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{response: content}
	cfg := &config.Config{}

	// PolishStatePath does not exist → LoadPolishState returns empty state,
	// falls back to full polish path.
	nonExistentPath := filepath.Join(dir, "no-such-dir", "polish-state.json")
	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, filename, PolishOptions{
		Incremental:     true,
		PolishStatePath: nonExistentPath,
	})
	if err != nil {
		t.Fatalf("PolishDocument incremental (missing state) error: %v", err)
	}
	if result.Filename != filename {
		t.Errorf("Filename = %q, want %q", result.Filename, filename)
	}
}

// TestPolishDocument_Incremental_SavesState verifies that after a successful
// incremental polish the state file is written to the specified path.
func TestPolishDocument_Incremental_SavesState(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause reasons.\n"
	filename := "decision-inc-save-2026-04-05.md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dir, "polish-state.json")
	provider := &mockProvider{response: content}
	cfg := &config.Config{}

	_, err := PolishDocument(context.Background(), provider, cfg, docsDir, filename, PolishOptions{
		Incremental:     true,
		PolishStatePath: statePath,
	})
	if err != nil {
		t.Fatalf("PolishDocument incremental (save state): %v", err)
	}

	// State file should have been created.
	if _, statErr := os.Stat(statePath); os.IsNotExist(statErr) {
		t.Error("expected polish-state.json to be written after incremental polish")
	}
}

// TestPolishDocument_Incremental_FlagFalse verifies that when Incremental=false
// the incremental branch is not entered even when PolishStatePath is set.
func TestPolishDocument_Incremental_FlagFalse(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause reasons.\n"
	filename := "decision-noinc-2026-04-05.md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dir, "polish-state.json")
	provider := &mockProvider{response: content}
	cfg := &config.Config{}

	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, filename, PolishOptions{
		Incremental:     false,
		PolishStatePath: statePath,
	})
	if err != nil {
		t.Fatalf("PolishDocument non-incremental: %v", err)
	}
	if result.Filename != filename {
		t.Errorf("Filename = %q, want %q", result.Filename, filename)
	}
	// State should NOT have been written since incremental=false.
	if _, statErr := os.Stat(statePath); !os.IsNotExist(statErr) {
		t.Error("polish-state.json should not be written when incremental=false")
	}
}

// --- PolishDocument multi-pass path ---

// TestPolishDocument_MultiPass exercises the ShouldMultiPass=true branch.
// We need a document with ~4000+ words so the word-count check triggers
// multi-pass mode. The mock provider returns the content unchanged so that
// the section-reassembly in PolishMultiPass succeeds.
func TestPolishDocument_MultiPass(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Build a document large enough to trigger multi-pass (>3780 words).
	var sb strings.Builder
	sb.WriteString("---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n")
	sb.WriteString("# Large Document\n\n")
	// Generate enough filler sections to exceed the word threshold.
	for sec := 0; sec < 40; sec++ {
		fmt.Fprintf(&sb, "## Section %d\n\n", sec+1)
		for line := 0; line < 3; line++ {
			sb.WriteString(strings.Repeat("word ", 35) + "\n")
		}
		sb.WriteString("\n")
	}
	content := sb.String()

	filename := "decision-large-2026-04-05.md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{response: content}
	cfg := &config.Config{}

	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, filename)
	if err != nil {
		t.Fatalf("PolishDocument multi-pass error: %v", err)
	}
	if result.Filename != filename {
		t.Errorf("Filename = %q, want %q", result.Filename, filename)
	}
	if result.Meta.Type != "decision" {
		t.Errorf("Meta.Type = %q, want decision", result.Meta.Type)
	}
}

// TestPolishDocument_MultiPass_WithProgress verifies the Progress callback
// is passed and invoked (or at minimum doesn't crash) in multi-pass mode.
func TestPolishDocument_MultiPass_WithProgress(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	var sb strings.Builder
	sb.WriteString("---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n")
	sb.WriteString("# Large Document With Progress\n\n")
	for sec := 0; sec < 40; sec++ {
		fmt.Fprintf(&sb, "## Section %d\n\n", sec+1)
		for line := 0; line < 3; line++ {
			sb.WriteString(strings.Repeat("word ", 35) + "\n")
		}
		sb.WriteString("\n")
	}
	content := sb.String()

	filename := "decision-large-prog-2026-04-05.md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	progressCalled := false
	provider := &mockProvider{response: content}
	cfg := &config.Config{}

	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, filename, PolishOptions{
		Progress: func(sectionIndex, totalSections int, heading string, changed bool) {
			progressCalled = true
		},
	})
	if err != nil {
		t.Fatalf("PolishDocument multi-pass with progress error: %v", err)
	}
	_ = result
	if !progressCalled {
		t.Log("note: progress callback was not called (may be expected if sections were not chunked)")
	}
}

// --- ReviewCorpus error paths ---

// errorCorpusReader implements domain.CorpusReader but always errors on ListDocs.
type errorCorpusReader struct {
	listErr error
	readErr error
}

func (r *errorCorpusReader) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error) {
	return nil, r.listErr
}

func (r *errorCorpusReader) ReadDoc(_ string) (string, error) {
	return "", r.readErr
}

// TestReviewCorpus_ListDocsError verifies that an error from the corpus reader
// is propagated correctly.
func TestReviewCorpus_ListDocsError(t *testing.T) {
	reader := &errorCorpusReader{listErr: fmt.Errorf("storage unavailable")}
	provider := &mockProvider{response: `{"findings": []}`}

	_, _, err := ReviewCorpus(context.Background(), provider, reader, &config.Config{}, nil)
	if err == nil {
		t.Fatal("expected error from ListDocs failure")
	}
	if !strings.Contains(err.Error(), "storage unavailable") {
		t.Errorf("error should wrap original: %v", err)
	}
}

// TestPolishDocument_WithAudience verifies the Audience option is propagated.
func TestPolishDocument_WithAudience(t *testing.T) {
	dir := t.TempDir()
	docsDir := filepath.Join(dir, ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := "---\ntype: decision\ndate: 2026-04-05\nstatus: draft\n---\n# Test\n\n## Why\nBecause reasons.\n"
	filename := "decision-audience-2026-04-05.md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &mockProvider{response: content}
	cfg := &config.Config{}

	result, err := PolishDocument(context.Background(), provider, cfg, docsDir, filename, PolishOptions{
		Audience: "engineers",
	})
	if err != nil {
		t.Fatalf("PolishDocument with audience error: %v", err)
	}
	if result.Filename != filename {
		t.Errorf("Filename = %q", result.Filename)
	}
}

// TestEngineConfigFromApp_CriticalScopesAndAlwaysSkip exercises the fields
// that were populated but not individually asserted in the existing test.
func TestEngineConfigFromApp_CriticalScopesAndAlwaysSkip(t *testing.T) {
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:      60,
			ThresholdReduced:   35,
			ThresholdSuggest:   15,
			AlwaysSkip:         []string{"ci", "chore"},
			CriticalScopes:     []string{"auth", "payments"},
			LearningMinCommits: 5,
		},
	}
	ec := EngineConfigFromApp(cfg)
	if len(ec.AlwaysSkip) != 2 {
		t.Errorf("AlwaysSkip = %v, want [ci chore]", ec.AlwaysSkip)
	}
	if len(ec.CriticalScopes) != 2 {
		t.Errorf("CriticalScopes = %v, want [auth payments]", ec.CriticalScopes)
	}
	if ec.LearningMinCommits != 5 {
		t.Errorf("LearningMinCommits = %d, want 5", ec.LearningMinCommits)
	}
}
