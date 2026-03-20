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
	prompt := BuildPolishPrompt(doc, meta, "- Require alternatives", "- [decision] auth.md", nil)

	if !strings.Contains(prompt, "Because.") {
		t.Error("prompt should contain document content")
	}
	if !strings.Contains(prompt, "Require alternatives") {
		t.Error("prompt should contain style guide")
	}
	if !strings.Contains(prompt, "auth.md") {
		t.Error("prompt should contain corpus summary")
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
	prompt := BuildPolishPrompt(doc, meta, "", "", personas)

	if !strings.Contains(prompt, "STORYTELLING LENS") {
		t.Error("prompt should contain storyteller directive")
	}
	if !strings.Contains(prompt, "ARCHITECTURE LENS") {
		t.Error("prompt should contain architect directive")
	}
	if !strings.Contains(prompt, "Affoue") {
		t.Error("prompt should contain Affoue display name")
	}
	if !strings.Contains(prompt, "Doumbia") {
		t.Error("prompt should contain Doumbia display name")
	}
	if !strings.Contains(prompt, "EXPERT TEAM") {
		t.Error("prompt should contain expert team header")
	}
}

func TestBuildPolishPrompt_NilPersonas_NoPersonaSection(t *testing.T) {
	doc := "---\ntype: decision\n---\n## Why\nBecause."
	meta := domain.DocMeta{Type: "decision"}
	prompt := BuildPolishPrompt(doc, meta, "", "", nil)

	if strings.Contains(prompt, "EXPERT TEAM") {
		t.Error("prompt should NOT contain persona section with nil personas")
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
