// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

// Default values for the Decision Engine. Exported so that config/defaults.go
// can reference them, avoiding value duplication across two files.
const (
	DefaultThresholdFull      = 60
	DefaultThresholdReduced   = 35
	DefaultThresholdSuggest   = 15
	DefaultLearningMinCommits = 20
)

// DefaultAlwaysAsk returns the default always_ask types (fresh copy each call).
func DefaultAlwaysAsk() []string { return []string{"feat", "breaking"} }

// DefaultAlwaysSkip returns the default always_skip types (fresh copy each call).
func DefaultAlwaysSkip() []string { return []string{"docs", "style", "ci", "build"} }

// EngineConfig holds Decision Engine configuration from .lorerc.
type EngineConfig struct {
	ThresholdFull      int
	ThresholdReduced   int
	ThresholdSuggest   int
	AlwaysAsk          []string
	AlwaysSkip         []string
	CriticalScopes     []string
	Learning           bool
	LearningMinCommits int
}

// DefaultConfig returns sensible defaults for the Decision Engine.
func DefaultConfig() EngineConfig {
	return EngineConfig{
		ThresholdFull:      DefaultThresholdFull,
		ThresholdReduced:   DefaultThresholdReduced,
		ThresholdSuggest:   DefaultThresholdSuggest,
		AlwaysAsk:          DefaultAlwaysAsk(),
		AlwaysSkip:         DefaultAlwaysSkip(),
		CriticalScopes:     nil,
		Learning:           true,
		LearningMinCommits: DefaultLearningMinCommits,
	}
}
