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
