// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

// StarPromptOpts controls whether the star prompt should be shown.
type StarPromptOpts struct {
	DocCount      int
	Threshold     int  // default 5, configurable via hooks.star_prompt_after
	AlreadyShown  bool // from EngagementState.StarPromptShown
	Enabled       bool // from hooks.star_prompt config (default true)
	IsTTY         bool
	IsQuiet       bool
}

// ShouldShowStarPrompt returns true if conditions are met for showing
// the one-time star prompt. It fires exactly once at the threshold.
func ShouldShowStarPrompt(opts StarPromptOpts) bool {
	if !opts.Enabled {
		return false
	}
	if opts.AlreadyShown {
		return false
	}
	if !opts.IsTTY {
		return false
	}
	if opts.IsQuiet {
		return false
	}
	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 5
	}
	return opts.DocCount >= threshold
}
