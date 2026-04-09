// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func testStreams() domain.IOStreams {
	return domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		In:  strings.NewReader(""),
	}
}

func TestIsTerminalWithBuffers(t *testing.T) {
	streams := testStreams()
	if IsTerminal(streams) {
		t.Error("expected IsTerminal=false for non-file streams")
	}
}

func TestIsTerminalWithDumbTerm(t *testing.T) {
	t.Setenv("TERM", "dumb")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	if IsTerminal(streams) {
		t.Error("expected IsTerminal=false when TERM=dumb")
	}
}

func TestIsTerminalWithLineMode(t *testing.T) {
	t.Setenv("LORE_LINE_MODE", "1")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	if IsTerminal(streams) {
		t.Error("expected IsTerminal=false when LORE_LINE_MODE=1")
	}
}

func TestColorEnabledWithBuffers(t *testing.T) {
	streams := testStreams()
	if ColorEnabled(streams) {
		t.Error("expected ColorEnabled=false for non-terminal streams")
	}
}

func TestColorEnabledWithNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	if ColorEnabled(streams) {
		t.Error("expected ColorEnabled=false when NO_COLOR is set")
	}
}

func TestColorEnabledWithDumbTerm(t *testing.T) {
	t.Setenv("TERM", "dumb")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	// TERM=dumb causes IsTerminal to return false, so ColorEnabled should be false
	if ColorEnabled(streams) {
		t.Error("expected ColorEnabled=false when TERM=dumb")
	}
}

func TestColorEnabledWithBuffers_NoPanic(t *testing.T) {
	// Ensure calling ColorEnabled with buffer-based streams does not panic
	streams := testStreams()
	result := ColorEnabled(streams)
	if result {
		t.Error("expected ColorEnabled=false for buffer-based streams")
	}
}

func TestColorEnabledWithNoColorEmpty(t *testing.T) {
	// NO_COLOR spec: presence of the variable matters, even if empty
	t.Setenv("NO_COLOR", "")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	if ColorEnabled(streams) {
		t.Error("expected ColorEnabled=false when NO_COLOR is set (even empty)")
	}
}
