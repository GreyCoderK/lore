package ui

import "fmt"

var colorEnabled = true

func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

func Success(text string) string {
	if !colorEnabled {
		return text
	}
	return fmt.Sprintf("\033[32m%s\033[0m", text)
}

func Warning(text string) string {
	if !colorEnabled {
		return text
	}
	return fmt.Sprintf("\033[33m%s\033[0m", text)
}

func Error(text string) string {
	if !colorEnabled {
		return text
	}
	return fmt.Sprintf("\033[31m%s\033[0m", text)
}

func Dim(text string) string {
	if !colorEnabled {
		return text
	}
	return fmt.Sprintf("\033[2m%s\033[0m", text)
}

func Bold(text string) string {
	if !colorEnabled {
		return text
	}
	return fmt.Sprintf("\033[1m%s\033[0m", text)
}
