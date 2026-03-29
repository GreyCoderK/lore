// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// DecisionMessages holds strings for calibration report and signal reasons.
type DecisionMessages struct {
	// calibration.go — report labels
	CalibrationTitle       string
	CalibrationTotalCommits string
	CalibrationAutoSkipped  string // args: count, pct
	CalibrationSuggestSkip  string
	CalibrationAskReduced   string
	CalibrationAskFull      string
	CalibrationQualityHdr   string
	CalibrationFalseNegRate string // arg: rate
	CalibrationFalsePosRate string // arg: rate
	CalibrationAskFullDoc   string // arg: rate
	CalibrationAutoSkipRate string // arg: rate
}
