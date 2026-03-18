// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
)

func makeItems(n int) []ui.ListItem {
	items := make([]ui.ListItem, n)
	for i := range items {
		items[i] = ui.ListItem{
			Type:  "decision",
			Title: "item-" + strings.Repeat("x", i%5+1),
			Date:  "2026-03-07",
		}
	}
	return items
}

func TestList_FiveItems_FormatCorrect(t *testing.T) {
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader("2\n"),
	}

	items := []ui.ListItem{
		{Type: "decision", Title: "auth-strategy", Date: "2026-03-07"},
		{Type: "feature", Title: "add-jwt", Date: "2026-03-05"},
		{Type: "bugfix", Title: "token-fix", Date: "2026-03-01"},
	}

	// Non-TTY mode (buffer-based streams) → no selection
	idx, err := ui.List(streams, items, "Select")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for non-TTY, got %d", idx)
	}

	// Items should be on stdout in non-TTY mode
	stdout := out.String()
	if !strings.Contains(stdout, "decision") {
		t.Error("expected 'decision' in stdout output")
	}
	if !strings.Contains(stdout, "auth-strategy") {
		t.Error("expected 'auth-strategy' in stdout output")
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
}

func TestList_TwentyItems_Truncation(t *testing.T) {
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	items := makeItems(20)

	_, err := ui.List(streams, items, "Select")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stdout := out.String()
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	// 15 items + 1 truncation message = 16 lines
	if len(lines) != 16 {
		t.Errorf("expected 16 lines (15 items + truncation), got %d", len(lines))
	}
	if !strings.Contains(stdout, "... and 5 more") {
		t.Error("expected truncation message '... and 5 more'")
	}
}

func TestList_ValidSelection_NonTTY(t *testing.T) {
	// Non-TTY always returns -1 (no interactive selection), but all items
	// must still appear on stdout for piping/scripting.
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader("2\n"),
	}

	items := []ui.ListItem{
		{Type: "decision", Title: "first", Date: "2026-03-07"},
		{Type: "feature", Title: "second", Date: "2026-03-05"},
		{Type: "bugfix", Title: "third", Date: "2026-03-01"},
	}

	idx, err := ui.List(streams, items, "Select")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for non-TTY, got %d", idx)
	}
	stdout := out.String()
	if !strings.Contains(stdout, "first") || !strings.Contains(stdout, "second") || !strings.Contains(stdout, "third") {
		t.Errorf("expected all items in stdout, got %q", stdout)
	}
}

func TestList_InvalidInput_NonTTY(t *testing.T) {
	// Non-TTY mode: invalid input is ignored, list is printed, returns -1.
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader("abc\n99\n2\n"),
	}

	items := []ui.ListItem{
		{Type: "decision", Title: "first", Date: "2026-03-07"},
		{Type: "feature", Title: "second", Date: "2026-03-05"},
	}

	idx, err := ui.List(streams, items, "Select")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for non-TTY, got %d", idx)
	}
	// Both items must appear
	stdout := out.String()
	if !strings.Contains(stdout, "first") || !strings.Contains(stdout, "second") {
		t.Errorf("expected all items in stdout, got %q", stdout)
	}
}

func TestList_Empty(t *testing.T) {
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  strings.NewReader(""),
	}

	idx, err := ui.List(streams, nil, "Select")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected -1 for empty list, got %d", idx)
	}
	if out.String() != "" {
		t.Errorf("expected no output for empty list, got %q", out.String())
	}
}
