package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/museigen/lore/internal/domain"
	"github.com/museigen/lore/internal/generator"
	"github.com/museigen/lore/internal/storage"
	loretemplate "github.com/museigen/lore/internal/template"
	"github.com/museigen/lore/internal/ui"
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
	return handleReactiveWithOpts(ctx, workDir, streams, gitAdapter, DetectOpts{})
}

func handleReactiveWithOpts(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, detectOpts DetectOpts) error {
	// --- 1. Resolve HEAD commit ---
	ref, err := gitAdapter.HeadRef()
	if err != nil {
		return fmt.Errorf("workflow: reactive: head ref: %w", err)
	}

	commit, _ := gitAdapter.Log(ref) // non-fatal; nil → flow uses defaults

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
	switch detection.Action {
	case "skip":
		if detection.Message != "" {
			fmt.Fprintln(streams.Err, detection.Message)
		}
		return nil
	case "defer":
		hash, msg := commitFields(commit)
		record := BuildPendingRecord(Answers{}, hash, msg, detection.Reason, "deferred")
		if err := SavePending(workDir, record); err != nil {
			return fmt.Errorf("workflow: reactive: defer: %w", err)
		}
		return nil
	case "amend":
		return handleAmend(ctx, workDir, streams, gitAdapter, commit)
	// "proceed" falls through to the normal interactive flow below.
	}

	// --- 2-5. Question flow + generate + persist (H2: shared via helper) ---
	result, err := runDocumentationFlow(ctx, workDir, streams, commit, "")
	if err != nil {
		return err
	}

	ui.Verb(streams, "Captured", result.Filename)
	// M6 fix: display path relative to workDir so it is correct regardless of CWD.
	displayPath, relErr := filepath.Rel(workDir, result.Path)
	if relErr != nil {
		displayPath = result.Path
	}
	fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(displayPath))

	return nil
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
//
// H2 fix: extracted from handleReactiveWithOpts and handleAmend to eliminate ~60
// lines of duplication between the two code paths.
func runDocumentationFlow(ctx context.Context, workDir string, streams domain.IOStreams, commit *domain.CommitInfo, overwritePath string) (storage.WriteResult, error) {
	renderer := NewRenderer(streams)
	flow := NewQuestionFlow(streams, renderer)

	answers, flowErr := flow.RunFlow(ctx, commit)
	if flowErr != nil {
		// Context cancelled → persist partial answers silently.
		if ctx.Err() != nil {
			hash, msg := commitFields(commit)
			record := BuildPendingRecord(answers, hash, msg, "interrupted", "partial")
			_ = SavePending(workDir, record) // best-effort
		}
		return storage.WriteResult{}, fmt.Errorf("workflow: question flow: %w", flowErr)
	}

	loreDir := filepath.Join(workDir, ".lore")
	engine, err := loretemplate.New(
		filepath.Join(loreDir, "templates"),
		loretemplate.GlobalDir(),
	)
	if err != nil {
		return storage.WriteResult{}, fmt.Errorf("workflow: template engine: %w", err)
	}

	// M7 fix: pass "hook" explicitly so the generated_by front-matter field is correct.
	// Story 2.7 proactive flow will pass "manual" via its own call path.
	input := answers.ToGenerateInput(commit, "hook")
	genResult, err := generator.Generate(ctx, engine, input)
	if err != nil {
		// H4 fix: save collected answers as pending so they are not silently lost
		// on template errors, engine failures, or context cancellation during render.
		hash, msg := commitFields(commit)
		record := BuildPendingRecord(answers, hash, msg, "interrupted", "partial")
		_ = SavePending(workDir, record) // best-effort
		return storage.WriteResult{}, fmt.Errorf("workflow: generate: %w", err)
	}

	// H1 fix: use genResult.Meta built inside Generate() — callers no longer
	// reconstruct domain.DocMeta independently (eliminates divergence risk).
	docsDir := filepath.Join(loreDir, "docs")
	if overwritePath != "" {
		// Atomic overwrite of an existing document (amend path).
		data, marshalErr := storage.Marshal(genResult.Meta, genResult.Body)
		if marshalErr != nil {
			return storage.WriteResult{}, fmt.Errorf("workflow: marshal: %w", marshalErr)
		}
		if writeErr := storage.AtomicWrite(overwritePath, data); writeErr != nil {
			return storage.WriteResult{}, fmt.Errorf("workflow: write: %w", writeErr)
		}
		return storage.WriteResult{Filename: filepath.Base(overwritePath), Path: overwritePath}, nil
	}

	result, err := storage.WriteDoc(docsDir, genResult.Meta, input.What, genResult.Body)
	if err != nil {
		return storage.WriteResult{}, fmt.Errorf("workflow: write doc: %w", err)
	}
	return result, nil
}

// handleAmend handles AC-4: when amending, find the pre-amend document and
// propose updating it instead of creating a new one.
//
// Strategy:
//  1. Read ORIG_HEAD from the git directory to get the pre-amend commit hash.
//  2. Scan .lore/docs/ for a document whose front-matter commit field matches.
//  3. If found: run the question flow and overwrite that document.
//  4. If not found: run the normal flow (create a new document).
func handleAmend(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, commit *domain.CommitInfo) error {
	loreDir := filepath.Join(workDir, ".lore")
	docsDir := filepath.Join(loreDir, "docs")

	// Locate pre-amend hash via ORIG_HEAD.
	origHash := readORIGHEAD(gitAdapter)

	// Search for an existing doc with the pre-amend hash.
	var existingFilename string
	if origHash != "" {
		store := &storage.CorpusStore{Dir: docsDir}
		docs, _ := store.ListDocs(domain.DocFilter{}) // non-fatal scan
		for _, doc := range docs {
			if doc.Commit == origHash {
				existingFilename = doc.Filename
				break
			}
		}
	}

	if existingFilename != "" {
		fmt.Fprintf(streams.Err, "Amend detected — updating existing document: %s\n", existingFilename)
	} else if origHash != "" {
		// L10 fix: inform the user when amend detection fired but no existing document
		// was found — the silent fallback to creating a new doc was confusing.
		shortHash := origHash
		if len(shortHash) > 7 {
			shortHash = shortHash[:7]
		}
		fmt.Fprintf(streams.Err, "Amend detected — no existing document found for %s, creating new document.\n", shortHash)
	}

	// H2 fix: delegate to the shared helper instead of duplicating the flow.
	var overwritePath string
	if existingFilename != "" {
		overwritePath = filepath.Join(docsDir, existingFilename)
	}
	result, err := runDocumentationFlow(ctx, workDir, streams, commit, overwritePath)
	if err != nil {
		return fmt.Errorf("workflow: amend: %w", err)
	}

	verb := "Captured"
	if existingFilename != "" {
		verb = "Updated"
	}
	ui.Verb(streams, verb, result.Filename)
	// M6 fix: display path relative to workDir.
	displayPath, relErr := filepath.Rel(workDir, result.Path)
	if relErr != nil {
		displayPath = result.Path
	}
	fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(displayPath))

	return nil
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
