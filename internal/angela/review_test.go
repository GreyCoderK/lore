// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// mockCorpusReader implements domain.CorpusReader for tests.
type mockCorpusReader struct {
	docs    []domain.DocMeta
	content map[string]string // filename -> content
	listErr error
	readErr error
}

func (m *mockCorpusReader) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error) {
	return m.docs, m.listErr
}

func (m *mockCorpusReader) ReadDoc(id string) (string, error) {
	if m.readErr != nil {
		return "", m.readErr
	}
	if content, ok := m.content[id]; ok {
		return content, nil
	}
	return "", fmt.Errorf("not found: %s", id)
}

func newMockProviderWith(response string, err error) *mockProvider {
	return &mockProvider{
		CompleteFunc: func(_ context.Context, _ string, _ ...domain.Option) (string, error) {
			return response, err
		},
	}
}

// --- BuildReviewPrompt tests ---

func TestBuildReviewPrompt_IncludesSummaries(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-auth-2026-03-01.md", Type: "decision", Date: "2026-03-01", Tags: []string{"auth"}, Summary: "[Why] Security | [What] JWT tokens"},
		{Filename: "feature-api-2026-03-02.md", Type: "feature", Date: "2026-03-02", Summary: "[Why] Speed | [What] REST API"},
	}
	sys, usr := BuildReviewPrompt(docs, "- Use consistent terminology", nil)

	if !strings.Contains(usr, "decision-auth-2026-03-01.md") {
		t.Error("user content should contain first doc filename")
	}
	if !strings.Contains(usr, "feature-api-2026-03-02.md") {
		t.Error("user content should contain second doc filename")
	}
	if !strings.Contains(usr, "JWT tokens") {
		t.Error("user content should contain summary content")
	}
	// TOON format: tags are comma-separated in pipe-delimited row
	if !strings.Contains(usr, "auth") {
		t.Error("user content should contain tags")
	}
	if !strings.Contains(usr, "consistent terminology") {
		t.Error("user content should contain style guide")
	}
	if !strings.Contains(usr, "<<<STYLE_GUIDE>>>") {
		t.Error("user content should contain style guide markers")
	}
	if !strings.Contains(sys, "Group documents by Type") {
		t.Error("system prompt should contain type grouping instruction")
	}
	if !strings.Contains(sys, "You are Angela") {
		t.Error("system prompt should contain Angela preamble")
	}
	// TOON format assertions
	if !strings.Contains(usr, "corpus:") {
		t.Error("user content should contain TOON corpus: header")
	}
	if !strings.Contains(usr, "filename|type|date|tags|branch|scope|summary") {
		t.Error("user content should contain TOON column headers")
	}
	if !strings.Contains(usr, "TOON (pipe-separated) format") {
		t.Error("user content should contain TOON preamble")
	}
}

func TestBuildReviewPrompt_NoStyleGuide(t *testing.T) {
	docs := []DocSummary{{Filename: "test.md", Type: "decision", Date: "2026-03-01"}}
	_, usr := BuildReviewPrompt(docs, "", nil)

	if strings.Contains(usr, "<<<STYLE_GUIDE>>>") {
		t.Error("user content should not contain style guide markers when empty")
	}
}

func TestBuildReviewPrompt_AllDocsIncluded(t *testing.T) {
	docs := make([]DocSummary, 60)
	for i := range docs {
		docs[i] = DocSummary{Filename: fmt.Sprintf("doc-%d.md", i), Type: "decision", Date: "2026-03-01"}
	}
	_, usr := BuildReviewPrompt(docs, "", nil)
	if !strings.Contains(usr, "doc-59.md") {
		t.Error("BuildReviewPrompt should include all docs passed to it")
	}
}

func TestBuildReviewPrompt_SystemStableAcrossCalls(t *testing.T) {
	docs1 := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01"}}
	docs2 := []DocSummary{{Filename: "b.md", Type: "feature", Date: "2026-02-01"}}
	signals1 := &CorpusSignals{PotentialPairs: []DocPair{{DocA: "x.md", DocB: "y.md", Type: "decision", Tags: "auth", DaysDiff: 30}}}
	sys1, _ := BuildReviewPrompt(docs1, "style1", signals1)
	sys2, _ := BuildReviewPrompt(docs2, "style2", nil)
	if sys1 != sys2 {
		t.Error("system prompt should be identical across different calls (stable/cacheable)")
	}
}

func TestBuildReviewPrompt_WithSignals(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Tags: []string{"auth"}}}
	signals := &CorpusSignals{
		PotentialPairs: []DocPair{{DocA: "a.md", DocB: "b.md", Type: "decision", Tags: "auth", DaysDiff: 60}},
		IsolatedDocs:   []string{"lonely.md"},
	}
	_, usr := BuildReviewPrompt(docs, "", signals)

	if !strings.Contains(usr, "signals:") {
		t.Error("user content should contain TOON signals section")
	}
	if !strings.Contains(usr, "contradiction|a.md,b.md|") {
		t.Error("user content should contain contradiction signal row")
	}
	if !strings.Contains(usr, "isolated|lonely.md|") {
		t.Error("user content should contain isolated signal row")
	}
}

// --- Review tests ---

func TestReview_WithMockProvider(t *testing.T) {
	provider := newMockProviderWith(
		`{"findings": [{"severity": "contradiction", "title": "Auth conflict", "description": "JWT vs session", "documents": ["a.md", "b.md"]}]}`,
		nil,
	)
	docs := []DocSummary{{Filename: "a.md"}, {Filename: "b.md"}}

	report, err := Review(context.Background(), provider, docs, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != "contradiction" {
		t.Errorf("severity = %q, want contradiction", report.Findings[0].Severity)
	}
	if report.Findings[0].Title != "Auth conflict" {
		t.Errorf("title = %q, want 'Auth conflict'", report.Findings[0].Title)
	}
	if report.DocCount != 2 {
		t.Errorf("doc count = %d, want 2", report.DocCount)
	}
}

func TestReview_PassesSystemAndMaxTokens(t *testing.T) {
	var receivedOpts domain.CallOptions
	provider := &mockProvider{
		CompleteFunc: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
			for _, opt := range opts {
				opt(&receivedOpts)
			}
			return `{"findings": []}`, nil
		},
	}

	_, err := Review(context.Background(), provider, []DocSummary{{Filename: "a.md"}}, "")
	if err != nil {
		t.Fatalf("Review: %v", err)
	}

	if receivedOpts.System == "" {
		t.Error("Review should pass system prompt via WithSystem")
	}
	if !strings.Contains(receivedOpts.System, "You are Angela") {
		t.Error("system prompt should contain Angela preamble")
	}
	if receivedOpts.MaxTokens != 1500 {
		t.Errorf("MaxTokens = %d, want 1500 for review mode", receivedOpts.MaxTokens)
	}
}

func TestReview_NilProvider(t *testing.T) {
	_, err := Review(context.Background(), nil, nil, "")
	if err == nil {
		t.Fatal("expected error for nil provider")
	}
	if !strings.Contains(err.Error(), "no AI provider configured") {
		t.Errorf("error = %q, want 'no AI provider configured'", err)
	}
}

func TestReview_InvalidJSON(t *testing.T) {
	provider := newMockProviderWith("This is not JSON at all", nil)
	_, err := Review(context.Background(), provider, []DocSummary{{}}, "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "could not parse AI response as JSON") {
		t.Errorf("error = %q, want parse error", err)
	}
}

func TestReview_JSONInCodeBlock(t *testing.T) {
	provider := newMockProviderWith(
		"Here is the analysis:\n```json\n{\"findings\": [{\"severity\": \"gap\", \"title\": \"Missing docs\", \"description\": \"No error handling doc\", \"documents\": [\"api.md\"]}]}\n```",
		nil,
	)

	report, err := Review(context.Background(), provider, []DocSummary{{}}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != "gap" {
		t.Errorf("severity = %q, want gap", report.Findings[0].Severity)
	}
}

func TestReview_ProviderError(t *testing.T) {
	provider := newMockProviderWith("", fmt.Errorf("network timeout"))
	_, err := Review(context.Background(), provider, []DocSummary{{}}, "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "network timeout") {
		t.Errorf("error = %q, want 'network timeout'", err)
	}
}

func TestReview_FindingsSortedBySeverity(t *testing.T) {
	provider := newMockProviderWith(
		`{"findings": [
			{"severity": "style", "title": "S1", "description": "", "documents": []},
			{"severity": "contradiction", "title": "C1", "description": "", "documents": []},
			{"severity": "obsolete", "title": "O1", "description": "", "documents": []},
			{"severity": "gap", "title": "G1", "description": "", "documents": []}
		]}`,
		nil,
	)

	report, err := Review(context.Background(), provider, []DocSummary{{}}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 4 {
		t.Fatalf("expected 4 findings, got %d", len(report.Findings))
	}

	expected := []string{"contradiction", "gap", "obsolete", "style"}
	for i, f := range report.Findings {
		if f.Severity != expected[i] {
			t.Errorf("finding[%d].Severity = %q, want %q", i, f.Severity, expected[i])
		}
	}
}

func TestReview_UnknownSeveritySortsLast(t *testing.T) {
	provider := newMockProviderWith(
		`{"findings": [
			{"severity": "info", "title": "I1", "description": "", "documents": []},
			{"severity": "contradiction", "title": "C1", "description": "", "documents": []},
			{"severity": "style", "title": "S1", "description": "", "documents": []}
		]}`,
		nil,
	)

	report, err := Review(context.Background(), provider, []DocSummary{{}}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != "contradiction" {
		t.Errorf("finding[0].Severity = %q, want contradiction", report.Findings[0].Severity)
	}
	if report.Findings[1].Severity != "style" {
		t.Errorf("finding[1].Severity = %q, want style", report.Findings[1].Severity)
	}
	if report.Findings[2].Severity != "style" {
		t.Errorf("finding[2].Severity = %q, want style (unknown normalized to style)", report.Findings[2].Severity)
	}
}

func TestReview_EmptyFindings(t *testing.T) {
	provider := newMockProviderWith(`{"findings": []}`, nil)
	report, err := Review(context.Background(), provider, []DocSummary{{}}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(report.Findings))
	}
}

// --- PrepareDocSummaries tests ---

func TestPrepareDocSummaries_LessThan5Docs(t *testing.T) {
	reader := &mockCorpusReader{
		docs: []domain.DocMeta{
			{Filename: "a.md", Date: "2026-01-01"},
			{Filename: "b.md", Date: "2026-01-02"},
		},
	}

	_, total, err := PrepareDocSummaries(reader)
	if err == nil {
		t.Fatal("expected error for < 5 docs")
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if !strings.Contains(err.Error(), "at least 5 documents required") {
		t.Errorf("error = %q, want 'At least 5 documents'", err)
	}
	if !strings.Contains(err.Error(), "you have 2") {
		t.Errorf("error should contain doc count, got: %q", err)
	}
	if !strings.Contains(err.Error(), "keep documenting") {
		t.Errorf("error should end with exclamation, got: %q", err)
	}
}

func TestPrepareDocSummaries_Exactly5Docs(t *testing.T) {
	docs := make([]domain.DocMeta, 5)
	content := make(map[string]string)
	for i := range docs {
		name := fmt.Sprintf("doc-%d.md", i)
		docs[i] = domain.DocMeta{Filename: name, Type: "decision", Date: fmt.Sprintf("2026-03-%02d", i+1)}
		content[name] = "---\ntype: decision\n---\n## What\nSomething important here\n\n## Why\nBecause reasons"
	}

	reader := &mockCorpusReader{docs: docs, content: content}
	summaries, total, err := PrepareDocSummaries(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(summaries) != 5 {
		t.Errorf("summaries = %d, want 5", len(summaries))
	}
	// Verify summary contains content from sections
	if summaries[0].Summary == "" {
		t.Error("summary should not be empty for docs with sections")
	}
}

func TestPrepareDocSummaries_MoreThan50Docs(t *testing.T) {
	docs := make([]domain.DocMeta, 60)
	content := make(map[string]string)
	for i := range docs {
		name := fmt.Sprintf("doc-%02d.md", i)
		docs[i] = domain.DocMeta{Filename: name, Type: "decision", Date: fmt.Sprintf("2026-%02d-%02d", (i/28)+1, (i%28)+1)}
		content[name] = "---\ntype: decision\n---\n## What\nContent here"
	}

	reader := &mockCorpusReader{docs: docs, content: content}
	summaries, total, err := PrepareDocSummaries(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 60 {
		t.Errorf("total = %d, want 60", total)
	}
	if len(summaries) != 50 {
		t.Errorf("summaries = %d, want 50 (25 newest + 25 oldest)", len(summaries))
	}
}

// --- severityRank unit tests ---

func TestSeverityRank_KnownSeverities(t *testing.T) {
	tests := []struct {
		sev  string
		want int
	}{
		{"contradiction", 0},
		{"gap", 1},
		{"obsolete", 2},
		{"style", 3},
	}
	for _, tt := range tests {
		got := severityRank(tt.sev)
		if got != tt.want {
			t.Errorf("severityRank(%q) = %d, want %d", tt.sev, got, tt.want)
		}
	}
}

func TestSeverityRank_UnknownReturnsHighValue(t *testing.T) {
	// Unknown severities should return len(severityOrder) to sort last.
	unknowns := []string{"info", "critical", "unknown", ""}
	expected := len(severityOrder)
	for _, sev := range unknowns {
		got := severityRank(sev)
		if got != expected {
			t.Errorf("severityRank(%q) = %d, want %d (high value for unknown)", sev, got, expected)
		}
		// Must be greater than any known severity rank
		if got <= severityRank("style") {
			t.Errorf("severityRank(%q) = %d, should be > severityRank(style)=%d", sev, got, severityRank("style"))
		}
	}
}

func TestTruncateRunes_Unicode(t *testing.T) {
	s := strings.Repeat("é", 250)
	result := truncateRunes(s, 200)
	if len([]rune(result)) != 200 {
		t.Errorf("rune count = %d, want 200", len([]rune(result)))
	}
}

// --- PrepareDocSummaries with ReviewFilter tests ---

func TestPrepareDocSummaries_PatternFilter_MatchesSubset(t *testing.T) {
	docs := make([]domain.DocMeta, 10)
	content := make(map[string]string)
	for i := range docs {
		name := fmt.Sprintf("doc-%02d.md", i)
		if i < 3 {
			name = fmt.Sprintf("decision-auth-%02d.md", i)
		}
		docs[i] = domain.DocMeta{Filename: name, Type: "decision", Date: fmt.Sprintf("2026-03-%02d", i+1)}
		content[name] = "---\ntype: decision\n---\n## What\nContent here\n\n## Why\nBecause reasons"
	}

	reader := &mockCorpusReader{docs: docs, content: content}
	pattern := regexp.MustCompile(`^decision-auth-`)
	summaries, total, err := PrepareDocSummaries(reader, ReviewFilter{Pattern: pattern})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3 (only matched docs)", total)
	}
	if len(summaries) != 3 {
		t.Errorf("summaries = %d, want 3", len(summaries))
	}
	for _, s := range summaries {
		if !strings.HasPrefix(s.Filename, "decision-auth-") {
			t.Errorf("unexpected filename %q in filtered results", s.Filename)
		}
	}
}

func TestPrepareDocSummaries_AllFlag_CorpusOver50(t *testing.T) {
	docs := make([]domain.DocMeta, 55)
	content := make(map[string]string)
	for i := range docs {
		name := fmt.Sprintf("doc-%03d.md", i)
		docs[i] = domain.DocMeta{Filename: name, Type: "feature", Date: fmt.Sprintf("2026-%02d-%02d", (i/28)+1, (i%28)+1)}
		content[name] = "---\ntype: feature\n---\n## What\nSomething\n\n## Why\nReasons"
	}

	reader := &mockCorpusReader{docs: docs, content: content}
	summaries, total, err := PrepareDocSummaries(reader, ReviewFilter{All: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 55 {
		t.Errorf("total = %d, want 55", total)
	}
	// With All=true, all 55 docs should be included (not 25+25 sampling)
	if len(summaries) != 55 {
		t.Errorf("summaries = %d, want 55 (All=true bypasses sampling)", len(summaries))
	}
}

func TestPrepareDocSummaries_PatternFilter_TooFewDocs_Error(t *testing.T) {
	docs := make([]domain.DocMeta, 10)
	content := make(map[string]string)
	for i := range docs {
		name := fmt.Sprintf("doc-%02d.md", i)
		docs[i] = domain.DocMeta{Filename: name, Type: "decision", Date: fmt.Sprintf("2026-03-%02d", i+1)}
		content[name] = "---\ntype: decision\n---\n## What\nContent"
	}

	reader := &mockCorpusReader{docs: docs, content: content}
	// Pattern that matches only 1 doc (less than minRequired=2 for filtered)
	pattern := regexp.MustCompile(`^doc-00\.md$`)
	_, total, err := PrepareDocSummaries(reader, ReviewFilter{Pattern: pattern})
	if err == nil {
		t.Fatal("expected error when fewer than 2 docs match pattern")
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if !strings.Contains(err.Error(), "2") {
		t.Errorf("error should mention minimum of 2, got: %q", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Review adaptive prompt — persona injection + agreement count
// ─────────────────────────────────────────────────────────────────────────────

// personaFixture returns two fake personas for tests. We construct them
// explicitly rather than pulling from GetRegistry() so the test assertions
// remain stable if the registry ordering or content changes.
func personaFixture() []PersonaProfile {
	return []PersonaProfile{
		{
			Name:            "security-senior",
			DisplayName:     "Security Senior",
			Icon:            "🔒",
			Expertise:       "Authentication, authorization, data protection",
			PromptDirective: "As Security Senior, flag missing auth flows and data exposure risks.",
			ReviewDirective: "SECURITY REVIEW LENS: flag cross-doc auth contradictions and security-section gaps.",
		},
		{
			Name:            "dx-lead",
			DisplayName:     "DX Lead",
			Icon:            "🧭",
			Expertise:       "Onboarding clarity and developer ergonomics",
			PromptDirective: "As DX Lead, flag jargon and missing context for new contributors.",
			ReviewDirective: "DX REVIEW LENS: flag onboarding gaps across the corpus and jargon inconsistencies.",
		},
	}
}

// TestBuildReviewPromptWithVHS_InjectsPersonas (AC-1, AC-2).
// When personas is non-nil and non-empty, the persona directive block must
// appear in the user content, and the system prompt must instruct the AI to
// attribute findings via "personas" and "agreement_count" fields.
func TestBuildReviewPromptWithVHS_InjectsPersonas(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-api.md", Type: "decision", Summary: "Pick REST vs gRPC."},
	}
	personas := personaFixture()

	sys, usr := BuildReviewPromptWithVHS(docs, "", nil, nil, personas)

	// System prompt — persona-aware block
	if !strings.Contains(sys, "PERSONA-AWARE REVIEW") {
		t.Error("system prompt missing PERSONA-AWARE REVIEW section")
	}
	if !strings.Contains(sys, "personas") || !strings.Contains(sys, "agreement_count") {
		t.Error("system prompt must instruct AI to emit personas and agreement_count fields")
	}
	if !strings.Contains(sys, "invariant I4") {
		t.Error("system prompt must call out I4 (zero hallucination) alongside persona rule")
	}

	// User content — persona names + review-specific directives must appear.
	// The review prompt uses BuildPersonaReviewPrompt, so we assert the
	// review-specific header AND the ReviewDirective text, NOT the draft
	// PromptDirective (which is explicitly not what review should inject).
	if !strings.Contains(usr, "YOUR EXPERT REVIEW TEAM") {
		t.Error("user content must include review-specific persona team header")
	}
	if !strings.Contains(usr, "Security Senior") {
		t.Error("user content must include first persona display name")
	}
	if !strings.Contains(usr, "DX Lead") {
		t.Error("user content must include second persona display name")
	}
	if !strings.Contains(usr, "SECURITY REVIEW LENS") {
		t.Error("user content must include first persona REVIEW directive, not draft directive")
	}
	if !strings.Contains(usr, "DX REVIEW LENS") {
		t.Error("user content must include second persona REVIEW directive, not draft directive")
	}
	// The draft-oriented PromptDirective text must NOT leak into the review
	// prompt — that was the whole point of the follow-up fix.
	if strings.Contains(usr, "flag missing auth flows and data exposure risks") {
		t.Error("review prompt leaked the DRAFT-oriented PromptDirective; must use ReviewDirective")
	}
}

// TestBuildReviewPromptWithVHS_NilPersonas_NoLeak (AC-5 prompt side).
// When personas is nil, neither the persona directive block NOR the output
// schema extension for personas must appear — this guarantees baseline
// prompt identity with the pre-8-19 behavior.
func TestBuildReviewPromptWithVHS_NilPersonas_NoLeak(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-api.md", Type: "decision", Summary: "Pick REST vs gRPC."},
	}

	sys, usr := BuildReviewPromptWithVHS(docs, "", nil, nil, nil)

	if strings.Contains(sys, "PERSONA-AWARE REVIEW") {
		t.Error("nil personas must NOT activate PERSONA-AWARE REVIEW block in system prompt")
	}
	if strings.Contains(sys, "agreement_count") {
		t.Error("nil personas must NOT mention agreement_count in system prompt")
	}
	if strings.Contains(usr, "YOUR EXPERT TEAM FOR THIS REVIEW") {
		t.Error("nil personas must NOT emit BuildPersonaPrompt header in user content")
	}
	if strings.Contains(usr, "YOUR EXPERT REVIEW TEAM") {
		t.Error("nil personas must NOT emit BuildPersonaReviewPrompt header in user content")
	}
}

// TestRegistryPersonas_AllCarryReviewDirective guards against adding a new
// persona to the registry without a review-specific directive. Without it,
// BuildPersonaReviewPrompt falls back to the draft-oriented PromptDirective
// with a WARNING line — acceptable as a graceful degradation, but the repo
// convention is that every registry persona ships a review directive.
func TestRegistryPersonas_AllCarryReviewDirective(t *testing.T) {
	for _, p := range GetRegistry() {
		if strings.TrimSpace(p.ReviewDirective) == "" {
			t.Errorf("persona %q (registry) is missing ReviewDirective — add one or rely on the fallback-with-warning path explicitly",
				p.Name)
		}
	}
}

// TestBuildPersonaReviewPrompt_MissingDirective_FallsBackWithWarning pins the
// fallback contract: a persona without ReviewDirective must produce the
// draft directive but precede it with an explicit WARNING line so the AI
// (and anyone reading the prompt) sees that this is a mis-targeted lens.
func TestBuildPersonaReviewPrompt_MissingDirective_FallsBackWithWarning(t *testing.T) {
	personas := []PersonaProfile{{
		Name:            "no-review-directive-persona",
		DisplayName:     "Missing Review",
		Icon:            "❓",
		Expertise:       "Has draft directive only",
		PromptDirective: "DRAFT-ONLY DIRECTIVE TEXT",
		// ReviewDirective: intentionally empty
	}}
	got := BuildPersonaReviewPrompt(personas)
	if !strings.Contains(got, "WARNING") {
		t.Errorf("fallback path must emit a WARNING line; got:\n%s", got)
	}
	if !strings.Contains(got, "DRAFT-ONLY DIRECTIVE TEXT") {
		t.Errorf("fallback path must include the PromptDirective; got:\n%s", got)
	}
}

func TestBuildPersonaReviewPrompt_EmptyInput_ReturnsEmpty(t *testing.T) {
	if got := BuildPersonaReviewPrompt(nil); got != "" {
		t.Errorf("nil input must yield empty string, got %q", got)
	}
	if got := BuildPersonaReviewPrompt([]PersonaProfile{}); got != "" {
		t.Errorf("empty slice must yield empty string, got %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Golden-file baseline guard: the system prompt produced for a no-persona
// review must remain byte-identical to a committed snapshot. Any change to
// the system prompt — including an innocuous whitespace refactor — breaks
// this test, forcing a deliberate decision (re-bless the golden or revert).
//
// The primary value is preventing silent drift that would invalidate
// upstream prompt-cache hits and violate the "zero-regression vs pre-
// persona" promise on the persona-aware review path.
//
// To re-bless after an intentional change:
//
//	LORE_UPDATE_GOLDEN=1 go test ./internal/angela/ -run TestSystemPromptBaseline_GoldenFile
// ─────────────────────────────────────────────────────────────────────────────

func TestSystemPromptBaseline_GoldenFile(t *testing.T) {
	// Fixed, minimal inputs so the golden stays small and reviewable.
	docs := []DocSummary{
		{Filename: "decision-api.md", Type: "decision", Date: "2026-03-01", Summary: "[Why] Choose REST | [What] HTTP 1.1"},
		{Filename: "feature-login.md", Type: "feature", Date: "2026-03-02", Summary: "[Why] Secure access | [What] JWT"},
	}
	sys, _ := BuildReviewPromptWithVHS(docs, "", nil, nil, nil)

	goldenPath := "testdata/review-system-prompt-baseline.golden"

	if os.Getenv("LORE_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(goldenPath, []byte(sys), 0644); err != nil {
			t.Fatalf("failed to write golden: %v", err)
		}
		t.Logf("golden updated: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("golden not found; re-run with LORE_UPDATE_GOLDEN=1 to create it: %v", err)
	}
	if string(want) != sys {
		t.Errorf("baseline system prompt drifted from golden file.\nTo re-bless an intentional change: LORE_UPDATE_GOLDEN=1 go test -run TestSystemPromptBaseline_GoldenFile ./internal/angela/")
	}
}

// TestReview_NoPersonas_Baseline (AC-5).
// Review with nil personas produces findings with zero-valued Personas and
// AgreementCount. Both must be omitted from JSON (json:",omitempty").
func TestReview_NoPersonas_Baseline(t *testing.T) {
	response := `{"findings":[{
		"severity":"gap",
		"title":"Missing onboarding",
		"description":"No doc explains setup.",
		"documents":["README.md"],
		"confidence":0.8
	}]}`
	provider := newMockProviderWith(response, nil)

	report, err := Review(context.Background(), provider, []DocSummary{{Filename: "README.md"}}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(report.Findings))
	}
	f := report.Findings[0]
	if len(f.Personas) != 0 {
		t.Errorf("baseline review must produce zero-length Personas, got %v", f.Personas)
	}
	if f.AgreementCount != 0 {
		t.Errorf("baseline review must produce AgreementCount=0, got %d", f.AgreementCount)
	}
}

// TestReview_AgreementCount_Aggregation (AC-3).
// When the AI returns a finding with multiple personas, AgreementCount in the
// parsed result must equal len(Personas). The test exercises the JSON round-trip.
func TestReview_AgreementCount_Aggregation(t *testing.T) {
	response := `{"findings":[{
		"severity":"gap",
		"title":"No auth documented",
		"description":"Missing auth section.",
		"documents":["feature-login.md"],
		"evidence":[{"file":"feature-login.md","quote":"POST /login"}],
		"confidence":0.9,
		"personas":["security-senior","dx-lead"],
		"agreement_count":2
	}]}`
	provider := newMockProviderWith(response, nil)

	report, err := Review(context.Background(), provider,
		[]DocSummary{{Filename: "feature-login.md", Summary: "POST /login"}}, "",
		ReviewOpts{Personas: personaFixture()},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(report.Findings))
	}
	f := report.Findings[0]
	if len(f.Personas) != 2 {
		t.Errorf("expected 2 personas on finding, got %d", len(f.Personas))
	}
	if f.AgreementCount != 2 {
		t.Errorf("expected AgreementCount=2, got %d", f.AgreementCount)
	}
}

// TestReview_PersonaOrderInvariant (AC-6).
// Given a deterministic stub (same response regardless of prompt), two runs
// with persona lists in reversed order must produce the same set of findings
// (by title). Order of the Personas slice INSIDE a finding can differ — what
// matters is the set equality.
func TestReview_PersonaOrderInvariant(t *testing.T) {
	response := `{"findings":[
		{"severity":"gap","title":"A","description":"x","documents":["d.md"],
		 "evidence":[{"file":"d.md","quote":"x"}],"confidence":0.8,
		 "personas":["security-senior"],"agreement_count":1},
		{"severity":"style","title":"B","description":"y","documents":["d.md"],
		 "evidence":[{"file":"d.md","quote":"x"}],"confidence":0.8,
		 "personas":["dx-lead"],"agreement_count":1}
	]}`

	personas := personaFixture()
	reversed := []PersonaProfile{personas[1], personas[0]}
	docs := []DocSummary{{Filename: "d.md", Summary: "x"}}

	r1, err1 := Review(context.Background(), newMockProviderWith(response, nil), docs, "",
		ReviewOpts{Personas: personas})
	r2, err2 := Review(context.Background(), newMockProviderWith(response, nil), docs, "",
		ReviewOpts{Personas: reversed})
	if err1 != nil || err2 != nil {
		t.Fatalf("review errors: %v, %v", err1, err2)
	}

	titles := func(rep *ReviewReport) map[string]bool {
		out := map[string]bool{}
		for _, f := range rep.Findings {
			out[f.Title] = true
		}
		return out
	}
	t1, t2 := titles(r1), titles(r2)
	if len(t1) != len(t2) {
		t.Fatalf("finding count differs: %d vs %d", len(t1), len(t2))
	}
	for k := range t1 {
		if !t2[k] {
			t.Errorf("finding title %q in run1 but not run2 — order invariance violated", k)
		}
	}
}

// TestReview_AdaptivePrompt_HonorsI4 (AC-4).
// A persona-attributed finding with an empty or missing quote must be
// rejected by the evidence validator just like a non-persona finding.
// Persona activation does NOT loosen I4.
func TestReview_AdaptivePrompt_HonorsI4(t *testing.T) {
	// Finding has personas attributed but evidence is empty (no quote) — I4 violation.
	response := `{"findings":[{
		"severity":"gap",
		"title":"Halucinated persona finding",
		"description":"Persona sees missing MFA but evidence is empty.",
		"documents":["feature-login.md"],
		"evidence":[{"file":"feature-login.md","quote":""}],
		"confidence":0.9,
		"personas":["security-senior"],
		"agreement_count":1
	}]}`
	provider := newMockProviderWith(response, nil)
	reader := &mockCorpusReader{
		docs:    []domain.DocMeta{{Filename: "feature-login.md", Type: "feature"}},
		content: map[string]string{"feature-login.md": "POST /login returns token"},
	}

	report, err := Review(context.Background(), provider,
		[]DocSummary{{Filename: "feature-login.md", Summary: "POST /login"}}, "",
		ReviewOpts{
			Personas: personaFixture(),
			Reader:   reader,
			Evidence: EvidenceValidation{Required: true, Mode: EvidenceModeStrict},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// I4: the finding must be rejected despite being persona-attributed.
	if len(report.Findings) != 0 {
		t.Errorf("I4 violated: persona-attributed finding with empty quote accepted; got %d findings", len(report.Findings))
	}
	if len(report.Rejected) == 0 {
		t.Error("I4: evidence-less persona finding must land in Rejected slice")
	}
}
