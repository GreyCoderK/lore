// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"bytes"
	"strings"
	"testing"

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
	if !strings.Contains(output, "╦") {
		t.Errorf("expected Unicode logo, got %q", output)
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
