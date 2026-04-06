// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestVerbAlignment(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	Verb(streams, "Captured", "decision-auth.md")
	got := buf.String()
	expected := "  Captured decision-auth.md\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestVerbWithColor(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	Verb(streams, "Done", "file.md")
	got := buf.String()
	if len(got) <= len("      Done file.md\n") {
		t.Errorf("expected colored output to be longer than plain, got %q", got)
	}
}

func TestVerbDelete(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	VerbDelete(streams, "old-doc.md")
	got := buf.String()
	if !strings.Contains(got, "old-doc.md") {
		t.Errorf("expected message in output, got %q", got)
	}
}

func TestVerbDeleteWithColor(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	VerbDelete(streams, "removed.md")
	got := buf.String()
	if !strings.Contains(got, "removed.md") {
		t.Errorf("expected message in output, got %q", got)
	}
}

func TestPrintLogo_ColorDisabled(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()

	t.Setenv("LANG", "en_US.UTF-8")

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	PrintLogo(streams)

	output := errBuf.String()
	// Logo should be present but without ANSI color codes
	if !strings.Contains(output, "██") {
		t.Errorf("expected Unicode logo in output, got %q", output)
	}
	if strings.Contains(output, "\033[1;36m") {
		t.Errorf("expected no cyan ANSI code with color disabled, got %q", output)
	}
}

func TestColorizeLogo_Passthrough(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()

	input := "LOGO TEXT"
	result := colorizeLogo(input)
	if result != input {
		t.Errorf("colorizeLogo with color disabled should return input unchanged, got %q", result)
	}
}
