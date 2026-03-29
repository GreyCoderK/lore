// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"fmt"
	"regexp"
	"strings"
)

// Compiled regex patterns replace inner loops over string slices.
// Each pattern is case-insensitive to match the previous strings.ToLower behavior.
var (
	testPatternRe      = regexp.MustCompile(`(?i)(_test\.go|/test/|test/|/mock/|mock/|/testdata/|testdata/)`)
	highValuePatternRe = regexp.MustCompile(`(?i)(/api/|api/|/schema/|schema/|/migration/|migration/|\.proto|\.graphql|/security/|security/)`)
)

// FileValueSignal scores modified files by their documentation value.
// High-value files (API, schema, migration, proto, graphql, security) get +5 each, capped at +15.
// If 100% of files are test/mock/testdata, score is -10.
func FileValueSignal(files []string) SignalScore {
	if len(files) == 0 {
		return SignalScore{Name: "file-value", Input: "0 files", Score: 0, Reason: "no files"}
	}

	highValue := 0
	testFiles := 0

	for _, f := range files {
		lower := strings.ToLower(f)

		if testPatternRe.MatchString(lower) {
			testFiles++
			continue
		}

		if highValuePatternRe.MatchString(lower) {
			highValue++
		}
	}

	// 100% test files → penalty
	if testFiles == len(files) {
		return SignalScore{
			Name: "file-value", Input: fmt.Sprintf("%d files (all tests)", len(files)),
			Score: -10, Reason: "100% test/mock files",
		}
	}

	score := highValue * 5
	if score > 15 {
		score = 15
	}

	reason := fmt.Sprintf("%d high-value files", highValue)
	if highValue == 0 {
		reason = "no high-value files"
	}

	return SignalScore{
		Name: "file-value", Input: fmt.Sprintf("%d files", len(files)),
		Score: score, Reason: reason,
	}
}
