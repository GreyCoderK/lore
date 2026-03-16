package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestActionableError(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	ActionableError(streams, "Hook not installed.", "lore hook install")
	got := buf.String()

	if !strings.Contains(got, "Error:") {
		t.Errorf("expected 'Error:' in output, got %q", got)
	}
	if !strings.Contains(got, "Hook not installed.") {
		t.Errorf("expected message in output, got %q", got)
	}
	if !strings.Contains(got, "lore hook install") {
		t.Errorf("expected command in output, got %q", got)
	}
	if !strings.Contains(got, "Run:") {
		t.Errorf("expected 'Run:' in output, got %q", got)
	}
}
