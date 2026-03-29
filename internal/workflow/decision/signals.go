// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"fmt"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// clock abstracts time for testability. Injected via Engine, not a mutable global.
type clock func() time.Time

// Scoring constants for Signal 1: Conventional Commit Type.
// Higher score = more likely to need documentation.
const (
	scoreBreakingChange = 45
	scoreNoConvType     = 20
	scoreUnknownType    = 20
)

var typeScores = map[string]int{
	"feat":     40, // New feature — almost always worth documenting
	"fix":      30, // Bug fix — often documents the "why"
	"perf":     25, // Performance improvement — architectural insight
	"revert":   15, // Revert — context on what went wrong
	"refactor": 10, // Refactoring — low doc value unless structural
	"chore":    5,  // Chores — rarely need documentation
	"test":     3,  // Test additions — very low doc value
	"docs":     0,  // Documentation changes — already documented
	"style":    0,  // Code style — no documentation needed
	"ci":       0,  // CI config — no documentation needed
	"build":    0,  // Build config — no documentation needed
}

// Scoring constants for Signal 2: Diff Size.
const (
	diffTrivialThreshold = 3   // lines: below this is trivial (score 0)
	diffSmallThreshold   = 20  // lines: small change (score 10)
	diffMediumThreshold  = 100 // lines: medium change (score 15)
	// Above medium is large (score 20)

	scoreDiffSmall  = 10
	scoreDiffMedium = 15
	scoreDiffLarge  = 20
)

// Scoring constants for Signal 5: LKS History.
const (
	scoreCriticalScope    = 20 // scope is in the critical list
	scoreNeverDocumented  = 15 // scope has no documentation history
	scoreStaleDoc         = 10 // last doc for scope is >30 days old
	scoreRecentDoc        = -10 // last doc for scope is <7 days old (reduce)
	scoreHighSkipRate     = -5  // skip rate >80% for this scope
	staleDaysThreshold    = 30  // days since last doc to consider stale
	recentDaysThreshold   = 7   // days since last doc to consider recent
	highSkipRateThreshold = 0.8 // skip rate above this triggers penalty
)

func scoreConvType(convType string, message string) SignalScore {
	// Check for breaking change
	if strings.HasSuffix(convType, "!") || strings.Contains(message, "BREAKING CHANGE") {
		return SignalScore{Name: "conv-type", Input: convType, Score: scoreBreakingChange, Reason: "breaking change"}
	}

	cleanType := strings.TrimSuffix(convType, "!")
	if cleanType == "" {
		return SignalScore{Name: "conv-type", Input: "(empty)", Score: scoreNoConvType, Reason: "no conventional type"}
	}

	score, ok := typeScores[cleanType]
	if !ok {
		return SignalScore{Name: "conv-type", Input: convType, Score: scoreUnknownType, Reason: "unknown type"}
	}
	return SignalScore{Name: "conv-type", Input: convType, Score: score, Reason: fmt.Sprintf("type %s", cleanType)}
}

// --- Signal 2: Diff Size ---

func scoreDiffSize(linesAdded, linesDeleted int) SignalScore {
	// Guard against negative values and overflow
	if linesAdded < 0 {
		linesAdded = 0
	}
	if linesDeleted < 0 {
		linesDeleted = 0
	}
	// Cap individual values to prevent integer overflow on addition
	const maxLines = 1<<31 - 1
	if linesAdded > maxLines {
		linesAdded = maxLines
	}
	if linesDeleted > maxLines {
		linesDeleted = maxLines
	}
	total := linesAdded + linesDeleted
	var score int
	var reason string

	switch {
	case total < diffTrivialThreshold:
		score = 0
		reason = fmt.Sprintf("%d lines (trivial)", total)
	case total <= diffSmallThreshold:
		score = scoreDiffSmall
		reason = fmt.Sprintf("%d lines (small)", total)
	case total <= diffMediumThreshold:
		score = scoreDiffMedium
		reason = fmt.Sprintf("%d lines (medium)", total)
	default:
		score = scoreDiffLarge
		reason = fmt.Sprintf("%d lines (large)", total)
	}

	return SignalScore{Name: "diff-size", Input: fmt.Sprintf("+%d/-%d", linesAdded, linesDeleted), Score: score, Reason: reason}
}

// --- Signal 5: LKS History ---

func scoreLKSHistory(store domain.LoreStore, scope string, config EngineConfig, now clock) SignalScore {
	if store == nil {
		return SignalScore{Name: "lks-history", Input: scope, Score: 0, Reason: "store unavailable"}
	}

	score := 0
	reasons := make([]string, 0, 8)

	// Critical scope check
	for _, cs := range config.CriticalScopes {
		if cs == scope {
			score += scoreCriticalScope
			reasons = append(reasons, fmt.Sprintf("critical scope +%d", scoreCriticalScope))
			break
		}
	}

	// Query scope history via SQL aggregation (avoids loading all commit records)
	stats, err := store.ScopeStats(scope, 365)
	if err != nil || stats.TotalCommits == 0 {
		if scope != "" {
			score += scoreNeverDocumented
			reasons = append(reasons, fmt.Sprintf("scope never documented +%d", scoreNeverDocumented))
		}
	} else {
		if stats.DocumentedCount == 0 && scope != "" {
			score += scoreNeverDocumented
			reasons = append(reasons, fmt.Sprintf("scope never documented +%d", scoreNeverDocumented))
		} else if stats.LastDocDate > 0 {
			nowUnix := now().Unix()
			daysSince := (nowUnix - stats.LastDocDate) / 86400
			if daysSince > staleDaysThreshold {
				score += scoreStaleDoc
				reasons = append(reasons, fmt.Sprintf("last doc %dd ago +%d", daysSince, scoreStaleDoc))
			} else if daysSince < recentDaysThreshold {
				score += scoreRecentDoc
				reasons = append(reasons, fmt.Sprintf("recent doc %dd ago %d", daysSince, scoreRecentDoc))
			}
		}

		// Skip rate
		total := stats.DocumentedCount + stats.SkippedCount
		if total > 0 {
			skipRate := float64(stats.SkippedCount) / float64(total)
			if skipRate > highSkipRateThreshold {
				score += scoreHighSkipRate
				reasons = append(reasons, fmt.Sprintf("skip rate %.0f%% %d", skipRate*100, scoreHighSkipRate))
			}
		}
	}

	// Warm-up: halve if < learning_min_commits
	totalCommits, countErr := store.CommitCountByDecision()
	if countErr != nil {
		reasons = append(reasons, "store error: warm-up check skipped")
		return SignalScore{Name: "lks-history", Input: scope, Score: score, Reason: strings.Join(reasons, ", ")}
	}
	totalAll := 0
	for _, c := range totalCommits {
		totalAll += c
	}
	if totalAll < config.LearningMinCommits {
		score = score / 2
		reasons = append(reasons, fmt.Sprintf("warm-up (%d/%d commits) halved", totalAll, config.LearningMinCommits))
	}

	reason := "no signals"
	if len(reasons) > 0 {
		reason = strings.Join(reasons, ", ")
	}

	return SignalScore{Name: "lks-history", Input: scope, Score: score, Reason: reason}
}
