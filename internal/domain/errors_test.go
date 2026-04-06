// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	sentinels := []error{
		ErrNotFound,
		ErrCorrupted,
		ErrAlreadyExists,
		ErrNotInitialized,
		ErrNotGitRepo,
		ErrNotInteractive,
		ErrNotImplemented,
	}
	for _, err := range sentinels {
		if err == nil {
			t.Error("sentinel error should not be nil")
		}
		if err.Error() == "" {
			t.Error("sentinel error should have a message")
		}
	}
}

func TestSentinelErrors_IsWrappable(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", ErrNotFound)
	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("wrapped ErrNotFound should match with errors.Is")
	}
	if errors.Is(wrapped, ErrCorrupted) {
		t.Error("wrapped ErrNotFound should not match ErrCorrupted")
	}
}
