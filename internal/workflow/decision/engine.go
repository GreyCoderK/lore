// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// SignalContext contains all inputs needed for scoring a commit.
type SignalContext struct {
	ConvType     string
	Scope        string
	Subject      string
	Message      string
	DiffContent  string
	FilesChanged []string
	LinesAdded   int
	LinesDeleted int
}

// DecisionResult holds the scoring outcome for a commit.
type DecisionResult struct {
	Score                 int
	Action                string  // ask-full, ask-reduced, suggest-skip, auto-skip
	Confidence            float64
	Signals               []SignalScore
	PrefilledWhat         string
	PrefilledWhy          string
	PrefilledWhyConfidence float64
}

// SignalScore records an individual signal's contribution.
type SignalScore struct {
	Name   string
	Input  string
	Score  int
	Reason string
}

// Engine evaluates commits using multi-signal scoring.
type Engine struct {
	store  domain.LoreStore // may be nil
	config EngineConfig
	now    clock // for testability; defaults to time.Now
}

// NewEngine creates a Decision Engine.
func NewEngine(store domain.LoreStore, config EngineConfig) *Engine {
	return &Engine{store: store, config: config, now: time.Now}
}

// SetClock overrides the time source (for testing only).
func (e *Engine) SetClock(fn func() time.Time) {
	e.now = fn
}

// Evaluate scores a commit and determines the documentation action.
func (e *Engine) Evaluate(ctx SignalContext) *DecisionResult {
	signals := make([]SignalScore, 0, 5)

	// Signal 1: Conventional Commit Type
	s1 := scoreConvType(ctx.ConvType, ctx.Message)
	signals = append(signals, s1)

	// Signal 2: Diff Size
	s2 := scoreDiffSize(ctx.LinesAdded, ctx.LinesDeleted)
	signals = append(signals, s2)

	// Signal 3: Diff Content Analysis
	s3 := ScanDiffContent(ctx.DiffContent)
	signals = append(signals, s3)

	// Signal 4: Modified Files Value
	s4 := FileValueSignal(ctx.FilesChanged)
	signals = append(signals, s4)

	// Signal 5: LKS History
	s5 := scoreLKSHistory(e.store, ctx.Scope, e.config, e.now)
	signals = append(signals, s5)

	// Sum scores
	total := 0
	for _, s := range signals {
		total += s.Score
	}

	// Clamp to 0-100
	if total < 0 {
		total = 0
	}
	if total > 100 {
		total = 100
	}

	// Confidence: based on available signals
	confidence := 1.0
	if e.store == nil {
		confidence = 0.8 // Signal 5 unavailable
	}

	// Apply overrides
	action := e.resolveAction(total, ctx.ConvType)

	// Pre-fill extraction
	what, why, whyConf := ExtractImplicitWhy(ctx.Subject)

	return &DecisionResult{
		Score:                  total,
		Action:                 action,
		Confidence:             confidence,
		Signals:                signals,
		PrefilledWhat:          what,
		PrefilledWhy:           why,
		PrefilledWhyConfidence: whyConf,
	}
}

func (e *Engine) resolveAction(score int, convType string) string {
	// Override: always_ask
	for _, t := range e.config.AlwaysAsk {
		if t == convType {
			return "ask-full"
		}
	}
	// Override: always_skip
	for _, t := range e.config.AlwaysSkip {
		if t == convType {
			return "auto-skip"
		}
	}

	// Threshold-based
	if score >= e.config.ThresholdFull {
		return "ask-full"
	}
	if score >= e.config.ThresholdReduced {
		return "ask-reduced"
	}
	if score >= e.config.ThresholdSuggest {
		return "suggest-skip"
	}
	return "auto-skip"
}
