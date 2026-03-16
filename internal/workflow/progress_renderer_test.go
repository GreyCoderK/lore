package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func newProgressRendererForTest() (*ProgressRenderer, *bytes.Buffer) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}
	return NewProgressRenderer(streams), stderr
}

func TestProgressRenderer_QuestionStart_WithDefault(t *testing.T) {
	r, buf := newProgressRendererForTest()
	r.QuestionStart("Type", "feature")
	got := buf.String()
	if !strings.Contains(got, "Type") {
		t.Errorf("output missing label: %q", got)
	}
	if !strings.Contains(got, "feature") {
		t.Errorf("output missing default: %q", got)
	}
	// ANSI bold/green should be present in TTY mode
	if !strings.Contains(got, "\033[") {
		t.Errorf("output missing ANSI escapes: %q", got)
	}
}

func TestProgressRenderer_QuestionStart_NoDefault(t *testing.T) {
	r, buf := newProgressRendererForTest()
	r.QuestionStart("Why was this approach chosen?", "")
	got := buf.String()
	if !strings.Contains(got, "Why was this approach chosen?") {
		t.Errorf("output missing question: %q", got)
	}
	if !strings.Contains(got, ">") {
		t.Errorf("output missing prompt '>': %q", got)
	}
	if r.lineCount != 2 {
		t.Errorf("lineCount = %d, want 2 for open question", r.lineCount)
	}
}

func TestProgressRenderer_QuestionConfirm_Condensation(t *testing.T) {
	r, buf := newProgressRendererForTest()
	// Confirm two answers — they should be on a single line
	r.QuestionConfirm("Type", "feature")
	r.QuestionConfirm("What", "add auth")
	got := buf.String()
	// Both confirmations should be present
	if !strings.Contains(got, "Type") || !strings.Contains(got, "feature") {
		t.Errorf("missing first confirmation: %q", got)
	}
	if !strings.Contains(got, "What") || !strings.Contains(got, "add auth") {
		t.Errorf("missing second confirmation: %q", got)
	}
	// Check that cursor-up ANSI escape is emitted (condensation)
	if !strings.Contains(got, "\033[A") {
		t.Errorf("missing cursor-up ANSI escape for condensation: %q", got)
	}
	if len(r.confirmed) != 2 {
		t.Errorf("confirmed count = %d, want 2", len(r.confirmed))
	}
}

func TestProgressRenderer_Progress(t *testing.T) {
	r, buf := newProgressRendererForTest()
	r.Progress(2, 3, "What")
	got := buf.String()
	if !strings.Contains(got, "##") {
		t.Errorf("progress bar missing hashes: %q", got)
	}
	if !strings.Contains(got, "What") {
		t.Errorf("progress missing label: %q", got)
	}
}

func TestProgressRenderer_ExpressSkip(t *testing.T) {
	r, buf := newProgressRendererForTest()
	r.QuestionConfirm("Type", "feature")
	r.QuestionConfirm("What", "add auth")
	r.QuestionConfirm("Why", "security")
	buf.Reset()
	r.ExpressSkip(2)
	got := buf.String()
	if !strings.Contains(got, "express") {
		t.Errorf("ExpressSkip missing 'express': %q", got)
	}
	if !strings.Contains(got, "2") {
		t.Errorf("ExpressSkip missing skipped count: %q", got)
	}
}

func TestProgressRenderer_ClearLines(t *testing.T) {
	r, buf := newProgressRendererForTest()
	// Simulate Progress + QuestionStart to build up lineCount
	r.Progress(1, 3, "Type")
	r.QuestionStart("Type", "feature")
	// After QuestionStart, clearLines should have been called and cursor-up emitted
	got := buf.String()
	if !strings.Contains(got, "\033[A\033[2K") {
		t.Errorf("expected cursor-up + clear line escape: %q", got)
	}
}
