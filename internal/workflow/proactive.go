package workflow

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/museigen/lore/internal/domain"
	"github.com/museigen/lore/internal/generator"
	"github.com/museigen/lore/internal/storage"
	loretemplate "github.com/museigen/lore/internal/template"
	"github.com/museigen/lore/internal/ui"
)

// ProactiveOpts holds pre-filled arguments from the CLI for lore new.
type ProactiveOpts struct {
	Type string // pre-filled type (may be empty)
	What string // pre-filled what (may be empty)
	Why  string // pre-filled why (may be empty)
}

// validDocTypes lists the accepted document types for argument validation.
var validDocTypes = map[string]bool{
	domain.DocTypeDecision: true,
	domain.DocTypeFeature:  true,
	domain.DocTypeBugfix:   true,
	domain.DocTypeRefactor: true,
	domain.DocTypeRelease:  true,
	domain.DocTypeNote:     true,
}

// HandleProactive runs the manual documentation flow for `lore new`.
// Unlike HandleReactive, there is no commit context, no express mode,
// and generated_by is "manual".
// CONSOLIDATE: extraire helper si duplication significative avec reactive.go
func HandleProactive(ctx context.Context, workDir string, streams domain.IOStreams, opts ProactiveOpts) error {
	renderer := NewRenderer(streams)
	// Express mode is structurally disabled: runProactiveQuestions calls Ask* methods
	// directly and never invokes RunFlow(), so no timer-based express skip can trigger.
	flow := NewQuestionFlow(streams, renderer)

	answers, err := runProactiveQuestions(ctx, flow, opts)
	if err != nil {
		// H2: save partial answers on Ctrl+C so they are not silently lost.
		// No commit hash in manual mode — SavePending uses "unknown-{timestamp}" fallback.
		if ctx.Err() != nil {
			record := BuildPendingRecord(answers, "", "", "interrupted", "partial")
			_ = SavePending(workDir, record) // best-effort
		}
		return fmt.Errorf("workflow: proactive: %w", err)
	}

	loreDir := filepath.Join(workDir, ".lore")
	engine, err := loretemplate.New(
		filepath.Join(loreDir, "templates"),
		loretemplate.GlobalDir(),
	)
	if err != nil {
		return fmt.Errorf("workflow: proactive: template engine: %w", err)
	}

	// M7: pass "manual" so generated_by front-matter is correct (AC-3).
	input := answers.ToGenerateInput(nil, "manual")
	genResult, err := generator.Generate(ctx, engine, input)
	if err != nil {
		return fmt.Errorf("workflow: proactive: generate: %w", err)
	}

	docsDir := filepath.Join(loreDir, "docs")
	result, err := storage.WriteDoc(docsDir, genResult.Meta, input.What, genResult.Body)
	if err != nil {
		return fmt.Errorf("workflow: proactive: write doc: %w", err)
	}

	ui.Verb(streams, "Captured", result.Filename)
	displayPath, relErr := filepath.Rel(workDir, result.Path)
	if relErr != nil {
		displayPath = result.Path
	}
	fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(displayPath))

	return nil
}

// runProactiveQuestions orchestrates the question flow for proactive mode.
// Pre-filled args skip the corresponding prompt; invalid types fall back
// to interactive. Express mode is disabled (no timer in manual mode).
func runProactiveQuestions(ctx context.Context, flow *QuestionFlow, opts ProactiveOpts) (Answers, error) {
	var answers Answers

	// --- Question 1: Type ---
	if opts.Type != "" && validDocTypes[opts.Type] {
		// Valid type provided → skip prompt
		answers.Type = opts.Type
		flow.renderer.QuestionConfirm("Type", opts.Type)
	} else {
		flow.renderer.Progress(1, 3, "Type")
		defaultType := "" // AC-1: no pre-fill in manual mode — "Type n'est PAS pre-rempli"
		if opts.Type != "" {
			// Invalid type provided → fallback to "note" default (Task 3.3)
			defaultType = "note"
		}
		t, err := flow.AskType(ctx, defaultType)
		if err != nil {
			return Answers{}, fmt.Errorf("question flow: type: %w", err)
		}
		answers.Type = t
		flow.renderer.QuestionConfirm("Type", t)
	}

	// --- Question 2: What ---
	if opts.What != "" {
		// What provided → skip prompt
		answers.What = opts.What
		flow.renderer.QuestionConfirm("What", opts.What)
	} else {
		flow.renderer.Progress(2, 3, "What")
		w, err := flow.AskWhat(ctx, "")
		if err != nil {
			return Answers{}, fmt.Errorf("question flow: what: %w", err)
		}
		answers.What = w
		flow.renderer.QuestionConfirm("What", w)
	}

	// --- Question 3: Why ---
	if opts.Why != "" {
		// Why provided → skip prompt
		answers.Why = opts.Why
		flow.renderer.QuestionConfirm("Why", opts.Why)
	} else {
		flow.renderer.Progress(3, 3, "Why")
		why, err := flow.AskWhy(ctx)
		if err != nil {
			return Answers{}, fmt.Errorf("question flow: why: %w", err)
		}
		answers.Why = why
		flow.renderer.QuestionConfirm("Why", why)
	}

	// --- Question 4: Alternatives (always interactive — no express skip) ---
	alt, err := flow.AskAlternatives(ctx)
	if err != nil {
		return Answers{}, fmt.Errorf("question flow: alternatives: %w", err)
	}
	answers.Alternatives = alt
	if alt != "" {
		flow.renderer.QuestionConfirm("Alternatives", alt)
	}

	// --- Question 5: Impact (always interactive — no express skip) ---
	imp, err := flow.AskImpact(ctx)
	if err != nil {
		return Answers{}, fmt.Errorf("question flow: impact: %w", err)
	}
	answers.Impact = imp
	if imp != "" {
		flow.renderer.QuestionConfirm("Impact", imp)
	}

	return answers, nil
}
