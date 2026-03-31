// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"fmt"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/engagement"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
)

// showMilestone displays a milestone reinforcement message on stderr when the
// document count in docsDir hits an exact threshold (3, 10, 25, 50).
// When tty is false the call is a no-op (AC-5: non-TTY suppression).
//
// Error handling: ListDocs returns (partialDocs, joinedParseErrors) when some
// files fail to parse, or (nil, nil) when the directory does not exist. Both
// cases are acceptable — we use len(docs) regardless (len(nil) == 0 in Go)
// so that a malformed or missing docs dir silently suppresses milestones
// rather than crashing the workflow.
func showMilestone(streams domain.IOStreams, docsDir string, tty bool) {
	if !tty {
		return
	}
	store := &storage.CorpusStore{Dir: docsDir}
	docs, _ := store.ListDocs(domain.DocFilter{})
	msg, ok := engagement.GetMilestoneMessage(len(docs), milestoneI18N)
	if !ok {
		return
	}
	_, _ = fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(msg))
}

// showStarPrompt displays the one-time star prompt after the milestone check.
// Best-effort: any error reading/writing state is silently ignored.
func showStarPrompt(streams domain.IOStreams, workDir, docsDir string, tty bool) {
	if !tty {
		return
	}
	// Count docs (reuse same pattern as showMilestone).
	store := &storage.CorpusStore{Dir: docsDir}
	docs, _ := store.ListDocs(domain.DocFilter{})
	docCount := len(docs)

	statePath := engagement.StatePath(workDir)
	state := engagement.LoadState(statePath)

	if !engagement.ShouldShowStarPrompt(engagement.StarPromptOpts{
		DocCount:     docCount,
		Threshold:    5, // default, will be overridden when config is threaded
		AlreadyShown: state.StarPromptShown,
		Enabled:      true, // default, will be overridden when config is threaded
		IsTTY:        tty,
		IsQuiet:      false,
	}) {
		return
	}

	msg := i18n.T().Engagement.StarPrompt
	if msg == "" {
		return
	}
	_, _ = fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(msg))

	// Persist so it never shows again.
	state.StarPromptShown = true
	_ = engagement.SaveState(statePath, state) // best-effort
}

// milestoneI18N maps milestone thresholds to i18n catalog strings.
func milestoneI18N(count int) string {
	t := i18n.T().Engagement
	switch count {
	case 3:
		return t.Milestone3
	case 8:
		return t.Milestone8
	case 21:
		return t.Milestone21
	case 55:
		return t.Milestone55
	default:
		return ""
	}
}
