package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/museigen/lore/internal/domain"
)

func TestWriteDoc_CreatesFileWithFrontMatter(t *testing.T) {
	dir := t.TempDir()
	meta := domain.DocMeta{
		Type:        "decision",
		Date:        domain.NewDateString(time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)),
		Commit:      "abc1234",
		Status:      "demo",
		Tags:        []string{"auth", "jwt"},
		GeneratedBy: "lore-demo",
	}
	body := "# Test\n\nContent.\n"

	filename, err := WriteDoc(dir, meta, "add JWT middleware", body)
	if err != nil {
		t.Fatalf("storage: write doc: %v", err)
	}

	if filename != "decision-add-jwt-middleware-2026-03-07.md" {
		t.Errorf("filename: got %q", filename)
	}

	data, err := os.ReadFile(filepath.Join(dir, filename))
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

func TestSlugify(t *testing.T) {
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
		got := domain.Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
