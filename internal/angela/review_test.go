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
