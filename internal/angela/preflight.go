// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// PreflightResult contains warnings and estimates before an API call.
type PreflightResult struct {
	EstimatedInputTokens int
	MaxOutputTokens      int
	Timeout              time.Duration
	EstimatedCost        float64 // -1 if unknown
	Warnings             []string
	ShouldAbort          bool   // true if the call will certainly fail
	AbortReason          string // human-readable reason for abort
}

// modelContextLimits maps known models to their max context window size.
var modelContextLimits = map[string]int{
	"claude-sonnet-4-20250514":  200000,
	"claude-haiku-4-5-20251001": 200000,
	"claude-opus-4-6":           200000,
	"gpt-4o":                    128000,
	"gpt-4o-mini":               128000,
	"gpt-4.1":                   128000,
	"gpt-4.1-mini":              128000,
	"llama3.2":                  8192,
	"llama3.1:8b":               8192,
	"llama3.1:70b":              8192,
	"mistral":                   32768,
	"codellama":                 16384,
	"gemma2":                    8192,
}

// modelSpeedTokPerSec maps known models to approximate output tokens/second.
var modelSpeedTokPerSec = map[string]float64{
	"claude-sonnet-4-20250514":  80,
	"claude-haiku-4-5-20251001": 150,
	"claude-opus-4-6":           60,
	"gpt-4o":                    90,
	"gpt-4o-mini":               130,
	"gpt-4.1":                   90,
	"gpt-4.1-mini":              130,
	"llama3.2":                  30,
	"llama3.1:8b":               25,
	"llama3.1:70b":              8,
	"mistral":                   35,
}

// estimateTokens gives a rough token count from text (~1 token per 3.5 chars).
func estimateTokens(text string) int {
	// Use rune count for accurate multi-byte character handling
	runes := len([]rune(text))
	return runes*10/35 + 1
}

// Preflight checks if the planned API call is likely to succeed.
// Returns warnings, cost estimate, and whether to abort.
func Preflight(doc string, systemPrompt string, model string, maxOutputTokens int, timeout time.Duration) *PreflightResult {
	t := i18n.T().Angela
	r := &PreflightResult{
		MaxOutputTokens: maxOutputTokens,
		Timeout:         timeout,
		EstimatedCost:   -1,
	}

	inputTokens := estimateTokens(doc) + estimateTokens(systemPrompt)
	r.EstimatedInputTokens = inputTokens

	// CRITICAL: if estimated input > max_output, the AI cannot return the full document
	if inputTokens > maxOutputTokens {
		suggested := inputTokens*2 + 500 // generous margin
		r.ShouldAbort = true
		r.AbortReason = fmt.Sprintf(t.UIInputExceedsMax, inputTokens, maxOutputTokens)
		r.Warnings = append(r.Warnings, fmt.Sprintf(t.UIInputExceedsHint, suggested))
		return r // no point checking further
	}

	// Check context window
	if ctxLimit, ok := modelContextLimits[model]; ok {
		totalNeeded := inputTokens + maxOutputTokens
		if totalNeeded > ctxLimit {
			r.ShouldAbort = true
			r.AbortReason = fmt.Sprintf(t.UIContextWarning, model, totalNeeded/1000, ctxLimit/1000)
		} else if float64(totalNeeded) > float64(ctxLimit)*0.85 {
			r.Warnings = append(r.Warnings, fmt.Sprintf(
				t.UIContextClose, totalNeeded*100/ctxLimit, model, totalNeeded/1000, ctxLimit/1000))
		}
	}

	// Check timeout vs expected generation time
	if speed, ok := modelSpeedTokPerSec[model]; ok {
		estimatedSeconds := float64(maxOutputTokens) / speed
		if timeout > 0 && estimatedSeconds > timeout.Seconds()*0.8 {
			r.Warnings = append(r.Warnings, fmt.Sprintf(
				t.UITimeoutWarning, timeout, model, speed, estimatedSeconds, maxOutputTokens))
		}
	}

	// Estimate cost
	r.EstimatedCost = EstimateCost(model, inputTokens, maxOutputTokens)

	return r
}

// PostCallAnalysis provides feedback after an API call completes.
type PostCallAnalysis struct {
	Lines []string // feedback lines to display
}

// modelCostPer1kInput maps models to cost per 1k input tokens (USD).
var modelCostPer1kInput = map[string]float64{
	"claude-sonnet-4-20250514":  0.003,
	"claude-haiku-4-5-20251001": 0.0008,
	"claude-opus-4-6":           0.015,
	"gpt-4o":                    0.0025,
	"gpt-4o-mini":               0.00015,
	"gpt-4.1":                   0.002,
	"gpt-4.1-mini":              0.0004,
}

// modelCostPer1kOutput maps models to cost per 1k output tokens (USD).
var modelCostPer1kOutput = map[string]float64{
	"claude-sonnet-4-20250514":  0.015,
	"claude-haiku-4-5-20251001": 0.004,
	"claude-opus-4-6":           0.075,
	"gpt-4o":                    0.01,
	"gpt-4o-mini":               0.0006,
	"gpt-4.1":                   0.008,
	"gpt-4.1-mini":              0.0016,
}

// EstimateCost returns the estimated cost in USD for an API call. Returns -1 if unknown.
func EstimateCost(model string, inputTokens, outputTokens int) float64 {
	inCost, hasIn := modelCostPer1kInput[model]
	outCost, hasOut := modelCostPer1kOutput[model]
	if !hasIn || !hasOut {
		return -1
	}
	return float64(inputTokens)/1000*inCost + float64(outputTokens)/1000*outCost
}

// AnalyzeUsage produces human-friendly feedback about token consumption.
func AnalyzeUsage(usage *domain.AIUsage, elapsed time.Duration, maxOutputTokens int) *PostCallAnalysis {
	if usage == nil {
		return &PostCallAnalysis{}
	}

	t := i18n.T().Angela
	a := &PostCallAnalysis{}
	model := usage.Model

	// Speed
	if elapsed.Seconds() > 0 && usage.OutputTokens > 0 {
		speed := float64(usage.OutputTokens) / elapsed.Seconds()
		if speed < 10 {
			a.Lines = append(a.Lines, fmt.Sprintf(t.UISpeedSlow, speed))
		} else if speed > 100 {
			a.Lines = append(a.Lines, fmt.Sprintf(t.UISpeedFast, speed))
		} else {
			a.Lines = append(a.Lines, fmt.Sprintf(t.UISpeedNormal, speed))
		}
	}

	// Cost
	cost := EstimateCost(model, usage.InputTokens, usage.OutputTokens)
	if cost >= 0 {
		costLine := fmt.Sprintf(t.UICost, cost)
		if cost < 0.001 {
			costLine += t.UICostCheap
		} else if cost > 0.05 {
			costLine += t.UICostExpensive
		}
		a.Lines = append(a.Lines, costLine)
	}

	// Truncation warning
	if usage.OutputTokens >= maxOutputTokens-10 {
		a.Lines = append(a.Lines, fmt.Sprintf(t.UITruncated, usage.OutputTokens, maxOutputTokens))
	}

	// Low output
	if usage.InputTokens > 0 {
		ratio := float64(usage.OutputTokens) / float64(usage.InputTokens)
		if ratio < 0.1 {
			a.Lines = append(a.Lines, t.UILowOutput)
		}
	}

	// Local model tip
	if strings.HasPrefix(model, "llama") || strings.HasPrefix(model, "gemma") || model == "mistral" {
		if usage.OutputTokens > 0 && usage.OutputTokens < 200 {
			a.Lines = append(a.Lines, t.UILocalModelTip)
		}
	}

	return a
}
