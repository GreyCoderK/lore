package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSavePending_CreatesFile(t *testing.T) {
	workDir := t.TempDir()

	record := PendingRecord{
		Commit:  "abc1234",
		Date:    "2026-03-07T14:30:00Z",
		Message: "feat(auth): add JWT middleware",
		Answers: PendingAnswers{
			Type: "feature",
			What: "add JWT auth middleware",
			Why:  "",
		},
		Status: "partial",
		Reason: "interrupted",
	}

	if err := SavePending(workDir, record); err != nil {
		t.Fatalf("SavePending: %v", err)
	}

	path := filepath.Join(workDir, ".lore", "pending", "abc1234.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("file not created: %v", err)
	}

	var got PendingRecord
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Commit != "abc1234" {
		t.Errorf("Commit = %q, want %q", got.Commit, "abc1234")
	}
	if got.Status != "partial" {
		t.Errorf("Status = %q, want %q", got.Status, "partial")
	}
	if got.Reason != "interrupted" {
		t.Errorf("Reason = %q, want %q", got.Reason, "interrupted")
	}
	if got.Answers.Type != "feature" {
		t.Errorf("Answers.Type = %q, want %q", got.Answers.Type, "feature")
	}
}

func TestSavePending_CreatesDirIfAbsent(t *testing.T) {
	workDir := t.TempDir()
	// No .lore/pending directory exists yet — SavePending must create it.
	record := PendingRecord{Commit: "deadbeef", Status: "partial", Reason: "interrupted"}
	if err := SavePending(workDir, record); err != nil {
		t.Fatalf("SavePending: %v", err)
	}
	if _, err := os.Stat(filepath.Join(workDir, ".lore", "pending")); err != nil {
		t.Errorf("pending dir not created: %v", err)
	}
}

func TestSavePending_EmptyCommitUsesTimestamp(t *testing.T) {
	workDir := t.TempDir()
	record := PendingRecord{Commit: "", Status: "partial", Reason: "interrupted"}
	if err := SavePending(workDir, record); err != nil {
		t.Fatalf("SavePending: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	name := entries[0].Name()
	if len(name) < 8 {
		t.Errorf("filename too short for timestamp-based name: %q", name)
	}
}

func TestBuildPendingRecord(t *testing.T) {
	answers := Answers{
		Type:         "feature",
		What:         "add auth",
		Why:          "security requirement",
		Alternatives: "none",
		Impact:       "users can log in",
	}
	rec := BuildPendingRecord(answers, "abc1234", "feat: add auth", "interrupted", "partial")
	if rec.Commit != "abc1234" {
		t.Errorf("Commit = %q", rec.Commit)
	}
	if rec.Answers.Type != "feature" {
		t.Errorf("Answers.Type = %q", rec.Answers.Type)
	}
	if rec.Answers.Impact != "users can log in" {
		t.Errorf("Answers.Impact = %q", rec.Answers.Impact)
	}
	if rec.Status != "partial" {
		t.Errorf("Status = %q", rec.Status)
	}
	if rec.Reason != "interrupted" {
		t.Errorf("Reason = %q", rec.Reason)
	}
	if rec.Date == "" {
		t.Error("Date should not be empty")
	}
}

func TestSavePending_NoOverwrite(t *testing.T) {
	workDir := t.TempDir()
	record1 := PendingRecord{Commit: "same1234", Status: "partial", Reason: "interrupted", Message: "first"}
	record2 := PendingRecord{Commit: "same1234", Status: "partial", Reason: "interrupted", Message: "second"}

	if err := SavePending(workDir, record1); err != nil {
		t.Fatalf("SavePending first: %v", err)
	}
	if err := SavePending(workDir, record2); err != nil {
		t.Fatalf("SavePending second: %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(workDir, ".lore", "pending"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) < 2 {
		t.Errorf("expected 2 files (no overwrite), got %d", len(entries))
	}
}

func TestSavePending_RelativePath(t *testing.T) {
	// L19 validation: SavePending works with relative workDir "."
	// We change CWD to a temp dir to simulate the git work tree scenario.
	workDir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(workDir); err != nil {
		t.Fatal(err)
	}

	record := PendingRecord{Commit: "reltest", Status: "partial", Reason: "interrupted"}
	if err := SavePending(".", record); err != nil {
		t.Fatalf("SavePending with relative path: %v", err)
	}

	path := filepath.Join(".", ".lore", "pending", "reltest.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not found at relative path: %v", err)
	}
}
