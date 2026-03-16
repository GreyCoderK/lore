package workflow

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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

// RunFlow orchestrates all 5 questions and returns Answers.
// commitInfo is used to pre-fill Type and What.
func (q *QuestionFlow) RunFlow(ctx context.Context, commit *domain.CommitInfo) (Answers, error) {
	var answers Answers
	var expressElapsed time.Duration

	// --- Question 1: Type ---
	q.renderer.Progress(1, 3, "Type")
	defaultType := "note"
	if commit != nil && commit.Type != "" {
		defaultType = MapCommitType(commit.Type)
	}
	start := time.Now()
	t, err := q.AskType(ctx, defaultType)
	if err != nil {
		return Answers{}, fmt.Errorf("workflow: question flow: type: %w", err)
	}
	expressElapsed += time.Since(start)
	answers.Type = t
	q.renderer.QuestionConfirm("Type", t)

	// --- Question 2: What ---
	q.renderer.Progress(2, 3, "What")
	defaultWhat := ""
	if commit != nil {
		defaultWhat = commit.Subject
	}
	start = time.Now()
	w, err := q.AskWhat(ctx, defaultWhat)
	if err != nil {
		return Answers{}, fmt.Errorf("workflow: question flow: what: %w", err)
	}
	expressElapsed += time.Since(start)
	answers.What = w
	q.renderer.QuestionConfirm("What", w)

	// --- Question 3: Why (required) ---
	q.renderer.Progress(3, 3, "Why")
	why, err := q.AskWhy(ctx)
	if err != nil {
		return Answers{}, fmt.Errorf("workflow: question flow: why: %w", err)
	}
	answers.Why = why
	q.renderer.QuestionConfirm("Why", why)

	// --- Express mode check ---
	if expressElapsed < q.opts.expressThreshold {
		q.renderer.ExpressSkip(2)
		return answers, nil
	}

	// --- Question 4: Alternatives (optional) ---
	alt, err := q.AskAlternatives(ctx)
	if err != nil {
		return Answers{}, fmt.Errorf("workflow: question flow: alternatives: %w", err)
	}
	answers.Alternatives = alt
	if alt != "" {
		q.renderer.QuestionConfirm("Alternatives", alt)
	}

	// --- Question 5: Impact (optional) ---
	imp, err := q.AskImpact(ctx)
	if err != nil {
		return Answers{}, fmt.Errorf("workflow: question flow: impact: %w", err)
	}
	answers.Impact = imp
	if imp != "" {
		q.renderer.QuestionConfirm("Impact", imp)
	}

	return answers, nil
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
func (q *QuestionFlow) readLine(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("workflow: read line: %w", err)
	}

	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	// M1 note: when ctx.Done() fires, the select exits but this goroutine
	// remains blocked on ReadString until the next keypress (or process exit).
	// Interrupting the underlying syscall would require streams.In to be an
	// *os.File so we can call SetDeadline — not guaranteed for all callers.
	go func() {
		line, err := q.reader.ReadString('\n')
		ch <- result{line, err}
	}()

	select {
	case <-ctx.Done():
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
