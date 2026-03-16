package workflow

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// testStreams builds mock IOStreams from a pre-defined input string.
func testStreams(input string) (domain.IOStreams, *bytes.Buffer) {
	stderr := &bytes.Buffer{}
	return domain.IOStreams{
		In:  strings.NewReader(input),
		Out: &bytes.Buffer{},
		Err: stderr,
	}, stderr
}

// lineRenderer returns a LineRenderer bound to the given streams.
func lineRenderer(streams domain.IOStreams) *LineRenderer {
	return NewLineRenderer(streams)
}

// --- MapCommitType ---

func TestMapCommitType_KnownTypes(t *testing.T) {
	cases := []struct {
		ccType string
		want   string
	}{
		{"feat", "feature"},
		{"FEAT", "feature"}, // case-insensitive
		{"fix", "bugfix"},
		{"refactor", "refactor"},
		{"docs", "note"},
		{"chore", "note"},
		{"test", "note"},
		{"perf", "feature"},
	}
	for _, tc := range cases {
		got := MapCommitType(tc.ccType)
		if got != tc.want {
			t.Errorf("MapCommitType(%q) = %q, want %q", tc.ccType, got, tc.want)
		}
	}
}

func TestMapCommitType_UnknownFallsBackToNote(t *testing.T) {
	for _, s := range []string{"style", "ci", "build", "", "unknown"} {
		if got := MapCommitType(s); got != "note" {
			t.Errorf("MapCommitType(%q) = %q, want %q", s, got, "note")
		}
	}
}

// --- RunFlow: full 5-question flow ---

func TestRunFlow_FullFlow_EnterUsesDefaults(t *testing.T) {
	// 5 lines: Enter (type default), Enter (what default), why, alternatives, impact
	input := "\n\nBecause it improves resilience\nOption A considered\nIncreased throughput\n"
	streams, _ := testStreams(input)
	commit := &domain.CommitInfo{Type: "feat", Subject: "add circuit breaker"}

	// threshold=0 → never express (elapsed always ≥ 0)
	flow := NewQuestionFlow(streams, lineRenderer(streams), WithExpressThreshold(0))
	answers, err := flow.RunFlow(context.Background(), commit)
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}
	if answers.Type != "feature" {
		t.Errorf("Type = %q, want %q", answers.Type, "feature")
	}
	if answers.What != "add circuit breaker" {
		t.Errorf("What = %q, want %q", answers.What, "add circuit breaker")
	}
	if answers.Why != "Because it improves resilience" {
		t.Errorf("Why = %q, want %q", answers.Why, "Because it improves resilience")
	}
	if answers.Alternatives != "Option A considered" {
		t.Errorf("Alternatives = %q, want %q", answers.Alternatives, "Option A considered")
	}
	if answers.Impact != "Increased throughput" {
		t.Errorf("Impact = %q, want %q", answers.Impact, "Increased throughput")
	}
}

func TestRunFlow_FullFlow_ExplicitValues(t *testing.T) {
	// User overrides type and what, then answers all optionals
	input := "bugfix\nfix the NPE in auth\nRoot cause was nil map\nNone\nNo prod impact\n"
	streams, _ := testStreams(input)
	commit := &domain.CommitInfo{Type: "feat", Subject: "original subject"}

	flow := NewQuestionFlow(streams, lineRenderer(streams), WithExpressThreshold(0))
	answers, err := flow.RunFlow(context.Background(), commit)
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}
	if answers.Type != "bugfix" {
		t.Errorf("Type = %q, want %q", answers.Type, "bugfix")
	}
	if answers.What != "fix the NPE in auth" {
		t.Errorf("What = %q, want %q", answers.What, "fix the NPE in auth")
	}
}

// --- RunFlow: express mode ---

func TestRunFlow_ExpressMode_SkipsOptionals(t *testing.T) {
	// threshold=1h → always express (answers are instant in tests)
	// Only 3 lines consumed: type, what, why
	input := "\n\nNeeded for security compliance\n"
	streams, _ := testStreams(input)
	commit := &domain.CommitInfo{Type: "fix", Subject: "patch XSS"}

	flow := NewQuestionFlow(streams, lineRenderer(streams), WithExpressThreshold(time.Hour))
	answers, err := flow.RunFlow(context.Background(), commit)
	if err != nil {
		t.Fatalf("RunFlow express: %v", err)
	}
	if answers.Type != "bugfix" {
		t.Errorf("Type = %q, want %q", answers.Type, "bugfix")
	}
	if answers.Alternatives != "" {
		t.Errorf("Alternatives should be empty in express mode, got %q", answers.Alternatives)
	}
	if answers.Impact != "" {
		t.Errorf("Impact should be empty in express mode, got %q", answers.Impact)
	}
}

func TestRunFlow_ExpressMode_EmptyOptionals(t *testing.T) {
	// threshold=0 → never express, but user presses Enter on both optionals
	input := "\n\nWhy answer\n\n\n"
	streams, _ := testStreams(input)
	commit := &domain.CommitInfo{Type: "docs", Subject: "update readme"}

	flow := NewQuestionFlow(streams, lineRenderer(streams), WithExpressThreshold(0))
	answers, err := flow.RunFlow(context.Background(), commit)
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}
	if answers.Alternatives != "" {
		t.Errorf("Alternatives should be empty when Enter pressed, got %q", answers.Alternatives)
	}
	if answers.Impact != "" {
		t.Errorf("Impact should be empty when Enter pressed, got %q", answers.Impact)
	}
}

// --- RunFlow: nil commit ---

func TestRunFlow_NilCommit_DefaultsToNote(t *testing.T) {
	// No commit: type defaults to "note", what defaults to ""
	input := "\nmy description\nMy reasoning\n\n\n"
	streams, _ := testStreams(input)

	flow := NewQuestionFlow(streams, lineRenderer(streams), WithExpressThreshold(0))
	answers, err := flow.RunFlow(context.Background(), nil)
	if err != nil {
		t.Fatalf("RunFlow nil commit: %v", err)
	}
	if answers.Type != "note" {
		t.Errorf("Type = %q, want %q", answers.Type, "note")
	}
	if answers.What != "my description" {
		t.Errorf("What = %q, want %q", answers.What, "my description")
	}
}

// --- Context cancellation ---

func TestRunFlow_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	streams, _ := testStreams("any\n")
	flow := NewQuestionFlow(streams, lineRenderer(streams))
	_, err := flow.RunFlow(ctx, nil)
	if err == nil {
		t.Error("expected error with cancelled context, got nil")
	}
}

// --- AskType / AskWhat helpers ---

func TestAskType_UsesDefaultOnEnter(t *testing.T) {
	streams, _ := testStreams("\n")
	flow := NewQuestionFlow(streams, lineRenderer(streams))
	got, err := flow.AskType(context.Background(), "feature")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "feature" {
		t.Errorf("AskType = %q, want %q", got, "feature")
	}
}

func TestAskType_ReplacesDefault(t *testing.T) {
	streams, _ := testStreams("refactor\n")
	flow := NewQuestionFlow(streams, lineRenderer(streams))
	got, err := flow.AskType(context.Background(), "feature")
	if err != nil {
		t.Fatalf("AskType: %v", err)
	}
	if got != "refactor" {
		t.Errorf("AskType = %q, want %q", got, "refactor")
	}
}

func TestAskWhat_UsesDefaultOnEnter(t *testing.T) {
	streams, _ := testStreams("\n")
	flow := NewQuestionFlow(streams, lineRenderer(streams))
	got, err := flow.AskWhat(context.Background(), "add auth")
	if err != nil {
		t.Fatalf("AskWhat: %v", err)
	}
	if got != "add auth" {
		t.Errorf("AskWhat = %q, want %q", got, "add auth")
	}
}

func TestAskWhy_ReturnsInput(t *testing.T) {
	streams, _ := testStreams("Because it is the right approach\n")
	flow := NewQuestionFlow(streams, lineRenderer(streams))
	got, err := flow.AskWhy(context.Background())
	if err != nil {
		t.Fatalf("AskWhy: %v", err)
	}
	if got != "Because it is the right approach" {
		t.Errorf("AskWhy = %q, want %q", got, "Because it is the right approach")
	}
}

// --- Answers.ToGenerateInput ---

func TestAnswers_ToGenerateInput_WithCommit(t *testing.T) {
	commit := &domain.CommitInfo{
		Hash:   "abc1234def",
		Author: "Alice",
		Date:   time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC),
	}
	a := Answers{
		Type:         "feature",
		What:         "add auth",
		Why:          "Security requirement",
		Alternatives: "None",
		Impact:       "Users can log in",
	}
	input := a.ToGenerateInput(commit, "hook")
	if input.DocType != "feature" {
		t.Errorf("DocType = %q", input.DocType)
	}
	if input.CommitInfo == nil || input.CommitInfo.Hash != "abc1234def" {
		t.Errorf("CommitInfo.Hash = %q", func() string {
			if input.CommitInfo != nil { return input.CommitInfo.Hash }
			return ""
		}())
	}
	if input.CommitInfo == nil || input.CommitInfo.Date != commit.Date {
		t.Errorf("CommitInfo.Date mismatch")
	}
	if input.Alternatives != "None" {
		t.Errorf("Alternatives = %q", input.Alternatives)
	}
}

func TestAnswers_ToGenerateInput_NilCommit(t *testing.T) {
	a := Answers{Type: "note", What: "quick note", Why: "To remember"}
	input := a.ToGenerateInput(nil, "hook")
	if input.CommitInfo != nil {
		t.Errorf("CommitInfo should be nil for nil commit")
	}
	if input.GeneratedBy != "hook" {
		t.Errorf("GeneratedBy = %q, want %q", input.GeneratedBy, "hook")
	}
}
