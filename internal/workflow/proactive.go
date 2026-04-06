// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
)

// ProactiveOpts holds pre-filled arguments from the CLI for lore new.
type ProactiveOpts struct {
	Type   string                       // pre-filled type (may be empty)
	What   string                       // pre-filled what (may be empty)
	Why    string                       // pre-filled why (may be empty)
	Commit *domain.CommitInfo           // retroactive mode: resolved commit info (nil → manual mode)
	IsTTY  func(domain.IOStreams) bool  // N4 fix: optional TTY override for testing (nil → IsInteractiveTTY)
}

// HandleProactive runs the manual or retroactive documentation flow for `lore new`.
// When opts.Commit is non-nil, retroactive mode pre-fills Type/What from the commit
// and sets generated_by to "retroactive" with the commit hash in front matter.
func HandleProactive(ctx context.Context, workDir string, streams domain.IOStreams, opts ProactiveOpts) error {
	var overwritePath string

	// Pre-fill from commit info in retroactive mode (AC-1)
	if opts.Commit != nil {
		if opts.Type == "" && opts.Commit.Type != "" {
			mapped := MapCommitType(opts.Commit.Type)
			if domain.ValidDocType(mapped) {
				opts.Type = mapped
			}
		}
		if opts.What == "" && opts.Commit.Subject != "" {
			opts.What = opts.Commit.Subject
		}

		// AC-4: check if commit is already documented
		docsDir := domain.DocsPath(workDir)
		existing, findErr := storage.FindDocByCommit(docsDir, opts.Commit.Hash)
		if findErr != nil {
			_, _ = fmt.Fprintf(streams.Err, "Warning: %v\n", findErr)
		}
		if existing != nil {
			fmt.Fprintf(streams.Err, "%s\n", i18n.T().Workflow.AlreadyDocumented)
			relPath, relErr := filepath.Rel(workDir, existing.Path)
			if relErr != nil {
				relPath = existing.Path
			}
			fmt.Fprintf(streams.Err, "  %s\n", relPath)

			tty := IsInteractiveTTY(streams)
			if opts.IsTTY != nil {
				tty = opts.IsTTY(streams)
			}
			if !tty {
				// AC-4: non-TTY safe default — do not create, show existing path
				return nil
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}
			fmt.Fprintf(streams.Err, "%s", i18n.T().Workflow.AmendChoicePrompt)
			choice, _ := readAmendAnswer(streams)
			switch choice {
			case "s", "skip", "i", "ignorer":
				return nil
			case "u", "update", "m":
				// Pre-fill from existing doc and overwrite
				if existing.Meta.Type != "" && opts.Type == "" {
					opts.Type = string(existing.Meta.Type)
				}
				if opts.What == "" {
					opts.What = existing.Title
				}
				if opts.Why == "" {
					if content, readErr := storage.ReadDocContent(existing.Path); readErr == nil {
						if why := extractWhy(content); why != "" {
							opts.Why = why
						}
					}
				}
				// Will overwrite existing doc below
				overwritePath = existing.Path
			default:
				// "c", "create", or Enter → create new doc (no overwrite)
			}
		}
	}

	// Pre-flight: verify pipeline can succeed before asking questions.
	if err := PreflightCheck(workDir); err != nil {
		return fmt.Errorf("workflow: proactive: %w", err)
	}

	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	// PreFilled skips questions entirely; CommitInfo provides interactive defaults
	// for any remaining empty fields. Both are set intentionally: retroactive mode
	// pre-fills Type/What above (skip), while CommitInfo covers edge cases like
	// non-conventional commits where pre-fill didn't set a value.
	qOpts := QuestionOpts{
		PreFilled: Answers{
			Type: opts.Type,
			What: opts.What,
			Why:  opts.Why,
		},
		CommitInfo: opts.Commit,
		// Express mode disabled in proactive — no timer-based skip.
	}

	answers, err := flow.AskQuestions(ctx, qOpts)
	if err != nil {
		// Save partial answers on Ctrl+C so they are not silently lost.
		if ctx.Err() != nil {
			commitHash := ""
			if opts.Commit != nil {
				commitHash = opts.Commit.Hash
			}
			record := BuildPendingRecord(answers, commitHash, "", "interrupted", "partial")
			if saveErr := SavePending(workDir, record); saveErr != nil {
				fmt.Fprintf(streams.Err, "warning: could not save pending answers: %v\n", saveErr)
			}
		}
		return fmt.Errorf("workflow: proactive: %w", err)
	}

	// Determine generatedBy and commit for generateAndWrite
	generatedBy := "manual"
	var commit *domain.CommitInfo
	if opts.Commit != nil {
		generatedBy = "retroactive"
		commit = opts.Commit
	}

	result, err := generateAndWrite(ctx, workDir, answers, commit, generatedBy, overwritePath)
	if err != nil {
		return fmt.Errorf("workflow: proactive: %w", err)
	}

	// N4 fix: use opts.IsTTY when available, fallback to IsInteractiveTTY.
	tty := IsInteractiveTTY(streams)
	if opts.IsTTY != nil {
		tty = opts.IsTTY(streams)
	}
	verb := "Captured"
	if overwritePath != "" {
		verb = "Updated"
	}
	displayCompletion(streams, result, verb, workDir, tty)

	return nil
}

