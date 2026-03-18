package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestDeleteDoc_Success(t *testing.T) {
	dir := t.TempDir()

	result, err := WriteDoc(dir, domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "published",
	}, "auth strategy", "# Auth\n")
	if err != nil {
		t.Fatalf("setup write: %v", err)
	}

	// Verify file exists before delete
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("file should exist before delete: %v", err)
	}

	if err := DeleteDoc(dir, result.Filename); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// File should be gone
	if _, err := os.Stat(result.Path); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestDeleteDoc_NotFound(t *testing.T) {
	dir := t.TempDir()

	err := DeleteDoc(dir, "nonexistent.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "storage: delete") {
		t.Errorf("error should follow storage pattern, got: %v", err)
	}
}

func TestDeleteDoc_PathTraversal(t *testing.T) {
	dir := t.TempDir()

	tests := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"subdir/file.md",
		"..",
	}
	for _, filename := range tests {
		err := DeleteDoc(dir, filename)
		if err == nil {
			t.Errorf("expected error for path traversal %q", filename)
		}
	}
}

func TestDeleteDoc_ProtectedREADME(t *testing.T) {
	dir := t.TempDir()
	err := DeleteDoc(dir, "README.md")
	if err == nil {
		t.Error("expected error when deleting README.md")
	}
	if !strings.Contains(err.Error(), "protected") {
		t.Errorf("expected 'protected' in error, got: %v", err)
	}
}

func TestFindReferencingDocs_WithRefs(t *testing.T) {
	dir := t.TempDir()

	// Create target doc
	if _, err := WriteDoc(dir, domain.DocMeta{
		Type: "decision", Date: "2026-03-07", Status: "published",
	}, "auth strategy", "# Auth Strategy\n"); err != nil {
		t.Fatalf("WriteDoc target: %v", err)
	}

	// Create doc that references the target
	if _, err := WriteDoc(dir, domain.DocMeta{
		Type: "feature", Date: "2026-03-08", Status: "published",
		Related: []string{"decision-auth-strategy-2026-03-07"},
	}, "login flow", "# Login Flow\n"); err != nil {
		t.Fatalf("WriteDoc referencing: %v", err)
	}

	// Create doc that doesn't reference the target
	if _, err := WriteDoc(dir, domain.DocMeta{
		Type: "note", Date: "2026-03-09", Status: "published",
	}, "unrelated", "# Unrelated\n"); err != nil {
		t.Fatalf("WriteDoc unrelated: %v", err)
	}

	refs, err := FindReferencingDocs(dir, "decision-auth-strategy-2026-03-07.md")
	if err != nil {
		t.Fatalf("FindReferencingDocs: %v", err)
	}
	if len(refs) != 1 {
		t.Fatalf("expected 1 referencing doc, got %d", len(refs))
	}
	if !strings.HasPrefix(refs[0], "feature-") {
		t.Errorf("expected feature doc in refs, got %q", refs[0])
	}
}

func TestFindReferencingDocs_NoRefs(t *testing.T) {
	dir := t.TempDir()

	_, _ = WriteDoc(dir, domain.DocMeta{
		Type: "decision", Date: "2026-03-07", Status: "published",
	}, "auth", "# Auth\n")
	_, _ = WriteDoc(dir, domain.DocMeta{
		Type: "note", Date: "2026-03-08", Status: "published",
	}, "other", "# Other\n")

	refs, err := FindReferencingDocs(dir, "decision-auth-2026-03-07.md")
	if err != nil {
		t.Fatalf("FindReferencingDocs: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestFindReferencingDocs_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	refs, err := FindReferencingDocs(dir, "anything.md")
	if err != nil {
		t.Fatalf("FindReferencingDocs: %v", err)
	}
	if len(refs) != 0 {
		t.Errorf("expected 0 refs, got %d", len(refs))
	}
}

func TestDeleteDoc_RegeneratesIndex(t *testing.T) {
	dir := t.TempDir()

	// Write two docs
	r1, _ := WriteDoc(dir, domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}, "one", "body\n")
	_, _ = WriteDoc(dir, domain.DocMeta{Type: "feature", Date: "2026-03-08", Status: "published"}, "two", "body\n")

	// README should show 2 docs
	data, _ := os.ReadFile(filepath.Join(dir, "README.md"))
	if !strings.Contains(string(data), "2 documents total") {
		t.Fatalf("expected 2 documents before delete, got:\n%s", string(data))
	}

	// Delete first doc
	if err := DeleteDoc(dir, r1.Filename); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// README should now show 1 doc
	data, _ = os.ReadFile(filepath.Join(dir, "README.md"))
	content := string(data)
	if !strings.Contains(content, "1 document total") {
		t.Errorf("expected 1 document after delete, got:\n%s", content)
	}
	if strings.Contains(content, r1.Filename) {
		t.Error("deleted doc should not appear in index")
	}
}
