package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
)

func TestNewCmd_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	// chdir required: cmd uses os.Getwd() to find .lore/
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	cfg := &config.Config{}

	cmd := newNewCmd(cfg, streams)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not initialized")
	}

	// AC-4: actionable error message
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in error output, got %q", errOutput)
	}
	if !strings.Contains(errOutput, "lore init") {
		t.Errorf("expected 'lore init' suggestion in error output, got %q", errOutput)
	}
}

func TestNewCmd_NoArgs_FullInteractiveFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// chdir required: cmd uses os.Getwd() to find .lore/
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// No args: type, what, why all prompted; alt+impact skipped with Enter.
	// H1 fix: no default for Type, user must type a value.
	input := "decision\nadd auth\nbecause JWT\n\n\n"
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(input),
	}
	cfg := &config.Config{}

	cmd := newNewCmd(cfg, streams)
	cmd.SetContext(t.Context())
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("lore new: %v", err)
	}

	// Verify document was created
	entries, _ := os.ReadDir(filepath.Join(dir, ".lore", "docs"))
	var docFound bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "decision-") && strings.HasSuffix(e.Name(), ".md") {
			docFound = true
		}
	}
	if !docFound {
		t.Error("expected decision-*.md document in .lore/docs/")
	}

	// AC-2: "Captured" verb in stderr
	if !strings.Contains(errBuf.String(), "Captured") {
		t.Errorf("expected 'Captured' in stderr, got: %q", errBuf.String())
	}

	// M6: dim path line must appear below "Captured"
	if !strings.Contains(errBuf.String(), ".lore/docs/") {
		t.Errorf("expected dim path '.lore/docs/' in stderr, got: %q", errBuf.String())
	}
}

func TestNewCmd_WithArgs_SkipsPrompts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// chdir required: cmd uses os.Getwd() to find .lore/
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// With all 3 args, only alternatives + impact remain → 2 Enter keys
	input := "\n\n"
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(input),
	}
	cfg := &config.Config{}

	cmd := newNewCmd(cfg, streams)
	cmd.SetContext(t.Context())
	cmd.SetArgs([]string{"feature", "add auth middleware", "JWT for stateless auth"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("lore new with args: %v", err)
	}

	// Verify feature-* document
	entries, _ := os.ReadDir(filepath.Join(".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "feature-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		t.Error("expected feature-*.md document in .lore/docs/")
	}
}

// M5: AC-2 — arguments manquants sont demandes interactivement (1-arg case).
func TestNewCmd_OneArg_AsksRemainingInteractively(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// chdir required: cmd uses os.Getwd() to find .lore/
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// 1 arg (type=note): what + why prompted interactively; alt + impact skipped.
	input := "my doc\nbecause reasons\n\n\n"
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		In:  strings.NewReader(input),
	}
	cfg := &config.Config{}

	cmd := newNewCmd(cfg, streams)
	cmd.SetContext(t.Context())
	cmd.SetArgs([]string{"note"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("lore new with 1 arg: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "note-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		t.Error("expected note-*.md document when only type arg is provided")
	}
}

// M5: AC-2 — arguments manquants sont demandes interactivement (2-args case).
func TestNewCmd_TwoArgs_AsksRemainingInteractively(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// chdir required: cmd uses os.Getwd() to find .lore/
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	// 2 args (type=bugfix, what="fix login"): only why prompted; alt + impact skipped.
	input := "login was broken\n\n\n"
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		In:  strings.NewReader(input),
	}
	cfg := &config.Config{}

	cmd := newNewCmd(cfg, streams)
	cmd.SetContext(t.Context())
	cmd.SetArgs([]string{"bugfix", "fix login"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("lore new with 2 args: %v", err)
	}

	entries, _ := os.ReadDir(filepath.Join(".lore", "docs"))
	var found bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "bugfix-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
		}
	}
	if !found {
		t.Error("expected bugfix-*.md document when type+what args are provided")
	}
}

func TestNewCmd_TooManyArgs(t *testing.T) {
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		In:  strings.NewReader(""),
	}
	cfg := &config.Config{}

	cmd := newNewCmd(cfg, streams)
	cmd.SetArgs([]string{"a", "b", "c", "d"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for too many args")
	}
}
