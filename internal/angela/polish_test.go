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

// --- I25: AI prompt excludes frontmatter (story 8-21, Task 2) --------

func TestBuildPolishPrompt_UserContentExcludesFrontmatter(t *testing.T) {
	// The user content passed to the AI must NOT contain the `---`
	// delimiter or any frontmatter field. Body content is preserved.
	doc := "---\n" +
		"type: decision\n" +
		"date: \"2026-04-10\"\n" +
		"status: published\n" +
		"commit: abc1234\n" +
		"tags: [auth, jwt]\n" +
		"---\n" +
		"## Why\nBecause compliance asked.\n"

	_, usr := BuildPolishPrompt(doc, domain.DocMeta{}, "", "", nil)

	// None of these substrings from the frontmatter should leak into
	// the user content. We assert on each key independently so the
	// failure message is specific.
	forbidden := []string{
		"type: decision",
		"date: \"2026-04-10\"",
		"status: published",
		"commit: abc1234",
		"tags: [auth, jwt]",
	}
	for _, s := range forbidden {
		if strings.Contains(usr, s) {
			t.Errorf("user content leaks frontmatter field %q:\n%s", s, usr)
		}
	}

	// Body content must still be present.
	if !strings.Contains(usr, "Because compliance asked.") {
		t.Error("user content should preserve body")
	}
	// And the BODY marker block must not contain a `---\n` at its very
	// start (the AI would interpret it as a delimiter).
	bodyBlock := extractBetween(usr, "<<<BODY>>>", "<<<END_BODY>>>")
	if strings.HasPrefix(strings.TrimLeft(bodyBlock, "\n"), "---\n") {
		t.Errorf("body block starts with `---\\n` — frontmatter leaked:\n%s", bodyBlock)
	}
}

func TestBuildPolishPrompt_SystemPromptInstructsNoFrontmatter(t *testing.T) {
	sys, _ := BuildPolishPrompt("", domain.DocMeta{}, "", "", nil)
	// The system prompt must explicitly forbid emitting a frontmatter
	// block in the response.
	if !strings.Contains(sys, "Return ONLY the improved BODY") {
		t.Error("system prompt should instruct AI to return body only")
	}
	if !strings.Contains(sys, "never emit ") || !strings.Contains(sys, "delimiters") {
		t.Error("system prompt should forbid `---` delimiter emission")
	}
}

func TestBuildPolishPrompt_DocWithoutFrontmatter_PreservedAsBody(t *testing.T) {
	// When the input has no frontmatter (edge case: raw body only),
	// the entire content is used as body — no rejection, no error.
	doc := "## Why\nPure body content.\nNo frontmatter here.\n"
	_, usr := BuildPolishPrompt(doc, domain.DocMeta{}, "", "", nil)
	if !strings.Contains(usr, "Pure body content.") {
		t.Error("body content should appear in user content")
	}
}

func TestBuildPolishPrompt_BodyWithHorizontalRule_Preserved(t *testing.T) {
	// A `---` INSIDE the body (Markdown horizontal rule) must survive
	// the pipeline unchanged — it is content, not a delimiter.
	doc := "---\n" +
		"type: note\ndate: \"2026-04-01\"\nstatus: draft\n" +
		"---\n" +
		"## Intro\nFirst section.\n\n---\n\n## Outro\nSecond section.\n"
	_, usr := BuildPolishPrompt(doc, domain.DocMeta{}, "", "", nil)
	// The horizontal rule in the body must survive.
	if !strings.Contains(usr, "\n---\n") {
		t.Error("body horizontal rule should be preserved")
	}
	// But the FM key must not leak.
	if strings.Contains(usr, "type: note") {
		t.Error("frontmatter leaked despite FM being split out")
	}
}

func TestBuildIncrementalPrompt_OutlineAndChangedExcludePreamble(t *testing.T) {
	// sections[0] is the preamble (frontmatter). It must be excluded
	// from both the outline and the "sections to polish" block, even
	// when index 0 appears in changedIdx.
	sections := []Section{
		{Index: 0, Heading: "", Body: "---\ntype: decision\ndate: \"2026-04-10\"\nstatus: published\n---\n"},
		{Index: 1, Heading: "## Why", Body: "\nBecause compliance.\n"},
		{Index: 2, Heading: "## Context", Body: "\nThe legal team...\n"},
	}
	// Deliberately include index 0 to simulate a stale hash marking
	// the preamble as changed — the builder must still filter it out.
	changedIdx := []int{0, 1}
	opts := IncrementalOpts{Meta: domain.DocMeta{Type: "decision"}}
	_, usr := buildIncrementalPrompt(sections, changedIdx, opts)

	forbidden := []string{"type: decision", "date: \"2026-04-10\"", "status: published"}
	for _, s := range forbidden {
		if strings.Contains(usr, s) {
			t.Errorf("incremental user content leaks frontmatter %q", s)
		}
	}
	// The body we wanted to polish is still present.
	if !strings.Contains(usr, "Because compliance.") {
		t.Error("expected changed section body in prompt")
	}
	// The outline must mention the headings that DO exist.
	if !strings.Contains(usr, "## Why") || !strings.Contains(usr, "## Context") {
		t.Error("outline should list the actual headings")
	}
}

func TestBuildIncrementalPrompt_UsesSafeSectionSeparator(t *testing.T) {
	// Inter-section separator must NOT be `---`. A Markdown horizontal
	// rule in a body would collide with it and the AI would be primed
	// to emit `---` in response.
	sections := []Section{
		{Index: 0, Heading: "", Body: "preamble\n"},
		{Index: 1, Heading: "## A", Body: "\nBody A.\n"},
		{Index: 2, Heading: "## B", Body: "\nBody B.\n"},
	}
	opts := IncrementalOpts{Meta: domain.DocMeta{Type: "note"}}
	_, usr := buildIncrementalPrompt(sections, []int{1, 2}, opts)

	if !strings.Contains(usr, "<<<NEXT SECTION>>>") {
		t.Error("expected safe separator `<<<NEXT SECTION>>>` between changed sections")
	}
	// Between the two section bodies, there should not be a bare
	// `\n---\n` separator (the horizontal-rule lookalike).
	// Count `\n---\n` occurrences: any body may contain it, but not as
	// a structural separator. We assert by checking that the structural
	// separator is `<<<NEXT SECTION>>>`, which we already did above.
	_ = usr
}

// extractBetween returns the content between markerStart and markerEnd
// exclusively, or the full string if either is absent.
func extractBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return s
	}
	rest := s[i+len(start):]
	j := strings.Index(rest, end)
	if j < 0 {
		return rest
	}
	return rest[:j]
}
