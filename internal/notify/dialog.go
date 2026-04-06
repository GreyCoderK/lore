// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import "os/exec"

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
	Branch      string // current branch at commit time (e.g. "feature/auth")
	Scope       string // conventional commit scope (e.g. "auth")

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
	LabelOK        string // "OK"
	LabelError     string // "Lore error: "
	LabelErrResolve string // "Failed to resolve pending"

	// IconPath is the absolute path to the Lore logo (PNG) for dialog/notification icons.
	// Resolved at call site; empty means no custom icon.
	IconPath string
}

// branchScopeContext builds a display string like "\nBranch: main · Scope: auth"
// for use in dialog prompts. Returns "" if both fields are empty.
func branchScopeContext(data DialogData) string {
	ctx := ""
	if data.Branch != "" {
		ctx = "Branch: " + data.Branch
	}
	if data.Scope != "" {
		if ctx != "" {
			ctx += " · "
		}
		ctx += "Scope: " + data.Scope
	}
	if ctx == "" {
		return ""
	}
	return "\\n" + sanitizeForShell(ctx)
}

// DialogOpts holds injectable dependencies for OS dialog notification.
type DialogOpts struct {
	// StartCommand launches a detached command. Defaults to defaultStartCommand.
	StartCommand func(name string, args []string, env []string) error

	// LookPath searches for a binary in PATH. Defaults to exec.LookPath.
	// Used on Linux to detect zenity/kdialog/yad.
	LookPath func(file string) (string, error)
}

func (o *DialogOpts) defaults() {
	if o.StartCommand == nil {
		o.StartCommand = defaultStartCommand
	}
	if o.LookPath == nil {
		o.LookPath = exec.LookPath
	}
}
