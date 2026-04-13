// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/fileutil"
)

// ReviewCache holds persisted review results for incremental tracking.
// Version field enables forward-compatible schema evolution (ADR-013).
type ReviewCache struct {
	Version    int              `json:"version"`
	LastReview time.Time        `json:"last_review"`
	DocCount   int              `json:"doc_count"`
	TotalDocs  int              `json:"total_docs"`
	Findings   []ReviewFinding  `json:"findings"`
}

const reviewCacheVersion = 1

// SaveReviewCache writes the review results to .lore/cache/review.json.
func SaveReviewCache(loreDir string, report *ReviewReport, totalDocs int) error {
	cacheDir := filepath.Join(loreDir, "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("angela: review cache: mkdir: %w", err)
	}

	cache := ReviewCache{
		Version:    reviewCacheVersion,
		LastReview: time.Now().UTC(),
		DocCount:   report.DocCount,
		TotalDocs:  totalDocs,
		Findings:   report.Findings,
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("angela: review cache: marshal: %w", err)
	}

	path := filepath.Join(cacheDir, "review.json")
	if err := fileutil.AtomicWrite(path, data, 0644); err != nil {
		return fmt.Errorf("angela: review cache: write: %w", err)
	}
	return nil
}

// LoadReviewCache reads the cached review results. Returns nil if no cache exists.
func LoadReviewCache(loreDir string) (*ReviewCache, error) {
	path := filepath.Join(loreDir, "cache", "review.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("angela: review cache: read: %w", err)
	}

	var cache ReviewCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("angela: review cache: parse: %w", err)
	}
	if cache.Version != reviewCacheVersion {
		// Incompatible version — treat as no cache (will be regenerated on next review)
		return nil, nil
	}
	return &cache, nil
}
