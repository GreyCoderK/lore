package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
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

	// Corpus provides doc existence checks for cherry-pick (AC-5) and amend
	// (AC-4) detection. When nil, these checks are skipped and the old
	// unconditional skip/amend behavior applies (backward compat for tests
	// that don't need doc existence verification).
	Corpus domain.CorpusReader
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

	// 5. Cherry-pick — AC-5: skip only when a doc exists for the source commit.
	// H1 fix: read the hash from CHERRY_PICK_HEAD and verify a matching doc
	// exists in the corpus before skipping. Without Corpus (nil), falls back
	// to unconditional skip (backward compat).
	sourceHash := cherryPickSourceHash(git)
	if sourceHash != "" {
		if opts.Corpus == nil || hasDocForCommit(opts.Corpus, sourceHash) {
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
		if origHash != "" && hasDocForCommit(opts.Corpus, origHash) {
			return DetectionResult{Action: "amend", Reason: "amend"}, nil
		}
		// Amend but no existing doc → proceed (create new doc).
		return DetectionResult{Action: "proceed"}, nil
	}

	// 7. Normal commit — proceed with interactive question flow.
	return DetectionResult{Action: "proceed"}, nil
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

// hasDocForCommit scans the corpus for a document matching the given commit hash.
func hasDocForCommit(corpus domain.CorpusReader, hash string) bool {
	docs, _ := corpus.ListDocs(domain.DocFilter{})
	for _, doc := range docs {
		if doc.Commit == hash {
			return true
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
