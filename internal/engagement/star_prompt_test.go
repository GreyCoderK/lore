// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldShowStarPrompt_ThresholdReached(t *testing.T) {
	assert.True(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 5, Threshold: 5, Enabled: true, IsTTY: true,
	}))
}

func TestShouldShowStarPrompt_BelowThreshold(t *testing.T) {
	assert.False(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 4, Threshold: 5, Enabled: true, IsTTY: true,
	}))
}

func TestShouldShowStarPrompt_AlreadyShown(t *testing.T) {
	assert.False(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 10, Threshold: 5, Enabled: true, IsTTY: true, AlreadyShown: true,
	}))
}

func TestShouldShowStarPrompt_Disabled(t *testing.T) {
	assert.False(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 10, Threshold: 5, Enabled: false, IsTTY: true,
	}))
}

func TestShouldShowStarPrompt_NonTTY(t *testing.T) {
	assert.False(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 10, Threshold: 5, Enabled: true, IsTTY: false,
	}))
}

func TestShouldShowStarPrompt_Quiet(t *testing.T) {
	assert.False(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 10, Threshold: 5, Enabled: true, IsTTY: true, IsQuiet: true,
	}))
}

func TestShouldShowStarPrompt_DefaultThreshold(t *testing.T) {
	// Threshold 0 → defaults to 5.
	assert.True(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 5, Threshold: 0, Enabled: true, IsTTY: true,
	}))
}

func TestShouldShowStarPrompt_AboveThreshold(t *testing.T) {
	// Above threshold but not yet shown → still show.
	assert.True(t, ShouldShowStarPrompt(StarPromptOpts{
		DocCount: 20, Threshold: 5, Enabled: true, IsTTY: true,
	}))
}
