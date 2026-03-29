// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import "testing"

func TestEvaluate_BoundaryScores(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.AlwaysAsk = nil  // disable overrides for threshold testing
	cfg.AlwaysSkip = nil

	tests := []struct {
		name       string
		ctx        SignalContext
		wantAction string
	}{
		// feat=40 + small diff(10) + no content + no files = 50 → ask-reduced (35-59)
		{"ask-reduced", SignalContext{ConvType: "feat", LinesAdded: 10}, "ask-reduced"},
		// feat=40 + large diff(20) + no content + no files = 60 → ask-full (>=60)
		{"ask-full at 60", SignalContext{ConvType: "feat", LinesAdded: 101}, "ask-full"},
		// chore=5 + small diff(10) = 15 → suggest-skip (15-34)
		{"suggest-skip at 15", SignalContext{ConvType: "chore", LinesAdded: 5}, "suggest-skip"},
		// test=3 + trivial diff(0) = 3 → auto-skip (<15)
		{"auto-skip", SignalContext{ConvType: "test", LinesAdded: 1}, "auto-skip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine(nil, cfg)
			result := e.Evaluate(tt.ctx)
			if result.Action != tt.wantAction {
				t.Errorf("Action = %q (score %d), want %q", result.Action, result.Score, tt.wantAction)
			}
		})
	}
}

func TestEvaluate_AlwaysAskOverride(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig() // always_ask includes "feat"
	e := NewEngine(nil, cfg)

	// Even with a low-scoring feat context, always_ask forces ask-full
	result := e.Evaluate(SignalContext{ConvType: "feat", LinesAdded: 1})
	if result.Action != "ask-full" {
		t.Errorf("Action = %q, want ask-full (always_ask override)", result.Action)
	}
}

func TestEvaluate_AlwaysSkipOverride(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig() // always_skip includes "docs"
	e := NewEngine(nil, cfg)

	result := e.Evaluate(SignalContext{ConvType: "docs", LinesAdded: 200})
	if result.Action != "auto-skip" {
		t.Errorf("Action = %q, want auto-skip (always_skip override)", result.Action)
	}
}

func TestEvaluate_NilStore_ReducedConfidence(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.AlwaysAsk = nil
	cfg.AlwaysSkip = nil
	e := NewEngine(nil, cfg)

	result := e.Evaluate(SignalContext{ConvType: "feat", LinesAdded: 50})
	if result.Confidence != 0.8 {
		t.Errorf("Confidence = %f, want 0.8 (nil store)", result.Confidence)
	}
}

func TestEvaluate_SignalCount(t *testing.T) {
	t.Parallel()
	cfg := DefaultConfig()
	cfg.AlwaysAsk = nil
	cfg.AlwaysSkip = nil
	e := NewEngine(nil, cfg)

	result := e.Evaluate(SignalContext{ConvType: "feat"})
	if len(result.Signals) != 5 {
		t.Errorf("Signals count = %d, want 5", len(result.Signals))
	}

	expectedNames := map[string]bool{
		"conv-type":    false,
		"diff-size":    false,
		"diff-content": false,
		"file-value":   false,
		"lks-history":  false,
	}
	for _, sig := range result.Signals {
		if _, ok := expectedNames[sig.Name]; !ok {
			t.Errorf("unexpected signal name %q", sig.Name)
		}
		expectedNames[sig.Name] = true
	}
	for name, found := range expectedNames {
		if !found {
			t.Errorf("missing expected signal %q", name)
		}
	}
}
