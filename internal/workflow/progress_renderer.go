// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// ProgressRenderer is the TTY renderer that condenses confirmed answers via ANSI.
// Budget: ~7 lines stderr max.
type ProgressRenderer struct {
	streams    domain.IOStreams
	confirmed  []string // accumulates "✓ Type: feature" entries
	lineCount  int      // lines currently on screen managed by this renderer
}

func NewProgressRenderer(streams domain.IOStreams) *ProgressRenderer {
	return &ProgressRenderer{streams: streams}
}

// QuestionStart prints the progress bar + question prompt.
// Overwrites previous lines via cursor-up escape.
func (r *ProgressRenderer) QuestionStart(question string, defaultVal string) {
	r.clearLines()
	if defaultVal != "" {
		_, _ = fmt.Fprintf(r.streams.Err, "\033[32m?\033[0m \033[1m%s\033[0m [\033[2m%s\033[0m]: ", question, defaultVal)
	} else {
		_, _ = fmt.Fprintf(r.streams.Err, "\033[32m?\033[0m \033[1m%s\033[0m\n  > ", question)
		r.lineCount = 2
		return
	}
	r.lineCount = 1
}

// QuestionConfirm updates the confirmed bar and redraws.
func (r *ProgressRenderer) QuestionConfirm(question string, answer string) {
	// Condense into single-line summary: "✓ Type: feature  ✓ What: add auth"
	r.confirmed = append(r.confirmed, fmt.Sprintf("\033[32m✓\033[0m %s: %s", question, answer))
	r.clearLines()
	_, _ = fmt.Fprintf(r.streams.Err, "  %s\n", strings.Join(r.confirmed, "  "))
	r.lineCount = 1
}

// Progress renders the progress bar [##·] N+ label.
func (r *ProgressRenderer) Progress(current, total int, label string) {
	bar := strings.Repeat("#", current) + strings.Repeat("·", total-current)
	_, _ = fmt.Fprintf(r.streams.Err, "  \033[2m[%s]\033[0m %d+ %s\n", bar, current, label)
	r.lineCount++
}

// ExpressSkip prints the express skip feedback.
func (r *ProgressRenderer) ExpressSkip(skipped int) {
	total := len(r.confirmed) + skipped
	bar := strings.Repeat("#", total)
	_, _ = fmt.Fprintf(r.streams.Err, "  \033[2m[%s]\033[0m %d/%d \033[2m(%d optional skipped — express)\033[0m\n",
		bar, len(r.confirmed), total, skipped)
}

// clearLines moves cursor up to erase previously printed renderer lines.
func (r *ProgressRenderer) clearLines() {
	for i := 0; i < r.lineCount; i++ {
		_, _ = fmt.Fprint(r.streams.Err, "\033[A\033[2K")
	}
	r.lineCount = 0
}
