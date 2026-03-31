// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// NotifyMessages holds strings for OS notification dialogs (internal/notify/).
type NotifyMessages struct {
	// Dialog titles and labels
	DialogTitle     string // "Lore"
	DialogTitleWhat string // "Lore — What"
	DialogTitleWhy  string // "Lore — Why"

	// Dialog prompts
	PromptType string // "Type:"
	PromptWhat string // "What did you change?"
	PromptWhy  string // "Why did you make this change?"

	// Dialog buttons
	ButtonCancel string // "Cancel"
	ButtonNext   string // "Next"
	ButtonSave   string // "Save"
	ButtonSkip   string // "Skip"

	// Simple notification
	SimplePending string // "lore pending — %s"

	// Dialog error handling
	ErrorPrefix string // "Lore error: "
	ErrorResolve string // "Failed to resolve pending"
	ButtonOK     string // "OK"

	// Pending resolve
	NoMatchingPending string // "No pending commit matching %q"
}
