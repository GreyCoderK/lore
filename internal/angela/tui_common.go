// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — tui_common.go
//
// Shared Bubbletea infrastructure used by the interactive review (8.13)
// and interactive draft (8.14) TUIs. Extracted here to avoid duplication.

package angela

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Shared lipgloss styles for both interactive TUIs.
var (
	TUIStyleTitle   = lipgloss.NewStyle().Bold(true)
	TUIStyleDim     = lipgloss.NewStyle().Faint(true)
	TUIStyleHelpKey = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	TUIStyleCursor  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	TUIStyleSpinner = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
)

// Severity colors shared by review and draft TUIs.
var (
	TUIStyleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	TUIStyleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	TUIStyleInfo    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
)

// IsTTYAvailable checks if stdout is a character device (TTY).
// Used by both --interactive flags for non-TTY fallback.
func IsTTYAvailable() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// splitEditorCmd splits $EDITOR into executable + args so values like
// "vim -u NONE" or "code --wait" work correctly. Returns nil if empty.
// Prevents passing "vim -u NONE" as a single binary name.
func splitEditorCmd(editor string) []string {
	editor = strings.TrimSpace(editor)
	if editor == "" {
		return nil
	}
	return strings.Fields(editor)
}

// isSafePath rejects path traversal and absolute paths in user/AI-provided
// filenames. Prevents opening ../../etc/passwd via $EDITOR.
func isSafePath(filename string) bool {
	if filename == "" {
		return false
	}
	if filepath.IsAbs(filename) {
		return false
	}
	cleaned := filepath.Clean(filename)
	return !strings.HasPrefix(cleaned, "..")
}
