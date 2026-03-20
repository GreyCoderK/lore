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
