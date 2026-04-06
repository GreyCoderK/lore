// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// TestStubs verifies that all Phase 1/2 stubs return ErrNotImplemented
// and do not panic.
func TestStubs_ReturnErrNotImplemented(t *testing.T) {
	s, _ := tempDB(t)

	tests := []struct {
		name string
		fn   func() error
	}{
		{"StoreSignatures", func() error { return s.StoreSignatures("hash", nil) }},
		{"FindBySignatureHash", func() error { _, err := s.FindBySignatureHash("hash"); return err }},
		{"SignaturesForCommit", func() error { _, err := s.SignaturesForCommit("hash"); return err }},
		{"EntityHistory", func() error { _, err := s.EntityHistory("name", "go"); return err }},
		{"RecordAIUsage", func() error { return s.RecordAIUsage(domain.AIUsageRecord{}) }},
		{"AIStatsSince", func() error { _, err := s.AIStatsSince(time.Now()); return err }},
		{"AIStatsByDay", func() error { _, err := s.AIStatsByDay(7); return err }},
		{"CacheReview", func() error { return s.CacheReview(domain.ReviewCacheEntry{}) }},
		{"GetCachedReview", func() error { _, err := s.GetCachedReview("hash"); return err }},
		{"ReviewHistory", func() error { _, err := s.ReviewHistory(10); return err }},
		{"UpdatePattern", func() error { return s.UpdatePattern("feat", "auth", "documented", 50, 75) }},
		{"GetPattern", func() error { _, err := s.GetPattern("feat", "auth"); return err }},
		{"AllPatterns", func() error { _, err := s.AllPatterns(); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if !errors.Is(err, domain.ErrNotImplemented) {
				t.Errorf("%s: got %v, want ErrNotImplemented", tt.name, err)
			}
		})
	}
}

func TestStubs_Rebuild(t *testing.T) {
	s, dbPath := tempDB(t)

	// Rebuild needs a valid docsDir (not the DB file).
	docsDir := filepath.Join(filepath.Dir(dbPath), "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	err := s.Rebuild(context.Background(), docsDir, nil)
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}
}

func TestStubs_Vacuum(t *testing.T) {
	s, _ := tempDB(t)
	if err := s.Vacuum(); err != nil {
		t.Fatalf("Vacuum: %v", err)
	}
}
