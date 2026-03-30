// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

func setupStatusTest(t *testing.T, docs []testutil.DocFixture) (string, domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	dir := testutil.SetupGitRepoWithHook(t)

	// Create .lore structure
	for _, sub := range []string{
		filepath.Join(".lore", "docs"),
		filepath.Join(".lore", "templates"),
		filepath.Join(".lore", "pending"),
	} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}

	// Create docs
	docsDir := filepath.Join(dir, ".lore", "docs")
	for _, d := range docs {
		meta := domain.DocMeta{Type: d.Type, Date: d.Date, Status: "draft", Tags: d.Tags}
		body := d.Body
		if body == "" {
			body = "# " + strings.ReplaceAll(d.Slug, "-", " ") + "\n\nTest document.\n"
		}
		data, err := storage.Marshal(meta, body)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		filename := d.Type + "-" + d.Slug + "-" + d.Date + ".md"
		os.WriteFile(filepath.Join(docsDir, filename), data, 0o644)
	}

	// Create README.md so health passes
	os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# Index\n"), 0o644)

	testutil.Chdir(t, dir)

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	return dir, streams, &out, &errBuf
}

func executeStatus(t *testing.T, streams domain.IOStreams, args ...string) error {
	t.Helper()
	cfg := &config.Config{}
	cmd := newStatusCmd(cfg, streams)
	cmd.SetArgs(args)
	return cmd.Execute()
}

// AC-1: Dashboard format
func TestStatusCmd_Dashboard(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
		{Type: "feature", Slug: "api", Date: "2026-03-08"},
	})

	// Disable colors for predictable output
	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stderr := errBuf.String()
	if !strings.Contains(stderr, "lore status") {
		t.Errorf("expected header, got %q", stderr)
	}
	// NOTE: Label assertions below are coupled to i18n English strings.
	// If we add multi-language support, switch these to use i18n.T() references.
	if !strings.Contains(stderr, "Hook:") {
		t.Errorf("expected Hook label, got %q", stderr)
	}
	if !strings.Contains(stderr, "Docs:") {
		t.Errorf("expected Docs label, got %q", stderr)
	}
	if !strings.Contains(stderr, "2 documented") {
		t.Errorf("expected '2 documented', got %q", stderr)
	}
	if !strings.Contains(stderr, "Angela:") {
		t.Errorf("expected Angela label, got %q", stderr)
	}
	if !strings.Contains(stderr, "Health:") {
		t.Errorf("expected Health label, got %q", stderr)
	}
}

// AC-2: Health check OK
func TestStatusCmd_HealthOK(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, nil)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NOTE: "all good" is an i18n English string; acceptable for MVP.
	if !strings.Contains(errBuf.String(), "all good") {
		t.Errorf("expected 'all good', got %q", errBuf.String())
	}
}

// AC-3: Health check issues
func TestStatusCmd_HealthIssues(t *testing.T) {
	dir, streams, _, errBuf := setupStatusTest(t, nil)

	// Create orphan .tmp file
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "orphan.tmp"), []byte(""), 0o644)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "issues") {
		t.Errorf("expected 'issues', got %q", stderr)
	}
	if !strings.Contains(stderr, "lore doctor") {
		t.Errorf("expected 'lore doctor' suggestion, got %q", stderr)
	}
}

// AC-4: Pending count
func TestStatusCmd_PendingCount(t *testing.T) {
	dir, streams, _, errBuf := setupStatusTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
	})

	// Create pending files
	pendingDir := filepath.Join(dir, ".lore", "pending")
	os.WriteFile(filepath.Join(pendingDir, "abc123.yaml"), []byte("pending"), 0o644)
	os.WriteFile(filepath.Join(pendingDir, "def456.yaml"), []byte("pending"), 0o644)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "2 pending") {
		t.Errorf("expected '2 pending', got %q", errBuf.String())
	}
}

// AC-6: Hook status displayed
func TestStatusCmd_HookInstalled(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, nil)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NOTE: i18n-coupled assertion; acceptable for MVP.
	if !strings.Contains(errBuf.String(), "installed (post-commit)") {
		t.Errorf("expected 'installed (post-commit)', got %q", errBuf.String())
	}
}

// AC-8: Quiet mode
func TestStatusCmd_Quiet(t *testing.T) {
	_, streams, out, errBuf := setupStatusTest(t, []testutil.DocFixture{
		{Type: "decision", Slug: "auth", Date: "2026-03-07"},
	})

	err := executeStatus(t, streams, "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
	stdout := out.String()
	if !strings.Contains(stdout, "hook=installed") {
		t.Errorf("expected 'hook=installed', got %q", stdout)
	}
	if !strings.Contains(stdout, "docs=1") {
		t.Errorf("expected 'docs=1', got %q", stdout)
	}
	if !strings.Contains(stdout, "health=ok") {
		t.Errorf("expected 'health=ok', got %q", stdout)
	}
	if !strings.Contains(stdout, "angela=draft") {
		t.Errorf("expected 'angela=draft', got %q", stdout)
	}
}

// AC-8 + AC-3: Quiet mode with health issues
func TestStatusCmd_QuietHealthIssues(t *testing.T) {
	dir, streams, out, errBuf := setupStatusTest(t, nil)

	// Create orphan .tmp file to trigger health issue
	os.WriteFile(filepath.Join(dir, ".lore", "docs", "orphan.tmp"), []byte(""), 0o644)

	err := executeStatus(t, streams, "--quiet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Errorf("expected no stderr in quiet mode, got %q", errBuf.String())
	}
	stdout := out.String()
	if !strings.Contains(stdout, "health=1-issues") {
		t.Errorf("expected 'health=1-issues', got %q", stdout)
	}
}

// AC-9: Not initialized
func TestStatusCmd_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	testutil.Chdir(t, dir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	err := executeStatus(t, streams)
	if err == nil {
		t.Fatal("expected error for not initialized")
	}
	if !errors.Is(err, domain.ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
	if !strings.Contains(errBuf.String(), "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized', got %q", errBuf.String())
	}
}

// Tagline present in dashboard
func TestStatusCmd_Tagline(t *testing.T) {
	_, streams, _, errBuf := setupStatusTest(t, nil)

	restore := ui.SaveAndDisableColor()
	defer restore()

	err := executeStatus(t, streams)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// NOTE: Tagline is an i18n English string; acceptable for MVP.
	if !strings.Contains(errBuf.String(), "Your code knows what. Lore knows why.") {
		t.Errorf("expected tagline, got %q", errBuf.String())
	}
}
