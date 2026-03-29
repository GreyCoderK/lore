// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

func TestEngineConfigFromApp_WithValues(t *testing.T) {
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:      70,
			ThresholdReduced:   40,
			ThresholdSuggest:   20,
			AlwaysAsk:          []string{"feat"},
			AlwaysSkip:         []string{"ci"},
			Learning:           true,
			LearningMinCommits: 30,
		},
	}
	ec := engineConfigFromApp(cfg)
	if ec.ThresholdFull != 70 {
		t.Errorf("ThresholdFull = %d, want 70", ec.ThresholdFull)
	}
	if ec.ThresholdReduced != 40 {
		t.Errorf("ThresholdReduced = %d, want 40", ec.ThresholdReduced)
	}
	if ec.LearningMinCommits != 30 {
		t.Errorf("LearningMinCommits = %d, want 30", ec.LearningMinCommits)
	}
}

func TestEngineConfigFromApp_ZeroThresholds_FallsBackToDefault(t *testing.T) {
	cfg := &config.Config{
		Decision: config.DecisionConfig{
			ThresholdFull:    0,
			ThresholdReduced: 0,
			ThresholdSuggest: 0,
		},
	}
	ec := engineConfigFromApp(cfg)
	def := decision.DefaultConfig()
	if ec.ThresholdFull != def.ThresholdFull {
		t.Errorf("ThresholdFull = %d, want default %d", ec.ThresholdFull, def.ThresholdFull)
	}
	if len(ec.AlwaysAsk) != len(def.AlwaysAsk) {
		t.Errorf("AlwaysAsk len = %d, want default %d", len(ec.AlwaysAsk), len(def.AlwaysAsk))
	}
}
