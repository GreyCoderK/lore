// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestComputeDiff_IdenticalDocs(t *testing.T) {
	doc := "line 1\nline 2\nline 3"
	hunks := ComputeDiff(doc, doc)
	if len(hunks) != 0 {
		t.Errorf("identical docs: expected 0 hunks, got %d", len(hunks))
	}
}

func TestComputeDiff_SingleLineChange(t *testing.T) {
	original := "line 1\nline 2\nline 3"
	modified := "line 1\nline 2 modified\nline 3"
	hunks := ComputeDiff(original, modified)
	if len(hunks) == 0 {
		t.Fatal("expected at least 1 hunk for single line change")
	}
	// The hunk should contain the original and modified lines
	found := false
	for _, h := range hunks {
		for _, l := range h.Modified {
			if l == "line 2 modified" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected modified line 'line 2 modified' in hunks")
	}
}

func TestComputeDiff_Addition(t *testing.T) {
	original := "line 1\nline 3"
	modified := "line 1\nline 2 new\nline 3"
	hunks := ComputeDiff(original, modified)
	if len(hunks) == 0 {
		t.Fatal("expected at least 1 hunk for addition")
	}
	found := false
	for _, h := range hunks {
		for _, l := range h.Modified {
			if l == "line 2 new" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected added line 'line 2 new' in hunks")
	}
}

func TestComputeDiff_Deletion(t *testing.T) {
	original := "line 1\nline 2\nline 3"
	modified := "line 1\nline 3"
	hunks := ComputeDiff(original, modified)
	if len(hunks) == 0 {
		t.Fatal("expected at least 1 hunk for deletion")
	}
	found := false
	for _, h := range hunks {
		for _, l := range h.Original {
			if l == "line 2" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected deleted line 'line 2' in hunks")
	}
}

func TestApplyDiff_AllAccepted(t *testing.T) {
	original := "line 1\nline 2\nline 3"
	modified := "line 1\nline 2 changed\nline 3"
	hunks := ComputeDiff(original, modified)
	accepted := make([]bool, len(hunks))
	for i := range accepted {
		accepted[i] = true
	}

	result := ApplyDiff(original, hunks, accepted)
	if result != modified {
		t.Errorf("ApplyDiff all accepted:\ngot:  %q\nwant: %q", result, modified)
	}
}

func TestApplyDiff_AllRejected(t *testing.T) {
	original := "line 1\nline 2\nline 3"
	modified := "line 1\nline 2 changed\nline 3"
	hunks := ComputeDiff(original, modified)
	accepted := make([]bool, len(hunks)) // all false

	result := ApplyDiff(original, hunks, accepted)
	if result != original {
		t.Errorf("ApplyDiff all rejected:\ngot:  %q\nwant: %q", result, original)
	}
}

func TestApplyDiff_PartialAcceptance(t *testing.T) {
	original := "aaa\nbbb\nccc\nddd\neee"
	modified := "aaa\nBBB\nccc\nDDD\neee"
	hunks := ComputeDiff(original, modified)

	if len(hunks) < 2 {
		t.Skipf("expected 2+ hunks for partial test, got %d", len(hunks))
	}

	// Accept first hunk only
	accepted := make([]bool, len(hunks))
	accepted[0] = true

	result := ApplyDiff(original, hunks, accepted)
	// First change applied, second not
	if result == original {
		t.Error("result should differ from original (first hunk accepted)")
	}
	if result == modified {
		t.Error("result should differ from modified (second hunk rejected)")
	}
}

func TestFormatDiff_Output(t *testing.T) {
	hunks := []DiffHunk{
		{
			ContextBefore: []string{"before"},
			Original:      []string{"old line"},
			Modified:      []string{"new line"},
			ContextAfter:  []string{"after"},
		},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	FormatDiff(hunks, streams)
	out := errBuf.String()
	if !strings.Contains(out, "old line") {
		t.Errorf("expected original line in output, got %q", out)
	}
	if !strings.Contains(out, "new line") {
		t.Errorf("expected modified line in output, got %q", out)
	}
}

func TestFormatDiff_MultipleHunks(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	FormatDiff(hunks, streams)
	if !strings.Contains(errBuf.String(), "---") {
		t.Error("expected separator between hunks")
	}
}

func TestInteractiveDiff_DryRun(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"old"}, Modified: []string{"new"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	accepted, err := InteractiveDiff(hunks, streams, DiffOptions{DryRun: true})
	if err != nil {
		t.Fatalf("InteractiveDiff dry run: %v", err)
	}
	if accepted[0] {
		t.Error("dry run should not accept any hunks")
	}
}

func TestInteractiveDiff_YesAll(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	accepted, err := InteractiveDiff(hunks, streams, DiffOptions{YesAll: true})
	if err != nil {
		t.Fatalf("InteractiveDiff yes-all: %v", err)
	}
	for i, a := range accepted {
		if !a {
			t.Errorf("hunk %d should be accepted with YesAll", i)
		}
	}
}

func TestInteractiveDiff_UserInput(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
		{Original: []string{"c"}, Modified: []string{"C"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("y\nn\nq\n"),
	}
	accepted, err := InteractiveDiff(hunks, streams, DiffOptions{})
	if err != nil {
		t.Fatalf("InteractiveDiff: %v", err)
	}
	if !accepted[0] {
		t.Error("hunk 0 should be accepted (user said y)")
	}
	if accepted[1] {
		t.Error("hunk 1 should be rejected (user said n)")
	}
	if accepted[2] {
		t.Error("hunk 2 should be rejected (user said q)")
	}
}

func TestInteractiveDiff_EOFInput(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("y\n"), // EOF before second hunk
	}
	accepted, err := InteractiveDiff(hunks, streams, DiffOptions{})
	if err != nil {
		t.Fatalf("InteractiveDiff: %v", err)
	}
	if !accepted[0] {
		t.Error("hunk 0 should be accepted")
	}
	if accepted[1] {
		t.Error("hunk 1 should not be accepted after EOF")
	}
}

func TestComputeDiff_EmptyOriginal(t *testing.T) {
	hunks := ComputeDiff("", "new content")
	if len(hunks) == 0 {
		t.Error("expected hunks for empty → non-empty")
	}
}

func TestComputeDiff_EmptyModified(t *testing.T) {
	hunks := ComputeDiff("original content", "")
	if len(hunks) == 0 {
		t.Error("expected hunks for non-empty → empty")
	}
}

func TestComputeDiff_BothEmpty(t *testing.T) {
	hunks := ComputeDiff("", "")
	if len(hunks) != 0 {
		t.Errorf("expected 0 hunks for empty → empty, got %d", len(hunks))
	}
}

func TestFindHunkPosition_PureInsertion(t *testing.T) {
	lines := []string{"a", "b", "c"}
	pos := findHunkPosition(lines, nil, 1)
	if pos != 1 {
		t.Errorf("expected position 1 for pure insertion, got %d", pos)
	}
}

func TestFindHunkPosition_FallbackSearch(t *testing.T) {
	lines := []string{"a", "b", "c", "d"}
	// Hint is wrong, should fallback to linear search
	pos := findHunkPosition(lines, []string{"c"}, 0)
	if pos != 2 {
		t.Errorf("expected position 2, got %d", pos)
	}
}

func TestMatchLines_DifferentLengths(t *testing.T) {
	if matchLines([]string{"a"}, []string{"a", "b"}) {
		t.Error("different lengths should not match")
	}
}

func TestSplitLines_Empty(t *testing.T) {
	result := splitLines("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}
