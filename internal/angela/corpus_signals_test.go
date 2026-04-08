// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
	"testing"
)

func TestAnalyzeCorpusSignals_PotentialPairs(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-auth-jwt-2026-01-01.md", Type: "decision", Date: "2026-01-01", Tags: []string{"auth"}},
		{Filename: "decision-auth-session-2026-03-01.md", Type: "decision", Date: "2026-03-01", Tags: []string{"auth"}},
		{Filename: "feature-api-2026-02-01.md", Type: "feature", Date: "2026-02-01", Tags: []string{"api"}},
	}

	signals := AnalyzeCorpusSignals(docs)

	if len(signals.PotentialPairs) != 1 {
		t.Fatalf("expected 1 potential pair, got %d", len(signals.PotentialPairs))
	}
	pair := signals.PotentialPairs[0]
	if pair.Type != "decision" {
		t.Errorf("pair type = %q, want decision", pair.Type)
	}
	if !strings.Contains(pair.Tags, "auth") {
		t.Errorf("pair tags = %q, want 'auth'", pair.Tags)
	}
	if pair.DaysDiff < 14 {
		t.Errorf("pair days diff = %d, want >= 14", pair.DaysDiff)
	}
}

func TestAnalyzeCorpusSignals_NoPairsWhenCloseInTime(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-a.md", Type: "decision", Date: "2026-03-01", Tags: []string{"api"}},
		{Filename: "decision-b.md", Type: "decision", Date: "2026-03-05", Tags: []string{"api"}},
	}

	signals := AnalyzeCorpusSignals(docs)

	if len(signals.PotentialPairs) != 0 {
		t.Errorf("expected 0 pairs for close dates, got %d", len(signals.PotentialPairs))
	}
}

func TestAnalyzeCorpusSignals_IsolatedDocs(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-auth.md", Type: "decision", Date: "2026-01-01", Tags: []string{"auth"}},
		{Filename: "decision-auth2.md", Type: "decision", Date: "2026-03-01", Tags: []string{"auth"}},
		{Filename: "feature-lonely.md", Type: "feature", Date: "2026-02-01", Tags: []string{"unique-tag"}},
	}

	signals := AnalyzeCorpusSignals(docs)

	if len(signals.IsolatedDocs) != 1 {
		t.Fatalf("expected 1 isolated doc, got %d", len(signals.IsolatedDocs))
	}
	if signals.IsolatedDocs[0] != "feature-lonely.md" {
		t.Errorf("isolated = %q, want feature-lonely.md", signals.IsolatedDocs[0])
	}
}

func TestAnalyzeCorpusSignals_TagClusters(t *testing.T) {
	docs := []DocSummary{
		{Filename: "a.md", Tags: []string{"auth", "api"}},
		{Filename: "b.md", Tags: []string{"auth"}},
		{Filename: "c.md", Tags: []string{"db"}},
	}

	signals := AnalyzeCorpusSignals(docs)

	if len(signals.TagClusters["auth"]) != 2 {
		t.Errorf("auth cluster = %d docs, want 2", len(signals.TagClusters["auth"]))
	}
	if len(signals.TagClusters["api"]) != 1 {
		t.Errorf("api cluster = %d docs, want 1", len(signals.TagClusters["api"]))
	}
	if len(signals.TagClusters["db"]) != 1 {
		t.Errorf("db cluster = %d docs, want 1", len(signals.TagClusters["db"]))
	}
}

func TestAnalyzeCorpusSignals_TypeDistribution(t *testing.T) {
	docs := []DocSummary{
		{Filename: "a.md", Type: "decision"},
		{Filename: "b.md", Type: "decision"},
		{Filename: "c.md", Type: "feature"},
	}

	signals := AnalyzeCorpusSignals(docs)

	if signals.TypeDistribution["decision"] != 2 {
		t.Errorf("decision count = %d, want 2", signals.TypeDistribution["decision"])
	}
	if signals.TypeDistribution["feature"] != 1 {
		t.Errorf("feature count = %d, want 1", signals.TypeDistribution["feature"])
	}
}

func TestAnalyzeCorpusSignals_EmptyCorpus(t *testing.T) {
	signals := AnalyzeCorpusSignals(nil)
	if len(signals.PotentialPairs) != 0 {
		t.Error("expected no pairs for empty corpus")
	}
	if len(signals.IsolatedDocs) != 0 {
		t.Error("expected no isolated docs for empty corpus")
	}
}

func TestAnalyzeCorpusSignals_LimitsPairsTo10(t *testing.T) {
	// Create 20 decisions with same tag but spread across dates
	docs := make([]DocSummary, 20)
	for i := range docs {
		docs[i] = DocSummary{
			Filename: strings.Replace("decision-X-2026-MM-01.md", "X", string(rune('a'+i)), 1),
			Type:     "decision",
			Date:     strings.Replace("2025-MM-01", "MM", padInt(i+1), 1),
			Tags:     []string{"common"},
		}
	}

	signals := AnalyzeCorpusSignals(docs)

	if len(signals.PotentialPairs) > 10 {
		t.Errorf("expected max 10 pairs, got %d", len(signals.PotentialPairs))
	}
}

func padInt(n int) string {
	return fmt.Sprintf("%02d", n)
}

func TestApproxDaysDiff(t *testing.T) {
	// ~60 days between Jan 1 and Mar 1
	diff := approxDaysDiff("2026-01-01", "2026-03-01")
	if diff < 50 || diff > 70 {
		t.Errorf("diff = %d, expected ~60", diff)
	}
}

func TestApproxDaysDiff_InvalidDates(t *testing.T) {
	diff := approxDaysDiff("short", "also-short")
	if diff != 0 {
		t.Errorf("diff = %d, expected 0 for invalid dates", diff)
	}
}

func TestAnalyzeCorpusSignals_EmptyDocsSlice(t *testing.T) {
	signals := AnalyzeCorpusSignals([]DocSummary{})
	if signals == nil {
		t.Fatal("expected non-nil signals for empty slice")
	}
	if len(signals.PotentialPairs) != 0 {
		t.Error("expected no pairs for empty slice")
	}
	if len(signals.IsolatedDocs) != 0 {
		t.Error("expected no isolated docs for empty slice")
	}
	if len(signals.TagClusters) != 0 {
		t.Error("expected empty tag clusters for empty slice")
	}
	if len(signals.ScopeClusters) != 0 {
		t.Error("expected empty scope clusters for empty slice")
	}
	if len(signals.TypeDistribution) != 0 {
		t.Error("expected empty type distribution for empty slice")
	}
}

func TestAnalyzeCorpusSignals_DocsWithNoScope(t *testing.T) {
	docs := []DocSummary{
		{Filename: "a.md", Type: "decision", Date: "2026-01-01", Tags: []string{"auth"}},
		{Filename: "b.md", Type: "decision", Date: "2026-06-01", Tags: []string{"auth"}},
	}

	signals := AnalyzeCorpusSignals(docs)

	// Should still detect pair via shared tags
	if len(signals.PotentialPairs) == 0 {
		t.Error("expected potential pair from shared tags even without scope")
	}
	// No scope clusters should exist
	if len(signals.ScopeClusters) != 0 {
		t.Error("expected empty scope clusters when no docs have scope")
	}
}

func TestAnalyzeCorpusSignals_DocsWithSameScope(t *testing.T) {
	docs := []DocSummary{
		{Filename: "a.md", Type: "feature", Date: "2026-01-01", Scope: "auth"},
		{Filename: "b.md", Type: "feature", Date: "2026-01-15", Scope: "auth"},
		{Filename: "c.md", Type: "feature", Date: "2026-01-02", Scope: "auth"},
	}

	signals := AnalyzeCorpusSignals(docs)

	// All 3 should be in the same scope cluster
	if len(signals.ScopeClusters["auth"]) != 3 {
		t.Errorf("expected 3 docs in auth scope cluster, got %d", len(signals.ScopeClusters["auth"]))
	}

	// Should detect unconsolidated scope (3 docs, no summary type)
	if len(signals.UnconsolidatedScopes) != 1 {
		t.Fatalf("expected 1 unconsolidated scope, got %d", len(signals.UnconsolidatedScopes))
	}
	if signals.UnconsolidatedScopes[0].Scope != "auth" {
		t.Errorf("unconsolidated scope = %q, want auth", signals.UnconsolidatedScopes[0].Scope)
	}
	if signals.UnconsolidatedScopes[0].DocCount != 3 {
		t.Errorf("unconsolidated doc count = %d, want 3", signals.UnconsolidatedScopes[0].DocCount)
	}

	// Same-scope pairs use 7-day threshold; a.md and b.md are 14 days apart
	hasPair := false
	for _, p := range signals.PotentialPairs {
		if (p.DocA == "a.md" && p.DocB == "b.md") || (p.DocA == "b.md" && p.DocB == "a.md") {
			hasPair = true
			if !strings.Contains(p.Tags, "scope:auth") {
				t.Errorf("same-scope pair should have tags containing 'scope:auth', got %q", p.Tags)
			}
		}
	}
	if !hasPair {
		t.Error("expected potential pair between a.md and b.md (14 days apart, same scope, 7-day threshold)")
	}
}
