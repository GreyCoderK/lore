// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"gopkg.in/yaml.v3"
)

func writePendingFile(t *testing.T, dir string, record PendingRecord) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	name := record.Commit
	if name == "" {
		name = "unknown"
	}
	data, err := yaml.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".yaml"), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestListPending_WithFiles(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	now := time.Now().UTC()
	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "abc1234",
		Date:    now.Add(-2 * 24 * time.Hour).Format(time.RFC3339),
		Message: "feat(auth): add JWT middleware",
		Answers: PendingAnswers{Type: "feature", What: "add JWT auth", Why: "security"},
		Status:  "partial",
		Reason:  "interrupted",
	})
	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "def5678",
		Date:    now.Add(-7 * 24 * time.Hour).Format(time.RFC3339),
		Message: "fix: token expiry",
		Answers: PendingAnswers{},
		Status:  "deferred",
		Reason:  "non-tty",
	})

	items, err := ListPending(context.Background(), pendingDir, nil)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	// Sorted by date descending — abc1234 (2 days ago) before def5678 (7 days ago)
	if items[0].CommitHash != "abc1234" {
		t.Errorf("first item hash = %q, want abc1234", items[0].CommitHash)
	}
	if items[1].CommitHash != "def5678" {
		t.Errorf("second item hash = %q, want def5678", items[1].CommitHash)
	}

	// Progress: abc1234 has type+what+why = 3/5
	if items[0].Progress != "3/5" {
		t.Errorf("progress = %q, want 3/5", items[0].Progress)
	}
	// Progress: def5678 has nothing = 0/5
	if items[1].Progress != "0/5" {
		t.Errorf("progress = %q, want 0/5", items[1].Progress)
	}
}

func TestListPending_Empty(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}

	items, err := ListPending(context.Background(), pendingDir, nil)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestListPending_NonExistentDir(t *testing.T) {
	items, err := ListPending(context.Background(), "/nonexistent/path", nil)
	if err != nil {
		t.Fatalf("ListPending should return nil for non-existent dir, got: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestListPending_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a corrupt YAML file
	if err := os.WriteFile(filepath.Join(pendingDir, "corrupt.yaml"), []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a valid file
	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "valid123",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "valid commit",
		Status:  "partial",
		Reason:  "interrupted",
	})

	var warnings []string
	warnWriter := func(msg string) { warnings = append(warnings, msg) }

	items, err := ListPending(context.Background(), pendingDir, warnWriter)
	if err != nil {
		t.Fatalf("ListPending: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 valid item, got %d", len(items))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(warnings))
	}
}

func TestSkipPending_Deletes(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "skip1234",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "feat: something",
		Status:  "partial",
		Reason:  "interrupted",
	})

	item, err := SkipPending(context.Background(), pendingDir, "skip1234")
	if err != nil {
		t.Fatalf("SkipPending: %v", err)
	}

	if item.CommitHash != "skip1234" {
		t.Errorf("returned item hash = %q, want skip1234", item.CommitHash)
	}

	// File should be gone
	if _, err := os.Stat(filepath.Join(pendingDir, "skip1234.yaml")); !os.IsNotExist(err) {
		t.Error("pending file should have been deleted")
	}
}

func TestSkipPending_NotFound(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := SkipPending(context.Background(), pendingDir, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent hash")
	}
}

func TestSkipPending_AmbiguousPrefix(t *testing.T) {
	dir := t.TempDir()
	pendingDir := filepath.Join(dir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit: "abc1234", Date: time.Now().UTC().Format(time.RFC3339),
		Message: "first", Status: "partial", Reason: "interrupted",
	})
	writePendingFile(t, pendingDir, PendingRecord{
		Commit: "abc5678", Date: time.Now().UTC().Format(time.RFC3339),
		Message: "second", Status: "partial", Reason: "interrupted",
	})

	_, err := SkipPending(context.Background(), pendingDir, "abc")
	if err == nil {
		t.Fatal("expected error for ambiguous prefix")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

func TestRelativeAge(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{2 * time.Minute, "just now"},
		{30 * time.Minute, "30 minutes ago"},
		{2 * time.Hour, "2 hours ago"},
		{1 * time.Hour, "1 hour ago"},
		{3 * 24 * time.Hour, "3 days ago"},
		{1 * 24 * time.Hour, "1 day ago"},
		{2 * 7 * 24 * time.Hour, "2 weeks ago"},
		{1 * 7 * 24 * time.Hour, "1 week ago"},
		{60 * 24 * time.Hour, "2 months ago"},
		{-3 * 24 * time.Hour, "3 days ago"}, // negative duration handled
	}

	for _, tt := range tests {
		got := RelativeAge(tt.duration)
		if got != tt.want {
			t.Errorf("RelativeAge(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestComputeProgress(t *testing.T) {
	tests := []struct {
		name    string
		answers PendingAnswers
		want    string
	}{
		{"all empty", PendingAnswers{}, "0/5"},
		{"all filled", PendingAnswers{Type: "feature", What: "x", Why: "y", Alternatives: "a", Impact: "i"}, "5/5"},
		{"partial", PendingAnswers{Type: "feature", What: "x"}, "2/5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeProgress(tt.answers)
			if got != tt.want {
				t.Errorf("computeProgress() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- ResolvePending tests ---

func newPendingWorkDir(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	for _, sub := range []string{".lore/docs", ".lore/templates", ".lore/pending"} {
		if err := os.MkdirAll(filepath.Join(workDir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	return workDir
}

func TestResolvePending_FullFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	// Create a pending file with partial answers (type + what filled)
	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "abc1234",
		Date:    time.Now().UTC().Add(-2 * 24 * time.Hour).Format(time.RFC3339),
		Message: "feat(auth): add JWT middleware",
		Answers: PendingAnswers{Type: "feature", What: "add JWT auth"},
		Status:  "partial",
		Reason:  "interrupted",
	})

	// Remaining questions: why, alternatives (skip), impact (skip)
	input := "because security\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	adapter := &mockGitAdapter{
		commit: &domain.CommitInfo{
			Hash:    "abc1234567890abcdef1234567890abcdef123456",
			Author:  "Test",
			Message: "feat(auth): add JWT middleware",
			Type:    "feat",
			Subject: "add JWT middleware",
		},
	}

	item := PendingItem{
		Filename:      "abc1234.yaml",
		CommitHash:    "abc1234",
		CommitMessage: "feat(auth): add JWT middleware",
		CommitDate:    time.Now().UTC().Add(-2 * 24 * time.Hour),
		Answers:       PendingAnswers{Type: "feature", What: "add JWT auth"},
		Progress:      "2/5",
	}

	err := ResolvePending(context.Background(), workDir, streams, item, adapter, ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolvePending: %v", err)
	}

	// Document created
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var docFound bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "feature-") && strings.HasSuffix(e.Name(), ".md") {
			docFound = true
			// Check generated_by: pending
			data, _ := os.ReadFile(filepath.Join(workDir, ".lore", "docs", e.Name()))
			if !strings.Contains(string(data), "generated_by: pending") {
				t.Errorf("expected generated_by: pending, got:\n%s", string(data))
			}
		}
	}
	if !docFound {
		t.Error("expected feature-*.md document to be created")
	}

	// Pending file deleted
	if _, statErr := os.Stat(filepath.Join(pendingDir, "abc1234.yaml")); !os.IsNotExist(statErr) {
		t.Error("pending file should have been deleted after resolve")
	}

	// Captured message in stderr
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' in stderr, got: %q", stderr.String())
	}
}

func TestResolvePending_PartialAnswersPreserved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	// All 3 required fields filled — only alt + impact remain
	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "full3456",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "fix: token bug",
		Answers: PendingAnswers{Type: "bugfix", What: "fix token", Why: "tokens expired"},
		Status:  "partial",
		Reason:  "interrupted",
	})

	// Only alternatives + impact (both skip with Enter)
	input := "\n\n"
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	adapter := &mockGitAdapter{
		commit: &domain.CommitInfo{
			Hash:    "ff003456abcdef1234567890abcdef1234567890",
			Author:  "Test",
			Message: "fix: token bug",
			Type:    "fix",
			Subject: "token bug",
		},
	}

	item := PendingItem{
		Filename:      "full3456.yaml",
		CommitHash:    "full3456",
		CommitMessage: "fix: token bug",
		CommitDate:    time.Now().UTC(),
		Answers:       PendingAnswers{Type: "bugfix", What: "fix token", Why: "tokens expired"},
		Progress:      "3/5",
	}

	err := ResolvePending(context.Background(), workDir, streams, item, adapter, ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolvePending: %v", err)
	}

	// Verify bugfix doc created
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bugfix-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		t.Error("expected bugfix-*.md document")
	}
}

// mockGitAdapterMissing simulates a commit that no longer exists.
type mockGitAdapterMissing struct {
	mockGitAdapter
}

func (m *mockGitAdapterMissing) CommitExists(_ string) (bool, error) { return false, nil }

func TestResolvePending_CommitGone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newPendingWorkDir(t)
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	writePendingFile(t, pendingDir, PendingRecord{
		Commit:  "gone1234",
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: "feat: vanished commit",
		Answers: PendingAnswers{Type: "feature", What: "vanished"},
		Status:  "partial",
		Reason:  "interrupted",
	})

	// Remaining: why, alt, impact
	input := "because reasons\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	adapter := &mockGitAdapterMissing{}

	item := PendingItem{
		Filename:      "gone1234.yaml",
		CommitHash:    "gone1234",
		CommitMessage: "feat: vanished commit",
		CommitDate:    time.Now().UTC(),
		Answers:       PendingAnswers{Type: "feature", What: "vanished"},
		Progress:      "2/5",
	}

	err := ResolvePending(context.Background(), workDir, streams, item, adapter, ResolveOpts{})
	if err != nil {
		t.Fatalf("ResolvePending with missing commit: %v", err)
	}

	// Warning about missing commit
	if !strings.Contains(stderr.String(), "no longer exists") {
		t.Errorf("expected 'no longer exists' warning, got: %q", stderr.String())
	}

	// Document still created
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md" {
			found = true
		}
	}
	if !found {
		t.Error("expected document to be created even with missing commit")
	}
}

// --- isValidDocType unit tests ---

func TestIsValidDocType(t *testing.T) {
	valid := []string{"feature", "bugfix", "decision", "refactor", "release", "note"}
	for _, dt := range valid {
		if !isValidDocType(dt) {
			t.Errorf("isValidDocType(%q) = false, want true", dt)
		}
	}

	invalid := []string{"", "feat", "bug", "unknown", "Feature"}
	for _, dt := range invalid {
		if isValidDocType(dt) {
			t.Errorf("isValidDocType(%q) = true, want false", dt)
		}
	}
}

// --- deletePendingFile unit tests ---

func TestDeletePendingFile_PathTraversal(t *testing.T) {
	pendingDir := filepath.Join(t.TempDir(), "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := deletePendingFile(pendingDir, "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestDeletePendingFile_NonExistent(t *testing.T) {
	pendingDir := filepath.Join(t.TempDir(), "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := deletePendingFile(pendingDir, "nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestDeletePendingFile_Success(t *testing.T) {
	pendingDir := filepath.Join(t.TempDir(), "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a file to delete
	path := filepath.Join(pendingDir, "test.yaml")
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := deletePendingFile(pendingDir, "test.yaml")
	if err != nil {
		t.Fatalf("deletePendingFile: %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Error("file should have been deleted")
	}
}
