// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
)

// ─────────────────────────────────────────────────────────────────
// ApplyDiff — DiffBoth and hunk-not-found paths
// ─────────────────────────────────────────────────────────────────

func TestApplyDiff_DiffBoth_NoLines(t *testing.T) {
	// DiffBoth with no Lines field → falls back to appending Modified.
	hunks := []DiffHunk{
		{
			Original: []string{"original"},
			Modified: []string{"modified"},
			Lines:    nil, // no detailed edit ops
		},
	}
	result := ApplyDiff("original\n", hunks, []DiffChoice{DiffBoth})
	// DiffBoth with nil Lines appends Modified after Original.
	if !strings.Contains(result, "original") {
		t.Errorf("DiffBoth should keep original, got: %q", result)
	}
	if !strings.Contains(result, "modified") {
		t.Errorf("DiffBoth should append modified, got: %q", result)
	}
}

func TestApplyDiff_DiffBoth_WithLines(t *testing.T) {
	// DiffBoth with Lines → only genuine additions from Modified are appended.
	hunks := []DiffHunk{
		{
			Original: []string{"line1"},
			Modified: []string{"line1", "added-line"},
			Lines: []DiffLine{
				{Kind: '=', Text: "line1"},
				{Kind: '+', Text: "added-line"},
			},
		},
	}
	result := ApplyDiff("line1\n", hunks, []DiffChoice{DiffBoth})
	if !strings.Contains(result, "line1") {
		t.Errorf("DiffBoth should keep original, got: %q", result)
	}
	if !strings.Contains(result, "added-line") {
		t.Errorf("DiffBoth should append added lines, got: %q", result)
	}
}

func TestApplyDiff_HunkNotFound_SkipsSafely(t *testing.T) {
	// A hunk whose Original doesn't exist in the document should be skipped.
	hunks := []DiffHunk{
		{
			Original: []string{"this-line-does-not-exist"},
			Modified: []string{"replacement"},
		},
	}
	original := "actual content\n"
	result := ApplyDiff(original, hunks, []DiffChoice{DiffAccept})
	// The original content should remain unchanged.
	if !strings.Contains(result, "actual content") {
		t.Errorf("unmatched hunk should leave content unchanged, got: %q", result)
	}
	if strings.Contains(result, "replacement") {
		t.Errorf("unmatched hunk should not inject replacement, got: %q", result)
	}
}

func TestApplyDiff_EmptyChoices_AllRejected(t *testing.T) {
	// No hunks, no choices → original preserved.
	result := ApplyDiff("content\n", nil, nil)
	// splitLines("content\n") → ["content"], joined back → "content\n" or "content"
	if !strings.HasPrefix(result, "content") {
		t.Errorf("no hunks should return original, got: %q", result)
	}
}
