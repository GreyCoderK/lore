package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// DocFixture describes a pre-existing document for test setup.
type DocFixture struct {
	Type string   // "decision", "feature", etc.
	Slug string   // "auth-strategy"
	Date string   // "2026-03-07"
	Tags []string // optional
	Body string   // optional — default body generated if empty
}

// Chdir changes the working directory to dir and registers a t.Cleanup to restore
// the original CWD. Use this when commands under test rely on os.Getwd() to find .lore/.
//
// WARNING: not goroutine-safe — tests using Chdir must NOT call t.Parallel().
// Process-wide CWD is shared state; parallel tests would race on it.
func Chdir(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("testutil: getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("testutil: chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

// SetupLoreDir creates a minimal .lore/ structure (docs/, templates/, pending/)
// in a t.TempDir(). Does NOT chdir. Returns the temp dir path.
func SetupLoreDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, sub := range []string{
		filepath.Join(".lore", "docs"),
		filepath.Join(".lore", "templates"),
		filepath.Join(".lore", "pending"),
	} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("testutil: create %s: %v", sub, err)
		}
	}

	return dir
}

// SetupLoreDirWithDocs creates .lore/ with pre-existing document files.
// Each DocFixture generates a properly formatted .md file in .lore/docs/.
// Returns the temp dir path.
func SetupLoreDirWithDocs(t *testing.T, docs []DocFixture) string {
	t.Helper()
	dir := SetupLoreDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	for _, d := range docs {
		meta := domain.DocMeta{
			Type:   d.Type,
			Date:   d.Date,
			Status: "draft",
			Tags:   d.Tags,
		}

		body := d.Body
		if body == "" {
			body = fmt.Sprintf("# %s\n\nTest document.\n", strings.ReplaceAll(d.Slug, "-", " "))
		}

		data, err := storage.Marshal(meta, body)
		if err != nil {
			t.Fatalf("testutil: marshal doc %s: %v", d.Slug, err)
		}

		filename := fmt.Sprintf("%s-%s-%s.md", d.Type, d.Slug, d.Date)
		path := filepath.Join(docsDir, filename)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("testutil: write doc %s: %v", filename, err)
		}
	}

	return dir
}
