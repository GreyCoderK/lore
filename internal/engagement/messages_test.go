// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

import "testing"

func TestGetMilestoneMessage_ExactThresholds(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{3, "3 decisions captured. They'll be here when you need them."},
		{10, "10 decisions documented. Your future self has a library now."},
		{25, "25 decisions. This repo has memory."},
		{50, "50 decisions documented. You're building institutional knowledge."},
	}
	for _, tt := range tests {
		msg, ok := GetMilestoneMessage(tt.count)
		if !ok {
			t.Errorf("GetMilestoneMessage(%d) = _, false; want true", tt.count)
		}
		if msg != tt.want {
			t.Errorf("GetMilestoneMessage(%d) = %q; want %q", tt.count, msg, tt.want)
		}
	}
}

func TestGetMilestoneMessage_NonThresholds(t *testing.T) {
	for _, count := range []int{-3, -1, 0, 1, 2, 4, 7, 9, 11, 24, 26, 49, 51, 100, 1000} {
		msg, ok := GetMilestoneMessage(count)
		if ok {
			t.Errorf("GetMilestoneMessage(%d) = %q, true; want false", count, msg)
		}
		if msg != "" {
			t.Errorf("GetMilestoneMessage(%d) = %q; want empty", count, msg)
		}
	}
}
