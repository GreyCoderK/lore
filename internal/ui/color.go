// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"
	"os"
	"sync/atomic"
)

var colorFlag atomic.Bool

func init() {
	// Respect NO_COLOR (https://no-color.org/) — disable ANSI if the variable is set.
	_, noColor := os.LookupEnv("NO_COLOR")
	colorFlag.Store(!noColor)
}

func SetColorEnabled(enabled bool) {
	colorFlag.Store(enabled)
}

// ResetColorFromEnv re-reads the NO_COLOR environment variable and updates the
// color flag accordingly. Call this in tests after t.Setenv("NO_COLOR", ...) to
// simulate the env-based initialization that init() performs once at startup.
func ResetColorFromEnv() {
	_, noColor := os.LookupEnv("NO_COLOR")
	colorFlag.Store(!noColor)
}

// SaveAndDisableColor saves the current color state, disables color output,
// and returns a restore function that sets colorFlag back to the saved state.
// Usage in tests: restore := ui.SaveAndDisableColor(); defer restore()
func SaveAndDisableColor() func() {
	prev := colorFlag.Load()
	colorFlag.Store(false)
	return func() { colorFlag.Store(prev) }
}

func isColorEnabled() bool {
	return colorFlag.Load()
}

func Success(text string) string {
	if !isColorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[32m%s\033[0m", text)
}

func Warning(text string) string {
	if !isColorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[33m%s\033[0m", text)
}

func Error(text string) string {
	if !isColorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[31m%s\033[0m", text)
}

func Dim(text string) string {
	if !isColorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[2m%s\033[0m", text)
}

func Bold(text string) string {
	if !isColorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[1m%s\033[0m", text)
}

// Info renders text in cyan. Used to mark content that came back FROM the
// AI (e.g. duplicate-section occurrences in the arbitrate prompt), so the
// user can distinguish AI-emitted material from the action keys they
// themselves need to type next.
func Info(text string) string {
	if !isColorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[36m%s\033[0m", text)
}
