// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"github.com/greycoderk/lore/internal/config"
)

// applySynthesizerFlags reconciles the per-run --synthesizers /
// --no-synthesizers CLI flags with cfg.Angela.Synthesizers.
// declared the flags on every angela sub-command so users can opt-in or
// opt-out per-run without editing .lorerc.
//
// Precedence rules:
//
//  1. --no-synthesizers wins over everything: clears Enabled.
//  2. --synthesizers <list> when explicitly set replaces Enabled with the
//     provided list (empty list means "no synthesizers this run", same as
//     --no-synthesizers but more discoverable in scripts).
//  3. When neither flag is touched, cfg.Angela.Synthesizers.Enabled is
//     preserved as loaded from .lorerc / built-in defaults.
//
// changedSynthesizersFlag distinguishes "user passed --synthesizers" (even
// with no values) from "user did not pass it" - cobra's Flag().Changed
// gives the right answer.
func applySynthesizerFlags(cfg *config.Config, listFlag []string, noFlag bool, changedSynthesizersFlag bool) {
	if cfg == nil {
		return
	}
	if noFlag {
		cfg.Angela.Synthesizers.Enabled = nil
		return
	}
	if changedSynthesizersFlag {
		cfg.Angela.Synthesizers.Enabled = listFlag
	}
}
