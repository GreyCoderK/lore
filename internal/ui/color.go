package ui

import (
	"fmt"
	"sync/atomic"
)

var colorFlag atomic.Bool

func init() {
	colorFlag.Store(true)
}

func SetColorEnabled(enabled bool) {
	colorFlag.Store(enabled)
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
