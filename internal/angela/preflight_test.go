// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// ---------------------------------------------------------------------------
// estimateTokens
// ---------------------------------------------------------------------------

func TestEstimateTokens_Empty(t *testing.T) {
	got := estimateTokens("")
	if got != 1 {
		t.Errorf("estimateTokens(\"\") = %d, want 1", got)
	}
}

func TestEstimateTokens_HelloWorld(t *testing.T) {
	got := estimateTokens("hello world")
	// 11 runes → 11*10/35+1 = 4. Just sanity-check it's small.
	if got < 1 || got > 20 {
		t.Errorf("estimateTokens(\"hello world\") = %d, want small number (1-20)", got)
	}
}

func TestEstimateTokens_ProportionalToLength(t *testing.T) {
	short := strings.Repeat("a", 100)
	long := strings.Repeat("a", 1000)
	tokShort := estimateTokens(short)
	tokLong := estimateTokens(long)
	// Long text should produce roughly 10x more tokens.
	if tokLong < tokShort*5 {
		t.Errorf("long text tokens (%d) should be much larger than short (%d)", tokLong, tokShort)
	}
}

// ---------------------------------------------------------------------------
// Preflight
// ---------------------------------------------------------------------------

func TestPreflight_NormalCase(t *testing.T) {
	doc := strings.Repeat("word ", 200)  // ~200 words, ~1000 chars
	sys := "You are a helpful assistant."
	r := Preflight(doc, sys, "claude-sonnet-4-20250514", 8192, 5*time.Minute)
	if r.ShouldAbort {
		t.Errorf("normal case should not abort, got AbortReason: %s", r.AbortReason)
	}
	if len(r.Warnings) != 0 {
		t.Errorf("normal case should have no warnings, got %v", r.Warnings)
	}
}

func TestPreflight_InputExceedsMaxOutput_Aborts(t *testing.T) {
	// Create a large doc whose estimated tokens exceed a tiny maxOutput.
	doc := strings.Repeat("word ", 5000) // ~25000 chars → ~7143 tokens
	r := Preflight(doc, "", "claude-sonnet-4-20250514", 100, 5*time.Minute)
	if !r.ShouldAbort {
		t.Error("expected ShouldAbort=true when input tokens > maxOutputTokens")
	}
	if r.AbortReason == "" {
		t.Error("expected non-empty AbortReason")
	}
}

func TestPreflight_KnownModel_HasCost(t *testing.T) {
	doc := "Some document text."
	r := Preflight(doc, "System prompt.", "claude-sonnet-4-20250514", 4096, 5*time.Minute)
	if r.EstimatedCost < 0 {
		t.Errorf("known model should have EstimatedCost >= 0, got %f", r.EstimatedCost)
	}
}

func TestPreflight_UnknownModel_CostNegative(t *testing.T) {
	doc := "Some document text."
	r := Preflight(doc, "System prompt.", "unknown-model-xyz", 4096, 5*time.Minute)
	if r.EstimatedCost != -1 {
		t.Errorf("unknown model should have EstimatedCost = -1, got %f", r.EstimatedCost)
	}
}

func TestPreflight_TightTimeout_Warning(t *testing.T) {
	// claude-sonnet-4-20250514 speed is 80 tok/s.
	// A ~12K char doc → ~3400 input tokens → expectedOutput = 3400*2+500 = 7300.
	// estimatedSeconds = 7300/80 = 91.25s → threshold = 90*0.9 = 81 → 91.25 > 81 → warn.
	doc := strings.Repeat("A long document with substantial content. ", 300) // ~12600 chars → ~3600 tokens
	r := Preflight(doc, "", "claude-sonnet-4-20250514", 16000, 90*time.Second)
	found := false
	for _, w := range r.Warnings {
		if strings.Contains(strings.ToLower(w), "timeout") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected timeout warning for large doc with tight timeout, got warnings: %v", r.Warnings)
	}
}

func TestPreflight_SmallDocNoTimeoutWarning(t *testing.T) {
	// A small doc (~10 chars → ~3 tokens) with generous timeout should
	// NOT trigger a timeout warning, even when max_tokens is large.
	// This was the false-positive bug that prompted the fix.
	doc := "Short doc."
	r := Preflight(doc, "", "claude-opus-4-6", 16000, 280*time.Second)
	for _, w := range r.Warnings {
		if strings.Contains(strings.ToLower(w), "timeout") {
			t.Errorf("small doc with generous timeout should NOT warn, got: %s", w)
		}
	}
}

// ---------------------------------------------------------------------------
// EstimateCost
// ---------------------------------------------------------------------------

func TestEstimateCost_KnownModel(t *testing.T) {
	cost := EstimateCost("claude-sonnet-4-20250514", 1000, 1000)
	if cost <= 0 {
		t.Errorf("known model cost should be positive, got %f", cost)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	cost := EstimateCost("totally-unknown-model", 1000, 1000)
	if cost != -1 {
		t.Errorf("unknown model cost should be -1, got %f", cost)
	}
}

// ---------------------------------------------------------------------------
// AnalyzeUsage
// ---------------------------------------------------------------------------

func TestAnalyzeUsage_FastSpeed(t *testing.T) {
	usage := &domain.AIUsage{
		InputTokens:  1000,
		OutputTokens: 2000,
		Model:        "claude-sonnet-4-20250514",
	}
	// 2000 tokens in 10s = 200 tok/s → fast
	a := AnalyzeUsage(usage, 10*time.Second, 8192)
	found := false
	for _, line := range a.Lines {
		if strings.Contains(strings.ToLower(line), "fast") || strings.Contains(line, "200") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'fast' line for >100 tok/s, got: %v", a.Lines)
	}
}

func TestAnalyzeUsage_SlowSpeed(t *testing.T) {
	usage := &domain.AIUsage{
		InputTokens:  1000,
		OutputTokens: 50,
		Model:        "claude-sonnet-4-20250514",
	}
	// 50 tokens in 10s = 5 tok/s → slow
	a := AnalyzeUsage(usage, 10*time.Second, 8192)
	found := false
	for _, line := range a.Lines {
		if strings.Contains(strings.ToLower(line), "slow") || strings.Contains(line, "5.0") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'slow' line for <10 tok/s, got: %v", a.Lines)
	}
}

func TestAnalyzeUsage_Truncation(t *testing.T) {
	maxTokens := 4096
	usage := &domain.AIUsage{
		InputTokens:  1000,
		OutputTokens: maxTokens - 5, // within 10 of max → truncation
		Model:        "claude-sonnet-4-20250514",
	}
	a := AnalyzeUsage(usage, 30*time.Second, maxTokens)
	found := false
	for _, line := range a.Lines {
		if strings.Contains(strings.ToLower(line), "trunc") || strings.Contains(line, "4091") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected truncation warning when output near max, got: %v", a.Lines)
	}
}

func TestAnalyzeUsage_NilUsage(t *testing.T) {
	a := AnalyzeUsage(nil, 5*time.Second, 4096)
	if len(a.Lines) != 0 {
		t.Errorf("nil usage should produce empty lines, got %v", a.Lines)
	}
}
