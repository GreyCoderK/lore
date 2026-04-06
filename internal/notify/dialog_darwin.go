// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build darwin

package notify

import "os"


// NotifyOSDialog launches a macOS AppleScript dialog for Lore documentation.
// Runs as a detached process — does not block the hook.
func NotifyOSDialog(data DialogData, opts DialogOpts) error {
	opts.defaults()

	// Resolve logo path for dialog icon.
	if data.IconPath == "" {
		data.IconPath = resolveLogoPath(data.RepoRoot)
	}

	script := buildAppleScript(data)
	return opts.StartCommand("osascript", []string{"-e", script}, nil)
}

// resolveLogoPath finds the Lore logo PNG for dialog icons.
// Checks repo-local assets first, then the installed binary's sibling.
func resolveLogoPath(repoRoot string) string {
	candidates := []string{
		repoRoot + "/assets/logo.png",
		repoRoot + "/docs/assets/logo.png",
	}
	for _, p := range candidates {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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

	// Branch Awareness: build context line for dialog prompt.
	branchCtx := ""
	if data.Branch != "" {
		branchCtx += "Branch: " + escapeAppleScript(sanitizeForShell(data.Branch))
	}
	if data.Scope != "" {
		if branchCtx != "" {
			branchCtx += " · "
		}
		branchCtx += "Scope: " + escapeAppleScript(sanitizeForShell(data.Scope))
	}

	// Build the prompt prefix: commit + diff + optional branch/scope line.
	promptPrefix := `"Commit: " & commitMsg & return & "Diff: " & diffStat`
	if branchCtx != "" {
		promptPrefix += ` & return & "` + branchCtx + `"`
	}

	// Build icon clause for display dialog (POSIX path → HFS alias).
	iconClause := ""
	if data.IconPath != "" {
		iconClause = ` with icon file (POSIX file "` + escapeAppleScript(sanitizeForShell(data.IconPath)) + `" as alias)`
	}

	return `
set commitMsg to "` + commitMsg + `"
set diffStat to "` + diffStat + `"

set docType to choose from list {"feature", "bugfix", "decision", "refactor", "release", "note"} with title "` + escapeAppleScript(title) + `" with prompt ` + promptPrefix + ` & return & return & "` + escapeAppleScript(labelType) + `" default items {"` + defaultType + `"}
if docType is false then return

set whatAnswer to text returned of (display dialog "` + escapeAppleScript(labelWhat) + `" default answer "` + prefillWhat + `" with title "` + escapeAppleScript(titleWhat) + `" buttons {"` + escapeAppleScript(btnCancel) + `", "` + escapeAppleScript(btnNext) + `"} default button "` + escapeAppleScript(btnNext) + `"` + iconClause + `)

set whyAnswer to text returned of (display dialog "` + escapeAppleScript(labelWhy) + `" default answer "` + prefillWhy + `" with title "` + escapeAppleScript(titleWhy) + `" buttons {"` + escapeAppleScript(btnCancel) + `", "` + escapeAppleScript(btnSave) + `"} default button "` + escapeAppleScript(btnSave) + `"` + iconClause + `)

try
	do shell script "cd " & quoted form of "` + escapeAppleScript(sanitizeForShell(data.RepoRoot)) + `" & " && " & quoted form of "` + escapeAppleScript(sanitizeForShell(data.LorePath)) + `" & " pending resolve --commit ` + hash + ` --type " & quoted form of (docType as text) & " --what " & quoted form of whatAnswer & " --why " & quoted form of whyAnswer
on error errMsg
	display dialog "` + escapeAppleScript(coalesce(data.LabelError, "Lore error: ")) + `" & errMsg with title "` + escapeAppleScript(coalesce(data.LabelTitle, "Lore")) + `" buttons {"` + escapeAppleScript(coalesce(data.LabelOK, "OK")) + `"} default button "` + escapeAppleScript(coalesce(data.LabelOK, "OK")) + `"` + iconClause + `
end try
`
}

