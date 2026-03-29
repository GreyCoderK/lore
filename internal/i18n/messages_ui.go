// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// UIMessages holds strings for terminal UI components (internal/ui/ package).
type UIMessages struct {
	// logo.go
	Tagline string // "your code knows what. lore knows why."

	// error.go
	ErrorPrefix string // "Error:"
	RunPrefix   string // "Run:"

	// verb.go
	VerbDeleted string // "Deleted"

	// list.go
	ListTruncated    string // "... and %d more. Refine your search." (arg: remaining count)
	ListPromptRange  string // "Please enter a number between 1 and %d." (arg: max)
	ListNoInput      string // "no input"

	// prompt.go
	PromptWithDefault string // "? %s [%s]: " (args: question, default)
	PromptNoDefault   string // "? %s\n> " (arg: question)
}
