// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
)

func setupShowTest(t *testing.T, docs []testutil.DocFixture) (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
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

func executeShow(t *testing.T, streams domain.IOStreams, args ...string) error {
	t.Helper()
	cfg := &config.Config{}
	cmd := newShowCmd(cfg, streams)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// AC-10: Repo not initialized
func TestShowCmd_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := executeShow(t, streams, "auth")
	if err == nil {
		t.Fatal("expected error for not initialized")
	}
	if !errors.Is(err, domain.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized', got %q", errOutput)
	}
	if !strings.Contains(errOutput, "lore init") {
		t.Errorf("expected 'lore init' suggestion, got %q", errOutput)
	}
}

// AC-5: Zero results
func TestShowCmd_ZeroResults(t *testing.T) {
	streams, out, errBuf := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
	})

	err := executeShow(t, streams, "nonexistent")
	if err == nil {
		t.Fatal("expected error for zero results")
	}
	if cli.ExitCodeFrom(err) != cli.ExitSkip {
		t.Errorf("expected exit code %d, got %d", cli.ExitSkip, cli.ExitCodeFrom(err))
	}
	if out.Len() != 0 {
		t.Errorf("expected no stdout output, got %q", out.String())
	}
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "No documents matching 'nonexistent'") {
		t.Errorf("expected zero-result message, got %q", errOutput)
	}
	if !strings.Contains(errOutput, "lore show --all") {
		t.Errorf("expected --all suggestion, got %q", errOutput)
	}
}

// AC-2: Single result → stdout direct
func TestShowCmd_SingleResult(t *testing.T) {
	streams, out, _ := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth-strategy", Date: "2026-03-07", Body: "# Auth Strategy\n\nWe chose JWT.\n"},
	})

	err := executeShow(t, streams, "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "# Auth Strategy") {
		t.Errorf("expected document body in stdout, got %q", stdout)
	}
	if !strings.Contains(stdout, "type: decision") {
		t.Errorf("expected front matter in stdout, got %q", stdout)
	}
}

// AC-3: Multiple results (2-15) → list on stderr in non-TTY
func TestShowCmd_MultipleResults(t *testing.T) {
	streams, out, _ := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth-jwt", Date: "2026-03-07", Body: "# Auth JWT\n"},
		{Type: "feature", Slug: "auth-api", Date: "2026-03-08", Body: "# Auth API\n"},
		{Type: "bugfix", Slug: "auth-fix", Date: "2026-03-01", Body: "# Auth Fix\n"},
	})

	err := executeShow(t, streams, "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Non-TTY: ui.List prints to stdout
	stdout := out.String()
	if !strings.Contains(stdout, "decision") {
		t.Errorf("expected 'decision' in list output, got %q", stdout)
	}
	if !strings.Contains(stdout, "feature") {
		t.Errorf("expected 'feature' in list output, got %q", stdout)
	}
}

// AC-4: 16+ results → truncation
func TestShowCmd_ManyResults_Truncation(t *testing.T) {
	docs := make([]testutil.DocFixture, 20)
	for i := range docs {
		docs[i] = testutil.DocFixture{
			Type: "decision",
			Slug: "auth-doc-" + string(rune('a'+i)),
			Date: "2026-03-07",
			Body: "# Auth doc\n\nContent about auth.\n",
		}
	}

	streams, out, _ := setupShowTest(t, docs)

	err := executeShow(t, streams, "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "... and 5 more") {
		t.Errorf("expected truncation message, got %q", stdout)
	}
}

// AC-6: --type feature filter
func TestShowCmd_TypeFilter(t *testing.T) {
	streams, out, _ := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth-decision", Date: "2026-03-07", Body: "# Auth Decision\n"},
		{Type: "feature", Slug: "auth-feature", Date: "2026-03-08", Body: "# Auth Feature\n"},
	})

	err := executeShow(t, streams, "--type", "feature", "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	// Single result (only the feature) → direct display
	if !strings.Contains(stdout, "# Auth Feature") {
		t.Errorf("expected feature doc content, got %q", stdout)
	}
}

// AC-6: --feature shorthand
func TestShowCmd_FeatureShorthand(t *testing.T) {
	streams, out, _ := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth-decision", Date: "2026-03-07", Body: "# Auth Decision\n"},
		{Type: "feature", Slug: "auth-feature", Date: "2026-03-08", Body: "# Auth Feature\n"},
	})

	err := executeShow(t, streams, "--feature", "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "# Auth Feature") {
		t.Errorf("expected feature doc content, got %q", stdout)
	}
}

// AC-7: --after date filter
func TestShowCmd_AfterFilter(t *testing.T) {
	streams, out, _ := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "old-auth", Date: "2026-02-01", Body: "# Old Auth\n"},
		{Type: "decision", Slug: "new-auth", Date: "2026-03-10", Body: "# New Auth\n"},
	})

	err := executeShow(t, streams, "--after", "2026-03", "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "# New Auth") {
		t.Errorf("expected new doc content, got %q", stdout)
	}
}

// AC-8: --quiet suppresses stderr
func TestShowCmd_Quiet(t *testing.T) {
	streams, _, errBuf := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07", Body: "# Auth\n"},
	})

	err := executeShow(t, streams, "--quiet", "nonexistent")
	if err == nil {
		t.Fatal("expected error for zero results")
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
}

// AC-8: --quiet with multiple results → parseable list on stdout, no stderr
func TestShowCmd_QuietMultiple(t *testing.T) {
	streams, out, errBuf := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth-jwt", Date: "2026-03-07", Body: "# Auth JWT\n"},
		{Type: "feature", Slug: "auth-api", Date: "2026-03-08", Body: "# Auth API\n"},
	})

	err := executeShow(t, streams, "--quiet", "auth")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
	stdout := out.String()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 tab-separated lines, got %d: %q", len(lines), stdout)
	}
	for _, line := range lines {
		if !strings.Contains(line, "\t") {
			t.Errorf("expected tab-separated output, got %q", line)
		}
	}
}

// AC-9: --all lists all documents
func TestShowCmd_All(t *testing.T) {
	streams, out, _ := setupShowTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07", Body: "# Auth\n"},
		{Type: "feature", Slug: "api", Date: "2026-03-08", Body: "# API\n"},
		{Type: "bugfix", Slug: "fix", Date: "2026-03-01", Body: "# Fix\n"},
	})

	err := executeShow(t, streams, "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stdout := out.String()
	// Non-TTY: all 3 listed on stdout
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), stdout)
	}
}

// No keyword and no --all → usage error
func TestShowCmd_NoKeywordNoAll(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := executeShow(t, streams)
	if err == nil {
		t.Fatal("expected error for no keyword and no --all")
	}
	if cli.ExitCodeFrom(err) != cli.ExitUserError {
		t.Errorf("expected exit code %d, got %d", cli.ExitUserError, cli.ExitCodeFrom(err))
	}
}
