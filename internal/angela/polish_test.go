// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

type mockProvider struct {
	CompleteFunc func(ctx context.Context, prompt string, opts ...domain.Option) (string, error)
}

func (m *mockProvider) Complete(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
	return m.CompleteFunc(ctx, prompt, opts...)
}

func TestBuildPolishPrompt_IncludesDocAndStyleGuide(t *testing.T) {
	doc := "---\ntype: decision\n---\n## Why\nBecause."
	meta := domain.DocMeta{Type: "decision"}
	sys, usr := BuildPolishPrompt(doc, meta, "- Require alternatives", "- [decision] auth.md", nil)

	if !strings.Contains(usr, "Because.") {
		t.Error("user content should contain document content")
	}
	if !strings.Contains(usr, "Require alternatives") {
		t.Error("user content should contain style guide")
	}
	if !strings.Contains(usr, "auth.md") {
		t.Error("user content should contain corpus summary")
	}
	if !strings.Contains(sys, "You are Angela") {
		t.Error("system prompt should contain Angela preamble")
	}
	if !strings.Contains(sys, "HARD RULES") {
		t.Error("system prompt should contain HARD RULES block")
	}
}

func TestBuildCorpusSummary_LimitedTo20(t *testing.T) {
	var corpus []domain.DocMeta
	for i := 0; i < 30; i++ {
		corpus = append(corpus, domain.DocMeta{
			Type:     "decision",
			Filename: fmt.Sprintf("doc-%d.md", i),
		})
	}
	summary := BuildCorpusSummary(corpus)
	if !strings.Contains(summary, "doc-19.md") {
		t.Error("should include doc-19 (20th)")
	}
	if strings.Contains(summary, "doc-20.md") {
		t.Error("should NOT include doc-20 (21st)")
	}
	if !strings.Contains(summary, "10 more") {
		t.Error("should mention remaining docs")
	}
}

func TestPolish_WithMockProvider(t *testing.T) {
	provider := &mockProvider{
		CompleteFunc: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
			return "---\ntype: decision\n---\n## Why\nImproved reason.", nil
		},
	}

	result, err := Polish(context.Background(), provider, "original", domain.DocMeta{}, "", "", nil)
	if err != nil {
		t.Fatalf("Polish: %v", err)
	}
	if !strings.Contains(result, "Improved reason") {
		t.Errorf("result = %q, want improved content", result)
	}
}

func TestPolish_PassesSystemAndMaxTokens(t *testing.T) {
	var receivedOpts domain.CallOptions
	provider := &mockProvider{
		CompleteFunc: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
			for _, opt := range opts {
				opt(&receivedOpts)
			}
			return "polished", nil
		},
	}

	// 1000-word document
	doc := strings.Repeat("word ", 1000)
	_, err := Polish(context.Background(), provider, doc, domain.DocMeta{}, "", "", nil)
	if err != nil {
		t.Fatalf("Polish: %v", err)
	}

	if receivedOpts.System == "" {
		t.Error("Polish should pass system prompt via WithSystem")
	}
	if !strings.Contains(receivedOpts.System, "You are Angela") {
		t.Error("system prompt should contain Angela preamble")
	}
	// 1000 words → 1000*1.3*1.8 = 2340
	if receivedOpts.MaxTokens != 2340 {
		t.Errorf("MaxTokens = %d, want 2340 for 1000-word doc", receivedOpts.MaxTokens)
	}
}

func TestPolish_NilProvider_Error(t *testing.T) {
	_, err := Polish(context.Background(), nil, "doc", domain.DocMeta{}, "", "", nil)
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
	if !strings.Contains(err.Error(), "no AI provider") {
		t.Errorf("error = %q, want 'no AI provider'", err)
	}
}

func TestBuildPolishPrompt_WithPersonas_ContainsDirectives(t *testing.T) {
	doc := "---\ntype: decision\n---\n## Why\nBecause."
	meta := domain.DocMeta{Type: "decision"}
	reg := GetRegistry()
	personas := []PersonaProfile{reg[0], reg[3]} // storyteller + architect
	_, usr := BuildPolishPrompt(doc, meta, "", "", personas)

	if !strings.Contains(usr, "STORYTELLING LENS") {
		t.Error("user content should contain storyteller directive")
	}
	if !strings.Contains(usr, "ARCHITECTURE LENS") {
		t.Error("user content should contain architect directive")
	}
	if !strings.Contains(usr, "Affoue") {
		t.Error("user content should contain Affoue display name")
	}
	if !strings.Contains(usr, "Doumbia") {
		t.Error("user content should contain Doumbia display name")
	}
	if !strings.Contains(usr, "EXPERT TEAM") {
		t.Error("user content should contain expert team header")
	}
}

func TestBuildPolishPrompt_NilPersonas_NoPersonaSection(t *testing.T) {
	doc := "---\ntype: decision\n---\n## Why\nBecause."
	meta := domain.DocMeta{Type: "decision"}
	_, usr := BuildPolishPrompt(doc, meta, "", "", nil)

	if strings.Contains(usr, "EXPERT TEAM") {
		t.Error("user content should NOT contain persona section with nil personas")
	}
}

func TestBuildPolishPrompt_SystemStableAcrossCalls(t *testing.T) {
	sys1, _ := BuildPolishPrompt("doc1", domain.DocMeta{}, "style1", "corpus1", nil)
	sys2, _ := BuildPolishPrompt("doc2", domain.DocMeta{}, "style2", "corpus2", nil)
	if sys1 != sys2 {
		t.Error("system prompt should be identical across different calls (stable/cacheable)")
	}
}

func TestBuildPolishPrompt_UserVariesWithInput(t *testing.T) {
	_, usr1 := BuildPolishPrompt("doc1", domain.DocMeta{}, "", "", nil)
	_, usr2 := BuildPolishPrompt("doc2", domain.DocMeta{}, "", "", nil)
	if usr1 == usr2 {
		t.Error("user content should vary with different input documents")
	}
}

func TestStripCodeFence(t *testing.T) {
	input := "```markdown\n---\ntype: decision\n---\nContent.\n```"
	result := stripCodeFence(input)
	if strings.Contains(result, "```") {
		t.Errorf("stripCodeFence should remove fences, got: %q", result)
	}
	if !strings.Contains(result, "Content.") {
		t.Errorf("stripCodeFence should preserve content, got: %q", result)
	}
}

func TestStripCodeFence_SingleLineBackticks(t *testing.T) {
	// A single line starting with ``` but no newline — should not strip.
	input := "```just backticks"
	result := stripCodeFence(input)
	if result != input {
		t.Errorf("stripCodeFence should return input unchanged for single line ```, got %q", result)
	}
}

func TestStripCodeFence_NoClosingFence(t *testing.T) {
	// Opening fence on first line but no closing ``` on its own line.
	input := "```markdown\nSome content here\nNo closing fence"
	result := stripCodeFence(input)
	if result != input {
		t.Errorf("stripCodeFence should return input unchanged when no closing fence, got %q", result)
	}
}

func TestStripCodeFence_WithLanguageTag(t *testing.T) {
	input := "```yaml\nkey: value\nother: data\n```"
	result := stripCodeFence(input)
	if strings.Contains(result, "```") {
		t.Errorf("stripCodeFence should remove fences with language tag, got %q", result)
	}
	if !strings.Contains(result, "key: value") {
		t.Errorf("stripCodeFence should preserve inner content, got %q", result)
	}
}

func TestStripCodeFence_AlreadyCleanText(t *testing.T) {
	input := "This is already clean text\nwith no code fences."
	result := stripCodeFence(input)
	if result != strings.TrimSpace(input) {
		t.Errorf("stripCodeFence should return clean text unchanged, got %q", result)
	}
}

// --- BuildCorpusSummary unit tests ---

func TestBuildCorpusSummary_EmptyCorpus(t *testing.T) {
	result := BuildCorpusSummary(nil)
	if result != "" {
		t.Errorf("BuildCorpusSummary(nil) = %q, want empty string", result)
	}

	result = BuildCorpusSummary([]domain.DocMeta{})
	if result != "" {
		t.Errorf("BuildCorpusSummary([]) = %q, want empty string", result)
	}
}

func TestBuildCorpusSummary_WithBranchScopeTags(t *testing.T) {
	corpus := []domain.DocMeta{
		{Type: "decision", Filename: "auth.md", Scope: "auth", Branch: "feat/auth", Tags: []string{"security", "api"}},
		{Type: "feature", Filename: "api.md", Branch: "main"},
	}
	result := BuildCorpusSummary(corpus)

	if !strings.Contains(result, "scope: auth") {
		t.Error("expected scope in summary")
	}
	if !strings.Contains(result, "branch: feat/auth") {
		t.Error("expected branch in summary")
	}
	if !strings.Contains(result, "tags: security, api") {
		t.Error("expected tags in summary")
	}
	// Branch "main" should be excluded
	if strings.Contains(result, "branch: main") {
		t.Error("branch 'main' should be excluded from summary")
	}
}

func TestBuildCorpusSummary_TruncationOver20(t *testing.T) {
	corpus := make([]domain.DocMeta, 25)
	for i := range corpus {
		corpus[i] = domain.DocMeta{Type: "decision", Filename: fmt.Sprintf("doc-%02d.md", i)}
	}

	result := BuildCorpusSummary(corpus)

	// Should include first 20
	if !strings.Contains(result, "doc-19.md") {
		t.Error("should include 20th doc (doc-19)")
	}
	// Should NOT include 21st
	if strings.Contains(result, "doc-20.md") {
		t.Error("should NOT include 21st doc (doc-20)")
	}
	// Should mention remaining
	if !strings.Contains(result, "5 more") {
		t.Errorf("should mention '5 more documents', got: %s", result)
	}
}
