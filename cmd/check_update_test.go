// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/ui"
)

func TestCheckUpdateCmd_Flags(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newCheckUpdateCmd(cfg, streams)

	if cmd.Use != "check-update" {
		t.Errorf("Use = %q, want %q", cmd.Use, "check-update")
	}

	if cmd.Args == nil {
		t.Error("expected Args validator")
	}

	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage")
	}

	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors")
	}
}

func TestUpgradeCmd_Flags(t *testing.T) {
	restore := ui.SaveAndDisableColor()
	defer restore()

	streams, _, _ := testStreams()
	cfg := &config.Config{}
	cmd := newUpgradeCmd(cfg, streams)

	if cmd.Use != "upgrade" {
		t.Errorf("Use = %q, want %q", cmd.Use, "upgrade")
	}

	versionFlag := cmd.Flag("version")
	if versionFlag == nil {
		t.Error("expected --version flag")
	}

	if !cmd.SilenceUsage {
		t.Error("expected SilenceUsage")
	}

	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors")
	}
}
