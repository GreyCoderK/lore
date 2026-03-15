package workflow

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

// newProactiveWorkDir creates a minimal .lore directory structure under a temp dir.
func newProactiveWorkDir(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	for _, sub := range []string{".lore/docs", ".lore/templates"} {
		if err := os.MkdirAll(filepath.Join(workDir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	return workDir
}

func TestHandleProactive_FullFlowNoArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// Simulate interactive answers: type=decision, what=add auth, why=because, alt=(skip), impact=(skip)
	input := "decision\nadd auth\nbecause JWT\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{})
	if err != nil {
		t.Fatalf("HandleProactive: %v", err)
	}

	// Verify a document was written under .lore/docs/
	entries, err := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	if err != nil {
		t.Fatalf("ReadDir docs: %v", err)
	}
	var docFound bool
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && strings.HasPrefix(e.Name(), "decision-") {
			docFound = true
		}
	}
	if !docFound {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected decision-*.md in docs, got: %v", names)
	}

	// AC-2: "Captured" verb appears in stderr
	if !strings.Contains(stderr.String(), "Captured") {
		t.Errorf("expected 'Captured' in stderr, got: %q", stderr.String())
	}

	// M6: dim path line must appear below "Captured" (AC-2 second display)
	if !strings.Contains(stderr.String(), ".lore/docs/") {
		t.Errorf("expected dim path '.lore/docs/' in stderr, got: %q", stderr.String())
	}
}

func TestHandleProactive_GeneratedByManual(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	input := "note\ntest doc\nfor testing\n\n\n"
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{})
	if err != nil {
		t.Fatalf("HandleProactive: %v", err)
	}

	// Read the generated document and verify front matter
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var docPath string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md" {
			docPath = filepath.Join(workDir, ".lore", "docs", e.Name())
			break
		}
	}
	if docPath == "" {
		t.Fatal("no .md file found in docs")
	}

	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	// AC-3: generated_by must be "manual"
	if !strings.Contains(content, "generated_by: manual") {
		t.Errorf("expected generated_by: manual in front matter, got:\n%s", content)
	}

	// AC-3: commit field must be absent (no commit associated)
	if strings.Contains(content, "commit:") {
		t.Errorf("expected no commit field in front matter for manual mode, got:\n%s", content)
	}
}

func TestHandleProactive_WithAllArgs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// Only alternatives + impact prompts remain → two Enter keys
	input := "\n\n"
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	opts := ProactiveOpts{
		Type: "feature",
		What: "add auth middleware",
		Why:  "JWT for stateless auth",
	}

	err := HandleProactive(context.Background(), workDir, streams, opts)
	if err != nil {
		t.Fatalf("HandleProactive with args: %v", err)
	}

	// Verify document was created with correct type prefix
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "feature-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected feature-*.md, got: %v", names)
	}
}

func TestHandleProactive_InvalidTypeAskInteractively(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// Invalid type "foobar" → falls to interactive: user types "bugfix"
	// then what, why, alt (skip), impact (skip)
	input := "bugfix\nfix login\nbug was critical\n\n\n"
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	opts := ProactiveOpts{Type: "foobar"}

	err := HandleProactive(context.Background(), workDir, streams, opts)
	if err != nil {
		t.Fatalf("HandleProactive invalid type: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bugfix-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected bugfix-*.md after interactive type correction, got: %v", names)
	}
}

func TestRunProactiveQuestions_SkipsPrefilledArgs(t *testing.T) {
	// Unit test: verify pre-filled args skip prompts (no stdin needed for skipped questions)
	// Only alternatives + impact prompts remain → two Enter keys
	input := "\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	renderer := NewLineRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	opts := ProactiveOpts{
		Type: "decision",
		What: "choose database",
		Why:  "PostgreSQL for ACID compliance",
	}

	answers, err := runProactiveQuestions(context.Background(), flow, opts)
	if err != nil {
		t.Fatalf("runProactiveQuestions: %v", err)
	}

	if answers.Type != "decision" {
		t.Errorf("Type = %q, want %q", answers.Type, "decision")
	}
	if answers.What != "choose database" {
		t.Errorf("What = %q, want %q", answers.What, "choose database")
	}
	if answers.Why != "PostgreSQL for ACID compliance" {
		t.Errorf("Why = %q, want %q", answers.Why, "PostgreSQL for ACID compliance")
	}

	// Stderr should show confirmations for skipped questions
	out := stderr.String()
	if !strings.Contains(out, "decision") {
		t.Errorf("expected type confirmation in stderr, got: %q", out)
	}
	if !strings.Contains(out, "choose database") {
		t.Errorf("expected what confirmation in stderr, got: %q", out)
	}
}

func TestRunProactiveQuestions_InvalidTypeDefaultsToNote(t *testing.T) {
	// Invalid type → interactive prompt with "note" default → user presses Enter.
	// What + Why are pre-filled (skipped). Interactive prompts remaining:
	//   Enter 1: Type (accept "note" default)
	//   Enter 2: Alternatives (skip)
	//   Enter 3: Impact (skip)
	input := "\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	renderer := NewLineRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	opts := ProactiveOpts{
		Type: "invalid-type",
		What: "some what",
		Why:  "some why",
	}

	answers, err := runProactiveQuestions(context.Background(), flow, opts)
	if err != nil {
		t.Fatalf("runProactiveQuestions: %v", err)
	}

	// Invalid type → user pressed Enter → default "note"
	if answers.Type != "note" {
		t.Errorf("Type = %q, want %q (default for invalid type)", answers.Type, "note")
	}
}

func TestToGenerateInput_ManualMode(t *testing.T) {
	answers := Answers{
		Type: "feature",
		What: "add auth",
		Why:  "needed",
	}

	input := answers.ToGenerateInput(nil, "manual")

	if input.GeneratedBy != "manual" {
		t.Errorf("GeneratedBy = %q, want %q", input.GeneratedBy, "manual")
	}
	if input.CommitInfo != nil {
		t.Errorf("CommitInfo = %v, want nil for manual mode", input.CommitInfo)
	}
}

// H2: context cancellation during lore new must save partial answers as pending.
func TestHandleProactive_ContextCancelled_SavesPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// Cancel context immediately — flow will fail on first readLine
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streams := domain.IOStreams{
		In:  strings.NewReader("any\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleProactive(ctx, workDir, streams, ProactiveOpts{})
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}

	// A pending file must have been written (no commit hash → "unknown-{ts}" filename)
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, readErr := os.ReadDir(pendingDir)
	if readErr != nil {
		t.Fatalf("ReadDir pending: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("expected a pending file to be created on context cancellation in lore new")
	}
}
