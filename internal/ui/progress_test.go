// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestProgress_Display(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	Progress(streams, 2, 5, "Processing")

	output := errBuf.String()
	if !strings.Contains(output, "##") {
		t.Error("expected filled portion")
	}
	if !strings.Contains(output, "2/5") {
		t.Error("expected counter")
	}
	if !strings.Contains(output, "Processing") {
		t.Error("expected label")
	}
}

func TestProgress_Complete(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	Progress(streams, 3, 3, "Done")

	output := errBuf.String()
	if !strings.Contains(output, "###") {
		t.Error("expected fully filled bar")
	}
	if strings.Contains(output, "·") {
		t.Error("expected no unfilled portion")
	}
}

func TestProgress_NegativeCurrent(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	Progress(streams, -1, 5, "Neg")
	output := errBuf.String()
	if !strings.Contains(output, "0/5") {
		t.Errorf("expected 0/5 for negative current, got %q", output)
	}
}

func TestProgress_CurrentExceedsTotal(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	Progress(streams, 10, 5, "Over")
	output := errBuf.String()
	if !strings.Contains(output, "5/5") {
		t.Errorf("expected 5/5 for current > total, got %q", output)
	}
}

func TestProgress_ZeroTotal(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	Progress(streams, 0, 0, "Empty")
	output := errBuf.String()
	if !strings.Contains(output, "0/0") {
		t.Errorf("expected 0/0, got %q", output)
	}
}

func TestProgress_NegativeTotal(t *testing.T) {
	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}

	Progress(streams, 1, -3, "NegTotal")
	output := errBuf.String()
	if !strings.Contains(output, "0/0") {
		t.Errorf("expected 0/0 for negative total, got %q", output)
	}
}
