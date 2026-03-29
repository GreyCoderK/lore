// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/i18n"

	"github.com/greycoderk/lore/internal/domain"
)

func TestPrintLogo_Unicode(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	PrintLogo(streams)

	output := errBuf.String()
	if !strings.Contains(output, "██") {
		t.Errorf("expected Unicode block logo, got %q", output)
	}
	if !strings.Contains(output, i18n.T().UI.Tagline) {
		t.Errorf("expected i18n.T().UI.Tagline in output, got %q", output)
	}
}

func TestPrintLogo_ASCIIFallback(t *testing.T) {
	t.Setenv("LANG", "C")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LC_ALL", "")

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	PrintLogo(streams)

	output := errBuf.String()
	if !strings.Contains(output, "+---+") {
		t.Errorf("expected ASCII logo fallback, got %q", output)
	}
	if !strings.Contains(output, i18n.T().UI.Tagline) {
		t.Errorf("expected i18n.T().UI.Tagline in output, got %q", output)
	}
}

func TestSupportsUnicode_UTF8(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	if !supportsUnicode() {
		t.Error("expected unicode support with UTF-8 LANG")
	}
}

func TestSupportsUnicode_NonUTF8(t *testing.T) {
	t.Setenv("LANG", "C")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LC_ALL", "")
	if supportsUnicode() {
		t.Error("expected no unicode support with non-UTF-8")
	}
}

func TestPickLogo_Unicode(t *testing.T) {
	t.Setenv("LANG", "en_US.UTF-8")
	logo := pickLogo()
	if !strings.Contains(logo, "██") {
		t.Errorf("expected large block logo, got %q", logo)
	}
}

func TestPickLogo_ASCII(t *testing.T) {
	t.Setenv("LANG", "C")
	t.Setenv("LC_CTYPE", "")
	t.Setenv("LC_ALL", "")
	logo := pickLogo()
	if !strings.Contains(logo, "+---+") {
		t.Errorf("expected ASCII logo, got %q", logo)
	}
}

func TestColorizeLogo_WithColor(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(true)

	result := colorizeLogo("TEST")
	if !strings.Contains(result, "\033[1;36m") {
		t.Errorf("expected cyan ANSI code, got %q", result)
	}
}

func TestColorizeLogo_NoColor(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()

	result := colorizeLogo("TEST")
	if strings.Contains(result, "\033[") {
		t.Errorf("expected no ANSI codes with color disabled, got %q", result)
	}
}
