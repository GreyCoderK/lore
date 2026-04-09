// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// AngelaMessages holds user-facing strings for the Angela AI reviewer.
// NOTE: PromptDirective, Principles, and ContentSignals remain in English.
type AngelaMessages struct {
	// persona.go — DraftCheck messages (user-facing suggestions)
	PersonaWhyTooListy     string
	PersonaLongParagraphs  string
	PersonaMissingVerify   string
	PersonaNoTradeoffs     string
	PersonaUxNoImpact      string
	PersonaBusinessNoValue string

	// draft.go — completeness checks
	DraftMissingWhat        string
	DraftMissingWhy         string
	DraftMissingAltWarn     string
	DraftMissingAltInfo     string
	DraftMissingImpact      string
	DraftBodyTooShort       string
	DraftBodyExceedsMax     string
	DraftAddTags            string
	DraftAddRelated         string
	DraftWhyTooBrief        string

	// coherence.go
	CoherencePossibleDup  string // args: filename, tags
	CoherenceRelatedFound string // args: filename, tags
	CoherenceMentionedBody string // arg: filename

	// diff.go
	DiffChangeHeader     string // args: current, total
	DiffApplyPrompt      string
	DiffApplyBothPrompt  string
	DiffInputEnded       string // arg: remaining
	DiffHunkLocation     string // args: startLine, lineCount
	DiffAutoAccept       string // args: desc, label
	DiffAutoReject       string // args: desc
	DiffAutoNeedsReview  string // args: current, total
	DiffAutoSummary      string // args: accepted, rejected, asked
	DiffWarnNetLoss      string // args: deleted, net
	DiffWarnSection      string // arg: names
	DiffWarnSections     string // args: count, names
	DiffWarnCodeBlocks   string // arg: count
	DiffWarnTableRows    string // arg: count

	// --for mode
	UIForPrompt     string // arg: audience
	UIForNewFile    string
	UIForOverwrite  string

	// review.go
	ReviewParseError string
	ReviewMinDocs    string // arg: count

	// style.go
	StyleUnknownRule string // arg: key

	// Runtime UI — polish/review progress messages
	UIMode              string // arg: audience
	UITokenEstimate     string // args: inputTokens, maxOutput, timeout
	UIPersonas          string // arg: personaList
	UIQuality           string // arg: score
	UIMultiPass         string // arg: sectionCount
	UITokenStats        string // args: input, output, model
	UISpeedFast         string // arg: speed
	UISpeedSlow         string // arg: speed
	UISpeedNormal       string // arg: speed
	UICost              string // arg: cost
	UICostCheap         string
	UICostExpensive     string
	UITruncated         string // args: used, limit
	UITruncatedHint     string
	UITimeoutErr        string // args: timeout, elapsed
	UITimeoutHint1      string
	UITimeoutHint2      string
	UIRewrittenFor      string // args: audience, filename
	UIOriginalUnchanged string
	UIChangesQuality    string // args: count, scoreBefore, scoreAfter, delta
	UIInputExceedsMax   string // args: inputTokens, maxOutput
	UIInputExceedsHint  string // arg: suggested max_tokens
	UIEstimatedCost     string // arg: cost
	UIContextWarning    string // args: model, needed, limit
	UIContextClose      string // args: pct, model, needed, limit
	UITimeoutWarning    string // args: timeout, model, speed, seconds, tokens
	UILowOutput         string
	UILocalModelTip     string
	UIReviewPreflight   string // args: docCount, inputTokens, maxOutput, timeout
}
