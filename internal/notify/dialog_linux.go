// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build linux

package notify

import (
	"fmt"
)

// NotifyOSDialog launches a Linux GUI dialog (zenity/kdialog/yad) for Lore documentation.
// Runs as a detached process — does not block the hook.
func NotifyOSDialog(data DialogData, opts DialogOpts) error {
	opts.defaults()

	tool := detectLinuxDialogTool(opts.LookPath)
	if tool == "" {
		return errUnsupportedOS
	}

	script := buildLinuxScript(data, tool)
	return opts.StartCommand("bash", []string{"-c", script}, nil)
}

func detectLinuxDialogTool(lookPath func(string) (string, error)) string {
	for _, tool := range []string{"zenity", "kdialog", "yad"} {
		if _, err := lookPath(tool); err == nil {
			return tool
		}
	}
	return ""
}

func buildLinuxScript(data DialogData, tool string) string {
	switch tool {
	case "zenity":
		return buildZenityScript(data)
	case "kdialog":
		return buildKDialogScript(data)
	default:
		return buildZenityScript(data) // yad is zenity-compatible
	}
}

func buildZenityScript(data DialogData) string {
	hash := sanitizeCommitHash(data.CommitHash)
	title := coalesce(data.LabelTitle, "Lore")
	titleWhat := coalesce(data.LabelTitleWhat, "Lore — What")
	titleWhy := coalesce(data.LabelTitleWhy, "Lore — Why")
	labelType := coalesce(data.LabelType, "Type:")
	labelWhat := coalesce(data.LabelWhat, "What did you change?")
	labelWhy := coalesce(data.LabelWhy, "Why did you make this change?")

	return fmt.Sprintf(`#!/bin/bash
COMMIT_MSG=%s
DIFF_STAT=%s
PREFILL_WHAT=%s
PREFILL_WHY=%s

DOC_TYPE=$(zenity --list --title=%s --text="Commit: $COMMIT_MSG\nDiff: $DIFF_STAT\n\n%s" --column="Type" feature bugfix decision refactor release note 2>/dev/null)
[ -z "$DOC_TYPE" ] && exit 0

WHAT=$(zenity --entry --title=%s --text=%s --entry-text="$PREFILL_WHAT" 2>/dev/null)
[ -z "$WHAT" ] && exit 0

WHY=$(zenity --entry --title=%s --text=%s --entry-text="$PREFILL_WHY" 2>/dev/null)
[ -z "$WHY" ] && exit 0

cd %s && %s pending resolve --commit '%s' --type "$DOC_TYPE" --what "$WHAT" --why "$WHY"
`,
		bashQuote(sanitizeForShell(data.CommitMsg)),
		bashQuote(sanitizeForShell(data.DiffStat)),
		bashQuote(sanitizeForShell(data.PrefillWhat)),
		bashQuote(sanitizeForShell(data.PrefillWhy)),
		bashQuote(title), labelType,
		bashQuote(titleWhat), bashQuote(labelWhat),
		bashQuote(titleWhy), bashQuote(labelWhy),
		bashQuote(sanitizeForShell(data.RepoRoot)),
		bashQuote(sanitizeForShell(data.LorePath)), hash,
	)
}

func buildKDialogScript(data DialogData) string {
	hash := sanitizeCommitHash(data.CommitHash)
	title := coalesce(data.LabelTitle, "Lore")
	titleWhat := coalesce(data.LabelTitleWhat, "Lore — What")
	titleWhy := coalesce(data.LabelTitleWhy, "Lore — Why")
	labelWhat := coalesce(data.LabelWhat, "What did you change?")
	labelWhy := coalesce(data.LabelWhy, "Why did you make this change?")

	return fmt.Sprintf(`#!/bin/bash
COMMIT_MSG=%s
DIFF_STAT=%s

DOC_TYPE=$(kdialog --combobox "Commit: $COMMIT_MSG\nDiff: $DIFF_STAT" feature bugfix decision refactor release note --default bugfix --title %s)
[ -z "$DOC_TYPE" ] && exit 0

WHAT=$(kdialog --inputbox %s %s --title %s)
[ -z "$WHAT" ] && exit 0

WHY=$(kdialog --inputbox %s %s --title %s)
[ -z "$WHY" ] && exit 0

cd %s && %s pending resolve --commit '%s' --type "$DOC_TYPE" --what "$WHAT" --why "$WHY"
`,
		bashQuote(sanitizeForShell(data.CommitMsg)),
		bashQuote(sanitizeForShell(data.DiffStat)),
		bashQuote(title),
		bashQuote(labelWhat), bashQuote(sanitizeForShell(data.PrefillWhat)), bashQuote(titleWhat),
		bashQuote(labelWhy), bashQuote(sanitizeForShell(data.PrefillWhy)), bashQuote(titleWhy),
		bashQuote(sanitizeForShell(data.RepoRoot)),
		bashQuote(sanitizeForShell(data.LorePath)), hash,
	)
}
