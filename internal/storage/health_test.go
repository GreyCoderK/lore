package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQuickHealthCheck_NoIssues(t *testing.T) {
	dir := t.TempDir()
	// Create a non-empty README.md
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := QuickHealthCheck(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issues != 0 {
		t.Errorf("expected 0 issues, got %d", issues)
	}
}

func TestQuickHealthCheck_OrphanTmpFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create orphan .tmp files
	for _, name := range []string{"write1.tmp", "write2.tmp"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	issues, err := QuickHealthCheck(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issues != 2 {
		t.Errorf("expected 2 issues (tmp files), got %d", issues)
	}
}

func TestQuickHealthCheck_MissingReadme(t *testing.T) {
	dir := t.TempDir()
	// No README.md

	issues, err := QuickHealthCheck(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issues != 1 {
		t.Errorf("expected 1 issue (missing README), got %d", issues)
	}
}

func TestQuickHealthCheck_EmptyReadme(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	issues, err := QuickHealthCheck(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issues != 1 {
		t.Errorf("expected 1 issue (empty README), got %d", issues)
	}
}

func TestQuickHealthCheck_MissingDir(t *testing.T) {
	issues, err := QuickHealthCheck("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issues != 1 {
		t.Errorf("expected 1 issue (missing dir), got %d", issues)
	}
}
