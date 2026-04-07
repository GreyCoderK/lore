// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"golang.org/x/term"
)

// typeSelectOptions are the valid document types displayed in the selector.
// Order is intentional: most common first.
var typeSelectOptions = []string{
	"feature",
	"bugfix",
	"decision",
	"refactor",
	"note",
	"release",
	"summary",
}

// selectType displays an interactive arrow-key selector for document type.
// Returns the selected type string.
// Falls back to text input if stdin is not a terminal.
func selectType(streams domain.IOStreams, defaultType string) (string, error) {
	inFile, ok := streams.In.(*os.File)
	if !ok {
		return defaultType, nil
	}
	fd := int(inFile.Fd())
	if !term.IsTerminal(fd) {
		return defaultType, nil
	}

	// Find initial cursor position based on default
	cursor := 0
	for i, opt := range typeSelectOptions {
		if opt == defaultType {
			cursor = i
			break
		}
	}

	// Switch to raw mode to capture arrow keys
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return defaultType, nil
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	renderSelect(streams, cursor, -1)

	for {
		b := make([]byte, 3)
		n, err := inFile.Read(b)
		if err != nil || n == 0 {
			_ = term.Restore(fd, oldState)
			return defaultType, nil
		}

		switch {
		case n == 1 && (b[0] == '\r' || b[0] == '\n'):
			// Enter — confirm selection
			clearSelectLines(streams, len(typeSelectOptions))
			_ = term.Restore(fd, oldState)
			return typeSelectOptions[cursor], nil

		case n == 1 && (b[0] == 'q' || b[0] == 3): // q or Ctrl+C
			clearSelectLines(streams, len(typeSelectOptions))
			_ = term.Restore(fd, oldState)
			return defaultType, nil

		case n == 3 && b[0] == 27 && b[1] == 91: // ESC [ sequence
			switch b[2] {
			case 'A': // Up arrow
				if cursor > 0 {
					cursor--
				}
			case 'B': // Down arrow
				if cursor < len(typeSelectOptions)-1 {
					cursor++
				}
			}

		case n == 1 && b[0] == 'k': // vim up
			if cursor > 0 {
				cursor--
			}
		case n == 1 && b[0] == 'j': // vim down
			if cursor < len(typeSelectOptions)-1 {
				cursor++
			}
		}

		clearSelectLines(streams, len(typeSelectOptions))
		renderSelect(streams, cursor, -1)
	}
}

// renderSelect draws the type selection list to stderr.
// Uses \r\n because terminal is in raw mode (LF alone doesn't do carriage return).
func renderSelect(streams domain.IOStreams, cursor int, _ int) {
	for i, opt := range typeSelectOptions {
		if i == cursor {
			_, _ = fmt.Fprintf(streams.Err, "  \033[32m❯\033[0m \033[1m%s\033[0m\r\n", opt)
		} else {
			_, _ = fmt.Fprintf(streams.Err, "    \033[2m%s\033[0m\r\n", opt)
		}
	}
}

// clearSelectLines moves cursor up N lines and clears each line.
func clearSelectLines(streams domain.IOStreams, n int) {
	for i := 0; i < n; i++ {
		_, _ = fmt.Fprint(streams.Err, "\033[A\033[2K\r")
	}
}

// validateType checks if the given type is valid. If not, returns an error message.
func validateType(t string) (string, bool) {
	t = strings.TrimSpace(strings.ToLower(t))
	if domain.ValidDocType(t) {
		return t, true
	}
	return t, false
}
