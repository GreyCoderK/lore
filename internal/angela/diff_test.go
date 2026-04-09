// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bytes"
	"fmt"
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
	choices := make([]DiffChoice, len(hunks))
	for i := range choices {
		choices[i] = DiffAccept
	}

	result := ApplyDiff(original, hunks, choices)
	if result != modified {
		t.Errorf("ApplyDiff all accepted:\ngot:  %q\nwant: %q", result, modified)
	}
}

func TestApplyDiff_AllRejected(t *testing.T) {
	original := "line 1\nline 2\nline 3"
	modified := "line 1\nline 2 changed\nline 3"
	hunks := ComputeDiff(original, modified)
	choices := make([]DiffChoice, len(hunks)) // all DiffReject (zero value)

	result := ApplyDiff(original, hunks, choices)
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
	choices := make([]DiffChoice, len(hunks))
	choices[0] = DiffAccept

	result := ApplyDiff(original, hunks, choices)
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
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{DryRun: true})
	if err != nil {
		t.Fatalf("InteractiveDiff dry run: %v", err)
	}
	if choices[0] != DiffReject {
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
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{YesAll: true})
	if err != nil {
		t.Fatalf("InteractiveDiff yes-all: %v", err)
	}
	for i, c := range choices {
		if c != DiffAccept {
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
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{})
	if err != nil {
		t.Fatalf("InteractiveDiff: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Error("hunk 0 should be accepted (user said y)")
	}
	if choices[1] != DiffReject {
		t.Error("hunk 1 should be rejected (user said n)")
	}
	if choices[2] != DiffReject {
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
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{})
	if err != nil {
		t.Fatalf("InteractiveDiff: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Error("hunk 0 should be accepted")
	}
	if choices[1] != DiffReject {
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

// --- analyzeHunk tests ---

func TestAnalyzeHunk_MajorDeletion(t *testing.T) {
	orig := make([]string, 30)
	for i := range orig {
		orig[i] = fmt.Sprintf("line %d", i)
	}
	h := DiffHunk{Original: orig, Modified: []string{"replacement"}}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "removes") && strings.Contains(w, "lines") {
			found = true
		}
	}
	if !found {
		t.Error("expected warning about major content removal")
	}
}

func TestAnalyzeHunk_SectionDeletion(t *testing.T) {
	h := DiffHunk{
		Original: []string{"## Export PDF", "content here", "more content"},
		Modified: []string{"replaced"},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "section") && strings.Contains(w, "Export PDF") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about section removal, got: %v", warnings)
	}
}

func TestAnalyzeHunk_CodeBlockDeletion(t *testing.T) {
	h := DiffHunk{
		Original: []string{"```java", "public class Foo {}", "```", "text"},
		Modified: []string{"text"},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "code block") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about code block removal, got: %v", warnings)
	}
}

func TestAnalyzeHunk_TableDeletion(t *testing.T) {
	h := DiffHunk{
		Original: []string{"| A | B |", "|---|---|", "| 1 | 2 |", "| 3 | 4 |", "| 5 | 6 |"},
		Modified: []string{"replaced"},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "table rows") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about table removal, got: %v", warnings)
	}
}

func TestAnalyzeHunk_NoWarnings(t *testing.T) {
	h := DiffHunk{
		Original: []string{"old line"},
		Modified: []string{"new line"},
	}
	warnings := analyzeHunk(h)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for small change, got: %v", warnings)
	}
}

// --- ClassifyHunk tests ---

func TestClassifyHunk_PureAddition(t *testing.T) {
	h := DiffHunk{Original: nil, Modified: []string{"new line 1", "new line 2"}}
	if got := ClassifyHunk(h); got != HunkPureAddition {
		t.Errorf("ClassifyHunk = %d, want HunkPureAddition", got)
	}
}

func TestClassifyHunk_PureDeletion(t *testing.T) {
	h := DiffHunk{Original: []string{"old line"}, Modified: nil}
	if got := ClassifyHunk(h); got != HunkPureDeletion {
		t.Errorf("ClassifyHunk = %d, want HunkPureDeletion", got)
	}
}

func TestClassifyHunk_Cosmetic(t *testing.T) {
	h := DiffHunk{
		Original: []string{"line with trailing space  "},
		Modified: []string{"line with trailing space"},
	}
	if got := ClassifyHunk(h); got != HunkCosmetic {
		t.Errorf("ClassifyHunk = %d, want HunkCosmetic", got)
	}
}

func TestClassifyHunk_MajorDeletion(t *testing.T) {
	orig := make([]string, 20)
	for i := range orig {
		orig[i] = fmt.Sprintf("line %d", i)
	}
	h := DiffHunk{Original: orig, Modified: []string{"single replacement"}}
	if got := ClassifyHunk(h); got != HunkMajorDeletion {
		t.Errorf("ClassifyHunk = %d, want HunkMajorDeletion", got)
	}
}

func TestClassifyHunk_Modification(t *testing.T) {
	h := DiffHunk{
		Original: []string{"old line 1", "old line 2"},
		Modified: []string{"new line 1", "new line 2", "new line 3"},
	}
	if got := ClassifyHunk(h); got != HunkModification {
		t.Errorf("ClassifyHunk = %d, want HunkModification", got)
	}
}
