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
	DiffChangeHeader string // args: current, total
	DiffApplyPrompt  string
	DiffInputEnded   string // arg: remaining

	// review.go
	ReviewParseError string
	ReviewMinDocs    string // arg: count

	// style.go
	StyleUnknownRule string // arg: key
}
