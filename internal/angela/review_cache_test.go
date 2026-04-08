// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveAndLoadReviewCache_Roundtrip(t *testing.T) {
	loreDir := t.TempDir()

	report := &ReviewReport{
		DocCount: 5,
		Findings: []ReviewFinding{
			{Severity: "warning", Title: "test finding", Description: "test desc", Documents: []string{"test.md"}},
		},
	}

	if err := SaveReviewCache(loreDir, report, 10); err != nil {
		t.Fatalf("SaveReviewCache: %v", err)
	}

	cache, err := LoadReviewCache(loreDir)
	if err != nil {
		t.Fatalf("LoadReviewCache: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if cache.DocCount != 5 {
		t.Errorf("DocCount = %d, want 5", cache.DocCount)
	}
	if cache.TotalDocs != 10 {
		t.Errorf("TotalDocs = %d, want 10", cache.TotalDocs)
	}
	if cache.Version != reviewCacheVersion {
		t.Errorf("Version = %d, want %d", cache.Version, reviewCacheVersion)
	}
	if len(cache.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(cache.Findings))
	}
	if cache.Findings[0].Title != "test finding" {
		t.Errorf("finding title = %q", cache.Findings[0].Title)
	}
	if cache.LastReview.IsZero() {
		t.Error("expected non-zero LastReview")
	}
}

func TestLoadReviewCache_NotExists(t *testing.T) {
	loreDir := t.TempDir()

	cache, err := LoadReviewCache(loreDir)
	if err != nil {
		t.Fatalf("LoadReviewCache: %v", err)
	}
	if cache != nil {
		t.Error("expected nil cache when no file exists")
	}
}

func TestLoadReviewCache_InvalidJSON(t *testing.T) {
	loreDir := t.TempDir()
	cacheDir := filepath.Join(loreDir, "cache")
	os.MkdirAll(cacheDir, 0o755)
	os.WriteFile(filepath.Join(cacheDir, "review.json"), []byte("not json"), 0o644)

	_, err := LoadReviewCache(loreDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadReviewCache_WrongVersion(t *testing.T) {
	loreDir := t.TempDir()
	cacheDir := filepath.Join(loreDir, "cache")
	os.MkdirAll(cacheDir, 0o755)

	cache := ReviewCache{Version: 999, DocCount: 1}
	data, _ := json.Marshal(cache)
	os.WriteFile(filepath.Join(cacheDir, "review.json"), data, 0o644)

	result, err := LoadReviewCache(loreDir)
	if err != nil {
		t.Fatalf("LoadReviewCache: %v", err)
	}
	if result != nil {
		t.Error("expected nil for incompatible version")
	}
}

func TestSaveReviewCache_NonExistentDir_Error(t *testing.T) {
	// Use a path that cannot be created (file as parent dir).
	tmpDir := t.TempDir()
	// Create a regular file where the "lore dir" should be a directory.
	fakePath := filepath.Join(tmpDir, "blocked")
	os.WriteFile(fakePath, []byte("not a dir"), 0o644)

	// Now try to save cache inside blocked/cache/ — MkdirAll should fail
	// because "blocked" is a file, not a directory.
	report := &ReviewReport{DocCount: 1, Findings: nil}
	err := SaveReviewCache(fakePath, report, 1)
	if err == nil {
		t.Fatal("expected error when writing to non-existent/blocked dir")
	}
	if !strings.Contains(err.Error(), "mkdir") {
		t.Errorf("error = %q, want mkdir error", err.Error())
	}
}

func TestSaveReviewCache_EmptyFindings(t *testing.T) {
	loreDir := t.TempDir()

	report := &ReviewReport{DocCount: 3, Findings: nil}
	if err := SaveReviewCache(loreDir, report, 3); err != nil {
		t.Fatalf("SaveReviewCache: %v", err)
	}

	cache, err := LoadReviewCache(loreDir)
	if err != nil {
		t.Fatalf("LoadReviewCache: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
	if len(cache.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(cache.Findings))
	}
}
