// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// WorkflowMessages holds strings for workflow orchestration (internal/workflow/).
type WorkflowMessages struct {
	// reactive.go
	SuggestSkipPrompt     string // "Document this commit? [y/N] "
	SuggestSkipSkipped    string // "Skipped."
	AmendUpdatingExisting string // arg: filename
	AmendNoDocCreatingNew string // arg: ref

	// detection.go
	MergeCommitSkipMsg string
	AutoSkipMsg        string // args: score, subject

	// proactive.go
	AlreadyDocumented string
	CreateAnother     string // "Create another document? [y/N] "

	// common.go
	IndexWarning string // arg: error

	// questions.go
	QuestionWhyLabel          string
	QuestionAlternativesLabel string
	QuestionImpactLabel       string

	// resolve.go
	ResolveHeader         string
	ResolveCommitLabel    string // arg: hash
	ResolveMessageLabel   string // arg: message
	ResolveDateLabel      string // arg: date
	ResolveCheckCommitW   string // args: hash, error
	ResolveCommitGoneW    string // arg: hash
	ResolveDeletePendingW string // arg: error

	// pending.go
	PendingReadWarn  string // args: path, error
	PendingParseWarn string // args: path, error

	// pending.go relative time
	RelativeAgeJustNow string
	RelativeAgeMinutes string // arg: count
	RelativeAge1Hour   string
	RelativeAgeHours   string // arg: count
	RelativeAge1Day    string
	RelativeAgeDays    string // arg: count
	RelativeAge1Week   string
	RelativeAgeWeeks   string // arg: count
	RelativeAge1Month  string
	RelativeAgeMonths  string // arg: count

	// line_renderer.go
	LineRendererConfirm    string // args: label, value
	LineRendererExpressSkip string // arg: count
}
