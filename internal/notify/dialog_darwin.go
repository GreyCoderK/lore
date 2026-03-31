// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build darwin

package notify

import (
	"fmt"
)

// NotifyOSDialog launches a macOS AppleScript dialog for Lore documentation.
// Runs as a detached process — does not block the hook.
func NotifyOSDialog(data DialogData, opts DialogOpts) error {
	opts.defaults()

	script := buildAppleScript(data)
	return opts.StartCommand("osascript", []string{"-e", script}, nil)
}

func buildAppleScript(data DialogData) string {
	commitMsg := escapeAppleScript(sanitizeForShell(data.CommitMsg))
	diffStat := escapeAppleScript(sanitizeForShell(data.DiffStat))
	prefillWhat := escapeAppleScript(sanitizeForShell(data.PrefillWhat))
	prefillWhy := escapeAppleScript(sanitizeForShell(data.PrefillWhy))
	hash := sanitizeCommitHash(data.CommitHash)

	defaultType := escapeAppleScript(sanitizeForShell(data.PrefillType))
	if defaultType == "" {
		defaultType = "note"
	}

	// Labels — use i18n values or fallback to English.
	title := coalesce(data.LabelTitle, "Lore")
	titleWhat := coalesce(data.LabelTitleWhat, "Lore — What")
	titleWhy := coalesce(data.LabelTitleWhy, "Lore — Why")
	labelType := coalesce(data.LabelType, "Type:")
	labelWhat := coalesce(data.LabelWhat, "What did you change?")
	labelWhy := coalesce(data.LabelWhy, "Why did you make this change?")
	btnCancel := coalesce(data.LabelCancel, "Cancel")
	btnNext := coalesce(data.LabelNext, "Next")
	btnSave := coalesce(data.LabelSave, "Save")

	return fmt.Sprintf(`
set commitMsg to "%s"
set diffStat to "%s"

set docType to choose from list {"feature", "bugfix", "decision", "refactor", "release", "note"} with title "%s" with prompt "Commit: " & commitMsg & return & "Diff: " & diffStat & return & return & "%s" default items {"%s"}
if docType is false then return

set whatAnswer to text returned of (display dialog "%s" default answer "%s" with title "%s" buttons {"%s", "%s"} default button "%s")

set whyAnswer to text returned of (display dialog "%s" default answer "%s" with title "%s" buttons {"%s", "%s"} default button "%s")

try
	do shell script "cd " & quoted form of "%s" & " && " & quoted form of "%s" & " pending resolve --commit %s --type " & quoted form of (docType as text) & " --what " & quoted form of whatAnswer & " --why " & quoted form of whyAnswer
on error errMsg
	display dialog "%s" & errMsg with title "%s" buttons {"%s"} default button "%s"
end try
`,
		commitMsg, diffStat,
		escapeAppleScript(title), escapeAppleScript(labelType), defaultType,
		escapeAppleScript(labelWhat), prefillWhat, escapeAppleScript(titleWhat), escapeAppleScript(btnCancel), escapeAppleScript(btnNext), escapeAppleScript(btnNext),
		escapeAppleScript(labelWhy), prefillWhy, escapeAppleScript(titleWhy), escapeAppleScript(btnCancel), escapeAppleScript(btnSave), escapeAppleScript(btnSave),
		escapeAppleScript(sanitizeForShell(data.RepoRoot)),
		escapeAppleScript(sanitizeForShell(data.LorePath)), hash,
		escapeAppleScript(coalesce(data.LabelError, "Lore error: ")),
		escapeAppleScript(coalesce(data.LabelTitle, "Lore")),
		escapeAppleScript(coalesce(data.LabelOK, "OK")),
		escapeAppleScript(coalesce(data.LabelOK, "OK")),
	)
}

