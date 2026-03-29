// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

// MilestoneThresholds are the Fibonacci thresholds that have assigned messages: 3, 8, 21, 55.
var MilestoneThresholds = []int{3, 8, 21, 55}

// IsFibonacciMilestone reports whether n is a Fibonacci number >= 3.
// Uses the generate-and-check approach with seed (1, 2) producing: 3, 5, 8, 13, 21, 34, 55, 89...
// Exported for use by the star prompt feature (Story 7f.3).
func IsFibonacciMilestone(n int) bool {
	if n < 3 {
		return false
	}
	a, b := 1, 2
	for b < n {
		a, b = b, a+b
	}
	return b == n
}

// IsMilestoneWithMessage returns true if docCount matches a threshold
// that has an assigned message (3, 8, 21, 55).
func IsMilestoneWithMessage(docCount int) bool {
	for _, t := range MilestoneThresholds {
		if docCount == t {
			return true
		}
	}
	return false
}

// GetMilestoneMessage returns the reinforcement message from the i18n catalog.
// The caller provides the message lookup function to keep this package free
// of i18n dependency. Returns ("", false) for non-milestone counts.
//
// Usage: engagement.GetMilestoneMessage(count, milestoneMessageFromI18N)
func GetMilestoneMessage(docCount int, lookup func(int) string) (string, bool) {
	if !IsMilestoneWithMessage(docCount) {
		return "", false
	}
	msg := lookup(docCount)
	if msg == "" {
		return "", false
	}
	return msg, true
}
