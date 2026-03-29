// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/store"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// BenchmarkFullHookOverhead simulates the complete hook sequence:
// RecordCommit + IndexDoc + Decision Engine Evaluate.
// Target: < 20ms/op combined (NFR26 validation).
func BenchmarkFullHookOverhead(b *testing.B) {
	// Setup store
	dir := b.TempDir()
	s, err := store.Open(filepath.Join(dir, "bench.db"))
	if err != nil {
		b.Fatalf("Open: %v", err)
	}
	b.Cleanup(func() { _ = s.Close() })

	// Setup engine (nil store for benchmark — avoids cross-dependency on store queries)
	cfg := decision.DefaultConfig()
	cfg.AlwaysAsk = nil
	cfg.AlwaysSkip = nil
	engine := decision.NewEngine(nil, cfg)

	// Prepare data
	rec := domain.CommitRecord{
		Date: time.Now(), Branch: "main", Scope: "auth", ConvType: "feat",
		Subject: "add OAuth2 flow for third-party providers",
		Message: "feat(auth): add OAuth2 flow for third-party providers",
		FilesChanged: 3, LinesAdded: 120, LinesDeleted: 30,
		Decision: "documented", QuestionMode: "full",
	}
	docEntry := domain.DocIndexEntry{
		Type: "feature", Date: "2026-03-28", Status: "draft",
		Tags: []string{"auth", "security"}, ContentHash: "sha256-bench",
		WordCount: 200, UpdatedAt: time.Now(),
	}
	sigCtx := decision.SignalContext{
		ConvType:     "feat",
		Scope:        "auth",
		Subject:      "feat(auth): add OAuth2 flow for third-party providers",
		Message:      "feat(auth): add OAuth2 flow for third-party providers",
		DiffContent:  realisticDiffForHook(150),
		FilesChanged: []string{"src/auth/oauth2.go", "src/api/routes.go", "tests/auth_test.go"},
		LinesAdded:   120,
		LinesDeleted: 30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Step 1: Decision Engine scoring
		engine.Evaluate(sigCtx)

		// Step 2: Record commit in store (fixed hash avoids changing data per iteration)
		rec.Hash = "hook-bench-fixed"
		if err := s.RecordCommit(rec); err != nil {
			b.Fatalf("RecordCommit: %v", err)
		}

		// Step 3: Index generated document (fixed filename avoids changing data per iteration)
		docEntry.Filename = "feature-bench-fixed.md"
		if err := s.IndexDoc(docEntry); err != nil {
			b.Fatalf("IndexDoc: %v", err)
		}
	}
}

func realisticDiffForHook(lines int) string {
	var b strings.Builder
	for i := 0; i < lines; i++ {
		switch {
		case i == 10:
			fmt.Fprintln(&b, "+\tfunc ValidateToken(token string) error {")
		case i == 50:
			fmt.Fprintln(&b, "+\t// TODO: add rate limiting")
		case i == 90:
			fmt.Fprintln(&b, "+\tdb.SetEndpoint(\"redis://localhost:6379\")")
		case i%2 == 0:
			fmt.Fprintf(&b, "+\t// line %d\n", i)
		default:
			fmt.Fprintf(&b, "-\t// line %d\n", i)
		}
	}
	return b.String()
}
