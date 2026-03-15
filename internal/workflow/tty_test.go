package workflow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/domain"
)

// mockStreams returns non-os.File streams (buffers/strings), which IsInteractiveTTY must treat as non-TTY.
func mockStreams() domain.IOStreams {
	return domain.IOStreams{
		In:  strings.NewReader(""),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
}

func TestIsInteractiveTTY_NonFileStreams(t *testing.T) {
	if IsInteractiveTTY(mockStreams()) {
		t.Error("IsInteractiveTTY should be false for non-os.File streams")
	}
}

func TestNewRenderer_NonTTY_ReturnsLineRenderer(t *testing.T) {
	r := NewRenderer(mockStreams())
	if _, ok := r.(*LineRenderer); !ok {
		t.Errorf("NewRenderer with non-TTY streams should return *LineRenderer, got %T", r)
	}
}
