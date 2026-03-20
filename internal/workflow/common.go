// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/generator"
	"github.com/greycoderk/lore/internal/storage"
	loretemplate "github.com/greycoderk/lore/internal/template"
	"github.com/greycoderk/lore/internal/ui"
)

// displayCompletion shows the "Captured" (or custom verb) message, the dim
// relative path, and the milestone reinforcement. Used by all 4 documentation
// paths after a successful document write.
func displayCompletion(streams domain.IOStreams, result storage.WriteResult, verb string, workDir string, tty bool) {
	if result.IndexErr != nil {
		_, _ = fmt.Fprintf(streams.Err, "Warning: index update failed: %v\n", result.IndexErr)
	}

	ui.Verb(streams, verb, result.Filename)
	displayPath, relErr := filepath.Rel(workDir, result.Path)
	if relErr != nil {
		displayPath = result.Path
	}
	_, _ = fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(displayPath))

	docsDir := filepath.Join(workDir, ".lore", "docs")
	showMilestone(streams, docsDir, tty)
}

// generateAndWrite handles the shared pipeline: template engine init → generate →
// write document. Both reactive (hook) and proactive (lore new) flows delegate here
// after collecting answers.
//
// Parameters:
//   - answers: collected user responses
//   - commit: commit context (nil for proactive/manual mode)
//   - generatedBy: "hook" or "manual" for front-matter
//   - overwritePath: non-empty to atomically overwrite an existing doc (amend path)
//
// On generate or write failure, answers are saved as pending (best-effort).
func generateAndWrite(ctx context.Context, workDir string, answers Answers, commit *domain.CommitInfo, generatedBy string, overwritePath string) (storage.WriteResult, error) {
	loreDir := filepath.Join(workDir, ".lore")
	engine, err := loretemplate.New(
		filepath.Join(loreDir, "templates"),
		loretemplate.GlobalDir(),
	)
	if err != nil {
		return storage.WriteResult{}, fmt.Errorf("workflow: template engine: %w", err)
	}

	input := answers.ToGenerateInput(commit, generatedBy)
	genResult, err := generator.Generate(ctx, engine, input)
	if err != nil {
		// Save collected answers as pending so they are not silently lost
		// on template errors, engine failures, or context cancellation during render.
		hash, msg := commitFields(commit)
		record := BuildPendingRecord(answers, hash, msg, "generate-error", "partial")
		_ = SavePending(workDir, record) // best-effort
		return storage.WriteResult{}, fmt.Errorf("workflow: generate: %w", err)
	}

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
		indexErr := storage.RegenerateIndex(docsDir)
		return storage.WriteResult{Filename: filepath.Base(overwritePath), Path: overwritePath, IndexErr: indexErr}, nil
	}

	result, err := storage.WriteDoc(docsDir, genResult.Meta, input.What, genResult.Body)
	if err != nil {
		// Save collected answers as pending so they are not silently lost
		// on WriteDoc failures (e.g., directory issues, permission errors).
		hash, msg := commitFields(commit)
		record := BuildPendingRecord(answers, hash, msg, "write-error", "partial")
		_ = SavePending(workDir, record) // best-effort
		return storage.WriteResult{}, fmt.Errorf("workflow: write doc: %w", err)
	}
	return result, nil
}
