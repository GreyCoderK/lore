// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
)

// DiffHunk represents a contiguous group of changes with surrounding context.
type DiffHunk struct {
	OrigStart     int      // 0-based line index in original
	OrigCount     int      // number of original lines in this hunk
	ModStart      int      // 0-based line index in modified
	ModCount      int      // number of modified lines in this hunk
	ContextBefore []string // up to 3 lines before the change
	Original      []string // lines removed/changed from original
	Modified      []string // lines added/changed in modified
	ContextAfter  []string // up to 3 lines after the change
}

// maxDiffLines is the maximum line count before falling back to full-document replacement.
// Prevents O(n*m) memory explosion from malicious AI responses.
const maxDiffLines = 5000

// ComputeDiff produces a list of diff hunks between original and modified text.
// Uses a simple LCS-based line diff. Returns nil if texts are identical.
// Falls back to a single whole-document hunk if either text exceeds maxDiffLines.
func ComputeDiff(original, modified string) []DiffHunk {
	origLines := splitLines(original)
	modLines := splitLines(modified)

	// Guard against O(n*m) memory explosion
	if len(origLines) > maxDiffLines || len(modLines) > maxDiffLines {
		return []DiffHunk{{
			OrigStart: 0, OrigCount: len(origLines),
			ModStart: 0, ModCount: len(modLines),
			Original: origLines, Modified: modLines,
		}}
	}

	// Compute LCS table
	lcs := computeLCS(origLines, modLines)

	// Backtrack to find edit script
	edits := backtrackEdits(origLines, modLines, lcs)

	// Group edits into hunks with context
	return groupHunks(origLines, modLines, edits, 3)
}

// editOp represents a single line-level edit operation.
type editOp struct {
	kind    byte // '=' equal, '-' delete, '+' insert
	origIdx int  // index in original (or -1 for inserts)
	modIdx  int  // index in modified (or -1 for deletes)
}

func computeLCS(a, b []string) [][]int {
	m, n := len(a), len(b)
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}
	return dp
}

func backtrackEdits(a, b []string, dp [][]int) []editOp {
	var ops []editOp
	i, j := len(a), len(b)
	for i > 0 || j > 0 {
		if i > 0 && j > 0 && a[i-1] == b[j-1] {
			ops = append(ops, editOp{'=', i - 1, j - 1})
			i--
			j--
		} else if j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]) {
			ops = append(ops, editOp{'+', -1, j - 1})
			j--
		} else {
			ops = append(ops, editOp{'-', i - 1, -1})
			i--
		}
	}
	// Reverse
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

func groupHunks(origLines, modLines []string, edits []editOp, contextSize int) []DiffHunk {
	// Find change regions (runs of non-equal ops)
	type region struct{ start, end int }
	var regions []region
	inChange := false
	start := 0
	for i, op := range edits {
		if op.kind != '=' {
			if !inChange {
				start = i
				inChange = true
			}
		} else if inChange {
			regions = append(regions, region{start, i})
			inChange = false
		}
	}
	if inChange {
		regions = append(regions, region{start, len(edits)})
	}

	var hunks []DiffHunk
	for _, r := range regions {
		hunk := DiffHunk{OrigStart: -1, ModStart: -1}

		// Collect changed lines
		for _, op := range edits[r.start:r.end] {
			if op.kind == '-' {
				hunk.Original = append(hunk.Original, origLines[op.origIdx])
				if hunk.OrigStart < 0 {
					hunk.OrigStart = op.origIdx
				}
				hunk.OrigCount++
			}
			if op.kind == '+' {
				hunk.Modified = append(hunk.Modified, modLines[op.modIdx])
				if hunk.ModStart < 0 {
					hunk.ModStart = op.modIdx
				}
				hunk.ModCount++
			}
		}
		// Ensure non-negative for pure insertions/deletions
		if hunk.OrigStart < 0 {
			hunk.OrigStart = 0
		}
		if hunk.ModStart < 0 {
			hunk.ModStart = 0
		}

		// Context before: look at edits before the region
		ctxStart := r.start - contextSize
		if ctxStart < 0 {
			ctxStart = 0
		}
		for i := ctxStart; i < r.start; i++ {
			if edits[i].kind == '=' {
				hunk.ContextBefore = append(hunk.ContextBefore, origLines[edits[i].origIdx])
			}
		}

		// Context after: look at edits after the region
		ctxEnd := r.end + contextSize
		if ctxEnd > len(edits) {
			ctxEnd = len(edits)
		}
		for i := r.end; i < ctxEnd; i++ {
			if edits[i].kind == '=' {
				hunk.ContextAfter = append(hunk.ContextAfter, origLines[edits[i].origIdx])
			}
		}

		hunks = append(hunks, hunk)
	}

	return hunks
}

// FormatDiff displays hunks to the writer with colored output.
func FormatDiff(hunks []DiffHunk, streams domain.IOStreams) {
	w := streams.Err
	for i, h := range hunks {
		if i > 0 {
			_, _ = fmt.Fprintln(w, "---")
		}
		for _, line := range h.ContextBefore {
			_, _ = fmt.Fprintf(w, " %s\n", ui.Dim(line))
		}
		for _, line := range h.Original {
			_, _ = fmt.Fprintf(w, "%s\n", ui.Error("- "+line))
		}
		for _, line := range h.Modified {
			_, _ = fmt.Fprintf(w, "%s\n", ui.Success("+ "+line))
		}
		for _, line := range h.ContextAfter {
			_, _ = fmt.Fprintf(w, " %s\n", ui.Dim(line))
		}
	}
}

// InteractiveDiff prompts the user for each hunk.
// Returns a slice of booleans indicating acceptance per hunk.
// dryRun: show diff only, no prompts. yesAll: accept all without prompting.
func InteractiveDiff(hunks []DiffHunk, streams domain.IOStreams, dryRun bool, yesAll bool) ([]bool, error) {
	accepted := make([]bool, len(hunks))

	if dryRun {
		FormatDiff(hunks, streams)
		return accepted, nil // all false — dry run doesn't apply
	}

	if yesAll {
		for i := range accepted {
			accepted[i] = true
		}
		FormatDiff(hunks, streams)
		return accepted, nil
	}

	scanner := bufio.NewScanner(streams.In)
	for i, h := range hunks {
		_, _ = fmt.Fprintf(streams.Err, "\n--- Change %d/%d ---\n", i+1, len(hunks))
		// Show this hunk
		for _, line := range h.ContextBefore {
			_, _ = fmt.Fprintf(streams.Err, " %s\n", ui.Dim(line))
		}
		for _, line := range h.Original {
			_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Error("- "+line))
		}
		for _, line := range h.Modified {
			_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Success("+ "+line))
		}
		for _, line := range h.ContextAfter {
			_, _ = fmt.Fprintf(streams.Err, " %s\n", ui.Dim(line))
		}

		_, _ = fmt.Fprintf(streams.Err, "Apply this change? [y/n/q] ")
		if !scanner.Scan() {
			remaining := len(hunks) - i - 1
			if remaining > 0 {
				_, _ = fmt.Fprintf(streams.Err, "\nInput ended. %d remaining changes rejected.\n", remaining)
			}
			break
		}
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch input {
		case "y", "yes":
			accepted[i] = true
		case "q", "quit":
			return accepted, nil
		default:
			accepted[i] = false
		}
	}

	return accepted, nil
}

// ApplyDiff applies accepted hunks to the original text.
// Hunks are applied in reverse order to preserve line offsets.
func ApplyDiff(original string, hunks []DiffHunk, accepted []bool) string {
	lines := splitLines(original)

	// Apply in reverse to preserve offsets
	for i := len(hunks) - 1; i >= 0; i-- {
		if !accepted[i] {
			continue
		}
		h := hunks[i]

		// Find the position in lines matching h.Original
		pos := findHunkPosition(lines, h.Original, h.OrigStart)
		if pos < 0 {
			continue // hunk can't be applied — skip safely
		}

		// Replace original lines with modified lines
		newLines := make([]string, 0, len(lines)-len(h.Original)+len(h.Modified))
		newLines = append(newLines, lines[:pos]...)
		newLines = append(newLines, h.Modified...)
		newLines = append(newLines, lines[pos+len(h.Original):]...)
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

// findHunkPosition finds where the hunk's original lines appear in the document.
// Uses hint (expected start position) for efficiency, falls back to linear search.
func findHunkPosition(lines, original []string, hint int) int {
	if len(original) == 0 {
		// Pure insertion — insert at hint position
		if hint >= 0 && hint <= len(lines) {
			return hint
		}
		return len(lines)
	}

	// Try hint first
	if hint >= 0 && hint+len(original) <= len(lines) {
		if matchLines(lines[hint:hint+len(original)], original) {
			return hint
		}
	}

	// Fallback: linear search
	for i := 0; i+len(original) <= len(lines); i++ {
		if matchLines(lines[i:i+len(original)], original) {
			return i
		}
	}
	return -1
}

func matchLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
