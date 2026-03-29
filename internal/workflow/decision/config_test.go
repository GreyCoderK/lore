// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import "testing"

func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ThresholdFull != 60 {
		t.Errorf("ThresholdFull = %d, want 60", cfg.ThresholdFull)
	}
	if cfg.ThresholdReduced != 35 {
		t.Errorf("ThresholdReduced = %d, want 35", cfg.ThresholdReduced)
	}
	if cfg.ThresholdSuggest != 15 {
		t.Errorf("ThresholdSuggest = %d, want 15", cfg.ThresholdSuggest)
	}
	if len(cfg.AlwaysAsk) != 2 || cfg.AlwaysAsk[0] != "feat" {
		t.Errorf("AlwaysAsk = %v, want [feat breaking]", cfg.AlwaysAsk)
	}
	if len(cfg.AlwaysSkip) != 4 {
		t.Errorf("AlwaysSkip = %v, want 4 items", cfg.AlwaysSkip)
	}
	if !cfg.Learning {
		t.Error("Learning should be true by default")
	}
	if cfg.LearningMinCommits != 20 {
		t.Errorf("LearningMinCommits = %d, want 20", cfg.LearningMinCommits)
	}
}

func TestEvaluate_AlwaysAskForces_AskFull(t *testing.T) {
	cfg := DefaultConfig()
	e := NewEngine(nil, cfg)

	// feat is in always_ask — should force ask-full even with zero diff
	result := e.Evaluate(SignalContext{ConvType: "feat"})
	if result.Action != "ask-full" {
		t.Errorf("always_ask feat: Action = %q, want ask-full", result.Action)
	}
}

func TestEvaluate_AlwaysSkipForces_AutoSkip(t *testing.T) {
	cfg := DefaultConfig()
	e := NewEngine(nil, cfg)

	// docs is in always_skip — should force auto-skip even with large diff
	result := e.Evaluate(SignalContext{ConvType: "docs", LinesAdded: 500})
	if result.Action != "auto-skip" {
		t.Errorf("always_skip docs: Action = %q, want auto-skip", result.Action)
	}
}

func TestEvaluate_CustomThresholds(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AlwaysAsk = nil
	cfg.AlwaysSkip = nil
	cfg.ThresholdFull = 80
	cfg.ThresholdReduced = 50
	cfg.ThresholdSuggest = 20

	e := NewEngine(nil, cfg)

	// feat(40) + large diff(20) = 60 → with threshold_full=80, this is ask-reduced (50-79)
	result := e.Evaluate(SignalContext{ConvType: "feat", LinesAdded: 101})
	if result.Action != "ask-reduced" {
		t.Errorf("custom threshold: Action = %q (score %d), want ask-reduced", result.Action, result.Score)
	}
}
