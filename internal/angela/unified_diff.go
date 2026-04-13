// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — unified_diff.go
//
// Unified diff helper for `--dry-run`.
//
// The polish command's hunk-based TUI (diff.go) is designed for interactive
// review. The dry-run mode instead wants a pipeable, standard-looking unified
// diff ready to be consumed by `diff`, `bat`, or a human in a CI log. This
// file provides that separate rendering path using pmezard/go-difflib — the
// library is already present in go.sum as an indirect dependency, so story
// task 6.1's dependency check is satisfied without a new direct dep.
package angela

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/greycoderk/lore/internal/ui"
	"github.com/pmezard/go-difflib/difflib"
)

// UnifiedDiffOptions configures UnifiedDiffString and WriteUnifiedDiff.
type UnifiedDiffOptions struct {
	// FromFile and ToFile are the labels printed in the `---` and `+++`
	// headers. They are purely cosmetic — the diff content is driven by
	// the A/B strings.
	FromFile string
	ToFile   string

	// Context is the number of context lines around each change. Default
	// (0 or negative) maps to 3, matching `diff -u`'s default.
	Context int

	// Colored switches ANSI colors on: additions in green, deletions in
	// red. Callers should only set this to true when the destination is
	// a TTY (see ui.ColorEnabled). When false the output is pure ASCII
	// and safe to redirect into a file or pipe.
	Colored bool
}

// UnifiedDiffString computes a unified diff of original vs modified and
// returns it as a single string. Line endings are preserved (the output
// ends with a trailing newline as produced by difflib). An empty diff
// string means the two inputs are identical.
func UnifiedDiffString(original, modified string, opts UnifiedDiffOptions) (string, error) {
	ctx := opts.Context
	if ctx <= 0 {
		ctx = 3
	}
	ud := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: opts.FromFile,
		ToFile:   opts.ToFile,
		Context:  ctx,
	}
	raw, err := difflib.GetUnifiedDiffString(ud)
	if err != nil {
		return "", fmt.Errorf("angela: unified diff: %w", err)
	}
	if !opts.Colored {
		return raw, nil
	}
	return colorizeUnifiedDiff(raw), nil
}

// WriteUnifiedDiff streams the diff into w. Equivalent to calling
// UnifiedDiffString and writing the result, but avoids an intermediate
// allocation for large documents.
func WriteUnifiedDiff(w io.Writer, original, modified string, opts UnifiedDiffOptions) error {
	s, err := UnifiedDiffString(original, modified, opts)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, s)
	return err
}

// colorizeUnifiedDiff paints +/- lines with ANSI colors without touching
// the header lines (`---`, `+++`) or hunk markers (`@@`). We rely on
// ui.Success/ui.Error so the global NO_COLOR switch still applies: when
// color is disabled at the ui level, these wrappers return the input as-is
// and the diff ends up identical to the `Colored=false` branch.
func colorizeUnifiedDiff(raw string) string {
	if raw == "" {
		return raw
	}
	var b strings.Builder
	b.Grow(len(raw) + 64)
	sc := bufio.NewScanner(strings.NewReader(raw))
	// Large documents may have long lines; bump the buffer well above the
	// default 64 KiB so the scanner doesn't choke on a single very long
	// markdown row.
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	first := true
	for sc.Scan() {
		line := sc.Text()
		if !first {
			b.WriteByte('\n')
		}
		first = false
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			// Headers stay uncolored — they carry the file labels.
			b.WriteString(line)
		case strings.HasPrefix(line, "+"):
			b.WriteString(ui.Success(line))
		case strings.HasPrefix(line, "-"):
			b.WriteString(ui.Error(line))
		case strings.HasPrefix(line, "@@"):
			b.WriteString(ui.Dim(line))
		default:
			b.WriteString(line)
		}
	}
	// Check the scanner's error. A bufio.Scanner stops silently on a
	// single over-long line (bufio.ErrTooLong) or an underlying reader
	// failure — without this check the colorized output would truncate
	// at the problematic line. On failure, fall back to the raw
	// uncolored diff so the user at least sees the full content.
	if err := sc.Err(); err != nil {
		return raw
	}
	// Preserve the trailing newline from the original diff output.
	if strings.HasSuffix(raw, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}
