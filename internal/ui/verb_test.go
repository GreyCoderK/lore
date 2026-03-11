package ui

import (
	"bytes"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

func TestVerbAlignment(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	Verb(streams, "Captured", "decision-auth.md")
	got := buf.String()
	expected := "  Captured decision-auth.md\n"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestVerbWithColor(t *testing.T) {
	SetColorEnabled(true)

	var buf bytes.Buffer
	streams := domain.IOStreams{
		Out: &buf,
		Err: &buf,
	}

	Verb(streams, "Done", "file.md")
	got := buf.String()
	if len(got) <= len("      Done file.md\n") {
		t.Errorf("expected colored output to be longer than plain, got %q", got)
	}
}
