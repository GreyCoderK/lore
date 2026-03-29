// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package service

import (
	"fmt"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// EvaluateCommitResult bundles the decision result with commit metadata
// needed for display.
type EvaluateCommitResult struct {
	Decision   *decision.DecisionResult
	CommitInfo *domain.CommitInfo
}

// EvaluateCommit sets up the decision engine and evaluates a commit.
func EvaluateCommit(store domain.LoreStore, cfg *config.Config, ref string, adapter domain.GitAdapter) (*EvaluateCommitResult, error) {
	commitInfo, err := adapter.Log(ref)
	if err != nil {
		return nil, fmt.Errorf("service: decision: log %s: %w", ref, err)
	}

	diffContent, diffErr := adapter.Diff(ref)
	if diffErr != nil {
		// Non-fatal: proceed without diff
		diffContent = ""
	}
	filesChanged := workflow.ExtractFilesFromDiff(diffContent)
	linesAdded, linesDeleted := workflow.CountDiffLines(diffContent)

	engineCfg := EngineConfigFromApp(cfg)

	engine := decision.NewEngine(store, engineCfg)
	ctx := decision.SignalContext{
		ConvType:     commitInfo.Type,
		Scope:        commitInfo.Scope,
		Subject:      commitInfo.Subject,
		Message:      commitInfo.Message,
		DiffContent:  diffContent,
		FilesChanged: filesChanged,
		LinesAdded:   linesAdded,
		LinesDeleted: linesDeleted,
	}

	result := engine.Evaluate(ctx)

	return &EvaluateCommitResult{
		Decision:   result,
		CommitInfo: commitInfo,
	}, nil
}

// EngineConfigFromApp builds an EngineConfig from the app config.
// Falls back to DefaultConfig() if all thresholds are zero. This happens when
// PersistentPreRunE is skipped (e.g. hook subprocess) and Viper defaults were
// not applied. A user intentionally setting all thresholds to 0 is not a
// supported use case — Viper defaults (60/35/15) always apply when config loads.
func EngineConfigFromApp(cfg *config.Config) decision.EngineConfig {
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
