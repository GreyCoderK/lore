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

// --- ClassifyHunk with Lines field (merged hunks) ---

func TestClassifyHunk_Lines_PureAddition(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '+', Text: "new line 1"},
			{Kind: '+', Text: "new line 2"},
		},
	}
	if got := ClassifyHunk(h); got != HunkPureAddition {
		t.Errorf("ClassifyHunk with Lines = %d, want HunkPureAddition", got)
	}
}

func TestClassifyHunk_Lines_PureDeletion(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "old line 1"},
			{Kind: '=', Text: "context"},
			{Kind: '-', Text: "old line 2"},
		},
	}
	if got := ClassifyHunk(h); got != HunkPureDeletion {
		t.Errorf("ClassifyHunk with Lines = %d, want HunkPureDeletion", got)
	}
}

func TestClassifyHunk_Lines_Cosmetic(t *testing.T) {
	h := DiffHunk{
		Original: []string{"hello  "},
		Modified: []string{"hello"},
		Lines: []DiffLine{
			{Kind: '-', Text: "hello  "},
			{Kind: '+', Text: "hello"},
		},
	}
	if got := ClassifyHunk(h); got != HunkCosmetic {
		t.Errorf("ClassifyHunk with Lines = %d, want HunkCosmetic", got)
	}
}

func TestClassifyHunk_Lines_MajorDeletion(t *testing.T) {
	var lines []DiffLine
	for i := 0; i < 20; i++ {
		lines = append(lines, DiffLine{Kind: '-', Text: fmt.Sprintf("line %d", i)})
	}
	lines = append(lines, DiffLine{Kind: '+', Text: "replacement"})
	h := DiffHunk{Lines: lines}
	if got := ClassifyHunk(h); got != HunkMajorDeletion {
		t.Errorf("ClassifyHunk with Lines = %d, want HunkMajorDeletion", got)
	}
}

// --- isCosmetic with Lines field ---

func TestIsCosmetic_WithLines_True(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "foo  \t"},
			{Kind: '+', Text: "foo"},
			{Kind: '=', Text: "context line"},
			{Kind: '-', Text: "bar   "},
			{Kind: '+', Text: "bar"},
		},
	}
	if !isCosmetic(h) {
		t.Error("expected cosmetic with Lines whitespace-only changes")
	}
}

func TestIsCosmetic_WithLines_False(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "foo"},
			{Kind: '+', Text: "bar"},
		},
	}
	if isCosmetic(h) {
		t.Error("expected non-cosmetic with Lines content changes")
	}
}

func TestIsCosmetic_WithLines_UnequalCount(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "foo"},
			{Kind: '+', Text: "foo"},
			{Kind: '+', Text: "extra"},
		},
	}
	if isCosmetic(h) {
		t.Error("expected non-cosmetic when del/add counts differ")
	}
}

// --- renderHunkBody tests ---

func TestRenderHunkBody_WithLines(t *testing.T) {
	var buf bytes.Buffer
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "removed"},
			{Kind: '=', Text: "context"},
			{Kind: '+', Text: "added"},
		},
	}
	renderHunkBody(&buf, h)
	out := buf.String()
	if !strings.Contains(out, "removed") {
		t.Error("expected removed line in output")
	}
	if !strings.Contains(out, "context") {
		t.Error("expected context line in output")
	}
	if !strings.Contains(out, "added") {
		t.Error("expected added line in output")
	}
}

func TestRenderHunkBody_WithoutLines(t *testing.T) {
	var buf bytes.Buffer
	h := DiffHunk{
		Original: []string{"old"},
		Modified: []string{"new"},
	}
	renderHunkBody(&buf, h)
	out := buf.String()
	if !strings.Contains(out, "old") {
		t.Error("expected original line in output")
	}
	if !strings.Contains(out, "new") {
		t.Error("expected modified line in output")
	}
}

// --- summarizeHunkContent tests ---

func TestSummarizeHunkContent_PureAddition_SingleLine(t *testing.T) {
	h := DiffHunk{Modified: []string{"This is a new line"}}
	got := summarizeHunkContent(h, HunkPureAddition)
	if !strings.HasPrefix(got, "+\"") {
		t.Errorf("single-line addition should start with +\", got %q", got)
	}
	if !strings.Contains(got, "This is a new line") {
		t.Errorf("should contain the line text, got %q", got)
	}
}

func TestSummarizeHunkContent_PureAddition_SingleLineTruncation(t *testing.T) {
	long := strings.Repeat("x", 60)
	h := DiffHunk{Modified: []string{long}}
	got := summarizeHunkContent(h, HunkPureAddition)
	if !strings.Contains(got, "…") {
		t.Errorf("long single-line addition should be truncated, got %q", got)
	}
}

func TestSummarizeHunkContent_PureAddition_MultiLine(t *testing.T) {
	h := DiffHunk{Modified: []string{"line 1", "line 2", "line 3"}}
	got := summarizeHunkContent(h, HunkPureAddition)
	if got != "+3 lines" {
		t.Errorf("expected '+3 lines', got %q", got)
	}
}

func TestSummarizeHunkContent_PureDeletion_WithSections(t *testing.T) {
	h := DiffHunk{Original: []string{"## Overview", "some text", "### Details", "more text"}}
	got := summarizeHunkContent(h, HunkPureDeletion)
	if !strings.Contains(got, "-4 lines") {
		t.Errorf("should mention line count, got %q", got)
	}
	if !strings.Contains(got, "## Overview") {
		t.Errorf("should mention section heading, got %q", got)
	}
	if !strings.Contains(got, "### Details") {
		t.Errorf("should mention subsection heading, got %q", got)
	}
}

func TestSummarizeHunkContent_PureDeletion_NoSections(t *testing.T) {
	h := DiffHunk{Original: []string{"line a", "line b"}}
	got := summarizeHunkContent(h, HunkPureDeletion)
	if got != "-2 lines" {
		t.Errorf("expected '-2 lines', got %q", got)
	}
}

func TestSummarizeHunkContent_Cosmetic(t *testing.T) {
	h := DiffHunk{Original: []string{"a  "}, Modified: []string{"a"}}
	got := summarizeHunkContent(h, HunkCosmetic)
	if got != "whitespace fix" {
		t.Errorf("expected 'whitespace fix', got %q", got)
	}
}

func TestSummarizeHunkContent_MermaidDiagram(t *testing.T) {
	h := DiffHunk{Modified: []string{"```mermaid", "graph TD", "A-->B", "```"}}
	got := summarizeHunkContent(h, HunkPureAddition)
	if got != "+mermaid diagram" {
		t.Errorf("expected '+mermaid diagram', got %q", got)
	}
}

func TestSummarizeHunkContent_Table(t *testing.T) {
	h := DiffHunk{Modified: []string{"| Col A | Col B |", "|---|---|", "| 1 | 2 |"}}
	got := summarizeHunkContent(h, HunkPureAddition)
	if got != "+table" {
		t.Errorf("expected '+table', got %q", got)
	}
}

func TestSummarizeHunkContent_WithLines(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '+', Text: "new line 1"},
			{Kind: '+', Text: "new line 2"},
			{Kind: '=', Text: "context"},
		},
	}
	got := summarizeHunkContent(h, HunkPureAddition)
	if got != "+2 lines" {
		t.Errorf("expected '+2 lines', got %q", got)
	}
}

// --- printAutoSummary tests ---

func TestPrintAutoSummary_Format(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	r := &AutoResult{Accepted: 3, Rejected: 1, Asked: 2}
	printAutoSummary(streams, r)
	out := errBuf.String()
	if !strings.Contains(out, "3") || !strings.Contains(out, "1") || !strings.Contains(out, "2") {
		t.Errorf("summary should contain counts 3/1/2, got %q", out)
	}
}

// --- analyzeHunk with Lines field tests ---

func TestAnalyzeHunk_WithLines_NetLoss(t *testing.T) {
	var lines []DiffLine
	for i := 0; i < 20; i++ {
		lines = append(lines, DiffLine{Kind: '-', Text: fmt.Sprintf("line %d", i)})
	}
	lines = append(lines, DiffLine{Kind: '+', Text: "single replacement"})
	h := DiffHunk{Lines: lines}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "removes") || strings.Contains(w, "lines") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected net-loss warning with Lines, got: %v", warnings)
	}
}

func TestAnalyzeHunk_WithLines_SectionDeletion(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "## Architecture"},
			{Kind: '-', Text: "content here"},
			{Kind: '+', Text: "replaced"},
		},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "section") && strings.Contains(w, "Architecture") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected section deletion warning with Lines, got: %v", warnings)
	}
}

func TestAnalyzeHunk_WithLines_CodeBlockRemoval(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "```go"},
			{Kind: '-', Text: "func main() {}"},
			{Kind: '-', Text: "```"},
			{Kind: '+', Text: "removed code"},
		},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "code block") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected code block warning with Lines, got: %v", warnings)
	}
}

func TestAnalyzeHunk_WithLines_TableRowsRemoval(t *testing.T) {
	h := DiffHunk{
		Lines: []DiffLine{
			{Kind: '-', Text: "| A | B |"},
			{Kind: '-', Text: "|---|---|"},
			{Kind: '-', Text: "| 1 | 2 |"},
			{Kind: '-', Text: "| 3 | 4 |"},
			{Kind: '+', Text: "replaced"},
		},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "table rows") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected table rows warning with Lines, got: %v", warnings)
	}
}

func TestAnalyzeHunk_MultipleSections(t *testing.T) {
	h := DiffHunk{
		Original: []string{"## First", "text", "## Second", "text"},
		Modified: []string{"replaced"},
	}
	warnings := analyzeHunk(h)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "2 sections") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected multi-section warning, got: %v", warnings)
	}
}

// --- InteractiveDiff extended tests ---

func TestInteractiveDiff_DryRun_AllHunksRejected(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
		{Original: []string{"c"}, Modified: []string{"C"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{DryRun: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, c := range choices {
		if c != DiffReject {
			t.Errorf("hunk %d: expected DiffReject in dry run, got %d", i, c)
		}
	}
}

func TestInteractiveDiff_YesAll_AllHunksAccepted(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
		{Original: []string{"c"}, Modified: []string{"C"}},
		{Original: []string{"d"}, Modified: []string{"D"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{YesAll: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, c := range choices {
		if c != DiffAccept {
			t.Errorf("hunk %d: expected DiffAccept with YesAll, got %d", i, c)
		}
	}
}

func TestInteractiveDiff_Standard_YNQ(t *testing.T) {
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
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Errorf("hunk 0: expected DiffAccept, got %d", choices[0])
	}
	if choices[1] != DiffReject {
		t.Errorf("hunk 1: expected DiffReject, got %d", choices[1])
	}
	// hunk 2 should remain DiffReject since q causes early return
	if choices[2] != DiffReject {
		t.Errorf("hunk 2: expected DiffReject after quit, got %d", choices[2])
	}
}

func TestInteractiveDiff_Standard_BothInput(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"old line"}, Modified: []string{"new line"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("b\n"),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffBoth {
		t.Errorf("hunk 0: expected DiffBoth, got %d", choices[0])
	}
}

func TestInteractiveDiff_Standard_InputEndedMidway(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"a"}, Modified: []string{"A"}},
		{Original: []string{"b"}, Modified: []string{"B"}},
		{Original: []string{"c"}, Modified: []string{"C"}},
		{Original: []string{"d"}, Modified: []string{"D"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("y\n"), // only 1 answer for 4 hunks
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Errorf("hunk 0: expected DiffAccept, got %d", choices[0])
	}
	// Remaining hunks should be DiffReject (zero value) since input ended
	for i := 1; i < len(choices); i++ {
		if choices[i] != DiffReject {
			t.Errorf("hunk %d: expected DiffReject after EOF, got %d", i, choices[i])
		}
	}
}

func TestInteractiveDiff_Auto_PureAdditionAccepted(t *testing.T) {
	hunks := []DiffHunk{
		{Original: nil, Modified: []string{"new line 1", "new line 2"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{Auto: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Errorf("pure addition in auto mode: expected DiffAccept, got %d", choices[0])
	}
}

func TestInteractiveDiff_Auto_PureDeletionRejected(t *testing.T) {
	hunks := []DiffHunk{
		{Original: []string{"deleted line 1", "deleted line 2"}, Modified: nil},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader(""),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{Auto: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffReject {
		t.Errorf("pure deletion in auto mode: expected DiffReject, got %d", choices[0])
	}
}

func TestInteractiveDiff_Auto_ModificationAsksUser(t *testing.T) {
	hunks := []DiffHunk{
		{
			Original: []string{"old line 1", "old line 2", "old line 3"},
			Modified: []string{"new line 1", "new line 2", "new line 3", "new line 4"},
		},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("y\n"),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{Auto: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Errorf("modification with 'y' input in auto mode: expected DiffAccept, got %d", choices[0])
	}
}

func TestInteractiveDiff_Auto_MixedHunkTypes(t *testing.T) {
	hunks := []DiffHunk{
		// Pure addition → auto-accept
		{Original: nil, Modified: []string{"added"}},
		// Pure deletion → auto-reject
		{Original: []string{"removed"}, Modified: nil},
		// Modification → asks user, user says n
		{Original: []string{"old"}, Modified: []string{"completely different new text"}},
	}
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("n\n"),
	}
	choices, err := InteractiveDiff(hunks, streams, DiffOptions{Auto: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if choices[0] != DiffAccept {
		t.Errorf("hunk 0 (addition): expected DiffAccept, got %d", choices[0])
	}
	if choices[1] != DiffReject {
		t.Errorf("hunk 1 (deletion): expected DiffReject, got %d", choices[1])
	}
	if choices[2] != DiffReject {
		t.Errorf("hunk 2 (modification, user said n): expected DiffReject, got %d", choices[2])
	}
}
