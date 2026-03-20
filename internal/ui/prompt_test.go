// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestConfirm_EnterKey(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("\n"),
	}

	err := Confirm(streams, "Press Enter to continue.")
	if err != nil {
		t.Fatalf("ui: confirm: %v", err)
	}
	if !strings.Contains(errBuf.String(), "Press Enter") {
		t.Error("expected prompt message on stderr")
	}
}

func TestPrompt_WithDefault(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("\n"),
	}

	answer, err := Prompt(streams, "Type", "feature")
	if err != nil {
		t.Fatalf("ui: prompt: %v", err)
	}
	if answer != "feature" {
		t.Errorf("expected 'feature', got %q", answer)
	}
}

func TestPrompt_WithUserInput(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("decision\n"),
	}

	answer, err := Prompt(streams, "Type", "feature")
	if err != nil {
		t.Fatalf("ui: prompt: %v", err)
	}
	if answer != "decision" {
		t.Errorf("expected 'decision', got %q", answer)
	}
}

func TestPrompt_NoDefault(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("my answer\n"),
	}

	answer, err := Prompt(streams, "Why was this approach chosen?", "")
	if err != nil {
		t.Fatalf("ui: prompt: %v", err)
	}
	if answer != "my answer" {
		t.Errorf("expected 'my answer', got %q", answer)
	}
	if !strings.Contains(errBuf.String(), "> ") {
		t.Error("expected '> ' prompt for no-default questions")
	}
}
