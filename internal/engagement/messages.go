// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

// milestones maps exact document counts to reinforcement messages.
// Thresholds: 3, 10, 25, 50 (per FR41 and AC-1 through AC-4).
var milestones = map[int]string{
	3:  "3 decisions captured. They'll be here when you need them.",
	10: "10 decisions documented. Your future self has a library now.",
	25: "25 decisions. This repo has memory.",
	50: "50 decisions documented. You're building institutional knowledge.",
}

// GetMilestoneMessage returns the reinforcement message if docCount matches
// an exact threshold (3, 10, 25, 50). Returns ("", false) otherwise.
// Formatting (dim/gray via ui.Dim) is the caller's responsibility —
// this package is pure logic with no external imports.
func GetMilestoneMessage(docCount int) (string, bool) {
	msg, ok := milestones[docCount]
	return msg, ok
}
