// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// TestShowMilestone_ThresholdReached verifies that showMilestone emits a
// reinforcement message when the doc count hits an exact threshold (count=3).
// Higher thresholds (10, 25, 50) are covered by the unit test in
// engagement/messages_test.go — no need to create 50 files on disk.
func TestShowMilestone_ThresholdReached(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	docsDir := filepath.Join(t.TempDir(), ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   "decision",
			Date:   "2026-03-15",
			Status: "published",
			Commit: fmt.Sprintf("aaa%037d", i),
		}, fmt.Sprintf("decision %d", i), fmt.Sprintf("# Decision %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	showMilestone(streams, docsDir, true)

	if !strings.Contains(stderr.String(), "3 decisions captured") {
		t.Errorf("expected milestone message for 3 docs, got: %q", stderr.String())
	}
}

// TestShowMilestone_NonTTY verifies milestone output is suppressed for non-TTY.
func TestShowMilestone_NonTTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	docsDir := filepath.Join(t.TempDir(), ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   "decision",
			Date:   "2026-03-15",
			Status: "published",
			Commit: fmt.Sprintf("bbb%037d", i),
		}, fmt.Sprintf("nontty decision %d", i), fmt.Sprintf("# Decision %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	showMilestone(streams, docsDir, false)

	if stderr.Len() != 0 {
		t.Errorf("expected no milestone output for non-TTY, got: %q", stderr.String())
	}
}

// TestShowMilestone_NoThreshold verifies no message when count is not a threshold.
func TestShowMilestone_NoThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	docsDir := filepath.Join(t.TempDir(), ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for i := 0; i < 2; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   "decision",
			Date:   "2026-03-15",
			Status: "published",
			Commit: fmt.Sprintf("ccc%037d", i),
		}, fmt.Sprintf("no threshold %d", i), fmt.Sprintf("# Decision %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	showMilestone(streams, docsDir, true)

	if stderr.Len() != 0 {
		t.Errorf("expected no milestone output for non-threshold count, got: %q", stderr.String())
	}
}

// TestShowMilestone_PassedThreshold (AC-6): verify that a passed threshold
// (count=4) does NOT emit the milestone-3 message.
func TestShowMilestone_PassedThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	docsDir := filepath.Join(t.TempDir(), ".lore", "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	for i := 0; i < 4; i++ {
		_, err := storage.WriteDoc(docsDir, domain.DocMeta{
			Type:   "decision",
			Date:   "2026-03-15",
			Status: "published",
			Commit: fmt.Sprintf("ddd%037d", i),
		}, fmt.Sprintf("passed threshold %d", i), fmt.Sprintf("# Decision %d\n\nBody.\n", i))
		if err != nil {
			t.Fatalf("setup WriteDoc[%d]: %v", i, err)
		}
	}

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	showMilestone(streams, docsDir, true)

	if stderr.Len() != 0 {
		t.Errorf("expected no milestone output for passed threshold (count=4), got: %q", stderr.String())
	}
}

// N9 fix: verify showMilestone is a no-op when docsDir does not exist.
func TestShowMilestone_MissingDocsDir(t *testing.T) {
	docsDir := filepath.Join(t.TempDir(), "nonexistent", "docs")

	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}

	showMilestone(streams, docsDir, true)

	if stderr.Len() != 0 {
		t.Errorf("expected no output for missing docsDir, got: %q", stderr.String())
	}
}
