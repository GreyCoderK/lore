// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/notify"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// HandleReactive runs the full interactive post-commit flow:
//  1. Detects context (merge, rebase, cherry-pick, amend, non-TTY, doc-skip).
//  2. Reads HEAD commit info via gitAdapter.
//  3. Presents the question flow on streams.
//  4. Generates and persists the document under .lore/docs/.
//  5. Prints "Captured {filename}" on stderr.
//
// On context cancellation (Ctrl+C forwarded via signal.NotifyContext in main),
// any partial answers collected before the interruption are saved to
// .lore/pending/{hash}.yaml (silent best-effort).
// HandleReactive runs the full interactive post-commit flow with default
// detection options. Use handleReactiveWithOpts for injection in tests.
func HandleReactive(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter) error {
	return HandleReactiveWithEngine(ctx, workDir, streams, gitAdapter, nil, nil)
}

// HandleReactiveWithEngine runs the full interactive post-commit flow with optional Decision Engine and store.
func HandleReactiveWithEngine(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, engine *decision.Engine, store domain.LoreStore) error {
	opts := DetectOpts{Store: store}
	if engine != nil {
		opts.Engine = engine
	}
	return handleReactiveWithOpts(ctx, workDir, streams, gitAdapter, opts, store)
}

func handleReactiveWithOpts(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, detectOpts DetectOpts, store domain.LoreStore) error {
	// --- 1. Resolve HEAD commit ---
	commit, ref, err := resolveHeadCommit(gitAdapter, streams)
	if err != nil {
		return err
	}

	// H1/H2 fix: provide corpus reader for doc existence checks (AC-4, AC-5).
	if detectOpts.Corpus == nil {
		detectOpts.Corpus = &storage.CorpusStore{Dir: domain.DocsPath(workDir)}
	}

	// Build SignalContext for Decision Engine if available
	if detectOpts.Engine != nil && commit != nil && detectOpts.SignalCtx == nil {
		diffContent, _ := gitAdapter.Diff(ref) // best-effort
		detectOpts.SignalCtx = buildSignalContext(commit, diffContent)
	}

	// N2 fix: normalize IsTTY BEFORE Detect so both Detect and showMilestone
	// share a single evaluation — avoids double-call divergence risk.
	if detectOpts.IsTTY == nil {
		isTTY := IsInteractiveTTY(streams)
		detectOpts.IsTTY = func(_ domain.IOStreams) bool { return isTTY }
	}
	tty := detectOpts.IsTTY(streams)

	// --- 1b. Contextual detection (AC-1 through AC-6) ---
	detection, err := Detect(ctx, ref, gitAdapter, streams, detectOpts)
	if err != nil {
		// Context cancelled during detection — save a pending record so the commit
		// is not silently lost (same behaviour as interruption mid-flow).
		if ctx.Err() != nil {
			hash, msg := commitFields(commit)
			record := BuildPendingRecord(Answers{}, hash, msg, "interrupted", "partial")
			_ = SavePending(workDir, record) // best-effort
		}
		return fmt.Errorf("workflow: reactive: detect: %w", err)
	}

	return handleDetectionResult(ctx, workDir, streams, gitAdapter, store, commit, detection, tty, detectOpts.NotifyConfig, detectOpts.AmendPrompt)
}

// resolveHeadCommit gets HEAD commit info in a single git call via HeadCommit().
// Falls back to separate HeadRef() + Log() if HeadCommit() fails.
// Returns the commit info (may be nil on log failure), the ref hash, and any fatal error.
func resolveHeadCommit(gitAdapter domain.GitAdapter, streams domain.IOStreams) (*domain.CommitInfo, string, error) {
	commit, err := gitAdapter.HeadCommit()
	if err == nil && commit != nil {
		return commit, commit.Hash, nil
	}

	// Fallback: HeadCommit failed, try separate calls.
	ref, err := gitAdapter.HeadRef()
	if err != nil {
		return nil, "", fmt.Errorf("workflow: reactive: head ref: %w", err)
	}

	commit, logErr := gitAdapter.Log(ref)
	if logErr != nil {
		// Log failure is non-fatal but worth surfacing — documents created without
		// commit metadata are harder to trace.
		_, _ = fmt.Fprintf(streams.Err, "Warning: could not read commit %s: %v\n", ref, logErr)
	}
	return commit, ref, nil
}

// buildSignalContext constructs a decision.SignalContext from commit info and diff content.
func buildSignalContext(commit *domain.CommitInfo, diffContent string) *decision.SignalContext {
	var filesChanged []string
	if diffContent != "" {
		filesChanged = ExtractFilesFromDiff(diffContent)
	}
	linesAdded, linesDeleted := CountDiffLines(diffContent)
	return &decision.SignalContext{
		ConvType:     commit.Type,
		Scope:        commit.Scope,
		Subject:      commit.Subject,
		Message:      commit.Message,
		DiffContent:  diffContent,
		FilesChanged: filesChanged,
		LinesAdded:   linesAdded,
		LinesDeleted: linesDeleted,
	}
}

// handleDetectionResult processes the detection action (skip, defer, amend, etc.)
// and either terminates the flow or continues to the documentation flow.
func handleDetectionResult(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, store domain.LoreStore, commit *domain.CommitInfo, detection DetectionResult, tty bool, notifyCfg *notify.NotifyConfig, amendPrompt *bool) error {
	switch detection.Action {
	case "skip":
		if detection.Message != "" {
			_, _ = fmt.Fprintln(streams.Err, detection.Message)
		}
		skipType := "skipped"
		if detection.Reason == "merge" {
			skipType = "merge-skipped"
		}
		recordDecision(store, commit, detection, skipType)
		return nil
	case "defer":
		hash, msg := commitFields(commit)
		record := BuildPendingRecord(Answers{}, hash, msg, detection.Reason, "deferred")
		if err := SavePending(workDir, record); err != nil {
			return fmt.Errorf("workflow: reactive: defer: %w", err)
		}
		recordDecision(store, commit, detection, "pending")

		// ADR-023: notify the developer via IDE terminal or OS dialog.
		// Best-effort: notification failure is not an error.
		if detection.Reason == "non-tty" {
			notifyAfterDefer(hash, msg, commit, detection, notifyCfg)
		}
		return nil
	case "amend":
		err := handleAmend(ctx, workDir, streams, gitAdapter, commit, tty, amendPrompt)
		if err == nil {
			recordDecision(store, commit, detection, "documented")
		}
		return err
	case "auto-skip":
		if detection.Message != "" {
			_, _ = fmt.Fprintln(streams.Err, detection.Message)
		}
		recordDecision(store, commit, detection, "auto-skipped")
		return nil
	case "suggest-skip":
		if ctx.Err() != nil {
			recordDecision(store, commit, detection, "skipped")
			return ctx.Err()
		}
		_, _ = fmt.Fprintf(streams.Err, "📝 %s", i18n.T().Workflow.SuggestSkipPrompt)
		reader := bufio.NewReader(streams.In)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(strings.ToLower(line))
		if line != "y" && line != "yes" && line != "o" && line != "oui" {
			_, _ = fmt.Fprintln(streams.Err, i18n.T().Workflow.SuggestSkipSkipped)
			recordDecision(store, commit, detection, "skipped")
			return nil
		}
		// User said yes → proceed as ask-reduced
		detection.QuestionMode = "reduced"
		// fall through to documentation flow
	case "ask-reduced", "ask-full", "proceed":
		// fall through to documentation flow
	default:
		// Unknown action — treat as proceed (defensive fallback)
		_, _ = fmt.Fprintf(streams.Err, "Warning: unknown detection action %q, proceeding with documentation flow\n", detection.Action)
	}

	// Pre-flight: verify pipeline can succeed before asking questions.
	if err := PreflightCheck(workDir); err != nil {
		hash, msg := commitFields(commit)
		record := BuildPendingRecord(Answers{}, hash, msg, "preflight-error", "deferred")
		_ = SavePending(workDir, record)
		_, _ = fmt.Fprintf(streams.Err, "%s: %v\n", i18n.T().Workflow.PreflightFailed, err)
		_, _ = fmt.Fprintln(streams.Err, i18n.T().Workflow.PreflightPendingSaved)
		return nil // don't block the git commit
	}

	// --- 2-5. Question flow + generate + persist ---
	result, err := runDocumentationFlow(ctx, workDir, streams, commit, "", detection)
	if err != nil {
		return err
	}

	recordDecision(store, commit, detection, "documented")
	displayCompletion(streams, result, "Captured", workDir, tty)

	return nil
}

// recordDecision persists the decision to the LKS store (best-effort, nil-safe).
func recordDecision(store domain.LoreStore, commit *domain.CommitInfo, detection DetectionResult, decisionType string) {
	if store == nil || commit == nil {
		return
	}
	rec := domain.CommitRecord{
		Hash:         commit.Hash,
		Date:         commit.Date,
		Branch:       commit.Branch,
		Scope:        commit.Scope,
		ConvType:     commit.Type,
		Subject:      commit.Subject,
		Message:      commit.Message,
		Decision:     decisionType,
		DecisionScore: detection.Score,
		QuestionMode: detection.QuestionMode,
		SkipReason:   detection.Reason,
	}
	_ = store.RecordCommit(rec) // best-effort, never block the hook
}

// commitFields extracts hash and message from a CommitInfo, safely handling nil.
func commitFields(commit *domain.CommitInfo) (hash, msg string) {
	if commit == nil {
		return "", ""
	}
	return commit.Hash, commit.Message
}

// runDocumentationFlow orchestrates the question flow, document generation, and
// persistence. If overwritePath is non-empty the document is written to that
// exact path (atomic overwrite) rather than creating a new file via WriteDoc.
// detection carries QuestionMode and pre-filled answers from the Decision Engine.
func runDocumentationFlow(ctx context.Context, workDir string, streams domain.IOStreams, commit *domain.CommitInfo, overwritePath string, detection ...DetectionResult) (storage.WriteResult, error) {
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	// Apply pre-fill and reduced mode from Decision Engine
	var prefill *DetectionResult
	if len(detection) > 0 {
		prefill = &detection[0]
	}
	answers, flowErr := flow.RunFlowWithMode(ctx, commit, prefill)
	if flowErr != nil {
		// Context cancelled → persist partial answers silently.
		if ctx.Err() != nil {
			hash, msg := commitFields(commit)
			record := BuildPendingRecord(answers, hash, msg, "interrupted", "partial")
			_ = SavePending(workDir, record) // best-effort
		}
		return storage.WriteResult{}, fmt.Errorf("workflow: question flow: %w", flowErr)
	}

	return generateAndWrite(ctx, workDir, answers, commit, "hook", overwritePath)
}

// handleAmend handles AC-4: when amending, find the pre-amend document and
// propose updating it instead of creating a new one.
//
// Strategy:
//  1. Read ORIG_HEAD from the git directory to get the pre-amend commit hash.
//  2. Scan .lore/docs/ for a document whose front-matter commit field matches.
//  3. If found: run the question flow and overwrite that document.
//  4. If not found: run the normal flow (create a new document).
func handleAmend(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, commit *domain.CommitInfo, tty bool, amendPrompt *bool) error {
	docsDir := domain.DocsPath(workDir)

	// Question 0: ask "Document this change?" before the amend flow.
	// Default: true (prompt). Set hooks.amend_prompt=false to skip.
	shouldPrompt := amendPrompt == nil || *amendPrompt
	if shouldPrompt && tty {
		_, _ = fmt.Fprintf(streams.Err, "%s", i18n.T().Workflow.AmendQuestion0)
		answer, readErr := readAmendAnswer(streams)
		if readErr != nil || answer == "n" || answer == "no" || answer == "non" {
			return nil // skip silently
		}
	}

	// Locate pre-amend hash via ORIG_HEAD.
	origHash := readORIGHEAD(gitAdapter)

	// Search for an existing doc with the pre-amend hash.
	var existingDoc *domain.DocMeta
	var existingFilename string
	if origHash != "" {
		corpusStore := &storage.CorpusStore{Dir: docsDir}
		docs, _ := corpusStore.ListDocs(domain.DocFilter{})
		for i, doc := range docs {
			if doc.Commit == origHash {
				existingFilename = doc.Filename
				existingDoc = &docs[i]
				break
			}
		}
	}

	if existingFilename != "" && tty {
		// [U]pdate / [C]reate / [S]kip choice
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Workflow.AmendUpdatingExisting+"\n", existingFilename)
		_, _ = fmt.Fprintf(streams.Err, "%s", i18n.T().Workflow.AmendChoicePrompt)
		choice, readErr := readAmendAnswer(streams)
		if readErr != nil {
			return nil
		}
		switch choice {
		case "s", "skip", "i", "ignorer":
			return nil
		case "c", "create":
			existingDoc = nil
			existingFilename = ""
		// "u", "update", "m" (mettre à jour), or default → update
		}
	} else if existingFilename != "" {
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Workflow.AmendUpdatingExisting+"\n", existingFilename)
	} else if origHash != "" {
		shortHash := origHash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Workflow.AmendNoDocCreatingNew+"\n", shortHash)
	}

	// H2 fix: delegate to the shared helper instead of duplicating the flow.
	var overwritePath string
	if existingFilename != "" {
		overwritePath = filepath.Join(docsDir, existingFilename)
	}

	// Pre-fill from existing doc when updating.
	var detection DetectionResult
	detection.QuestionMode = "reduced"
	if existingDoc != nil {
		detection.PrefilledWhat = storage.ExtractSlug(existingDoc.Filename)
		// Read body to extract existing Why section for pre-fill.
		if content, readErr := storage.ReadDocContent(filepath.Join(docsDir, existingFilename)); readErr == nil {
			if why := extractWhy(content); why != "" {
				detection.PrefilledWhy = why
				detection.PrefilledWhyConfidence = 0.9
			}
		}
	}

	result, err := runDocumentationFlow(ctx, workDir, streams, commit, overwritePath, detection)
	if err != nil {
		return fmt.Errorf("workflow: amend: %w", err)
	}

	verb := "Captured"
	if existingFilename != "" {
		verb = "Updated"
	}
	displayCompletion(streams, result, verb, workDir, tty)

	return nil
}

// readAmendAnswer reads a single line from stdin for amend prompt responses.
func readAmendAnswer(streams domain.IOStreams) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := streams.In.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				break
			}
			buf = append(buf, b[0])
		}
		if err != nil {
			return strings.TrimSpace(strings.ToLower(string(buf))), err
		}
	}
	return strings.TrimSpace(strings.ToLower(string(buf))), nil
}

// extractWhy extracts the "Why" section content from a doc body.
func extractWhy(content string) string {
	lines := strings.Split(content, "\n")
	inWhy := false
	var whyLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "## Why") {
			inWhy = true
			continue
		}
		if inWhy && strings.HasPrefix(line, "## ") {
			break
		}
		if inWhy {
			whyLines = append(whyLines, line)
		}
	}
	return strings.TrimSpace(strings.Join(whyLines, "\n"))
}

// readORIGHEAD reads the pre-amend commit hash from .git/ORIG_HEAD.
// Returns empty string if not found or on error.
func readORIGHEAD(gitAdapter domain.GitAdapter) string {
	gitDir, err := gitAdapter.GitDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(gitDir, "ORIG_HEAD"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// ExtractFilesFromDiff parses file names from unified diff output (+++ b/ lines).
func ExtractFilesFromDiff(diff string) []string {
	seen := make(map[string]bool, 16)
	files := make([]string, 0, 16)
	for _, line := range strings.Split(diff, "\n") {
		if len(line) > 6 && line[:6] == "+++ b/" {
			f := line[6:]
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	return files
}

// CountDiffLines counts added and deleted lines from unified diff output.
// Limited to first 10000 lines for performance on very large diffs.
func CountDiffLines(diff string) (added, deleted int) {
	lines := strings.SplitN(diff, "\n", 10001)
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deleted++
		}
	}
	return
}

// notifyAfterDefer sends a notification to the developer after a commit is
// deferred to pending in a non-TTY environment (ADR-023).
// Best-effort: runs in the current goroutine but does not block (all commands
// are detached via cmd.Start()). Errors are silently ignored.
func notifyAfterDefer(hash, commitMsg string, commit *domain.CommitInfo, detection DetectionResult, notifyCfg *notify.NotifyConfig) {
	env := notify.DetectEnvironment(notify.EnvOpts{})

	// CI environments are already handled by Detect() (action=defer with reason=non-tty).
	// But double-check: never notify in CI.
	if env == notify.EnvCI {
		return
	}

	prefillType := ""
	if commit != nil {
		prefillType = mapConvTypeToDocType(commit.Type)
	}

	cfg := notify.DefaultNotifyConfig()
	if notifyCfg != nil {
		cfg = *notifyCfg
	}

	notify.NotifyNonTTY(
		hash, env, commitMsg, "",
		prefillType,
		detection.PrefilledWhat,
		detection.PrefilledWhy,
		notify.NotifyOpts{
			Config: cfg,
			I18nLabels: func(data *notify.DialogData) {
				n := i18n.T().Notify
				data.LabelTitle = n.DialogTitle
				data.LabelTitleWhat = n.DialogTitleWhat
				data.LabelTitleWhy = n.DialogTitleWhy
				data.LabelType = n.PromptType
				data.LabelWhat = n.PromptWhat
				data.LabelWhy = n.PromptWhy
				data.LabelCancel = n.ButtonCancel
				data.LabelNext = n.ButtonNext
				data.LabelSave = n.ButtonSave
				data.LabelSkip = n.ButtonSkip
				data.LabelOK = n.ButtonOK
				data.LabelError = n.ErrorPrefix
				data.LabelErrResolve = n.ErrorResolve
				// Branch Awareness: propagate branch/scope to dialog context.
				if commit != nil {
					data.Branch = commit.Branch
					data.Scope = commit.Scope
				}
			},
		},
	)
}

// mapConvTypeToDocType maps conventional commit types to Lore doc types.
func mapConvTypeToDocType(convType string) string {
	switch convType {
	case "feat":
		return "feature"
	case "fix":
		return "bugfix"
	case "refactor":
		return "refactor"
	case "docs", "style", "ci", "build", "chore":
		return "note"
	default:
		return ""
	}
}
