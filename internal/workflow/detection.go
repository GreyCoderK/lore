// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/notify"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// DetectionAction represents the possible outcomes of commit detection.
type DetectionAction = string

const (
	ActionProceed     DetectionAction = "proceed"
	ActionSkip        DetectionAction = "skip"
	ActionDefer       DetectionAction = "defer"
	ActionAmend       DetectionAction = "amend"
	ActionAutoSkip    DetectionAction = "auto-skip"
	ActionSuggestSkip DetectionAction = "suggest-skip"
	ActionAskReduced  DetectionAction = "ask-reduced"
	ActionAskFull     DetectionAction = "ask-full"
)

// QuestionMode controls the depth of interactive questioning.
type QuestionMode = string

const (
	QModeFull    QuestionMode = "full"
	QModeReduced QuestionMode = "reduced"
	QModeConfirm QuestionMode = "confirm"
	QModeNone    QuestionMode = "none"
)

// DetectionResult describes how the hook should handle the current commit.
type DetectionResult struct {
	Action                string // "proceed", "skip", "defer", "amend", "auto-skip", "suggest-skip", "ask-reduced", "ask-full"
	Reason                string
	Message               string  // human-readable message for stderr (empty = silent)
	Score                 int     // 0-100 from Decision Engine (0 if no scoring)
	QuestionMode          string  // full, reduced, confirm, none
	PrefilledWhat         string
	PrefilledWhy          string
	PrefilledWhyConfidence float64
}

// DetectOpts holds injectable dependencies for testability (NOTE m22).
type DetectOpts struct {
	// GetEnv reads an environment variable. Defaults to os.Getenv.
	GetEnv func(string) string

	// IsTTY reports whether the given streams represent an interactive TTY.
	// Defaults to IsInteractiveTTY (which delegates to ui.IsTerminal).
	// M2 fix: replaces the ForceInteractive bool antipattern — tests inject
	// func(_ domain.IOStreams) bool { return true } to bypass TTY detection
	// without polluting production code with a test-only flag.
	IsTTY func(domain.IOStreams) bool

	// Corpus provides doc existence checks for cherry-pick (AC-5) and amend
	// (AC-4) detection. When nil, these checks are skipped and the old
	// unconditional skip/amend behavior applies (backward compat for tests
	// that don't need doc existence verification).
	Corpus domain.CorpusReader

	// Store provides O(1) doc lookup by commit hash. When nil, falls back to corpus scan.
	Store domain.LoreStore

	// Engine is the Decision Engine for multi-signal scoring.
	// When nil, step 7 is skipped and fallback proceed applies (backward compat).
	Engine *decision.Engine

	// SignalCtx holds pre-built signal context for the Decision Engine.
	// Only used when Engine is non-nil.
	SignalCtx *decision.SignalContext

	// NotifyConfig holds notification preferences from .lorerc.
	// Used by handleDetectionResult to configure non-TTY notifications (ADR-023).
	// When nil, DefaultNotifyConfig() is used.
	NotifyConfig *notify.NotifyConfig

	// AmendPrompt controls whether to ask "Document this change?" (Question 0)
	// before the amend flow in TTY mode. Defaults to true.
	// Set to false via hooks.amend_prompt=false in .lorerc to skip the prompt.
	AmendPrompt *bool
}

// Detect determines the appropriate action for the current commit context.
//
// Detection order (first match wins — priority is deterministic per Dev Notes):
//  1. [doc-skip] marker   → skip silently  (explicit developer intent, exit 0)
//  2. Non-TTY / TERM=dumb → defer pending  (CI must never block)
//  3. Rebase in progress  → defer pending  (avoid questionnaire per replay)
//  4. Merge commit        → skip with 1-line stderr message
//  5. Cherry-pick         → skip silently  (CHERRY_PICK_HEAD present)
//  6. Amend               → propose modification of existing doc
//  7. Otherwise           → proceed with normal interactive flow
//
// GitAdapter methods do NOT accept ctx (per architecture.md NOTE C2).
// Cancellation is managed at the workflow/cobra level.
func Detect(ctx context.Context, ref string, git domain.GitAdapter, streams domain.IOStreams, opts DetectOpts) (DetectionResult, error) {
	// L1 fix: check for context cancellation before performing any I/O.
	if err := ctx.Err(); err != nil {
		return DetectionResult{}, err
	}

	// Validate ref: an empty ref cannot be resolved by any git command, so bail
	// out early rather than producing confusing downstream errors.
	if ref == "" {
		return DetectionResult{Action: ActionSkip, Reason: "empty-ref", Message: "Warning: empty commit ref, skipping"}, nil
	}

	if opts.GetEnv == nil {
		opts.GetEnv = os.Getenv
	}

	// 1. [doc-skip] — highest priority: explicit developer opt-out.
	docSkip, err := git.CommitMessageContains(ref, "[doc-skip]")
	if err != nil {
		return DetectionResult{}, fmt.Errorf("workflow: detect: doc-skip: %w", err)
	}
	if docSkip {
		return DetectionResult{Action: "skip", Reason: "doc-skip"}, nil
	}

	// 2. Non-TTY / TERM=dumb — CI/pipe environments must never block.
	isTTY := opts.IsTTY
	if isTTY == nil {
		isTTY = IsInteractiveTTY
	}
	if !isTTY(streams) {
		return DetectionResult{Action: "defer", Reason: "non-tty"}, nil
	}

	// 3. Rebase in progress — batch-defer all replayed commits.
	rebase, err := git.IsRebaseInProgress()
	if err != nil {
		return DetectionResult{}, fmt.Errorf("workflow: detect: rebase: %w", err)
	}
	if rebase {
		return DetectionResult{Action: "defer", Reason: "rebase"}, nil
	}

	// 4. Merge commit — skip with informational message on stderr.
	merge, err := git.IsMergeCommit(ref)
	if err != nil {
		return DetectionResult{}, fmt.Errorf("workflow: detect: merge: %w", err)
	}
	if merge {
		return DetectionResult{
			Action:  "skip",
			Reason:  "merge",
			Message: i18n.T().Workflow.MergeCommitSkipMsg,
		}, nil
	}

	// 5. Cherry-pick — AC-5: skip only when a doc exists for the source commit.
	// H1 fix: read the hash from CHERRY_PICK_HEAD and verify a matching doc
	// exists in the corpus before skipping. Without Corpus (nil), falls back
	// to unconditional skip (backward compat).
	sourceHash := cherryPickSourceHash(git)
	if sourceHash != "" {
		if opts.Corpus == nil || hasDocForCommit(opts.Store, opts.Corpus, sourceHash) {
			return DetectionResult{Action: "skip", Reason: "cherry-pick"}, nil
		}
		// Cherry-pick in progress but no doc for source → proceed normally.
	}

	// 6. Amend — AC-4: propose modification only when a doc exists for the
	// pre-amend commit. H2 fix: read ORIG_HEAD and verify a matching doc
	// exists before returning "amend". Without Corpus (nil), falls back to
	// unconditional amend (backward compat).
	if isAmendCommit(opts.GetEnv) {
		if opts.Corpus == nil {
			return DetectionResult{Action: "amend", Reason: "amend"}, nil
		}
		origHash := readORIGHEAD(git)
		if origHash != "" && hasDocForCommit(opts.Store, opts.Corpus, origHash) {
			return DetectionResult{Action: "amend", Reason: "amend"}, nil
		}
		// Amend but no existing doc → proceed (create new doc).
		return DetectionResult{Action: "proceed"}, nil
	}

	// 7. Decision Engine — multi-signal scoring.
	if opts.Engine != nil && opts.SignalCtx != nil {
		result := opts.Engine.Evaluate(*opts.SignalCtx)
		dr := DetectionResult{
			Action:                 result.Action,
			Reason:                 "decision-engine",
			Score:                  result.Score,
			QuestionMode:           questionModeFromAction(result.Action),
			PrefilledWhat:          result.PrefilledWhat,
			PrefilledWhy:           result.PrefilledWhy,
			PrefilledWhyConfidence: result.PrefilledWhyConfidence,
		}
		if result.Action == "auto-skip" {
			dr.Message = fmt.Sprintf("⏭ "+i18n.T().Workflow.AutoSkipMsg, result.Score, opts.SignalCtx.Subject)
		}
		return dr, nil
	}

	// 8. Normal commit — proceed with interactive question flow (fallback when no engine).
	return DetectionResult{Action: "proceed"}, nil
}

func questionModeFromAction(action string) string {
	switch action {
	case "ask-full":
		return "full"
	case "ask-reduced":
		return "reduced"
	case "suggest-skip":
		return "none"
	case "auto-skip":
		return "none"
	default:
		return "full"
	}
}

// cherryPickSourceHash returns the source commit hash if a cherry-pick is in
// progress, or empty string otherwise. Reads CHERRY_PICK_HEAD from the git dir.
// H1 fix: replaces isCherryPickInProgress (bool) — callers now get the hash
// so they can verify whether a doc exists for the source commit (AC-5).
func cherryPickSourceHash(git domain.GitAdapter) string {
	gitDir, err := git.GitDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// hasDocForCommit checks if a document exists for the given commit hash.
// Uses store (O(1) indexed lookup) when available, falls back to corpus scan.
func hasDocForCommit(store domain.LoreStore, corpus domain.CorpusReader, hash string) bool {
	// O(1) path via store
	if store != nil {
		docs, err := store.DocsByCommitHash(hash)
		if err == nil && len(docs) > 0 {
			return true
		}
		// Store error or no match — fall through to corpus scan
	}
	// O(n) fallback via filesystem scan
	if corpus != nil {
		docs, _ := corpus.ListDocs(domain.DocFilter{})
		for _, doc := range docs {
			if doc.Commit == hash {
				return true
			}
		}
	}
	return false
}

// isAmendCommit reports whether the current commit is an amend.
// Uses the injectable getEnv function (pattern from NOTE m22) so tests can
// supply a custom env reader without modifying real environment variables.
// Uses exact equality ("amend") to avoid false positives from rebase operations
// that set GIT_REFLOG_ACTION to values like "rewriting (amend)".
func isAmendCommit(getEnv func(string) string) bool {
	action := getEnv("GIT_REFLOG_ACTION")
	return action == "amend"
}
