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

// TestAskType_NonTTY_ValidInput tests the non-TTY text input path with a valid type.
func TestAskType_NonTTY_ValidInput(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader("bugfix\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	got, err := flow.AskType(context.Background(), "note")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "bugfix" {
		t.Errorf("AskType = %q, want bugfix", got)
	}
}

// TestAskType_NonTTY_DefaultOnEmpty tests that pressing Enter returns the default.
func TestAskType_NonTTY_DefaultOnEmpty(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader("\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	got, err := flow.AskType(context.Background(), "feature")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "feature" {
		t.Errorf("AskType = %q, want feature (default)", got)
	}
}

// TestAskType_NonTTY_InvalidThenValid tests the retry loop with an invalid type followed by a valid one.
func TestAskType_NonTTY_InvalidThenValid(t *testing.T) {
	// First line is invalid, second line is valid
	streams := domain.IOStreams{
		In:  strings.NewReader("invalid\ndecision\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	got, err := flow.AskType(context.Background(), "note")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "decision" {
		t.Errorf("AskType = %q, want decision", got)
	}
	// Verify error message was written to stderr
	stderr := streams.Err.(*bytes.Buffer).String()
	if !strings.Contains(stderr, "invalide") {
		t.Errorf("expected validation error in stderr, got: %q", stderr)
	}
}

// TestAskType_NonTTY_CaseNormalization tests that uppercase input is normalized.
func TestAskType_NonTTY_CaseNormalization(t *testing.T) {
	streams := domain.IOStreams{
		In:  strings.NewReader("REFACTOR\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	got, err := flow.AskType(context.Background(), "note")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "refactor" {
		t.Errorf("AskType = %q, want refactor", got)
	}
}

// TestAskType_NonTTY_ContextCancelled tests that cancellation returns an error.
func TestAskType_NonTTY_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streams := domain.IOStreams{
		In:  strings.NewReader("feature\n"),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	_, err := flow.AskType(ctx, "note")
	if err == nil {
		t.Fatal("expected error with cancelled context")
	}
}
