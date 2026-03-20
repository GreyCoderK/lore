// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
)

func setupListTest(t *testing.T, docs []testutil.DocFixture) (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := testutil.SetupLoreDirWithDocs(t, docs)
	testutil.Chdir(t, dir)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	return streams, &out, &errBuf
}

func executeList(t *testing.T, streams domain.IOStreams, args ...string) error {
	t.Helper()
	cfg := &config.Config{}
	cmd := newListCmd(cfg, streams)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// AC-1: Full listing — parseable output on stdout
func TestListCmd_FullListing(t *testing.T) {
	streams, out, _ := setupListTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth-strategy", Date: "2026-03-07", Tags: []string{"auth", "jwt", "security"}},
		{Type: "feature", Slug: "add-jwt-middleware", Date: "2026-03-05", Tags: []string{"jwt"}},
		{Type: "bugfix", Slug: "token-expiry-fix", Date: "2026-03-01"},
	})

	err := executeList(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stdout := out.String()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), stdout)
	}

	// Each line should contain type, slug, date, tag count
	if !strings.Contains(lines[0], "decision") || !strings.Contains(lines[0], "auth-strategy") {
		t.Errorf("expected first line to contain decision/auth-strategy, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "3 tags") {
		t.Errorf("expected '3 tags' in first line, got %q", lines[0])
	}
	if !strings.Contains(lines[2], "0 tags") {
		t.Errorf("expected '0 tags' in last line, got %q", lines[2])
	}
}

// AC-1: Single tag uses singular "tag"
func TestListCmd_SingleTagSingular(t *testing.T) {
	streams, out, _ := setupListTest(t, []testutil.DocFixture{
		{Type: "feature", Slug: "one-tag", Date: "2026-03-05", Tags: []string{"jwt"}},
	})

	err := executeList(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "1 tag\n") {
		t.Errorf("expected '1 tag' (singular), got %q", out.String())
	}
}

// AC-2: Empty corpus → stderr message
func TestListCmd_EmptyCorpus(t *testing.T) {
	streams, out, errBuf := setupListTest(t, nil)

	err := executeList(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no stdout, got %q", out.String())
	}
	if !strings.Contains(errBuf.String(), "No documents yet") {
		t.Errorf("expected empty corpus message, got %q", errBuf.String())
	}
	if !strings.Contains(errBuf.String(), "lore new") {
		t.Errorf("expected 'lore new' suggestion, got %q", errBuf.String())
	}
}

// AC-3: --quiet suppresses stderr
func TestListCmd_Quiet(t *testing.T) {
	streams, _, errBuf := setupListTest(t, nil)

	err := executeList(t, streams, "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
}

// AC-3: --quiet compatible with wc -l (newline terminated)
func TestListCmd_QuietNewlineTerminated(t *testing.T) {
	streams, out, _ := setupListTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
		{Type: "feature", Slug: "api", Date: "2026-03-08"},
	})

	err := executeList(t, streams, "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	if !strings.HasSuffix(stdout, "\n") {
		t.Errorf("expected output to end with newline, got %q", stdout)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

// AC-4: --type filter
func TestListCmd_TypeFilter(t *testing.T) {
	streams, out, _ := setupListTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
		{Type: "feature", Slug: "api", Date: "2026-03-08"},
		{Type: "bugfix", Slug: "fix", Date: "2026-03-01"},
	})

	err := executeList(t, streams, "--type", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line, got %d: %q", len(lines), stdout)
	}
	if !strings.Contains(lines[0], "feature") {
		t.Errorf("expected feature type, got %q", lines[0])
	}
}

// AC-4: --type filtering to empty results (different from empty corpus)
func TestListCmd_TypeFilterNoMatch(t *testing.T) {
	streams, out, errBuf := setupListTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
		{Type: "bugfix", Slug: "fix", Date: "2026-03-01"},
	})

	err := executeList(t, streams, "--type", "feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no stdout, got %q", out.String())
	}
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "No documents of type 'feature'") {
		t.Errorf("expected type-specific empty message, got %q", errOutput)
	}
	// Should NOT say "No documents yet" when corpus has docs
	if strings.Contains(errOutput, "No documents yet") {
		t.Errorf("should not show 'No documents yet' when corpus has docs: %q", errOutput)
	}
}

// AC-5: Sort by date descending
func TestListCmd_SortDateDescending(t *testing.T) {
	streams, out, _ := setupListTest(t, []testutil.DocFixture{
		{Type: "bugfix", Slug: "old-fix", Date: "2026-01-01"},
		{Type: "decision", Slug: "mid-auth", Date: "2026-02-15"},
		{Type: "feature", Slug: "new-api", Date: "2026-03-10"},
	})

	err := executeList(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	// Most recent first
	if !strings.Contains(lines[0], "2026-03-10") {
		t.Errorf("expected newest date first, got %q", lines[0])
	}
	if !strings.Contains(lines[2], "2026-01-01") {
		t.Errorf("expected oldest date last, got %q", lines[2])
	}
}

// AC-6: Repo not initialized
func TestListCmd_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := executeList(t, streams)
	if err == nil {
		t.Fatal("expected error for not initialized")
	}
	if cli.ExitCodeFrom(err) != cli.ExitError {
		t.Errorf("expected exit code %d, got %d", cli.ExitError, cli.ExitCodeFrom(err))
	}
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized', got %q", errOutput)
	}
	if !strings.Contains(errOutput, "lore init") {
		t.Errorf("expected 'lore init' suggestion, got %q", errOutput)
	}
}
