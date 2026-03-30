// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

// DialogData holds the pre-filled data for OS dialog questions.
type DialogData struct {
	CommitHash  string
	CommitMsg   string
	DiffStat    string
	LorePath    string
	RepoRoot    string
	PrefillType string // pre-selected doc type (e.g. "bugfix")
	PrefillWhat string // pre-filled What answer
	PrefillWhy  string // pre-filled Why answer (if confidence >= 0.6)

	// I18n labels — populated from i18n.T().Notify by the caller.
	LabelTitle     string // "Lore"
	LabelTitleWhat string // "Lore — What"
	LabelTitleWhy  string // "Lore — Why"
	LabelType      string // "Type:"
	LabelWhat      string // "What did you change?"
	LabelWhy       string // "Why did you make this change?"
	LabelCancel    string // "Cancel"
	LabelNext      string // "Next"
	LabelSave      string // "Save"
	LabelSkip      string // "Skip"
}

// DialogOpts holds injectable dependencies for OS dialog notification.
type DialogOpts struct {
	// StartCommand launches a detached command. Defaults to defaultStartCommand.
	StartCommand func(name string, args []string, env []string) error
}

func (o *DialogOpts) defaults() {
	if o.StartCommand == nil {
		o.StartCommand = defaultStartCommand
	}
}
