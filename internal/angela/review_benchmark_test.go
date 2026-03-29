// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// ============================================================================
// BENCHMARK CORPUS — Known contradictions, gaps, and style issues.
//
// This corpus is the quality baseline for Angela review improvements.
// Any change to ExtractAdaptiveSummary, AnalyzeCorpusSignals, or BuildReviewPrompt
// MUST be validated against this benchmark.
//
// PLANTED ISSUES:
//   C1: Contradiction — JWT vs session auth (same type, shared tag, 2 months apart)
//   C2: Contradiction — REST vs GraphQL API (same type, shared tag, 3 months apart)
//   C3: Obsolete — old caching decision superseded by newer one
//   G1: Gap — "monitoring" referenced in feature-api but no monitoring doc exists
//   S1: Style — inconsistent terminology "endpoint" vs "route"
// ============================================================================

func benchmarkCorpus() ([]domain.DocMeta, map[string]string) {
	docs := []domain.DocMeta{
		// C1: JWT auth decision (old)
		{Filename: "decision-auth-jwt-2026-01-15.md", Type: "decision", Date: "2026-01-15", Tags: []string{"auth", "security"}},
		// C1: Session auth decision (new, contradicts JWT)
		{Filename: "decision-auth-session-2026-03-10.md", Type: "decision", Date: "2026-03-10", Tags: []string{"auth", "security"}},
		// C2: REST API decision (old)
		{Filename: "decision-api-rest-2025-12-01.md", Type: "decision", Date: "2025-12-01", Tags: []string{"api", "architecture"}},
		// C2: GraphQL API decision (new, contradicts REST)
		{Filename: "decision-api-graphql-2026-03-05.md", Type: "decision", Date: "2026-03-05", Tags: []string{"api", "architecture"}},
		// C3: Old caching decision (obsolete)
		{Filename: "decision-cache-redis-2025-11-01.md", Type: "decision", Date: "2025-11-01", Tags: []string{"cache", "performance"}},
		// C3: New caching decision (supersedes old)
		{Filename: "decision-cache-valkey-2026-02-20.md", Type: "decision", Date: "2026-02-20", Tags: []string{"cache", "performance"}},
		// G1: Feature referencing monitoring (gap — no monitoring doc)
		{Filename: "feature-api-gateway-2026-03-01.md", Type: "feature", Date: "2026-03-01", Tags: []string{"api", "gateway"}},
		// S1: Uses "endpoint" terminology
		{Filename: "feature-user-endpoints-2026-02-15.md", Type: "feature", Date: "2026-02-15", Tags: []string{"api", "users"}},
		// S1: Uses "route" terminology (inconsistent with above)
		{Filename: "refactor-routing-2026-02-20.md", Type: "refactor", Date: "2026-02-20", Tags: []string{"api", "routing"}},
		// Clean doc (no issues, provides baseline)
		{Filename: "bugfix-login-timeout-2026-03-12.md", Type: "bugfix", Date: "2026-03-12", Tags: []string{"auth", "bugfix"}},
	}

	content := map[string]string{
		"decision-auth-jwt-2026-01-15.md": `---
type: decision
date: "2026-01-15"
tags: [auth, security]
---
## What
We chose JWT tokens for API authentication. All endpoints require a Bearer token.

## Why
Stateless authentication scales better with microservices. No session store needed.

## Alternatives
Session-based auth was considered but rejected due to scalability concerns.`,

		"decision-auth-session-2026-03-10.md": `---
type: decision
date: "2026-03-10"
tags: [auth, security]
---
## What
We are migrating to session-based authentication with Redis session store.

## Why
JWT token size is causing performance issues with large payloads. Sessions keep auth data server-side.

## Impact
All API clients must update to use session cookies instead of Bearer tokens.`,

		"decision-api-rest-2025-12-01.md": `---
type: decision
date: "2025-12-01"
tags: [api, architecture]
---
## What
REST API with JSON responses. Standard HTTP methods (GET, POST, PUT, DELETE).

## Why
REST is the team standard. Well-understood, good tooling support.

## Alternatives
GraphQL was considered but deemed overkill for our use case.`,

		"decision-api-graphql-2026-03-05.md": `---
type: decision
date: "2026-03-05"
tags: [api, architecture]
---
## What
We are adopting GraphQL for all new API endpoints. Existing REST routes will be deprecated.

## Why
Frontend team needs flexible queries. REST over-fetching is causing performance problems.

## Impact
REST endpoints will be maintained for 6 months then removed.`,

		"decision-cache-redis-2025-11-01.md": `---
type: decision
date: "2025-11-01"
tags: [cache, performance]
---
## What
Redis for all caching needs. Single instance, no clustering.

## Why
Simple setup, team familiarity.`,

		"decision-cache-valkey-2026-02-20.md": `---
type: decision
date: "2026-02-20"
tags: [cache, performance]
---
## What
Migrating from Redis to Valkey. Cluster mode enabled for high availability.

## Why
Redis licensing changes. Valkey is the community fork with better clustering support.

## Impact
All Redis connection strings must be updated. Cluster-aware client required.`,

		"feature-api-gateway-2026-03-01.md": `---
type: feature
date: "2026-03-01"
tags: [api, gateway]
---
## What
API gateway with rate limiting and request routing.

## Why
Centralized entry point for all microservices. Monitoring integration planned for Q2.

## Impact
All internal services must register with the gateway.`,

		"feature-user-endpoints-2026-02-15.md": `---
type: feature
date: "2026-02-15"
tags: [api, users]
---
## What
User management endpoints: create, read, update, delete users via REST.

## Why
Core functionality for the platform. All endpoints return JSON.`,

		"refactor-routing-2026-02-20.md": `---
type: refactor
date: "2026-02-20"
tags: [api, routing]
---
## What
Refactored API routes to use centralized router. All routes now follow RESTful conventions.

## Why
Previous routing was scattered across files. Centralization improves maintainability.`,

		"bugfix-login-timeout-2026-03-12.md": `---
type: bugfix
date: "2026-03-12"
tags: [auth, bugfix]
---
## What
Fixed login timeout issue where sessions expired after 5 minutes.

## Why
The timeout was hardcoded instead of using the configured value from environment.`,
	}

	return docs, content
}

// --- Benchmark: AnalyzeCorpusSignals detection ---

func TestBenchmark_SignalsDetectPlantedPairs(t *testing.T) {
	docs, content := benchmarkCorpus()

	// Build summaries like the real pipeline
	summaries := make([]DocSummary, len(docs))
	for i, meta := range docs {
		summary := ExtractAdaptiveSummary(content[meta.Filename], 450)
		summaries[i] = DocSummary{
			Filename: meta.Filename,
			Type:     meta.Type,
			Date:     meta.Date,
			Tags:     meta.Tags,
			Summary:  summary,
		}
	}

	signals := AnalyzeCorpusSignals(summaries)

	// C1: JWT vs session auth — same type "decision", shared tag "auth", ~54 days apart
	foundC1 := findPair(signals.PotentialPairs, "decision-auth-jwt", "decision-auth-session")
	if !foundC1 {
		t.Error("RECALL FAIL [C1]: AnalyzeCorpusSignals did not detect JWT vs session auth pair")
	}

	// C2: REST vs GraphQL — same type "decision", shared tag "api", ~94 days apart
	foundC2 := findPair(signals.PotentialPairs, "decision-api-rest", "decision-api-graphql")
	if !foundC2 {
		t.Error("RECALL FAIL [C2]: AnalyzeCorpusSignals did not detect REST vs GraphQL pair")
	}

	// C3: Redis vs Valkey — same type "decision", shared tag "cache", ~112 days apart
	foundC3 := findPair(signals.PotentialPairs, "decision-cache-redis", "decision-cache-valkey")
	if !foundC3 {
		t.Error("RECALL FAIL [C3]: AnalyzeCorpusSignals did not detect Redis vs Valkey pair")
	}

	// Recall: 3/3 contradictions detected = 100%
	detected := 0
	if foundC1 {
		detected++
	}
	if foundC2 {
		detected++
	}
	if foundC3 {
		detected++
	}
	t.Logf("BENCHMARK RECALL: %d/3 planted contradictions detected by AnalyzeCorpusSignals", detected)

	// Precision: check false positive rate
	// Expected pairs: C1, C2, C3 = 3. Some additional pairs may appear (same type + shared tag)
	// but they should be few.
	t.Logf("BENCHMARK PRECISION: %d total pairs detected (3 expected, rest are false positives)", len(signals.PotentialPairs))
	if len(signals.PotentialPairs) > 10 {
		t.Errorf("Too many pairs (%d) — pre-analysis is too noisy", len(signals.PotentialPairs))
	}
}

// --- Benchmark: Prompt contains critical content ---

func TestBenchmark_PromptContainsCriticalContent(t *testing.T) {
	docs, content := benchmarkCorpus()

	summaries := make([]DocSummary, len(docs))
	for i, meta := range docs {
		summary := ExtractAdaptiveSummary(content[meta.Filename], 450)
		summaries[i] = DocSummary{
			Filename: meta.Filename,
			Type:     meta.Type,
			Date:     meta.Date,
			Tags:     meta.Tags,
			Summary:  summary,
		}
	}

	signals := AnalyzeCorpusSignals(summaries)
	sys, usr := BuildReviewPrompt(summaries, "", signals)
	prompt := sys + usr // combine for content checks

	// Verify critical content is in the prompt for each planted issue

	// C1: Auth contradiction — both docs must be present with their opposing content
	checks := []struct {
		id    string
		terms []string
	}{
		{"C1-jwt", []string{"JWT", "decision-auth-jwt"}},
		{"C1-session", []string{"session", "decision-auth-session"}},
		{"C2-rest", []string{"REST", "decision-api-rest"}},
		{"C2-graphql", []string{"GraphQL", "decision-api-graphql"}},
		{"C3-redis", []string{"Redis", "decision-cache-redis"}},
		{"C3-valkey", []string{"Valkey", "decision-cache-valkey"}},
		// G1: "monitoring" is in a short ## Why section — adaptive summary scores Why at 30
		{"G1-gateway", []string{"feature-api-gateway"}},
		{"S1-endpoint", []string{"endpoint", "feature-user-endpoints"}},
		{"S1-route", []string{"route", "refactor-routing"}},
	}

	for _, check := range checks {
		for _, term := range check.terms {
			if !strings.Contains(prompt, term) {
				t.Errorf("PROMPT QUALITY [%s]: missing %q — AI won't be able to detect this issue", check.id, term)
			}
		}
	}

	// Verify TOON signals are embedded in corpus section
	if !strings.Contains(usr, "contradiction|") {
		t.Error("PROMPT QUALITY: missing TOON contradiction signals")
	}

	// Verify type grouping instruction
	if !strings.Contains(sys, "Group documents by Type") {
		t.Error("PROMPT QUALITY: missing type grouping instruction")
	}

	// Verify TOON format
	if !strings.Contains(usr, "corpus:") {
		t.Error("PROMPT QUALITY: missing TOON corpus: header")
	}

	t.Logf("BENCHMARK PROMPT: %d tokens (approx %d chars)", len(prompt)/4, len(prompt))
}

// --- Benchmark: Mock AI recall simulation ---

func TestBenchmark_MockAIRecall(t *testing.T) {
	// Simulate an AI that correctly identifies all planted issues
	idealResponse := `{"findings": [
		{"severity": "contradiction", "title": "Conflicting auth approaches", "description": "JWT vs session-based auth", "documents": ["decision-auth-jwt-2026-01-15.md", "decision-auth-session-2026-03-10.md"]},
		{"severity": "contradiction", "title": "REST vs GraphQL API", "description": "Conflicting API strategies", "documents": ["decision-api-rest-2025-12-01.md", "decision-api-graphql-2026-03-05.md"]},
		{"severity": "obsolete", "title": "Redis caching superseded", "description": "Valkey replaces Redis", "documents": ["decision-cache-redis-2025-11-01.md", "decision-cache-valkey-2026-02-20.md"]},
		{"severity": "gap", "title": "Missing monitoring documentation", "description": "Monitoring referenced in API gateway but no doc exists", "documents": ["feature-api-gateway-2026-03-01.md"]},
		{"severity": "style", "title": "Inconsistent terminology", "description": "endpoint vs route", "documents": ["feature-user-endpoints-2026-02-15.md", "refactor-routing-2026-02-20.md"]}
	]}`

	provider := newMockProviderWith(idealResponse, nil)
	docs, content := benchmarkCorpus()

	summaries := make([]DocSummary, len(docs))
	for i, meta := range docs {
		summary := ExtractAdaptiveSummary(content[meta.Filename], 450)
		summaries[i] = DocSummary{
			Filename: meta.Filename,
			Type:     meta.Type,
			Date:     meta.Date,
			Tags:     meta.Tags,
			Summary:  summary,
		}
	}

	report, err := Review(context.Background(), provider, summaries, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify sorting: contradiction > gap > obsolete > style
	if len(report.Findings) != 5 {
		t.Fatalf("expected 5 findings, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != "contradiction" {
		t.Errorf("finding[0] = %q, want contradiction", report.Findings[0].Severity)
	}
	// After sorting: contradiction, contradiction, gap, obsolete, style
	if report.Findings[2].Severity != "gap" {
		t.Errorf("finding[2] = %q, want gap (sorted after contradictions)", report.Findings[2].Severity)
	}
	if report.Findings[3].Severity != "obsolete" {
		t.Errorf("finding[3] = %q, want obsolete", report.Findings[3].Severity)
	}

	// Count by severity
	counts := map[string]int{}
	for _, f := range report.Findings {
		counts[f.Severity]++
	}

	t.Logf("BENCHMARK AI RECALL: %d contradiction, %d gap, %d obsolete, %d style",
		counts["contradiction"], counts["gap"], counts["obsolete"], counts["style"])
	t.Logf("BENCHMARK: Ideal AI detects 5/5 planted issues (100%% recall)")
}

// --- Benchmark: ExtractAdaptiveSummary captures contradiction evidence ---

func TestBenchmark_SummaryCaptures_ContradictionEvidence(t *testing.T) {
	_, content := benchmarkCorpus()

	// C1: JWT doc should have "JWT" or "stateless" in summary
	jwtSummary := ExtractAdaptiveSummary(content["decision-auth-jwt-2026-01-15.md"], 450)
	if !strings.Contains(jwtSummary, "JWT") && !strings.Contains(jwtSummary, "stateless") {
		t.Errorf("C1: JWT summary missing key term. Got: %q", jwtSummary)
	}

	// C1: Session doc should have "session" or "migrating" in summary
	sessionSummary := ExtractAdaptiveSummary(content["decision-auth-session-2026-03-10.md"], 450)
	if !strings.Contains(sessionSummary, "session") {
		t.Errorf("C1: Session summary missing 'session'. Got: %q", sessionSummary)
	}

	// C2: GraphQL doc should have "GraphQL" in summary
	graphqlSummary := ExtractAdaptiveSummary(content["decision-api-graphql-2026-03-05.md"], 450)
	if !strings.Contains(graphqlSummary, "GraphQL") {
		t.Errorf("C2: GraphQL summary missing 'GraphQL'. Got: %q", graphqlSummary)
	}

	// G1: Gateway doc should have "monitoring" in summary — adaptive scores Why at 30
	gatewaySummary := ExtractAdaptiveSummary(content["feature-api-gateway-2026-03-01.md"], 450)
	if !strings.Contains(strings.ToLower(gatewaySummary), "monitoring") {
		t.Errorf("G1: Gateway summary missing 'monitoring'. Got: %q", gatewaySummary)
	}
}

// --- Helpers ---

func findPair(pairs []DocPair, substringA, substringB string) bool {
	for _, p := range pairs {
		aInA := strings.Contains(p.DocA, substringA)
		bInB := strings.Contains(p.DocB, substringB)
		aInB := strings.Contains(p.DocA, substringB)
		bInA := strings.Contains(p.DocB, substringA)
		if (aInA && bInB) || (aInB && bInA) {
			return true
		}
	}
	return false
}

// --- Benchmark summary helper ---

func TestBenchmark_PrintSummary(t *testing.T) {
	docs, content := benchmarkCorpus()

	summaries := make([]DocSummary, len(docs))
	for i, meta := range docs {
		summary := ExtractAdaptiveSummary(content[meta.Filename], 450)
		summaries[i] = DocSummary{
			Filename: meta.Filename,
			Type:     meta.Type,
			Date:     meta.Date,
			Tags:     meta.Tags,
			Summary:  summary,
		}
	}

	signals := AnalyzeCorpusSignals(summaries)
	sys, usr := BuildReviewPrompt(summaries, "", signals)
	prompt := sys + usr

	// Signal detection recall
	c1 := findPair(signals.PotentialPairs, "decision-auth-jwt", "decision-auth-session")
	c2 := findPair(signals.PotentialPairs, "decision-api-rest", "decision-api-graphql")
	c3 := findPair(signals.PotentialPairs, "decision-cache-redis", "decision-cache-valkey")
	signalRecall := 0
	for _, found := range []bool{c1, c2, c3} {
		if found {
			signalRecall++
		}
	}

	t.Logf("═══════════════════════════════════════")
	t.Logf("  ANGELA REVIEW QUALITY BENCHMARK")
	t.Logf("═══════════════════════════════════════")
	t.Logf("  Corpus:    %d docs (%d planted issues)", len(docs), 5)
	t.Logf("  Signals:   %d/3 contradictions detected (recall: %d%%)", signalRecall, signalRecall*100/3)
	t.Logf("  Pairs:     %d total (%d false positives)", len(signals.PotentialPairs), len(signals.PotentialPairs)-signalRecall)
	t.Logf("  Isolated:  %d docs", len(signals.IsolatedDocs))
	t.Logf("  Prompt:    ~%d tokens (%d chars)", len(prompt)/4, len(prompt))
	t.Logf("═══════════════════════════════════════")
}
