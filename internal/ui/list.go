package ui

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

const maxListItems = 15

// ListItem represents one entry in a numbered selection list.
type ListItem struct {
	Type  string // "decision", "feature", etc.
	Title string // slug or heading
	Date  string // "2026-03-07"
}

// List displays a numbered list on stderr and reads selection from stdin.
// If items > 15, truncates with "... and N more. Refine your search."
// Returns the selected index (0-based).
// In non-TTY mode: prints list to stdout (parseable), returns -1 (no selection).
func List(streams domain.IOStreams, items []ListItem, prompt string) (int, error) {
	if len(items) == 0 {
		return -1, nil
	}

	isTTY := IsTerminal(streams)

	displayCount := len(items)
	truncated := false
	if displayCount > maxListItems {
		displayCount = maxListItems
		truncated = true
	}

	// Render the list
	out := streams.Err
	if !isTTY {
		out = streams.Out
	}
	for i := 0; i < displayCount; i++ {
		item := items[i]
		_, _ = fmt.Fprintf(out, "%3d  %-10s %-22s %s\n", i+1, item.Type, item.Title, item.Date)
	}

	if truncated {
		remaining := len(items) - maxListItems
		_, _ = fmt.Fprintf(out, "... and %d more. Refine your search.\n", remaining)
	}

	// Non-TTY: no interactive selection
	if !isTTY {
		return -1, nil
	}

	// Interactive selection
	scanner := bufio.NewScanner(streams.In)
	for {
		fmt.Fprintf(streams.Err, "%s [1-%d]: ", prompt, displayCount)
		if !scanner.Scan() {
			return -1, fmt.Errorf("ui: list: no input")
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		n, err := strconv.Atoi(text)
		if err != nil || n < 1 || n > displayCount {
			fmt.Fprintf(streams.Err, "Please enter a number between 1 and %d.\n", displayCount)
			continue
		}
		return n - 1, nil
	}
}
