// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// engineConfigFromApp builds an EngineConfig from the app config.
// Falls back to DefaultConfig() if all thresholds are zero. This happens when
// PersistentPreRunE is skipped (e.g. hook subprocess) and Viper defaults were
// not applied. A user intentionally setting all thresholds to 0 is not a
// supported use case — Viper defaults (60/35/15) always apply when config loads.
func engineConfigFromApp(cfg *config.Config) decision.EngineConfig {
	ec := decision.EngineConfig{
		ThresholdFull:      cfg.Decision.ThresholdFull,
		ThresholdReduced:   cfg.Decision.ThresholdReduced,
		ThresholdSuggest:   cfg.Decision.ThresholdSuggest,
		AlwaysAsk:          cfg.Decision.AlwaysAsk,
		AlwaysSkip:         cfg.Decision.AlwaysSkip,
		CriticalScopes:     cfg.Decision.CriticalScopes,
		Learning:           cfg.Decision.Learning,
		LearningMinCommits: cfg.Decision.LearningMinCommits,
	}
	if ec.ThresholdFull == 0 && ec.ThresholdReduced == 0 && ec.ThresholdSuggest == 0 {
		return decision.DefaultConfig()
	}
	return ec
}
