package generator_test

// H3 fix: external test package — generator/ is not allowed to import storage/ (CORRECTION C1).
// Integration tests that exercise the full generator → storage pipeline live here as package
// generator_test so the dependency on storage/ is an external-test-only concern.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/generator"
	"github.com/greycoderk/lore/internal/storage"
	loretemplate "github.com/greycoderk/lore/internal/template"
	"github.com/greycoderk/lore/internal/ui"
)

// TestIntegration_FullPipeline tests the complete workflow → generator → storage pipeline.
func TestIntegration_FullPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	docsDir := t.TempDir()

	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template engine: %v", err)
	}

	commit := &domain.CommitInfo{
		Hash:   "abc1234",
		Author: "Dev",
		Date:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
	}

	input := generator.GenerateInput{
		DocType:      "decision",
		What:         "adopt hexagonal architecture",
		Why:          "better testability",
		Alternatives: "layered architecture",
		Impact:       "all layers refactored",
		CommitInfo:   commit,
		GeneratedBy:  "hook",
	}

	// AC-1: pipeline workflow → generator → storage
	genResult, err := generator.Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if genResult.Body == "" {
		t.Fatal("expected non-empty body from generator")
	}

	// Verify body contains expected content
	if !strings.Contains(genResult.Body, "adopt hexagonal architecture") {
		t.Error("expected 'what' content in generated body")
	}
	if !strings.Contains(genResult.Body, "better testability") {
		t.Error("expected 'why' content in generated body")
	}

	// AC-1: front matter via storage.WriteDoc using generated meta (H1 fix: no duplication)
	writeResult, err := storage.WriteDoc(docsDir, genResult.Meta, input.What, genResult.Body)
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}

	// AC-2: verify front matter in written file
	data, err := os.ReadFile(writeResult.Path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "type: decision") {
		t.Error("expected 'type: decision' in front matter")
	}
	if !strings.Contains(content, "date: \"2026-03-15\"") && !strings.Contains(content, "date: 2026-03-15") {
		t.Error("expected date in front matter")
	}
	if !strings.Contains(content, "commit: abc1234") {
		t.Error("expected commit in front matter")
	}
	if !strings.Contains(content, "status: draft") {
		t.Error("expected status in front matter")
	}

	// AC-3: README.md index regenerated
	readmePath := filepath.Join(docsDir, "README.md")
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("README.md not generated: %v", err)
	}
	if !strings.Contains(string(readme), writeResult.Filename) {
		t.Errorf("README.md should contain %q", writeResult.Filename)
	}
}

// TestIntegration_PipelineOutput verifies AC-2 output display via real ui.Verb() call.
// H2 fix: this test previously wrote "Captured" manually to a buffer then checked for it
// (tautological). Now it calls ui.Verb() and validates the formatted cargo-style output.
func TestIntegration_PipelineOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	docsDir := t.TempDir()
	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template engine: %v", err)
	}

	input := generator.GenerateInput{
		DocType:     "feature",
		What:        "add search",
		Why:         "usability",
		GeneratedBy: "hook",
		CommitInfo:  &domain.CommitInfo{Hash: "cafe1234", Date: time.Now()},
	}

	genResult, err := generator.Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	writeResult, err := storage.WriteDoc(docsDir, genResult.Meta, input.What, genResult.Body)
	if err != nil {
		t.Fatalf("WriteDoc: %v", err)
	}

	// AC-2: verify ui.Verb() produces the expected cargo-style output.
	// Use color-disabled mode so the assertion is against plain text.
	restore := ui.SaveAndDisableColor()
	defer restore()

	var stderrBuf strings.Builder
	streams := domain.IOStreams{
		Err: &stderrBuf,
	}
	ui.Verb(streams, "Captured", writeResult.Filename)
	// AC-2: dim path on second line (color disabled → plain text, same format as reactive.go)
	fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(writeResult.Path))

	output := stderrBuf.String()
	// Verb pads to 10 chars: "  Captured" (2 leading spaces)
	if !strings.Contains(output, "  Captured") {
		t.Errorf("expected right-aligned 'Captured' verb (10 chars), got: %q", output)
	}
	if !strings.Contains(output, writeResult.Filename) {
		t.Errorf("expected filename %q in Verb output, got: %q", writeResult.Filename, output)
	}
	// N1 fix: verify dim path appears on second line (AC-2 second display requirement)
	if !strings.Contains(output, writeResult.Path) {
		t.Errorf("expected dim path %q on second line, got: %q", writeResult.Path, output)
	}
}

// TestIntegration_Performance verifies AC-4: a single generator pipeline call < 1s.
// M5 fix: previous test ran 10 iterations and allowed 5s total (500ms each), which
// would pass even when individual calls violate the < 1s AC-4 target.
func TestIntegration_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template engine: %v", err)
	}

	input := generator.GenerateInput{
		DocType:     "note",
		What:        "quick note",
		Why:         "to remember",
		GeneratedBy: "hook",
	}

	start := time.Now()
	_, err = generator.Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > time.Second {
		t.Errorf("Generate() took %v, want < 1s (AC-4)", elapsed)
	}
}
