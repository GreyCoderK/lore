package workflow

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// LineRenderer is the non-TTY renderer for CI/pipe environments.
// One line per event, no ANSI rewriting — CI-compatible.
type LineRenderer struct {
	streams  domain.IOStreams
	answered int
}

func NewLineRenderer(streams domain.IOStreams) *LineRenderer {
	return &LineRenderer{streams: streams}
}

func (r *LineRenderer) QuestionStart(question string, defaultVal string) {
	if defaultVal != "" {
		_, _ = fmt.Fprintf(r.streams.Err, "? %s [%s]: ", question, defaultVal)
	} else {
		_, _ = fmt.Fprintf(r.streams.Err, "? %s\n> ", question)
	}
}

func (r *LineRenderer) QuestionConfirm(question string, answer string) {
	r.answered++
	_, _ = fmt.Fprintf(r.streams.Err, "✓ %s: %s\n", question, answer)
}

func (r *LineRenderer) Progress(current, total int, label string) {
	bar := strings.Repeat("#", current) + strings.Repeat("·", total-current)
	_, _ = fmt.Fprintf(r.streams.Err, "[%s] %d+ %s\n", bar, current, label)
}

func (r *LineRenderer) ExpressSkip(skipped int) {
	total := r.answered + skipped
	bar := strings.Repeat("#", total)
	_, _ = fmt.Fprintf(r.streams.Err, "[%s] %d/%d (%d optional skipped — express)\n",
		bar, r.answered, total, skipped)
}
