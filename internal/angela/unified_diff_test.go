// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/ui"
)

// TestUnifiedDiffString_BasicShape checks the minimum viable output: the
// standard `---` / `+++` headers, a `@@` hunk marker, and +/- lines for the
// two differing rows. Anything beyond that is difflib's job to get right.
func TestUnifiedDiffString_BasicShape(t *testing.T) {
	defer ui.SaveAndDisableColor()()
	orig := "alpha\nbeta\ngamma\n"
	mod := "alpha\nBETA\ngamma\n"

	got, err := UnifiedDiffString(orig, mod, UnifiedDiffOptions{
		FromFile: "original",
		ToFile:   "modified",
		Context:  3,
	})
	if err != nil {
		t.Fatalf("UnifiedDiffString: %v", err)
	}
	for _, need := range []string{"--- original", "+++ modified", "@@", "-beta", "+BETA"} {
		if !strings.Contains(got, need) {
			t.Errorf("diff missing %q:\n%s", need, got)
		}
	}
}

// TestUnifiedDiffString_EmptyWhenIdentical: difflib returns an empty string
// for equal inputs and the dry-run command relies on that signal to print
// "no changes" instead of a blank diff.
func TestUnifiedDiffString_EmptyWhenIdentical(t *testing.T) {
	defer ui.SaveAndDisableColor()()
	got, err := UnifiedDiffString("same\nlines\n", "same\nlines\n", UnifiedDiffOptions{
		FromFile: "a", ToFile: "b",
	})
	if err != nil {
		t.Fatalf("UnifiedDiffString: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty diff for identical inputs, got %q", got)
	}
}

// TestUnifiedDiffString_DefaultContextThree: the story AC-9 pins context at
// 3 lines. Passing 0 (zero value) must still produce 3 lines of context so
// callers don't have to remember the default.
func TestUnifiedDiffString_DefaultContextThree(t *testing.T) {
	defer ui.SaveAndDisableColor()()
	// Six-line document with one change in the middle.
	orig := "l1\nl2\nl3\nl4\nl5\nl6\nl7\n"
	mod := "l1\nl2\nl3\nL4\nl5\nl6\nl7\n"

	got, err := UnifiedDiffString(orig, mod, UnifiedDiffOptions{Context: 0})
	if err != nil {
		t.Fatalf("UnifiedDiffString: %v", err)
	}
	// With context=3 the hunk should span the whole 7-line doc.
	for _, need := range []string{" l1", " l2", " l3", "-l4", "+L4", " l5", " l6", " l7"} {
		if !strings.Contains(got, need) {
			t.Errorf("expected context line %q:\n%s", need, got)
		}
	}
}

// TestUnifiedDiffString_ColoredNoColor confirms the Colored flag is a no-op
// when ui.SetColorEnabled is false — this is how the dry-run flow behaves
// when stderr is not a TTY (NO_COLOR or pipe).
func TestUnifiedDiffString_ColoredNoColor(t *testing.T) {
	defer ui.SaveAndDisableColor()()
	plain, err := UnifiedDiffString("x\n", "y\n", UnifiedDiffOptions{Colored: false})
	if err != nil {
		t.Fatal(err)
	}
	colored, err := UnifiedDiffString("x\n", "y\n", UnifiedDiffOptions{Colored: true})
	if err != nil {
		t.Fatal(err)
	}
	// Colors are disabled globally, so the two calls should be byte-equal.
	if plain != colored {
		t.Errorf("with color disabled, Colored=true should match Colored=false\nplain=%q\ncolored=%q", plain, colored)
	}
	// And neither should contain any ANSI escape.
	if strings.Contains(plain, "\x1b[") {
		t.Errorf("plain output contains ANSI escape: %q", plain)
	}
}

// TestUnifiedDiffString_ColoredWithColor: when color is enabled, +/- lines
// must carry ANSI markers but the header lines must stay bare — otherwise
// the diff is unparseable by downstream tools.
func TestUnifiedDiffString_ColoredWithColor(t *testing.T) {
	ui.SetColorEnabled(true)
	defer ui.SetColorEnabled(false)

	got, err := UnifiedDiffString("keep\nold\n", "keep\nnew\n", UnifiedDiffOptions{
		FromFile: "a", ToFile: "b", Colored: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "\x1b[32m+new") {
		t.Errorf("expected green '+' line, got %q", got)
	}
	if !strings.Contains(got, "\x1b[31m-old") {
		t.Errorf("expected red '-' line, got %q", got)
	}
	if strings.Contains(got, "\x1b[32m+++") || strings.Contains(got, "\x1b[31m---") {
		t.Errorf("headers must remain uncolored, got %q", got)
	}
}
