package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestWriteDoc_CreatesFileWithFrontMatter(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{
		Type:        "decision",
		Date:        "2026-03-07",
		Commit:      "abc1234",
		Status:      "demo",
		Tags:        []string{"auth", "jwt"},
		GeneratedBy: "lore-demo",
	}
	body := "# Test\n\nContent.\n"

	result, err := WriteDoc(dir, meta, "add JWT middleware", body)
	if err != nil {
		t.Fatalf("storage: write doc: %v", err)
	}

	if result.Filename != "decision-add-jwt-middleware-2026-03-07.md" {
		t.Errorf("filename: got %q", result.Filename)
	}
	expectedPath := filepath.Join(dir, result.Filename)
	if result.Path != expectedPath {
		t.Errorf("path: got %q, want %q", result.Path, expectedPath)
	}

	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("storage: read written file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type: decision") {
		t.Error("expected front matter to contain 'type: decision'")
	}
	if !strings.Contains(content, "status: demo") {
		t.Error("expected front matter to contain 'status: demo'")
	}
	if !strings.Contains(content, "date: \"2026-03-07\"") && !strings.Contains(content, "date: 2026-03-07") {
		t.Error("expected date in YYYY-MM-DD format")
	}
	if !strings.Contains(content, "# Test") {
		t.Error("expected body content")
	}
}

func TestWriteDoc_EmptySubject(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{
		Type:   "note",
		Date:   "2026-03-07",
		Status: "published",
	}

	result, err := WriteDoc(dir, meta, "", "# Note\n")
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(result.Filename, "untitled") {
		t.Errorf("empty subject should produce 'untitled', got %q", result.Filename)
	}
}

func TestSlugify_Basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Add JWT Middleware", "add-jwt-middleware"},
		{"Hello, World!", "hello-world"},
		{"  spaces  ", "spaces"},
		{"already-slugged", "already-slugged"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSlugify_Accents(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"décision stratégique", "decision-strategique"},
		{"café résumé", "cafe-resume"},
		{"über cool", "uber-cool"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSlugify_MaxLength(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := slugify(long)
	if len(got) > 50 {
		t.Errorf("slugify should truncate to 50 chars, got %d", len(got))
	}
}

func TestSlugify_TrailingDashAfterTruncation(t *testing.T) {
	// Create input that will have a dash at position 50 after slugify
	input := strings.Repeat("abcde ", 20) // "abcde-abcde-..." after slugify
	got := slugify(input)
	if len(got) > 50 {
		t.Errorf("len should be <= 50, got %d", len(got))
	}
	if strings.HasSuffix(got, "-") {
		t.Errorf("should not end with dash, got %q", got)
	}
}

func TestSlugify_DedupDashes(t *testing.T) {
	got := slugify("foo---bar")
	if got != "foo-bar" {
		t.Errorf("slugify should dedup dashes, got %q", got)
	}
}

func TestSlugify_SpecialChars(t *testing.T) {
	got := slugify("hello@world#2026!")
	if got != "hello-world-2026" {
		t.Errorf("slugify(%q) = %q", "hello@world#2026!", got)
	}
}

func TestAtomicWrite_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	data := []byte("hello world")

	if err := AtomicWrite(path, data); err != nil {
		t.Fatalf("atomic write: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != "hello world" {
		t.Errorf("content: got %q", string(got))
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0644 {
		t.Errorf("permissions: got %o, want 0644", info.Mode().Perm())
	}
}

func TestAtomicWrite_NoTmpLeftover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := AtomicWrite(path, []byte("data")); err != nil {
		t.Fatalf("write: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".lore-") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left over: %s", e.Name())
		}
	}
}

func TestWriteDoc_ValidatesMeta(t *testing.T) {
	dir := t.TempDir()

	// Missing type
	_, err := WriteDoc(dir, domain.DocMeta{Date: "2026-03-07", Status: "published"}, "test", "body\n")
	if err == nil {
		t.Error("expected error for missing type")
	}

	// Bad date format
	_, err = WriteDoc(dir, domain.DocMeta{Type: "note", Date: "not-a-date", Status: "published"}, "test", "body\n")
	if err == nil {
		t.Error("expected error for bad date format")
	}
}

func TestWriteDoc_CollisionReturnsError(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{Type: "decision", Date: "2026-03-07", Status: "published"}

	_, err := WriteDoc(dir, meta, "same subject", "body\n")
	if err != nil {
		t.Fatalf("first write: %v", err)
	}

	_, err = WriteDoc(dir, meta, "same subject", "different body\n")
	if err == nil {
		t.Error("expected error for filename collision")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error should mention collision, got: %v", err)
	}
}

func TestWriteDoc_IndexErrSurfaced(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{Type: "note", Date: "2026-03-07", Status: "published"}

	result, err := WriteDoc(dir, meta, "test", "body\n")
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	// Index should succeed in normal case
	if result.IndexErr != nil {
		t.Errorf("expected nil IndexErr, got: %v", result.IndexErr)
	}
}
