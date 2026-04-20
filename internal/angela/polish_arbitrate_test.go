// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// testStreams builds an IOStreams backed by bytes.Buffer for Out/Err
// and a string for In. Returns the streams plus handles to assert on
// rendered output.
func testStreams(stdin string) (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	return domain.IOStreams{
		In:  strings.NewReader(stdin),
		Out: &out,
		Err: &errBuf,
	}, &out, &errBuf
}

// --- ValidArbitrationRule / parsePromptInput --------------------------

func TestValidArbitrationRule(t *testing.T) {
	valid := []string{"", "first", "second", "both", "abort"}
	for _, v := range valid {
		if !ValidArbitrationRule(v) {
			t.Errorf("ValidArbitrationRule(%q) = false, want true", v)
		}
	}
	invalid := []string{"First", "BOTH", "keep-first", "yes", "no", "third"}
	for _, v := range invalid {
		if ValidArbitrationRule(v) {
			t.Errorf("ValidArbitrationRule(%q) = true, want false", v)
		}
	}
}

func TestParsePromptInput(t *testing.T) {
	cases := []struct {
		in       string
		numOccs  int
		want     ArbitrateChoice
		wantOK   bool
		wantDash bool // true iff the parser returned -1 (the [d] sentinel)
	}{
		{"1", 2, ChoiceFirst, true, false},
		{"2", 2, ChoiceSecond, true, false},
		{"b", 2, ChoiceBoth, true, false},
		{"B", 2, ChoiceBoth, true, false}, // case-insensitive
		{"a", 2, ChoiceAbort, true, false},
		{"A", 2, ChoiceAbort, true, false},
		{"d", 2, 0, true, true}, // [d] returns -1
		{"D", 2, 0, true, true},
		{"", 2, 0, false, false},
		{"3", 2, 0, false, false},
		{"yes", 2, 0, false, false},
		// Defensive: numOccurrences=1 falls back from ChoiceSecond → ChoiceFirst.
		{"2", 1, ChoiceFirst, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.in+"_"+strings.Repeat("n", tc.numOccs), func(t *testing.T) {
			got, ok := parsePromptInput(tc.in, tc.numOccs)
			if ok != tc.wantOK {
				t.Errorf("ok=%v, want %v", ok, tc.wantOK)
				return
			}
			if !ok {
				return
			}
			if tc.wantDash {
				if got != -1 {
					t.Errorf("got=%d, want -1 (the [d] sentinel)", got)
				}
				return
			}
			if got != tc.want {
				t.Errorf("got=%d, want %d", got, tc.want)
			}
		})
	}
}

// --- applyDuplicateResolutions ----------------------------------------

func TestApplyDuplicateResolutions_ChoiceFirst_KeepsFirstRemovesOthers(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n## Why\nC.\n")
	groups := detectDuplicateSections(body)
	if len(groups) != 1 {
		t.Fatalf("detect: want 1 group, got %d", len(groups))
	}
	res := []Resolution{{Heading: "## Why", Choice: ChoiceFirst}}
	got := applyDuplicateResolutions(body, groups, res)
	want := "## Why\nA.\n"
	if string(got) != want {
		t.Errorf("\n got: %q\nwant: %q", string(got), want)
	}
}

func TestApplyDuplicateResolutions_ChoiceSecond_KeepsSecondRemovesOthers(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n## Why\nC.\n")
	groups := detectDuplicateSections(body)
	res := []Resolution{{Heading: "## Why", Choice: ChoiceSecond}}
	got := applyDuplicateResolutions(body, groups, res)
	want := "## Why\nB.\n"
	if string(got) != want {
		t.Errorf("\n got: %q\nwant: %q", string(got), want)
	}
}

func TestApplyDuplicateResolutions_ChoiceBoth_PreservesAllInOrder(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n## Why\nC.\n")
	groups := detectDuplicateSections(body)
	res := []Resolution{{Heading: "## Why", Choice: ChoiceBoth}}
	got := applyDuplicateResolutions(body, groups, res)
	if string(got) != string(body) {
		t.Errorf("body mutated on ChoiceBoth:\n got: %q\nwant: %q", string(got), string(body))
	}
}

func TestApplyDuplicateResolutions_MultipleGroups_Independent(t *testing.T) {
	body := []byte(
		"## Why\nA1.\n## Context\nB1.\n## Why\nA2.\n## Context\nB2.\n## Other\nC1.\n")
	groups := detectDuplicateSections(body)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// Keep first for Why, both for Context.
	res := []Resolution{
		{Heading: "## Why", Choice: ChoiceFirst},
		{Heading: "## Context", Choice: ChoiceBoth},
	}
	got := string(applyDuplicateResolutions(body, groups, res))
	want := "## Why\nA1.\n## Context\nB1.\n## Context\nB2.\n## Other\nC1.\n"
	if got != want {
		t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

func TestApplyDuplicateResolutions_NonDuplicateSectionsUntouched(t *testing.T) {
	// The preamble before the first heading and any non-duplicated
	// section must pass through verbatim.
	body := []byte(
		"preamble line\n\n## Intro\nOnly once.\n## Why\nA.\n## Why\nB.\n## Outro\nOnce.\n")
	groups := detectDuplicateSections(body)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	res := []Resolution{{Heading: "## Why", Choice: ChoiceFirst}}
	got := string(applyDuplicateResolutions(body, groups, res))
	want := "preamble line\n\n## Intro\nOnly once.\n## Why\nA.\n## Outro\nOnce.\n"
	if got != want {
		t.Errorf("\n got: %q\nwant: %q", got, want)
	}
}

func TestApplyDuplicateResolutions_EmptyGroups_NoOp(t *testing.T) {
	body := []byte("## A\n1.\n## B\n2.\n")
	got := applyDuplicateResolutions(body, nil, nil)
	if string(got) != string(body) {
		t.Errorf("body mutated when no groups")
	}
}

// --- arbitrateDuplicates rule paths -----------------------------------

func TestArbitrateDuplicates_RuleAbort_ReturnsErrArbitrateAbort(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("")
	_, err := arbitrateDuplicates(groups, body, RuleAbort, true, streams, ArbitrateOptions{})
	if !errors.Is(err, ErrArbitrateAbort) {
		t.Errorf("err=%v, want ErrArbitrateAbort", err)
	}
}

func TestArbitrateDuplicates_RuleFirst_AppliesPerGroup(t *testing.T) {
	body := []byte("## A\nx1.\n## B\ny1.\n## A\nx2.\n## B\ny2.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("")
	got, err := arbitrateDuplicates(groups, body, RuleFirst, false, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resolutions, got %d", len(got))
	}
	for _, r := range got {
		if r.Choice != ChoiceFirst {
			t.Errorf("resolution %q Choice=%d, want ChoiceFirst", r.Heading, r.Choice)
		}
	}
}

func TestArbitrateDuplicates_RuleSecond_AppliesPerGroup(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("")
	got, err := arbitrateDuplicates(groups, body, RuleSecond, false, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 1 || got[0].Choice != ChoiceSecond {
		t.Errorf("got=%+v, want [{## Why, ChoiceSecond}]", got)
	}
}

func TestArbitrateDuplicates_RuleBoth_AppliesPerGroup(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("")
	got, err := arbitrateDuplicates(groups, body, RuleBoth, false, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 1 || got[0].Choice != ChoiceBoth {
		t.Errorf("got=%+v, want [{## Why, ChoiceBoth}]", got)
	}
}

func TestArbitrateDuplicates_NonTTYWithoutRule_Refuses(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("")
	_, err := arbitrateDuplicates(groups, body, RuleNone, false, streams, ArbitrateOptions{})
	if !errors.Is(err, ErrArbitrateRefused) {
		t.Errorf("err=%v, want ErrArbitrateRefused", err)
	}
}

func TestArbitrateDuplicates_NoGroups_Empty(t *testing.T) {
	streams, _, _ := testStreams("")
	res, err := arbitrateDuplicates(nil, nil, RuleNone, false, streams, ArbitrateOptions{})
	if err != nil {
		t.Errorf("err=%v, want nil on empty groups", err)
	}
	if len(res) != 0 {
		t.Errorf("resolutions=%+v, want empty", res)
	}
}

// --- arbitrateDuplicates TTY interactive paths ------------------------

func TestArbitrateDuplicates_TTY_UserPicksFirst(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, stderr := testStreams("1\n")
	got, err := arbitrateDuplicates(groups, body, RuleNone, true, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 1 || got[0].Choice != ChoiceFirst {
		t.Errorf("resolutions=%+v, want [ChoiceFirst]", got)
	}
	// The prompt rendering should mention the heading.
	if !strings.Contains(stderr.String(), "## Why") {
		t.Errorf("prompt did not mention the heading; stderr=\n%s", stderr.String())
	}
}

func TestArbitrateDuplicates_TTY_UserAborts(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("a\n")
	_, err := arbitrateDuplicates(groups, body, RuleNone, true, streams, ArbitrateOptions{})
	if !errors.Is(err, ErrArbitrateAbort) {
		t.Errorf("err=%v, want ErrArbitrateAbort", err)
	}
}

func TestArbitrateDuplicates_TTY_InvalidThenValid_Loops(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	streams, _, stderr := testStreams("xyz\n1\n")
	got, err := arbitrateDuplicates(groups, body, RuleNone, true, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 1 || got[0].Choice != ChoiceFirst {
		t.Errorf("resolutions=%+v, want [ChoiceFirst]", got)
	}
	if !strings.Contains(stderr.String(), "invalid choice") {
		t.Errorf("stderr should mention invalid choice:\n%s", stderr.String())
	}
}

func TestArbitrateDuplicates_TTY_ShowFullThenPick(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nBB.\n")
	groups := detectDuplicateSections(body)
	streams, _, stderr := testStreams("d\n2\n")
	got, err := arbitrateDuplicates(groups, body, RuleNone, true, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 1 || got[0].Choice != ChoiceSecond {
		t.Errorf("resolutions=%+v, want [ChoiceSecond]", got)
	}
	// The [d] action should have dumped both occurrences to stderr.
	s := stderr.String()
	if !strings.Contains(s, "occurrence 1 of 2") || !strings.Contains(s, "occurrence 2 of 2") {
		t.Errorf("expected [d] to dump full content, stderr=\n%s", s)
	}
}

func TestArbitrateDuplicates_TTY_EOFBeforeChoice_AbortsCleanly(t *testing.T) {
	body := []byte("## Why\nA.\n## Why\nB.\n")
	groups := detectDuplicateSections(body)
	// Empty stdin → ReadString returns EOF → treat as abort.
	streams, _, _ := testStreams("")
	_, err := arbitrateDuplicates(groups, body, RuleNone, true, streams, ArbitrateOptions{})
	if !errors.Is(err, ErrArbitrateAbort) {
		t.Errorf("err=%v, want ErrArbitrateAbort on EOF", err)
	}
}

func TestArbitrateDuplicates_TTY_MultipleGroups_EachPromptedIndependently(t *testing.T) {
	body := []byte("## A\nx1.\n## B\ny1.\n## A\nx2.\n## B\ny2.\n")
	groups := detectDuplicateSections(body)
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	// First group (## A) → pick 1, second group (## B) → pick b.
	streams, _, stderr := testStreams("1\nb\n")
	got, err := arbitrateDuplicates(groups, body, RuleNone, true, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resolutions, got %d", len(got))
	}
	if got[0].Heading != "## A" || got[0].Choice != ChoiceFirst {
		t.Errorf("first resolution=%+v", got[0])
	}
	if got[1].Heading != "## B" || got[1].Choice != ChoiceBoth {
		t.Errorf("second resolution=%+v", got[1])
	}
	// Prompt should have mentioned both group numbers.
	s := stderr.String()
	if !strings.Contains(s, "[group 1/2]") || !strings.Contains(s, "[group 2/2]") {
		t.Errorf("expected both group headers, stderr=\n%s", s)
	}
}

// --- end-to-end: arbitrate then apply ---------------------------------

func TestArbitrateAndApply_RuleFirst_EndToEnd(t *testing.T) {
	body := []byte("preamble\n\n## Why\nA.\n## Context\nB.\n## Why\nA2.\n## Why\nA3.\n")
	groups := detectDuplicateSections(body)
	streams, _, _ := testStreams("")
	res, err := arbitrateDuplicates(groups, body, RuleFirst, false, streams, ArbitrateOptions{})
	if err != nil {
		t.Fatalf("arbitrate: %v", err)
	}
	cleaned := string(applyDuplicateResolutions(body, groups, res))
	want := "preamble\n\n## Why\nA.\n## Context\nB.\n"
	if cleaned != want {
		t.Errorf("\n got: %q\nwant: %q", cleaned, want)
	}
}
