// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// --- Additional validateType edge cases ---

func TestValidateType_WhitespaceCaseCombination(t *testing.T) {
	// Validates that both whitespace trimming and case normalization happen together.
	cases := []struct {
		input string
		want  string
	}{
		{"  RELEASE  ", "release"},
		{"\tDECISION\n", "decision"},
		{"  Summary ", "summary"},
		{" \t BUGFIX \t ", "bugfix"},
	}
	for _, tc := range cases {
		t.Run(tc.want+"_combo", func(t *testing.T) {
			got, ok := validateType(tc.input)
			if !ok {
				t.Errorf("validateType(%q) = (_, false), want (_, true)", tc.input)
			}
			if got != tc.want {
				t.Errorf("validateType(%q) = (%q, _), want (%q, _)", tc.input, got, tc.want)
			}
		})
	}
}

func TestValidateType_EmptyAndBlank(t *testing.T) {
	blanks := []string{"", " ", "\t", "\n", "  \t  "}
	for _, b := range blanks {
		_, ok := validateType(b)
		if ok {
			t.Errorf("validateType(%q) = (_, true), want (_, false)", b)
		}
	}
}

// --- Additional AskType non-TTY edge cases ---

func TestAskType_NonTTY_AllSevenTypes(t *testing.T) {
	validTypes := []string{"decision", "feature", "bugfix", "refactor", "release", "note", "summary"}
	for _, typ := range validTypes {
		t.Run(typ, func(t *testing.T) {
			streams := domain.IOStreams{
				In:  strings.NewReader(typ + "\n"),
				Out: &bytes.Buffer{},
				Err: &bytes.Buffer{},
			}
			flow := NewQuestionFlow(streams, NewRenderer(streams))
			got, err := flow.AskType(context.Background(), "note")
			if err != nil {
				t.Fatalf("AskType(%q): %v", typ, err)
			}
			if got != typ {
				t.Errorf("AskType = %q, want %q", got, typ)
			}
		})
	}
}

func TestAskType_NonTTY_WhitespaceInput(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader("  feature  \n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	flow := NewQuestionFlow(streams, NewRenderer(streams))
	got, err := flow.AskType(context.Background(), "note")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	// askWithDefault trims space, then validateType also trims+lowercases
	if got != "feature" {
		t.Errorf("AskType = %q, want %q", got, "feature")
	}
}

func TestAskType_NonTTY_MultipleInvalidThenValid(t *testing.T) {
	// Three invalid types, then a valid one
	streams := domain.IOStreams{
		In:  strings.NewReader("foo\nbar\nbaz\nsummary\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	flow := NewQuestionFlow(streams, NewRenderer(streams))
	got, err := flow.AskType(context.Background(), "note")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "summary" {
		t.Errorf("AskType = %q, want %q", got, "summary")
	}
	stderr := streams.Err.(*bytes.Buffer).String()
	// Each invalid attempt should produce an error message
	if !strings.Contains(stderr, "foo") {
		t.Errorf("expected error for 'foo' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "bar") {
		t.Errorf("expected error for 'bar' in stderr, got: %q", stderr)
	}
	if !strings.Contains(stderr, "baz") {
		t.Errorf("expected error for 'baz' in stderr, got: %q", stderr)
	}
}
