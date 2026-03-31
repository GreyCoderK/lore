// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import (
	"fmt"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// CalibrationReport holds quality metrics for the Decision Engine.
type CalibrationReport struct {
	TotalCommits     int
	AutoSkipped      int
	SuggestSkipped   int
	AskReduced       int
	AskFull          int
	Documented       int
	FalseNegativeRate float64 // auto-skip then manually documented / total auto-skip
	FalsePositiveRate float64 // ask-full then skipped / total ask-full
	AskFullDocRate    float64 // documented / total ask-full
	AutoSkipRate      float64 // auto-skip / total commits
}

// ComputeCalibration calculates Decision Engine quality metrics from stored commits.
// Uses a proxy for false negatives: commits with decision='documented' and question_mode='none'
// indicate an auto-skip that was later resolved via `lore new --commit`.
func ComputeCalibration(store domain.LoreStore) (*CalibrationReport, error) {
	if store == nil {
		return nil, fmt.Errorf("store is nil")
	}

	counts, err := store.CommitCountByDecision()
	if err != nil {
		return nil, fmt.Errorf("CommitCountByDecision: %w", err)
	}

	r := &CalibrationReport{}

	// Count by decision type
	r.AutoSkipped = counts["auto-skipped"]
	r.Documented = counts["documented"]

	// Use SQL aggregation instead of loading all commits into memory.
	// Query mode-level stats via CommitsSince with a reasonable lookback.
	commits, err := store.CommitsSince(time.Time{})
	if err != nil {
		return nil, fmt.Errorf("CommitsSince: %w", err)
	}

	// Detailed analysis from individual commits
	var (
		askFullTotal      int
		askFullSkipped    int
		askFullDocumented int
		suggestSkipTotal  int
		askReducedTotal   int
		falseNegatives    int
	)

	for _, c := range commits {
		r.TotalCommits++

		switch c.QuestionMode {
		case "full":
			askFullTotal++
			switch c.Decision {
			case "skipped":
				askFullSkipped++
			case "documented":
				askFullDocumented++
			}
		case "reduced":
			askReducedTotal++
		case "none":
			// Proxy for false negative: documented with mode=none means
			// auto-skipped then later resolved via `lore new --commit`
			if c.Decision == "documented" {
				falseNegatives++
			}
		}

		// Count suggest-skip decisions
		if c.Decision == "skipped" && c.QuestionMode == "" {
			suggestSkipTotal++
		}
	}

	r.AskFull = askFullTotal
	r.AskReduced = askReducedTotal
	r.SuggestSkipped = suggestSkipTotal

	// Compute rates (avoid division by zero)
	if r.AutoSkipped > 0 {
		r.FalseNegativeRate = float64(falseNegatives) / float64(r.AutoSkipped+falseNegatives)
	}
	if askFullTotal > 0 {
		r.FalsePositiveRate = float64(askFullSkipped) / float64(askFullTotal)
		r.AskFullDocRate = float64(askFullDocumented) / float64(askFullTotal)
	}
	if r.TotalCommits > 0 {
		r.AutoSkipRate = float64(r.AutoSkipped) / float64(r.TotalCommits)
	}

	return r, nil
}

// FormatCalibration returns a human-readable summary of calibration metrics.
func FormatCalibration(r *CalibrationReport) string {
	t := i18n.T().Decision
	return fmt.Sprintf("%s\n===========================\n"+
		t.CalibrationTotalCommits+"\n"+
		t.CalibrationAutoSkipped+"\n"+
		t.CalibrationSuggestSkip+"\n"+
		t.CalibrationAskReduced+"\n"+
		t.CalibrationAskFull+"\n\n"+
		t.CalibrationQualityHdr+"\n"+
		t.CalibrationFalseNegRate+"\n"+
		t.CalibrationFalsePosRate+"\n"+
		t.CalibrationAskFullDoc+"\n"+
		t.CalibrationAutoSkipRate,
		t.CalibrationTitle,
		r.TotalCommits,
		r.AutoSkipped, r.AutoSkipRate*100,
		r.SuggestSkipped,
		r.AskReduced,
		r.AskFull,
		r.FalseNegativeRate*100,
		r.FalsePositiveRate*100,
		r.AskFullDocRate*100,
		r.AutoSkipRate*100,
	)
}
