// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/greycoderk/lore/internal/config"
)

func TestAngelaPolishCmd_NoArgs(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}

	cmd := newAngelaPolishCmd(cfg, streams)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing filename arg")
	}
}

func TestAngelaPolishCmd_Flags(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := &config.Config{}

	cmd := newAngelaPolishCmd(cfg, streams)

	if cmd.Use != "polish <filename>" {
		t.Errorf("Use = %q", cmd.Use)
	}

	dryRunFlag := cmd.Flag("dry-run")
	if dryRunFlag == nil {
		t.Error("expected --dry-run flag")
	}

	yesFlag := cmd.Flag("yes")
	if yesFlag == nil {
		t.Error("expected --yes flag")
	}
}
