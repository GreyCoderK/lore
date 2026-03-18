package workflow

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
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

func TestAskQuestions_SkipsPrefilledArgs(t *testing.T) {
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

	qOpts := QuestionOpts{
		PreFilled: Answers{
			Type: "decision",
			What: "choose database",
			Why:  "PostgreSQL for ACID compliance",
		},
	}

	answers, err := flow.AskQuestions(context.Background(), qOpts)
	if err != nil {
		t.Fatalf("AskQuestions: %v", err)
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

func TestAskQuestions_InvalidTypeDefaultsToNote(t *testing.T) {
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

	qOpts := QuestionOpts{
		PreFilled: Answers{
			Type: "invalid-type",
			What: "some what",
			Why:  "some why",
		},
	}

	answers, err := flow.AskQuestions(context.Background(), qOpts)
	if err != nil {
		t.Fatalf("AskQuestions: %v", err)
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

// N1: generate failure (broken local template) must save partial answers as pending.
func TestHandleProactive_GenerateFails_SavesPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// Write a syntactically invalid Go template to the local templates dir.
	// Local templates are resolved lazily at Render time (not at New time),
	// so this causes generator.Generate() to fail — not loretemplate.New().
	brokenTmpl := filepath.Join(workDir, ".lore", "templates", "note.md.tmpl")
	if err := os.WriteFile(brokenTmpl, []byte("{{.Bad template syntax"), 0o644); err != nil {
		t.Fatalf("write broken template: %v", err)
	}

	// All args pre-filled → only alt+impact remain (2 Enters).
	streams := domain.IOStreams{
		In:  strings.NewReader("\n\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	opts := ProactiveOpts{Type: "note", What: "test doc", Why: "for testing"}
	err := HandleProactive(context.Background(), workDir, streams, opts)
	if err == nil {
		t.Fatal("expected error from broken template")
	}

	// N1: pending file must be saved even on generate failure.
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, readErr := os.ReadDir(pendingDir)
	if readErr != nil {
		t.Fatalf("ReadDir pending: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("expected pending file saved on generate failure, got none")
	}
}

// N2: WriteDoc failure must save partial answers as pending.
// Trigger: .lore/docs exists as a file — os.MkdirAll inside WriteDoc fails.
func TestHandleProactive_WriteDocFails_SavesPending(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := t.TempDir()
	// .lore/templates must exist for the template engine to initialise.
	os.MkdirAll(filepath.Join(workDir, ".lore", "templates"), 0o755)
	// .lore/docs as a regular FILE — WriteDoc's os.MkdirAll will fail.
	if err := os.WriteFile(filepath.Join(workDir, ".lore", "docs"), []byte("not-a-dir"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	streams := domain.IOStreams{
		In:  strings.NewReader("\n\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	opts := ProactiveOpts{Type: "note", What: "test doc", Why: "for testing"}
	err := HandleProactive(context.Background(), workDir, streams, opts)
	if err == nil {
		t.Fatal("expected error when docs dir is a file")
	}

	// N2: pending file must be saved even on WriteDoc failure.
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	entries, readErr := os.ReadDir(pendingDir)
	if readErr != nil {
		t.Fatalf("ReadDir pending: %v", readErr)
	}
	if len(entries) == 0 {
		t.Error("expected pending file saved on WriteDoc failure, got none")
	}
}

// N4 fix: end-to-end test for proactive milestone — pre-create 2 docs, run
// HandleProactive (creates 3rd), verify milestone-3 message via IsTTY injection.
func TestHandleProactive_MilestoneAtThreshold3(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	// Pre-create 2 documents so the proactive flow's 3rd triggers the milestone.
	for i := 0; i < 2; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   "decision",
			Date:   "2026-03-15",
			Status: "published",
			Commit: fmt.Sprintf("aaa%037d", i),
		}, fmt.Sprintf("proactive pre %d", i), fmt.Sprintf("# Pre %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	input := "note\nthird doc\nbecause milestone\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("HandleProactive milestone: %v", err)
	}

	output := stderr.String()
	if !strings.Contains(output, "3 decisions captured") {
		t.Errorf("expected milestone-3 message in proactive stderr, got: %q", output)
	}

	// Verify milestone appears AFTER "Captured".
	capturedIdx := strings.Index(output, "Captured")
	milestoneIdx := strings.Index(output, "3 decisions captured")
	if capturedIdx >= 0 && milestoneIdx >= 0 && milestoneIdx <= capturedIdx {
		t.Errorf("milestone should appear after Captured line")
	}
}

// --- Retroactive mode tests (Story 4.1) ---

func TestHandleProactive_RetroactivePreFill(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// CommitInfo with conventional commit type "feat" → maps to doc type "feature"
	commit := &domain.CommitInfo{
		Hash:    "abc1234567890abcdef1234567890abcdef123456",
		Author:  "Test Author",
		Message: "feat: add auth middleware",
		Type:    "feat",
		Subject: "add auth middleware",
	}

	// Type + What pre-filled from commit → only why, alt, impact prompted
	input := "because JWT\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	opts := ProactiveOpts{
		Commit: commit,
	}

	err := HandleProactive(context.Background(), workDir, streams, opts)
	if err != nil {
		t.Fatalf("HandleProactive retroactive: %v", err)
	}

	// Verify document was created with correct type
	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var docPath string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "feature-") && strings.HasSuffix(e.Name(), ".md") {
			docPath = filepath.Join(workDir, ".lore", "docs", e.Name())
		}
	}
	if docPath == "" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected feature-*.md in docs, got: %v", names)
	}

	// Verify front matter
	data, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	// AC-2: generated_by must be "retroactive"
	if !strings.Contains(content, "generated_by: retroactive") {
		t.Errorf("expected generated_by: retroactive, got:\n%s", content)
	}

	// AC-2: commit hash must be present
	if !strings.Contains(content, "commit: abc1234567890abcdef1234567890abcdef123456") {
		t.Errorf("expected commit hash in front matter, got:\n%s", content)
	}
}

func TestHandleProactive_RetroactiveNonConventionalCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)

	// Non-conventional commit: Type is empty, Subject is the full message
	commit := &domain.CommitInfo{
		Hash:    "def456789012345678901234567890abcdef1234",
		Author:  "Test Author",
		Message: "just a plain commit message",
		Type:    "",      // not conventional
		Subject: "just a plain commit message",
	}

	// Type not pre-filled (invalid) → user must type one; What is pre-filled
	input := "note\nbecause testing\n\n\n"
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}

	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{
		Commit: commit,
	})
	if err != nil {
		t.Fatalf("HandleProactive non-conventional: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(workDir, ".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "note-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		t.Error("expected note-*.md for non-conventional commit retroactive flow")
	}
}

func TestToGenerateInput_RetroactiveMode(t *testing.T) {
	commit := &domain.CommitInfo{
		Hash:    "abc1234567890abcdef1234567890abcdef123456",
		Message: "feature: add auth",
		Type:    "feature",
		Subject: "add auth",
	}

	answers := Answers{
		Type: "feature",
		What: "add auth",
		Why:  "needed",
	}

	input := answers.ToGenerateInput(commit, "retroactive")

	if input.GeneratedBy != "retroactive" {
		t.Errorf("GeneratedBy = %q, want %q", input.GeneratedBy, "retroactive")
	}
	if input.CommitInfo == nil {
		t.Fatal("CommitInfo should not be nil for retroactive mode")
	}
	if input.CommitInfo.Hash != commit.Hash {
		t.Errorf("CommitInfo.Hash = %q, want %q", input.CommitInfo.Hash, commit.Hash)
	}
}

// AC-4: already documented commit — user confirms creation of another doc.
func TestHandleProactive_AlreadyDocumented_Confirm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	commitHash := "abc1234567890abcdef1234567890abcdef123456"

	// Pre-create a document for this commit (different date to avoid filename collision)
	storage.WriteDoc(docsDir, domain.DocMeta{
		Type:        "feature",
		Date:        "2026-03-15",
		Status:      "published",
		Commit:      commitHash,
		GeneratedBy: "retroactive",
	}, "initial setup", "# Initial Setup\n")

	commit := &domain.CommitInfo{
		Hash:    commitHash,
		Author:  "Test Author",
		Message: "feat: initial setup",
		Type:    "feat",
		Subject: "initial setup",
	}

	// "y" confirms, then why + alt + impact
	input := "y\nbecause update\n\n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{
		Commit: commit,
		IsTTY:  func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("HandleProactive already documented (y): %v\nstderr: %s", err, stderr.String())
	}

	// Warning message
	if !strings.Contains(stderr.String(), "A document already exists for this commit") {
		t.Errorf("expected warning, got: %q", stderr.String())
	}

	// Second document created
	entries, _ := os.ReadDir(docsDir)
	var docCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md" {
			docCount++
		}
	}
	if docCount < 2 {
		t.Errorf("expected at least 2 documents, got %d", docCount)
	}
}

// AC-4: already documented commit — user declines.
func TestHandleProactive_AlreadyDocumented_Decline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	commitHash := "abc1234567890abcdef1234567890abcdef123456"

	storage.WriteDoc(docsDir, domain.DocMeta{
		Type:        "feature",
		Date:        "2026-03-15",
		Status:      "published",
		Commit:      commitHash,
		GeneratedBy: "retroactive",
	}, "initial setup", "# Initial Setup\n")

	commit := &domain.CommitInfo{
		Hash:    commitHash,
		Author:  "Test Author",
		Message: "feat: initial setup",
		Type:    "feat",
		Subject: "initial setup",
	}

	input := "n\n"
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{
		Commit: commit,
		IsTTY:  func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("HandleProactive already documented (n): %v", err)
	}

	// Only original doc should exist
	entries, _ := os.ReadDir(docsDir)
	var docCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md" {
			docCount++
		}
	}
	if docCount != 1 {
		t.Errorf("expected 1 document after declining, got %d", docCount)
	}
}

// AC-4: already documented commit — non-TTY safe default (do not create).
func TestHandleProactive_AlreadyDocumented_NonTTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	workDir := newProactiveWorkDir(t)
	docsDir := filepath.Join(workDir, ".lore", "docs")

	commitHash := "abc1234567890abcdef1234567890abcdef123456"

	storage.WriteDoc(docsDir, domain.DocMeta{
		Type:        "feature",
		Date:        "2026-03-16",
		Status:      "published",
		Commit:      commitHash,
		GeneratedBy: "retroactive",
	}, "initial setup", "# Initial Setup\n")

	commit := &domain.CommitInfo{
		Hash:    commitHash,
		Author:  "Test Author",
		Message: "feat: initial setup",
		Type:    "feat",
		Subject: "initial setup",
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	// IsTTY returns false → non-TTY safe default
	err := HandleProactive(context.Background(), workDir, streams, ProactiveOpts{
		Commit: commit,
		IsTTY:  func(_ domain.IOStreams) bool { return false },
	})
	if err != nil {
		t.Fatalf("HandleProactive already documented (non-TTY): %v", err)
	}

	// Warning shown
	if !strings.Contains(stderr.String(), "A document already exists for this commit") {
		t.Errorf("expected warning, got: %q", stderr.String())
	}

	// No new document created
	entries, _ := os.ReadDir(docsDir)
	var docCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".md") && e.Name() != "README.md" {
			docCount++
		}
	}
	if docCount != 1 {
		t.Errorf("expected 1 document (non-TTY safe default), got %d", docCount)
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
