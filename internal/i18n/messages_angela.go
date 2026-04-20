// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// AngelaMessages holds user-facing strings for the Angela AI reviewer.
// NOTE: PromptDirective, Principles, and ContentSignals remain in English.
type AngelaMessages struct {
	// persona.go — DraftCheck messages (user-facing suggestions)
	PersonaWhyTooListy       string
	PersonaLongParagraphs    string
	PersonaMissingVerify     string
	PersonaNoTradeoffs       string
	PersonaUxNoImpact        string
	PersonaBusinessNoValue   string
	PersonaAPINoExample      string // api-designer: endpoints listed without a request example
	PersonaAPIMissingErrors  string // api-designer: endpoints without HTTP error responses
	PersonaAPIMissingRequired string // api-designer: DTO fields without a required/optional column

	// draft.go — completeness checks
	DraftMissingWhat    string
	DraftMissingWhy     string
	DraftMissingAltWarn string
	DraftMissingAltInfo string
	DraftMissingImpact  string
	DraftBodyTooShort   string
	DraftBodyExceedsMax string
	DraftAddScope       string
	DraftAddTags        string
	DraftAddRelated     string
	DraftWhyTooBrief    string

	// coherence.go
	CoherencePossibleDup       string // args: filename, tags
	CoherenceRelatedFound      string // args: filename, tags
	CoherenceSameScopeOverlap  string // args: scope, filename
	CoherenceSameScopeRelated  string // args: scope, filename, type
	CoherenceMentionedBody     string // arg: filename

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

	// polish — hallucination check
	PolishHallucinationWarn   string // no args
	PolishHallucinationReject string // no args
	PolishHallucinationHint   string // no args

	// review — differential summary
	ReviewDiffSummary  string // args: new, persisting, regressed, resolved
	ReviewDiffResolved string // no args

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

	// Preview report (lore angela review --preview)
	UIReviewPreviewHeader           string // "Review preview" + separator rule
	UIReviewPreviewCorpus           string // args: count (int), size (string like "245 KB")
	UIReviewPreviewModel            string // arg: model name
	UIReviewPreviewPersonasBaseline string // "baseline (no personas)"
	UIReviewPreviewPersonasList     string // args: count (int), pluralSuffix (string "s" or ""), comma-separated names
	UIReviewPreviewAudienceNone     string // "(none)"
	UIReviewPreviewAudience         string // arg: audience
	UIReviewPreviewTokens           string // args: input tokens (string formatted), max output tokens (string formatted)
	UIReviewPreviewContextWindow    string // arg: pct (float)
	UIReviewPreviewCost             string // arg: USD amount (string formatted)
	UIReviewPreviewCostUnknown      string // "unknown (model pricing not registered)"
	UIReviewPreviewExpectedTime     string // arg: seconds (int)
	UIReviewPreviewWarningsHeader   string // just "Warnings:" heading, no args
	UIReviewPreviewAbort            string // arg: abort reason

	// Persona opt-in prompt & UX
	UIPersonaConfiguredHeader        string // args: count (int), pluralSuffix (string), comma-separated names
	UIPersonaCostDeltaBaselineLabel  string // arg: rendered line (string)
	UIPersonaCostDeltaAugmentedLabel string // args: count (int), pluralSuffix, rendered line
	UIPersonaAddContext              string // explanation line, no args
	UIPersonaPromptQuestion          string // "   Use them for this review? [y/N] "
	UIPersonaNonTTYInfo              string // args: icon (string), comma-separated names
	UIPersonaCostBaselineZero        string // "~$0 (zero-token baseline)"
	UIPersonaCostDeltaUndefined      string // arg: USD amount — used when baseline cost is 0
	UIPersonaCostDeltaPct            string // arg: percent (int) — like "(+18%)"
	UIPersonaCostSameRoughly         string // "(~same cost)"
	UIPersonaCostUnknown             string // "(cost unknown — model pricing not registered)"
	UIPersonaInputTokens             string // arg: formatted token count — "~%s input tokens"
	UIPersonaCostInline              string // arg: USD — " · ~$%s"

	// Review report header + per-finding + TUI
	UIReviewAngleHeader              string // args: count (int), pluralSuffix
	UIReviewAnglePersonaRow          string // args: icon, displayName, expertise
	UIReviewFlaggedBy                string // arg: comma-separated "icon name" pairs
	UIReviewAgreementConcur          string // args: concurrentCount (int), poolSize (int)
	UIReviewAgreementLineFormat      string // arg: agreement tag — "(%d/%d)"

	// Story 8-21 — duplicate-section arbitration prompts (TTY).
	// Used by internal/angela/polish_arbitrate.go.
	ArbitrateGroupHeader        string // args: groupIdx (int), total (int), heading (%q), occCount (int)
	ArbitrateInvalidChoice      string // no args
	ArbitratePrompt             string // no args — "  → "
	ArbitratePreviewLine        string // args: idx (int), line (int), words (int)
	ArbitratePreviewTruncated   string // args: shown (int), total (int)
	ArbitrateOptKeepBoth        string // no args
	ArbitrateOptKeepAll         string // arg: count (int)
	ArbitrateOptsLine           string // arg: bothOrAll label (already localized)
	ArbitrateOccurrenceBanner   string // args: i (int), total (int), line (int)
	ArbitrateEmptyPreview       string // no args — "(empty section)"
}
