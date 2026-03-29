// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import "testing"

// --- Signal 1: Conventional Commit Type ---

func TestScoreConvType_AllTypes(t *testing.T) {
	tests := []struct {
		convType string
		message  string
		want     int
	}{
		{"feat", "", 40},
		{"fix", "", 30},
		{"perf", "", 25},
		{"revert", "", 15},
		{"refactor", "", 10},
		{"chore", "", 5},
		{"test", "", 3},
		{"docs", "", 0},
		{"style", "", 0},
		{"ci", "", 0},
		{"build", "", 0},
		{"", "", 20},                                // empty type
		{"feat!", "", 45},                           // breaking via !
		{"feat", "BREAKING CHANGE: new API", 45},    // breaking via body
		{"unknown", "", 20},                         // unknown type
	}
	for _, tt := range tests {
		t.Run(tt.convType, func(t *testing.T) {
			s := scoreConvType(tt.convType, tt.message)
			if s.Score != tt.want {
				t.Errorf("scoreConvType(%q) = %d, want %d", tt.convType, s.Score, tt.want)
			}
		})
	}
}

// --- Signal 2: Diff Size ---

func TestScoreDiffSize_Boundaries(t *testing.T) {
	tests := []struct {
		added, deleted int
		want           int
	}{
		{1, 0, 0},   // 1 total < 3
		{2, 0, 0},   // 2 total < 3
		{3, 0, 10},  // 3 total = 3 (boundary)
		{10, 10, 10}, // 20 total <= 20
		{11, 10, 15}, // 21 total > 20
		{50, 50, 15}, // 100 total <= 100
		{51, 50, 20}, // 101 total > 100
		{200, 0, 20}, // large
	}
	for _, tt := range tests {
		s := scoreDiffSize(tt.added, tt.deleted)
		if s.Score != tt.want {
			t.Errorf("scoreDiffSize(%d, %d) = %d, want %d", tt.added, tt.deleted, s.Score, tt.want)
		}
	}
}
