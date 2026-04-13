// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"testing"
)

// TestCheckHallucinations_UnchangedDocNoClaims — AC-10: identical
// original and polished → no claims at all.
func TestCheckHallucinations_UnchangedDocNoClaims(t *testing.T) {
	doc := "## Why\n\nWe chose Go for its simplicity.\n\n## How\n\nStep one.\n"
	hc := CheckHallucinations(doc, doc, "")
	if len(hc.NewFactualClaims) != 0 {
		t.Errorf("unchanged doc should have 0 claims, got %d", len(hc.NewFactualClaims))
	}
	if len(hc.Unsupported) != 0 {
		t.Errorf("unchanged doc should have 0 unsupported, got %d", len(hc.Unsupported))
	}
}

// TestCheckHallucinations_OriginalNumberSurvivesReformat — AC-9:
// "200 ms" in the original should support "200ms" in the polished.
func TestCheckHallucinations_OriginalNumberSurvivesReformat(t *testing.T) {
	original := "## Performance\n\nLatency is 200 ms on average.\n"
	polished := "## Performance\n\nAverage latency is 200ms.\n"
	hc := CheckHallucinations(original, polished, "")
	for _, u := range hc.Unsupported {
		if u.Core == "200ms" || u.Core == "200 ms" {
			t.Errorf("'200ms' should be supported (original has '200 ms'), got unsupported")
		}
	}
}

// TestCheckHallucinations_InventedLatencyMetricDetected — AC-10:
// a metric not in the original is flagged.
func TestCheckHallucinations_InventedLatencyMetricDetected(t *testing.T) {
	original := "## Performance\n\nWe improved latency.\n"
	polished := "## Performance\n\nLatency was reduced from 200ms to 45ms.\n"
	hc := CheckHallucinations(original, polished, "")
	if len(hc.Unsupported) == 0 {
		t.Fatal("expected unsupported claims for invented metrics")
	}
	found := false
	for _, u := range hc.Unsupported {
		if u.Type == "metric" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one 'metric' type unsupported claim")
	}
}

// TestCheckHallucinations_InventedVersionDetected — AC-10: a version
// string not in the original is flagged.
func TestCheckHallucinations_InventedVersionDetected(t *testing.T) {
	original := "## Stack\n\nWe use PostgreSQL.\n"
	polished := "## Stack\n\nWe use PostgreSQL 15.3 for persistence.\n"
	hc := CheckHallucinations(original, polished, "")
	found := false
	for _, u := range hc.Unsupported {
		if u.Type == "version" && u.Core == "15.3" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unsupported version claim '15.3', got %+v", hc.Unsupported)
	}
}

// TestCheckHallucinations_InventedProperNounDetected — AC-10: a tech
// proper noun not in the original is flagged.
func TestCheckHallucinations_InventedProperNounDetected(t *testing.T) {
	original := "## Stack\n\nWe use a relational database.\n"
	polished := "## Stack\n\nWe use PostgreSQL as our relational database.\n"
	hc := CheckHallucinations(original, polished, "")
	found := false
	for _, u := range hc.Unsupported {
		if u.Type == "proper-noun" && u.Core == "PostgreSQL" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unsupported proper-noun 'PostgreSQL', got %+v", hc.Unsupported)
	}
}

// TestCheckHallucinations_ClaimInCorpusSummaryIsSupported — AC-10:
// a claim that appears in the corpus summary is considered supported.
func TestCheckHallucinations_ClaimInCorpusSummaryIsSupported(t *testing.T) {
	original := "## Stack\n\nWe use a database.\n"
	polished := "## Stack\n\nWe use Redis for caching.\n"
	corpusSummary := "The project uses Redis for session caching."
	hc := CheckHallucinations(original, polished, corpusSummary)
	for _, u := range hc.Unsupported {
		if u.Core == "Redis" {
			t.Error("'Redis' should be supported via corpus summary")
		}
	}
}

// TestExtractClaims_MetricPatterns verifies the metric regex.
func TestExtractClaims_MetricPatterns(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Latency dropped to 45ms.", "45ms"},
		{"We saved 30% on costs.", "30%"},
		{"Throughput reached 1000 req/s.", "1000 req/s"},
		{"File size was 512MB.", "512MB"},
	}
	for _, tc := range cases {
		claims := extractClaims(tc.input, "")
		found := false
		for _, c := range claims {
			if c.Core == tc.want && c.Type == "metric" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("input=%q: expected metric claim %q, got %+v", tc.input, tc.want, claims)
		}
	}
}

// TestExtractClaims_VersionPatterns verifies the version regex.
func TestExtractClaims_VersionPatterns(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Uses Go v1.22.0 for builds.", "v1.22.0"},
		{"PostgreSQL 15.3 is required.", "15.3"},
		{"Compatible with 2.0 API.", "2.0"},
	}
	for _, tc := range cases {
		claims := extractClaims(tc.input, "")
		found := false
		for _, c := range claims {
			if c.Core == tc.want && c.Type == "version" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("input=%q: expected version claim %q, got %+v", tc.input, tc.want, claims)
		}
	}
}

// TestIsSupported_SpaceInsensitive verifies AC-9 normalization.
func TestIsSupported_SpaceInsensitive(t *testing.T) {
	origNorm := normalizeForClaim("response time is 200 ms")
	c := FactualClaim{Core: "200ms", Type: "metric"}
	if !isSupported(c, origNorm, "") {
		t.Error("'200ms' should match '200 ms' after normalization")
	}
}

// TestNewSentences_BasicDiff verifies that only truly new sentences
// are detected.
func TestNewSentences_BasicDiff(t *testing.T) {
	orig := "We use Go. It is fast."
	polished := "We use Go. It is fast. It compiles quickly."
	ns := newSentences(orig, polished)
	if len(ns) != 1 {
		t.Fatalf("expected 1 new sentence, got %d: %v", len(ns), ns)
	}
	if ns[0] != "It compiles quickly." {
		t.Errorf("unexpected new sentence: %q", ns[0])
	}
}
