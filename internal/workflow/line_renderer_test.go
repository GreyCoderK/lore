package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

func newLineRendererForTest() (*LineRenderer, *bytes.Buffer) {
	stderr := &bytes.Buffer{}
	streams := domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: stderr,
	}
	return NewLineRenderer(streams), stderr
}

func TestLineRenderer_QuestionStart_WithDefault(t *testing.T) {
	r, buf := newLineRendererForTest()
	r.QuestionStart("Type", "feature")
	got := buf.String()
	if !strings.Contains(got, "Type") {
		t.Errorf("QuestionStart output missing label: %q", got)
	}
	if !strings.Contains(got, "feature") {
		t.Errorf("QuestionStart output missing default: %q", got)
	}
}

func TestLineRenderer_QuestionStart_NoDefault(t *testing.T) {
	r, buf := newLineRendererForTest()
	r.QuestionStart("Why was this approach chosen?", "")
	got := buf.String()
	if !strings.Contains(got, "Why was this approach chosen?") {
		t.Errorf("QuestionStart output missing question: %q", got)
	}
	if !strings.Contains(got, ">") {
		t.Errorf("QuestionStart output missing prompt '>': %q", got)
	}
}

func TestLineRenderer_QuestionConfirm(t *testing.T) {
	r, buf := newLineRendererForTest()
	r.QuestionConfirm("Type", "feature")
	got := buf.String()
	if !strings.Contains(got, "✓") {
		t.Errorf("QuestionConfirm output missing checkmark: %q", got)
	}
	if !strings.Contains(got, "Type") || !strings.Contains(got, "feature") {
		t.Errorf("QuestionConfirm output missing label/answer: %q", got)
	}
}

func TestLineRenderer_Progress(t *testing.T) {
	r, buf := newLineRendererForTest()
	r.Progress(2, 3, "What")
	got := buf.String()
	if !strings.Contains(got, "##") {
		t.Errorf("Progress bar missing hashes: %q", got)
	}
	if !strings.Contains(got, "What") {
		t.Errorf("Progress missing label: %q", got)
	}
}

func TestLineRenderer_ExpressSkip(t *testing.T) {
	r, buf := newLineRendererForTest()
	// Simulate 3 confirmed answers before express skip
	r.QuestionConfirm("Type", "feature")
	r.QuestionConfirm("What", "add auth")
	r.QuestionConfirm("Why", "security")
	buf.Reset() // only check ExpressSkip output
	r.ExpressSkip(2)
	got := buf.String()
	if !strings.Contains(got, "express") {
		t.Errorf("ExpressSkip output missing 'express': %q", got)
	}
	if !strings.Contains(got, "2") {
		t.Errorf("ExpressSkip output missing skipped count: %q", got)
	}
}
