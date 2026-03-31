// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/testutil"
	"github.com/greycoderk/lore/internal/ui"
)

// writeDocWithCommit creates a document file with a commit in front matter.
func writeDocWithCommit(t *testing.T, docsDir string, docType, slug, date, commit string) {
	t.Helper()
	meta := domain.DocMeta{
		Type:   docType,
		Date:   date,
		Status: "draft",
		Commit: commit,
	}
	body := "# " + slug + "\n\nTest document.\n"
	data, err := storage.Marshal(meta, body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	filename := docType + "-" + slug + "-" + date + ".md"
	if err := os.WriteFile(filepath.Join(docsDir, filename), data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestRelease_WithDocuments(t *testing.T) {
	ui.SetColorEnabled(false)
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	commitA := "aaaa1111222233334444555566667777aaaabbbb"
	commitB := "bbbb1111222233334444555566667777aaaabbbb"

	writeDocWithCommit(t, docsDir, "feature", "jwt-middleware", "2026-03-05", commitA)
	writeDocWithCommit(t, docsDir, "bugfix", "token-expiry", "2026-03-06", commitB)

	streams, out, errBuf := testStreams()

	mock := &mockGitAdapter{
		CommitRangeFunc: func(from, to string) ([]string, error) {
			return []string{commitA, commitB}, nil
		},
		LatestTagFunc: func() (string, error) {
			return "v1.0.0", nil
		},
	}

	err := runRelease(streams, mock, "v0.9.0", "v1.0.0", "v1.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// NOTE: "Released" is an i18n English string; acceptable for MVP.
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "Released") {
		t.Errorf("expected 'Released' verb in stderr, got: %s", errOutput)
	}

	// stdout should be empty in non-quiet mode
	if out.String() != "" {
		t.Errorf("expected empty stdout in non-quiet mode, got: %s", out.String())
	}

	// Verify releases.json was created
	data, readErr := os.ReadFile(filepath.Join(dir, ".lore", "releases.json"))
	if readErr != nil {
		t.Fatalf("read releases.json: %v", readErr)
	}
	var entries []storage.ReleaseEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(entries) != 1 || entries[0].Version != "v1.0.0" {
		t.Errorf("unexpected releases.json content: %+v", entries)
	}

	// Verify CHANGELOG.md was created
	changelogData, readErr := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if readErr != nil {
		t.Fatalf("read CHANGELOG.md: %v", readErr)
	}
	changelog := string(changelogData)
	if !strings.Contains(changelog, "## [v1.0.0]") {
		t.Errorf("CHANGELOG.md missing release section: %s", changelog)
	}

	// L4: Verify README.md index was regenerated with release doc
	indexData, readErr := os.ReadFile(filepath.Join(docsDir, "README.md"))
	if readErr != nil {
		t.Fatalf("read README.md index: %v", readErr)
	}
	indexContent := string(indexData)
	if !strings.Contains(indexContent, "release-v1.0.0") {
		t.Errorf("README.md index should contain release doc, got: %s", indexContent)
	}
}

func TestRelease_NoDocuments(t *testing.T) {
	ui.SetColorEnabled(false)
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		CommitRangeFunc: func(from, to string) ([]string, error) {
			return []string{"deadbeef"}, nil
		},
		LatestTagFunc: func() (string, error) {
			return "v1.0.0", nil
		},
	}

	err := runRelease(streams, mock, "v0.9.0", "v1.0.0", "v1.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// NOTE: i18n-coupled assertion; acceptable for MVP.
	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "No documented changes") {
		t.Errorf("expected 'No documented changes' message, got: %s", errOutput)
	}
}

func TestRelease_VersionFlag(t *testing.T) {
	ui.SetColorEnabled(false)
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	commitA := "aaaa1111222233334444555566667777aaaabbbb"
	writeDocWithCommit(t, docsDir, "feature", "jwt-middleware", "2026-03-05", commitA)

	streams, _, errBuf := testStreams()

	mock := &mockGitAdapter{
		CommitRangeFunc: func(from, to string) ([]string, error) {
			return []string{commitA}, nil
		},
		LatestTagFunc: func() (string, error) {
			return "v1.0.0", nil
		},
	}

	err := runRelease(streams, mock, "v0.9.0", "HEAD", "v2.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "release-v2.0.0") {
		t.Errorf("expected version v2.0.0 in output, got: %s", errOutput)
	}
}

func TestRelease_FromToFlags(t *testing.T) {
	ui.SetColorEnabled(false)
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	commitA := "aaaa1111222233334444555566667777aaaabbbb"
	writeDocWithCommit(t, docsDir, "feature", "jwt-middleware", "2026-03-05", commitA)

	streams, _, _ := testStreams()

	var capturedFrom, capturedTo string
	mock := &mockGitAdapter{
		CommitRangeFunc: func(from, to string) ([]string, error) {
			capturedFrom = from
			capturedTo = to
			return []string{commitA}, nil
		},
		LatestTagFunc: func() (string, error) {
			return "v1.0.0", nil
		},
	}

	err := runRelease(streams, mock, "v0.5.0", "v1.0.0", "v1.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFrom != "v0.5.0" {
		t.Errorf("expected from=v0.5.0, got %s", capturedFrom)
	}
	if capturedTo != "v1.0.0" {
		t.Errorf("expected to=v1.0.0, got %s", capturedTo)
	}
}

func TestRelease_QuietMode(t *testing.T) {
	ui.SetColorEnabled(false)
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)
	docsDir := filepath.Join(dir, ".lore", "docs")

	commitA := "aaaa1111222233334444555566667777aaaabbbb"
	writeDocWithCommit(t, docsDir, "feature", "jwt-middleware", "2026-03-05", commitA)

	streams, out, errBuf := testStreams()

	mock := &mockGitAdapter{
		CommitRangeFunc: func(from, to string) ([]string, error) {
			return []string{commitA}, nil
		},
		LatestTagFunc: func() (string, error) {
			return "v1.0.0", nil
		},
	}

	err := runRelease(streams, mock, "v0.9.0", "HEAD", "v1.0.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// AC-7: stdout = path only
	stdout := strings.TrimSpace(out.String())
	if !strings.HasSuffix(stdout, ".md") {
		t.Errorf("expected .md path on stdout, got: %s", stdout)
	}
	// Use filepath.Join for cross-platform path separator.
	expectedRelPath := filepath.Join(".lore", "docs", "release-v1.0.0")
	if !strings.Contains(stdout, expectedRelPath) {
		t.Errorf("expected release file path on stdout, got: %s", stdout)
	}

	// No messages on stderr
	if errBuf.String() != "" {
		t.Errorf("expected no stderr in quiet mode, got: %s", errBuf.String())
	}
}

func TestRelease_NotInitialized(t *testing.T) {
	ui.SetColorEnabled(false)
	dir := t.TempDir() // no .lore/
	testutil.Chdir(t, dir)

	streams, _, errBuf := testStreams()
	mock := &mockGitAdapter{}

	err := runRelease(streams, mock, "", "", "v1.0.0", false)
	if err == nil {
		t.Fatal("expected error for not initialized")
	}
	// NOTE: i18n-coupled assertions; acceptable for MVP.
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("expected 'not initialized' error, got: %v", err)
	}
	if !strings.Contains(errBuf.String(), "lore init") {
		t.Errorf("expected actionable error with 'lore init', got: %s", errBuf.String())
	}
}
