// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
)

// DiffLine represents a single line in a diff hunk with its edit operation.
type DiffLine struct {
	Kind byte   // '=' unchanged, '-' removed, '+' added
	Text string // line content
}

// DiffHunk represents a contiguous group of changes with surrounding context.
type DiffHunk struct {
	OrigStart     int        // 0-based line index in original
	OrigCount     int        // number of original lines in this hunk
	ModStart      int        // 0-based line index in modified
	ModCount      int        // number of modified lines in this hunk
	ContextBefore []string   // up to 3 lines before the change
	Original      []string   // lines removed/changed from original (includes = lines for merged hunks)
	Modified      []string   // lines added/changed in modified (includes = lines for merged hunks)
	ContextAfter  []string   // up to 3 lines after the change
	Lines         []DiffLine // ordered edit operations for display (nil for non-merged hunks)
}

// maxDiffLines is the maximum line count before falling back to full-document replacement.
// Prevents O(n*m) memory explosion: 2000×2000 = 4M ints ≈ 32MB (acceptable for CLI).
const maxDiffLines = 2000

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
	var rawRegions []region
	inChange := false
	start := 0
	for i, op := range edits {
		if op.kind != '=' {
			if !inChange {
				start = i
				inChange = true
			}
		} else if inChange {
			rawRegions = append(rawRegions, region{start, i})
			inChange = false
		}
	}
	if inChange {
		rawRegions = append(rawRegions, region{start, len(edits)})
	}

	// Merge regions that are close together (gap < 2*contextSize equal lines).
	// This avoids showing 46 micro-hunks for a single logical change.
	mergeGap := 2 * contextSize
	var regions []region
	for _, r := range rawRegions {
		if len(regions) > 0 {
			prev := &regions[len(regions)-1]
			gap := 0
			for k := prev.end; k < r.start; k++ {
				if edits[k].kind == '=' {
					gap++
				}
			}
			if gap <= mergeGap {
				prev.end = r.end
				continue
			}
		}
		regions = append(regions, r)
	}

	var hunks []DiffHunk
	for _, r := range regions {
		hunk := DiffHunk{OrigStart: -1, ModStart: -1}

		// Collect all lines in the region: changed lines AND equal lines between sub-changes.
		// Equal lines within a merged region appear in both Original and Modified
		// so that ApplyDiff can match the full block correctly.
		// Lines tracks edit ops for proper display rendering.
		for _, op := range edits[r.start:r.end] {
			switch op.kind {
			case '-':
				hunk.Original = append(hunk.Original, origLines[op.origIdx])
				hunk.Lines = append(hunk.Lines, DiffLine{Kind: '-', Text: origLines[op.origIdx]})
				if hunk.OrigStart < 0 {
					hunk.OrigStart = op.origIdx
				}
				hunk.OrigCount++
			case '+':
				hunk.Modified = append(hunk.Modified, modLines[op.modIdx])
				hunk.Lines = append(hunk.Lines, DiffLine{Kind: '+', Text: modLines[op.modIdx]})
				if hunk.ModStart < 0 {
					hunk.ModStart = op.modIdx
				}
				hunk.ModCount++
			case '=':
				// Equal line inside a merged region — include in both sides
				hunk.Original = append(hunk.Original, origLines[op.origIdx])
				hunk.Modified = append(hunk.Modified, modLines[op.modIdx])
				hunk.Lines = append(hunk.Lines, DiffLine{Kind: '=', Text: origLines[op.origIdx]})
				if hunk.OrigStart < 0 {
					hunk.OrigStart = op.origIdx
				}
				if hunk.ModStart < 0 {
					hunk.ModStart = op.modIdx
				}
				hunk.OrigCount++
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
		// Show hunk location
		_, _ = fmt.Fprintf(w, "%s\n", ui.Dim(fmt.Sprintf(i18n.T().Angela.DiffHunkLocation, h.OrigStart+1, h.OrigCount)))
		for _, line := range h.ContextBefore {
			_, _ = fmt.Fprintf(w, " %s\n", ui.Dim(line))
		}
		renderHunkBody(w, h)
		for _, line := range h.ContextAfter {
			_, _ = fmt.Fprintf(w, " %s\n", ui.Dim(line))
		}
	}
}

// renderHunkBody writes the hunk body using Lines (merged hunks) or Original/Modified (simple hunks).
func renderHunkBody(w interface{ Write([]byte) (int, error) }, h DiffHunk) {
	if len(h.Lines) > 0 {
		for _, dl := range h.Lines {
			switch dl.Kind {
			case '-':
				_, _ = fmt.Fprintf(w, "%s\n", ui.Error("- "+dl.Text))
			case '+':
				_, _ = fmt.Fprintf(w, "%s\n", ui.Success("+ "+dl.Text))
			case '=':
				_, _ = fmt.Fprintf(w, " %s\n", ui.Dim(dl.Text))
			}
		}
		return
	}
	for _, line := range h.Original {
		_, _ = fmt.Fprintf(w, "%s\n", ui.Error("- "+line))
	}
	for _, line := range h.Modified {
		_, _ = fmt.Fprintf(w, "%s\n", ui.Success("+ "+line))
	}
}

// DiffChoice represents the user's decision for a hunk.
type DiffChoice int

const (
	DiffReject DiffChoice = iota // n — discard the change
	DiffAccept                   // y — apply the change (replace original with modified)
	DiffBoth                     // b — keep both original and modified lines
)

// HunkClass categorizes a hunk for auto-mode decisions.
type HunkClass int

const (
	HunkModification  HunkClass = iota // balanced change — needs review
	HunkPureAddition                    // only additions, no deletions
	HunkPureDeletion                    // only deletions, no additions
	HunkCosmetic                        // whitespace-only change
	HunkMajorDeletion                   // net loss > 15 lines
)

// ClassifyHunk determines the category of a hunk for auto-mode.
func ClassifyHunk(h DiffHunk) HunkClass {
	var dels, adds int
	if len(h.Lines) > 0 {
		for _, dl := range h.Lines {
			switch dl.Kind {
			case '-':
				dels++
			case '+':
				adds++
			}
		}
	} else {
		dels = len(h.Original)
		adds = len(h.Modified)
	}

	if dels == 0 && adds > 0 {
		return HunkPureAddition
	}
	if adds == 0 && dels > 0 {
		return HunkPureDeletion
	}
	if dels > 0 && adds > 0 && isCosmetic(h) {
		return HunkCosmetic
	}
	if dels-adds > 15 {
		return HunkMajorDeletion
	}
	return HunkModification
}

// isCosmetic returns true if every changed line pair differs only by trailing whitespace.
func isCosmetic(h DiffHunk) bool {
	orig := h.Original
	mod := h.Modified
	if len(h.Lines) > 0 {
		orig, mod = nil, nil
		for _, dl := range h.Lines {
			switch dl.Kind {
			case '-':
				orig = append(orig, dl.Text)
			case '+':
				mod = append(mod, dl.Text)
			}
		}
	}
	if len(orig) != len(mod) {
		return false
	}
	for i := range orig {
		if strings.TrimRight(orig[i], " \t") != strings.TrimRight(mod[i], " \t") {
			return false
		}
	}
	return true
}

// AutoResult holds the summary of auto-mode decisions.
type AutoResult struct {
	Accepted int
	Rejected int
	Asked    int
	Details  []string // human-readable lines for summary
}

// DiffOptions controls the behavior of InteractiveDiff.
type DiffOptions struct {
	DryRun bool
	YesAll bool
	Auto   bool // auto-accept additions, auto-reject deletions, ask modifications
}

// InteractiveDiff prompts the user for each hunk.
// Returns a slice of DiffChoice indicating the decision per hunk.
func InteractiveDiff(hunks []DiffHunk, streams domain.IOStreams, opts DiffOptions) ([]DiffChoice, error) {
	choices := make([]DiffChoice, len(hunks))

	if opts.DryRun {
		FormatDiff(hunks, streams)
		return choices, nil // all DiffReject — dry run doesn't apply
	}

	if opts.YesAll {
		for i := range choices {
			choices[i] = DiffAccept
		}
		FormatDiff(hunks, streams)
		return choices, nil
	}

	// Auto mode: classify hunks and auto-decide where possible
	if opts.Auto {
		ta := i18n.T().Angela
		autoResult := &AutoResult{}
		scanner := bufio.NewScanner(streams.In)

		for i, h := range hunks {
			class := ClassifyHunk(h)

			switch class {
			case HunkPureAddition, HunkCosmetic:
				choices[i] = DiffAccept
				autoResult.Accepted++
				label := "addition"
				if class == HunkCosmetic {
					label = "cosmetic"
				}
				desc := summarizeHunkContent(h, class)
				autoResult.Details = append(autoResult.Details, fmt.Sprintf("✓ %s (%s)", desc, label))
				_, _ = fmt.Fprintln(streams.Err, fmt.Sprintf(ta.DiffAutoAccept, desc, label))

			case HunkPureDeletion, HunkMajorDeletion:
				choices[i] = DiffReject
				autoResult.Rejected++
				desc := summarizeHunkContent(h, class)
				autoResult.Details = append(autoResult.Details, fmt.Sprintf("✗ %s (rejected)", desc))
				_, _ = fmt.Fprintln(streams.Err, fmt.Sprintf(ta.DiffAutoReject, desc))

			default: // HunkModification — ask interactively
				autoResult.Asked++
				_, _ = fmt.Fprintln(streams.Err, fmt.Sprintf(ta.DiffAutoNeedsReview, i+1, len(hunks)))
				_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Dim(fmt.Sprintf(ta.DiffHunkLocation, h.OrigStart+1, h.OrigCount)))
				for _, line := range h.ContextBefore {
					_, _ = fmt.Fprintf(streams.Err, " %s\n", ui.Dim(line))
				}
				renderHunkBody(streams.Err, h)
				for _, line := range h.ContextAfter {
					_, _ = fmt.Fprintf(streams.Err, " %s\n", ui.Dim(line))
				}

				warnings := analyzeHunk(h)
				for _, w := range warnings {
					_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Warning("⚠ "+w))
				}

				hasBoth := len(h.Original) > 0 && len(h.Modified) > 0
				if hasBoth {
					_, _ = fmt.Fprint(streams.Err, ta.DiffApplyBothPrompt)
				} else {
					_, _ = fmt.Fprint(streams.Err, ta.DiffApplyPrompt)
				}
				if !scanner.Scan() {
					break
				}
				input := strings.TrimSpace(strings.ToLower(scanner.Text()))
				switch input {
				case "y", "yes", "o", "oui":
					choices[i] = DiffAccept
				case "b", "both", "l":
					choices[i] = DiffBoth
				case "q", "quit", "quitter":
					printAutoSummary(streams, autoResult)
					return choices, nil
				default:
					choices[i] = DiffReject
				}
			}
		}

		printAutoSummary(streams, autoResult)
		return choices, nil
	}

	// Standard interactive mode
	ta := i18n.T().Angela
	scanner := bufio.NewScanner(streams.In)
	for i, h := range hunks {
		_, _ = fmt.Fprintln(streams.Err, "\n"+fmt.Sprintf(ta.DiffChangeHeader, i+1, len(hunks)))
		_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Dim(fmt.Sprintf(ta.DiffHunkLocation, h.OrigStart+1, h.OrigCount)))
		for _, line := range h.ContextBefore {
			_, _ = fmt.Fprintf(streams.Err, " %s\n", ui.Dim(line))
		}
		renderHunkBody(streams.Err, h)
		for _, line := range h.ContextAfter {
			_, _ = fmt.Fprintf(streams.Err, " %s\n", ui.Dim(line))
		}

		warnings := analyzeHunk(h)
		for _, w := range warnings {
			_, _ = fmt.Fprintf(streams.Err, "%s\n", ui.Warning("⚠ "+w))
		}

		hasBoth := len(h.Original) > 0 && len(h.Modified) > 0
		if hasBoth {
			_, _ = fmt.Fprint(streams.Err, ta.DiffApplyBothPrompt)
		} else {
			_, _ = fmt.Fprint(streams.Err, ta.DiffApplyPrompt)
		}
		if !scanner.Scan() {
			remaining := len(hunks) - i - 1
			if remaining > 0 {
				_, _ = fmt.Fprintf(streams.Err, "\n%s\n", fmt.Sprintf(ta.DiffInputEnded, remaining))
			}
			break
		}
		input := strings.TrimSpace(strings.ToLower(scanner.Text()))
		switch input {
		case "y", "yes", "o", "oui":
			choices[i] = DiffAccept
		case "b", "both", "l":
			choices[i] = DiffBoth
		case "q", "quit", "quitter":
			return choices, nil
		default:
			choices[i] = DiffReject
		}
	}

	return choices, nil
}

// summarizeHunkContent produces a brief description of a hunk for auto-mode logging.
func summarizeHunkContent(h DiffHunk, class HunkClass) string {
	var adds, dels []string
	if len(h.Lines) > 0 {
		for _, dl := range h.Lines {
			if dl.Kind == '+' {
				adds = append(adds, dl.Text)
			} else if dl.Kind == '-' {
				dels = append(dels, dl.Text)
			}
		}
	} else {
		adds = h.Modified
		dels = h.Original
	}

	// Check for specific content types
	joined := strings.Join(adds, "\n")
	if strings.Contains(joined, "```mermaid") {
		return "+mermaid diagram"
	}
	if strings.Contains(joined, "|---") || strings.Contains(joined, "| ---") {
		return "+table"
	}

	switch class {
	case HunkPureAddition:
		if len(adds) == 1 {
			text := strings.TrimSpace(adds[0])
			if len(text) > 50 {
				text = text[:50] + "…"
			}
			return fmt.Sprintf("+\"%s\"", text)
		}
		return fmt.Sprintf("+%d lines", len(adds))
	case HunkPureDeletion, HunkMajorDeletion:
		// Check for section headings in deleted content
		var sections []string
		for _, l := range dels {
			if strings.HasPrefix(strings.TrimSpace(l), "## ") || strings.HasPrefix(strings.TrimSpace(l), "### ") {
				sections = append(sections, strings.TrimSpace(l))
			}
		}
		if len(sections) > 0 {
			return fmt.Sprintf("-%d lines including %s", len(dels), strings.Join(sections, ", "))
		}
		return fmt.Sprintf("-%d lines", len(dels))
	case HunkCosmetic:
		return "whitespace fix"
	default:
		return fmt.Sprintf("±%d lines", len(adds)+len(dels))
	}
}

// printAutoSummary displays the auto-mode summary.
func printAutoSummary(streams domain.IOStreams, r *AutoResult) {
	_, _ = fmt.Fprintln(streams.Err, fmt.Sprintf(i18n.T().Angela.DiffAutoSummary, r.Accepted, r.Rejected, r.Asked))
}

// ApplyDiff applies chosen hunks to the original text.
// Hunks are applied in reverse order to preserve line offsets.
// DiffAccept replaces original with modified, DiffBoth keeps both.
func ApplyDiff(original string, hunks []DiffHunk, choices []DiffChoice) string {
	lines := splitLines(original)

	// Apply in reverse to preserve offsets
	for i := len(hunks) - 1; i >= 0; i-- {
		if choices[i] == DiffReject {
			continue
		}
		h := hunks[i]

		// Find the position in lines matching h.Original
		pos := findHunkPosition(lines, h.Original, h.OrigStart)
		if pos < 0 {
			continue // hunk can't be applied — skip safely
		}

		var replacement []string
		switch choices[i] {
		case DiffAccept:
			replacement = h.Modified
		case DiffBoth:
			// Keep original lines, then append only genuinely new lines from Modified.
			// Use Lines (edit ops) to identify which modified lines are truly additions
			// vs equal lines already present in Original.
			replacement = append(replacement, h.Original...)
			if len(h.Lines) > 0 {
				for _, dl := range h.Lines {
					if dl.Kind == '+' {
						replacement = append(replacement, dl.Text)
					}
				}
			} else {
				replacement = append(replacement, h.Modified...)
			}
		}

		newLines := make([]string, 0, len(lines)-len(h.Original)+len(replacement))
		newLines = append(newLines, lines[:pos]...)
		newLines = append(newLines, replacement...)
		newLines = append(newLines, lines[pos+len(h.Original):]...)
		lines = newLines
	}

	return strings.Join(lines, "\n")
}

// analyzeHunk inspects a hunk for potential coherence/readability issues
// and returns warning messages to show the user before they decide.
func analyzeHunk(h DiffHunk) []string {
	ta := i18n.T().Angela
	var warnings []string

	// Count real deletions and additions (excluding equal lines in merged hunks)
	var delLines, addLines int
	var deletedSections []string

	if len(h.Lines) > 0 {
		for _, dl := range h.Lines {
			switch dl.Kind {
			case '-':
				delLines++
				if strings.HasPrefix(dl.Text, "## ") || strings.HasPrefix(dl.Text, "### ") {
					deletedSections = append(deletedSections, strings.TrimSpace(dl.Text))
				}
			case '+':
				addLines++
			}
		}
	} else {
		delLines = len(h.Original)
		addLines = len(h.Modified)
		for _, line := range h.Original {
			if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
				deletedSections = append(deletedSections, strings.TrimSpace(line))
			}
		}
	}

	// Warn: large content removal
	netLoss := delLines - addLines
	if netLoss > 15 {
		warnings = append(warnings, fmt.Sprintf(ta.DiffWarnNetLoss, delLines, netLoss))
	}

	// Warn: section headings being deleted
	if len(deletedSections) > 0 {
		names := strings.Join(deletedSections, ", ")
		if len(deletedSections) == 1 {
			warnings = append(warnings, fmt.Sprintf(ta.DiffWarnSection, names))
		} else {
			warnings = append(warnings, fmt.Sprintf(ta.DiffWarnSections, len(deletedSections), names))
		}
	}

	// Warn: code blocks being removed
	var codeBlocksRemoved int
	inCodeBlock := false
	removed := h.Original
	if len(h.Lines) > 0 {
		removed = nil
		for _, dl := range h.Lines {
			if dl.Kind == '-' {
				removed = append(removed, dl.Text)
			}
		}
	}
	for _, line := range removed {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				codeBlocksRemoved++
			}
			inCodeBlock = !inCodeBlock
		}
	}
	if codeBlocksRemoved > 0 {
		warnings = append(warnings, fmt.Sprintf(ta.DiffWarnCodeBlocks, codeBlocksRemoved))
	}

	// Warn: table rows being removed
	var tableRowsRemoved int
	for _, line := range removed {
		if strings.HasPrefix(strings.TrimSpace(line), "|") && strings.Contains(line, "|") {
			tableRowsRemoved++
		}
	}
	if tableRowsRemoved > 3 {
		warnings = append(warnings, fmt.Sprintf(ta.DiffWarnTableRows, tableRowsRemoved))
	}

	return warnings
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
