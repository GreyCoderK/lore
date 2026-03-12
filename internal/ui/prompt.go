package ui

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/museigen/lore/internal/domain"
)

// Confirm waits for Enter key press.
func Confirm(streams domain.IOStreams, message string) error {
	fmt.Fprintf(streams.Err, "%s", message)
	reader := bufio.NewReader(streams.In)
	_, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("ui: confirm: %w", err)
	}
	return nil
}

// Prompt asks a question with optional default, returns answer.
func Prompt(streams domain.IOStreams, question string, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(streams.Err, "? %s [%s]: ", question, defaultVal)
	} else {
		fmt.Fprintf(streams.Err, "? %s\n> ", question)
	}
	reader := bufio.NewReader(streams.In)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("ui: prompt: %w", err)
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return defaultVal, nil
	}
	return answer, nil
}
