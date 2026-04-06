// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

import "testing"

// testLookup simulates the i18n lookup for testing.
var testLookup = func(count int) string {
	msgs := map[int]string{
		3:  "3 decisions captured. They'll be here when you need them.",
		8:  "8 decisions documented. Your future self has a library now.",
		21: "21 decisions. This repo has memory.",
		55: "55 decisions documented. You're building institutional knowledge.",
	}
	return msgs[count]
}

func TestGetMilestoneMessage_FibonacciThresholds(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{3, "3 decisions captured. They'll be here when you need them."},
		{8, "8 decisions documented. Your future self has a library now."},
		{21, "21 decisions. This repo has memory."},
		{55, "55 decisions documented. You're building institutional knowledge."},
	}
	for _, tt := range tests {
		msg, ok := GetMilestoneMessage(tt.count, testLookup)
		if !ok {
			t.Errorf("GetMilestoneMessage(%d) = _, false; want true", tt.count)
		}
		if msg != tt.want {
			t.Errorf("GetMilestoneMessage(%d) = %q; want %q", tt.count, msg, tt.want)
		}
	}
}

func TestGetMilestoneMessage_OldThresholdsNoLongerTrigger(t *testing.T) {
	// Old thresholds 10, 25, 50 must NOT trigger anymore.
	for _, count := range []int{10, 25, 50} {
		msg, ok := GetMilestoneMessage(count, testLookup)
		if ok {
			t.Errorf("GetMilestoneMessage(%d) = %q, true; old threshold should no longer trigger", count, msg)
		}
	}
}

func TestGetMilestoneMessage_NonThresholds(t *testing.T) {
	for _, count := range []int{-3, -1, 0, 1, 2, 4, 5, 7, 9, 11, 20, 22, 54, 56, 100, 1000} {
		msg, ok := GetMilestoneMessage(count, testLookup)
		if ok {
			t.Errorf("GetMilestoneMessage(%d) = %q, true; want false", count, msg)
		}
		if msg != "" {
			t.Errorf("GetMilestoneMessage(%d) = %q; want empty", count, msg)
		}
	}
}

func TestGetMilestoneMessage_FibonacciBeyond55_NoMessage(t *testing.T) {
	// Fibonacci numbers beyond 55 are milestones but have no message yet.
	for _, count := range []int{89, 144, 233, 377, 610} {
		msg, ok := GetMilestoneMessage(count, testLookup)
		if ok {
			t.Errorf("GetMilestoneMessage(%d) = %q, true; Fibonacci >55 should have no message yet", count, msg)
		}
	}
}

func TestGetMilestoneMessage_LookupReturnsEmpty(t *testing.T) {
	emptyLookup := func(count int) string { return "" }
	msg, ok := GetMilestoneMessage(3, emptyLookup)
	if ok {
		t.Errorf("expected false when lookup returns empty, got msg=%q", msg)
	}
	if msg != "" {
		t.Errorf("expected empty msg, got %q", msg)
	}
}

func TestIsFibonacciMilestone(t *testing.T) {
	// Fibonacci numbers in range [3, 250]: 3, 5, 8, 13, 21, 34, 55, 89, 144, 233
	fibs := map[int]bool{
		3: true, 5: true, 8: true, 13: true, 21: true,
		34: true, 55: true, 89: true, 144: true, 233: true,
	}
	for n := 0; n <= 250; n++ {
		got := IsFibonacciMilestone(n)
		want := fibs[n]
		if got != want {
			t.Errorf("IsFibonacciMilestone(%d) = %v; want %v", n, got, want)
		}
	}
}

func TestIsFibonacciMilestone_Negative(t *testing.T) {
	for _, n := range []int{-1, -3, -8} {
		if IsFibonacciMilestone(n) {
			t.Errorf("IsFibonacciMilestone(%d) = true; want false", n)
		}
	}
}

func TestSeuil5_ReservedForStarPrompt(t *testing.T) {
	// 5 is Fibonacci but must NOT have a milestone message (reserved for star prompt 7f.3).
	if IsFibonacciMilestone(5) != true {
		t.Error("5 should be recognized as Fibonacci")
	}
	msg, ok := GetMilestoneMessage(5, testLookup)
	if ok {
		t.Errorf("GetMilestoneMessage(5) = %q, true; 5 is reserved for star prompt, not milestone", msg)
	}
}
