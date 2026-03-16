package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/generator"
	"github.com/greycoderk/lore/internal/storage"
	loretemplate "github.com/greycoderk/lore/internal/template"
)

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
		return storage.WriteResult{Filename: filepath.Base(overwritePath), Path: overwritePath}, nil
	}

	result, err := storage.WriteDoc(docsDir, genResult.Meta, input.What, genResult.Body)
	if err != nil {
		return storage.WriteResult{}, fmt.Errorf("workflow: write doc: %w", err)
	}
	return result, nil
}
