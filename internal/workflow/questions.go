// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

const defaultExpressThreshold = 3 * time.Second

// ccTypeMap maps Conventional Commit prefixes to Lore document types.
var ccTypeMap = map[string]string{
	"feat":     "feature",
	"fix":      "bugfix",
	"refactor": "refactor",
	"docs":     "note",
	"chore":    "note",
	"test":     "note",
	"perf":     "feature",
}

// MapCommitType converts a Conventional Commit type to a Lore document type.
// Falls back to "note" for unknown types.
func MapCommitType(ccType string) string {
	if t, ok := ccTypeMap[strings.ToLower(ccType)]; ok {
		return t
	}
	return "note"
}

// flowOptions holds injectable options for the question flow.
type flowOptions struct {
	expressThreshold time.Duration
}

// Option is a functional option for QuestionFlow.
type Option func(*flowOptions)

// WithExpressThreshold sets the cumulative time threshold for express mode.
func WithExpressThreshold(d time.Duration) Option {
	return func(o *flowOptions) { o.expressThreshold = d }
}

// QuestionFlow orchestrates the inverse funnel question sequence.
type QuestionFlow struct {
	streams  domain.IOStreams
	reader   *bufio.Reader // reused across calls to avoid losing buffered bytes
	renderer Renderer
	opts     flowOptions
}

// NewQuestionFlow creates a QuestionFlow with the given renderer and options.
func NewQuestionFlow(streams domain.IOStreams, renderer Renderer, opts ...Option) *QuestionFlow {
	o := flowOptions{expressThreshold: defaultExpressThreshold}
	for _, opt := range opts {
		opt(&o)
	}
	return &QuestionFlow{
		streams:  streams,
		reader:   bufio.NewReader(streams.In),
		renderer: renderer,
		opts:     o,
	}
}

// QuestionOpts controls the behavior of AskQuestions for all 4 documentation paths.
type QuestionOpts struct {
	PreFilled  Answers            // partial or full pre-filled answers (resolve/proactive)
	Express    bool               // enable express mode — timer-based skip for Alternatives+Impact (reactive only)
	CommitInfo *domain.CommitInfo // for pre-fill defaults (type from MapCommitType, what from Subject)
}

// AskQuestions is the unified question flow for all documentation paths.
// It handles pre-filled answers, express mode, and commit-based defaults.
//
// On error (including context cancellation), AskQuestions returns the partial
// answers collected so far. Callers are responsible for saving these as pending
// via BuildPendingRecord + SavePending — AskQuestions does not persist state
// because it lacks the commit hash and context needed for the pending record.
//
// Behavior per field:
//   - Pre-filled + valid → confirm and skip (no prompt)
//   - Pre-filled but invalid type → interactive with "note" default
//   - Empty → interactive prompt with commit-derived defaults when available
//   - Express mode → if first 3 Qs answered within expressThreshold, skip Alternatives+Impact
func (q *QuestionFlow) AskQuestions(ctx context.Context, opts QuestionOpts) (Answers, error) {
	answers := opts.PreFilled
	var expressElapsed time.Duration
	questionNum := 0
	totalRequired := countInteractive(opts)

	// --- Question 1: Type ---
	if answers.Type != "" && domain.ValidDocType(answers.Type) {
		// Valid type pre-filled → skip prompt
		q.renderer.QuestionConfirm("Type", answers.Type)
	} else {
		questionNum++
		q.renderer.Progress(questionNum, totalRequired, "Type")
		defaultType := ""
		if answers.Type != "" {
			// Invalid type provided → fallback to "note" default
			defaultType = "note"
		} else if opts.CommitInfo != nil && opts.CommitInfo.Type != "" {
			defaultType = MapCommitType(opts.CommitInfo.Type)
		} else {
			defaultType = "note"
		}
		start := time.Now()
		t, err := q.AskType(ctx, defaultType)
		if err != nil {
			return answers, fmt.Errorf("workflow: question flow: type: %w", err)
		}
		if opts.Express {
			expressElapsed += time.Since(start)
		}
		answers.Type = t
		q.renderer.QuestionConfirm("Type", t)
	}

	// --- Question 2: What ---
	if answers.What != "" {
		q.renderer.QuestionConfirm("What", answers.What)
	} else {
		questionNum++
		q.renderer.Progress(questionNum, totalRequired, "What")
		defaultWhat := ""
		if opts.CommitInfo != nil {
			defaultWhat = opts.CommitInfo.Subject
		}
		start := time.Now()
		w, err := q.AskWhat(ctx, defaultWhat)
		if err != nil {
			return answers, fmt.Errorf("workflow: question flow: what: %w", err)
		}
		if opts.Express {
			expressElapsed += time.Since(start)
		}
		answers.What = w
		q.renderer.QuestionConfirm("What", w)
	}

	// --- Question 3: Why ---
	if answers.Why != "" {
		q.renderer.QuestionConfirm("Why", answers.Why)
	} else {
		questionNum++
		q.renderer.Progress(questionNum, totalRequired, "Why")
		why, err := q.AskWhy(ctx)
		if err != nil {
			return answers, fmt.Errorf("workflow: question flow: why: %w", err)
		}
		answers.Why = why
		q.renderer.QuestionConfirm("Why", why)
	}

	// --- Express mode check ---
	// Only apply express skip if at least one of the timed questions (Type, What)
	// was actually asked interactively. If all were pre-filled, expressElapsed is 0
	// which would falsely trigger the skip.
	if opts.Express && expressElapsed > 0 && expressElapsed < q.opts.expressThreshold {
		q.renderer.ExpressSkip(2)
		return answers, nil
	}

	// --- Question 4: Alternatives ---
	if answers.Alternatives != "" {
		q.renderer.QuestionConfirm("Alternatives", answers.Alternatives)
	} else {
		alt, err := q.AskAlternatives(ctx)
		if err != nil {
			return answers, fmt.Errorf("workflow: question flow: alternatives: %w", err)
		}
		answers.Alternatives = alt
		if alt != "" {
			q.renderer.QuestionConfirm("Alternatives", alt)
		}
	}

	// --- Question 5: Impact ---
	if answers.Impact != "" {
		q.renderer.QuestionConfirm("Impact", answers.Impact)
	} else {
		imp, err := q.AskImpact(ctx)
		if err != nil {
			return answers, fmt.Errorf("workflow: question flow: impact: %w", err)
		}
		answers.Impact = imp
		if imp != "" {
			q.renderer.QuestionConfirm("Impact", imp)
		}
	}

	return answers, nil
}

// countInteractive counts how many of the 3 required questions (Type, What, Why)
// will be asked interactively (not pre-filled). Used for progress display.
func countInteractive(opts QuestionOpts) int {
	count := 0
	if opts.PreFilled.Type == "" || !domain.ValidDocType(opts.PreFilled.Type) {
		count++
	}
	if opts.PreFilled.What == "" {
		count++
	}
	if opts.PreFilled.Why == "" {
		count++
	}
	return count
}

// RunFlow orchestrates all 5 questions with express mode and returns Answers.
// commitInfo is used to pre-fill Type and What defaults.
// Package-internal: called by runDocumentationFlow in reactive.go.
func (q *QuestionFlow) RunFlow(ctx context.Context, commit *domain.CommitInfo) (Answers, error) {
	return q.AskQuestions(ctx, QuestionOpts{
		Express:    true,
		CommitInfo: commit,
	})
}

// AskType prompts for document type with a pre-filled default.
// Enter confirms, any input replaces.
func (q *QuestionFlow) AskType(ctx context.Context, defaultType string) (string, error) {
	return q.askWithDefault(ctx, "Type", defaultType)
}

// AskWhat prompts for what was done, pre-filled from commit subject.
func (q *QuestionFlow) AskWhat(ctx context.Context, defaultWhat string) (string, error) {
	return q.askWithDefault(ctx, "What", defaultWhat)
}

// AskWhy prompts for the reason — the single true question of the flow.
func (q *QuestionFlow) AskWhy(ctx context.Context) (string, error) {
	return q.askOpen(ctx, "Why was this approach chosen?")
}

// AskAlternatives prompts for alternatives considered (optional).
func (q *QuestionFlow) AskAlternatives(ctx context.Context) (string, error) {
	return q.askWithDefault(ctx, "Alternatives considered (optional, Enter to skip)", "")
}

// AskImpact prompts for impact (optional).
func (q *QuestionFlow) AskImpact(ctx context.Context) (string, error) {
	return q.askWithDefault(ctx, "Impact (optional, Enter to skip)", "")
}

// askWithDefault shows "? label [default]: " and returns the answer or default.
func (q *QuestionFlow) askWithDefault(ctx context.Context, label, defaultVal string) (string, error) {
	q.renderer.QuestionStart(label, defaultVal)
	answer, err := q.readLine(ctx)
	if err != nil {
		return "", fmt.Errorf("workflow: ask %s: %w", label, err)
	}
	if answer == "" {
		return defaultVal, nil
	}
	return answer, nil
}

// askOpen shows "? question\n  > " and reads a single line.
func (q *QuestionFlow) askOpen(ctx context.Context, question string) (string, error) {
	q.renderer.QuestionStart(question, "")
	answer, err := q.readLine(ctx)
	if err != nil {
		return "", fmt.Errorf("workflow: ask open %q: %w", question, err)
	}
	return answer, nil
}

// readLine reads a single line from streams.In, respecting context cancellation.
// q.reader is reused across calls to avoid losing bytes buffered by bufio.
// The read runs in a goroutine so that ctx.Done() can interrupt a blocking read.
//
// Goroutine lifecycle: the channel is buffered (size 1), so the goroutine can
// always send its result even if nobody is listening after ctx cancellation.
// If streams.In is an *os.File (the normal CLI case), we set a past read
// deadline to unblock ReadString and let the goroutine exit promptly.
// Otherwise (e.g. tests using *strings.Reader), the goroutine stays blocked
// until the reader returns — acceptable for a CLI tool that will exit soon.
func (q *QuestionFlow) readLine(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("workflow: read line: %w", err)
	}

	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := q.reader.ReadString('\n')
		ch <- result{line, err}
	}()

	select {
	case <-ctx.Done():
		// Try to unblock the goroutine by setting a past deadline on the
		// underlying file descriptor. This works when streams.In is *os.File
		// (real stdin). For other reader types, the goroutine will remain
		// blocked until the reader returns — the buffered channel ensures it
		// won't leak a reference to ch.
		if f, ok := q.streams.In.(*os.File); ok {
			_ = f.SetReadDeadline(time.Now())
		}
		return "", fmt.Errorf("workflow: read line: %w", ctx.Err())
	case r := <-ch:
		if r.err != nil {
			// M5 fix: EOF means stdin was closed (piped input ended).
			// bufio.ReadString returns a partial line + io.EOF when the final
			// line has no trailing newline — treat it as valid input.
			if errors.Is(r.err, io.EOF) {
				return strings.TrimSpace(r.line), nil
			}
			return "", fmt.Errorf("workflow: read line: %w", r.err)
		}
		return strings.TrimSpace(r.line), nil
	}
}
