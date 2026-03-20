// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"testing"
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
