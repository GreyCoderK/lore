package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

func testStreams() domain.IOStreams {
	return domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
		In:  strings.NewReader(""),
	}
}

func TestIsTerminalWithBuffers(t *testing.T) {
	streams := testStreams()
	// Buffers are not *os.File, so IsTerminal must return false
	if IsTerminal(streams) {
		t.Error("expected IsTerminal=false for non-file streams")
	}
}

func TestIsTerminalWithDumbTerm(t *testing.T) {
	os.Setenv("TERM", "dumb")
	defer os.Unsetenv("TERM")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	if IsTerminal(streams) {
		t.Error("expected IsTerminal=false when TERM=dumb")
	}
}

func TestIsTerminalWithLineMode(t *testing.T) {
	os.Setenv("LORE_LINE_MODE", "1")
	defer os.Unsetenv("LORE_LINE_MODE")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	if IsTerminal(streams) {
		t.Error("expected IsTerminal=false when LORE_LINE_MODE=1")
	}
}

func TestColorEnabledWithBuffers(t *testing.T) {
	streams := testStreams()
	// Non-terminal streams → color disabled
	if ColorEnabled(streams) {
		t.Error("expected ColorEnabled=false for non-terminal streams")
	}
}

func TestColorEnabledWithNoColor(t *testing.T) {
	os.Setenv("NO_COLOR", "1")
	defer os.Unsetenv("NO_COLOR")

	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}
	// NO_COLOR set → color disabled regardless of terminal
	if ColorEnabled(streams) {
		t.Error("expected ColorEnabled=false when NO_COLOR is set")
	}
}
