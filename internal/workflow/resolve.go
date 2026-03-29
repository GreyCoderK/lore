// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// ResolveOpts holds options for ResolvePending.
type ResolveOpts struct {
	IsTTY func(domain.IOStreams) bool // optional TTY override for testing
}

// ResolvePending resolves a pending item: displays commit context, asks only
// remaining questions (preserving partial answers), generates the document
// via the standard pipeline, and deletes the pending file.
func ResolvePending(ctx context.Context, workDir string, streams domain.IOStreams, item PendingItem, gitAdapter domain.GitAdapter, opts ResolveOpts) error {
	pendingDir := filepath.Join(workDir, ".lore", "pending")

	// --- Display commit context ---
	fmt.Fprintf(streams.Err, "\n%s\n", i18n.T().Workflow.ResolveHeader)
	fmt.Fprintf(streams.Err, "  "+i18n.T().Workflow.ResolveCommitLabel+"\n", item.CommitHash)
	fmt.Fprintf(streams.Err, "  "+i18n.T().Workflow.ResolveMessageLabel+"\n", item.CommitMessage)
	fmt.Fprintf(streams.Err, "  "+i18n.T().Workflow.ResolveDateLabel+"\n", item.CommitDate.Format("2006-01-02 15:04"))
	fmt.Fprintf(streams.Err, "\n")

	// --- Try to retrieve full commit info ---
	var commit *domain.CommitInfo
	if item.CommitHash != "" {
		exists, existsErr := gitAdapter.CommitExists(item.CommitHash)
		if existsErr != nil {
			fmt.Fprintf(streams.Err, i18n.T().Workflow.ResolveCheckCommitW+"\n", item.CommitHash, existsErr)
		}
		if exists {
			info, logErr := gitAdapter.Log(item.CommitHash)
			if logErr == nil {
				commit = info
			}
		} else if existsErr == nil {
			fmt.Fprintf(streams.Err, i18n.T().Workflow.ResolveCommitGoneW+"\n\n", item.CommitHash)
		}
	}

	// --- Build pre-filled answers from partial data ---
	answers := Answers{
		Type:         item.Answers.Type,
		What:         item.Answers.What,
		Why:          item.Answers.Why,
		Alternatives: item.Answers.Alternatives,
		Impact:       item.Answers.Impact,
	}

	// --- Ask only remaining questions (pre-filled answers are preserved) ---
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	remaining, err := flow.AskQuestions(ctx, QuestionOpts{
		PreFilled:  answers,
		CommitInfo: commit,
	})
	if err != nil {
		return fmt.Errorf("workflow: resolve pending: %w", err)
	}

	// --- Generate and write document ---
	result, err := generateAndWrite(ctx, workDir, remaining, commit, "pending", "")
	if err != nil {
		return fmt.Errorf("workflow: resolve pending: %w", err)
	}

	// --- Delete pending file only after successful write ---
	if delErr := deletePendingFile(pendingDir, item.Filename); delErr != nil {
		fmt.Fprintf(streams.Err, i18n.T().Workflow.ResolveDeletePendingW+"\n", delErr)
	}

	tty := IsInteractiveTTY(streams)
	if opts.IsTTY != nil {
		tty = opts.IsTTY(streams)
	}
	displayCompletion(streams, result, "Captured", workDir, tty)

	return nil
}

