package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/museigen/lore/internal/domain"
)

// DetectionResult describes how the hook should handle the current commit.
type DetectionResult struct {
	Action  string // "proceed", "skip", "defer", "amend"
	Reason  string // "doc-skip", "non-tty", "rebase", "merge", "cherry-pick", "amend"
	Message string // human-readable message for stderr (empty = silent)
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
			Message: "Merge commit detected — documentation skipped.",
		}, nil
	}

	// 5. Cherry-pick — skip silently when CHERRY_PICK_HEAD is present.
	if isCherryPickInProgress(git) {
		return DetectionResult{Action: "skip", Reason: "cherry-pick"}, nil
	}

	// 6. Amend — propose modification of existing document.
	if isAmendCommit(opts.GetEnv) {
		return DetectionResult{Action: "amend", Reason: "amend"}, nil
	}

	// 7. Normal commit — proceed with interactive question flow.
	return DetectionResult{Action: "proceed"}, nil
}

// isCherryPickInProgress reports whether a cherry-pick is in progress by
// checking for CHERRY_PICK_HEAD in the git directory. Uses GitAdapter.GitDir()
// so detection.go never hardcodes the .git/ path (per task 1.5).
func isCherryPickInProgress(git domain.GitAdapter) bool {
	gitDir, err := git.GitDir()
	if err != nil {
		return false
	}
	_, statErr := os.Stat(filepath.Join(gitDir, "CHERRY_PICK_HEAD"))
	return statErr == nil
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
