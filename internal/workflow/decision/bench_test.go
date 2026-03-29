// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkEvaluateFull benchmarks Evaluate() with all 5 signals active.
// Target: < 10ms/op (NFR26 budget: Decision Engine portion).
func BenchmarkEvaluateFull(b *testing.B) {
	cfg := DefaultConfig()
	cfg.AlwaysAsk = nil
	cfg.AlwaysSkip = nil
	e := NewEngine(nil, cfg) // nil store — Signal 5 returns 0

	ctx := SignalContext{
		ConvType:     "feat",
		Scope:        "auth",
		Subject:      "feat(auth): add OAuth2 flow for third-party providers",
		Message:      "feat(auth): add OAuth2 flow for third-party providers\n\nImplements OAuth2 with PKCE.",
		DiffContent:  realisticDiff(150),
		FilesChanged: []string{"src/auth/oauth2.go", "src/api/routes.go", "tests/auth_test.go"},
		LinesAdded:   120,
		LinesDeleted: 30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Evaluate(ctx)
	}
}

// BenchmarkEvaluateNilStore benchmarks Evaluate() with store=nil (Signals 1-4 only).
// Target: <= full-store latency.
func BenchmarkEvaluateNilStore(b *testing.B) {
	cfg := DefaultConfig()
	cfg.AlwaysAsk = nil
	cfg.AlwaysSkip = nil
	e := NewEngine(nil, cfg)

	ctx := SignalContext{
		ConvType:     "refactor",
		Scope:        "core",
		Subject:      "refactor(core): extract helper for validation",
		DiffContent:  realisticDiff(80),
		FilesChanged: []string{"src/core/validation.go", "src/core/helpers.go"},
		LinesAdded:   60,
		LinesDeleted: 20,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := e.Evaluate(ctx)
		if result.Confidence != 0.8 {
			b.Fatalf("expected confidence 0.8 for nil store, got %f", result.Confidence)
		}
	}
}

// BenchmarkScanDiffContent benchmarks regex scanning on a realistic 500-line diff.
// Target: < 5ms/op.
func BenchmarkScanDiffContent(b *testing.B) {
	diff := realisticDiff(500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ScanDiffContent(diff)
	}
}

// BenchmarkExtractImplicitWhy benchmarks pre-fill extraction on mixed FR/EN subjects.
// Target: < 1ms/op.
func BenchmarkExtractImplicitWhy(b *testing.B) {
	subjects := []string{
		"feat(auth): add OAuth2 flow for third-party providers",
		"fix(api): resolve timeout because of connection pool exhaustion",
		"refactor(core): extract helper in order to reduce duplication",
		"feat(ui): ajouter le menu pour ameliorer la navigation",
		"fix(db): corriger la requete parce que le join etait incorrect",
		"feat(auth): add JWT validation so that tokens are verified",
		"chore: update dependencies due to security advisory",
		"feat(api): implement rate limiting",
		"refactor: simplify error handling",
		"fix(cache): invalidate stale entries",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractImplicitWhy(subjects[i%len(subjects)])
	}
}

// realisticDiff generates a synthetic diff with N lines containing patterns
// that trigger the content analysis signals.
func realisticDiff(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		switch {
		case i == 10:
			fmt.Fprintln(&b, "+\tfunc ValidateToken(token string) error {")
		case i == 20:
			fmt.Fprintln(&b, "+\t\tif apiKey == \"\" { return errNoKey }")
		case i == 50:
			fmt.Fprintln(&b, "+\t// TODO: add rate limiting")
		case i == 70:
			fmt.Fprintln(&b, "-\tfunc oldHandler(w http.ResponseWriter, r *http.Request) {")
		case i == 90:
			fmt.Fprintln(&b, "+\tdb.SetEndpoint(\"redis://localhost:6379\")")
		case i%2 == 0:
			fmt.Fprintf(&b, "+\t// line %d: added code\n", i)
		default:
			fmt.Fprintf(&b, "-\t// line %d: removed code\n", i)
		}
	}
	return b.String()
}
