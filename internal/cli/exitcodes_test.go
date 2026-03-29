// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"errors"
	"fmt"
	"testing"
)

func TestExitCodeConstants(t *testing.T) {
	if ExitOK != 0 {
		t.Errorf("ExitOK = %d, want 0", ExitOK)
	}
	if ExitError != 1 {
		t.Errorf("ExitError = %d, want 1", ExitError)
	}
	if ExitSkip != 2 {
		t.Errorf("ExitSkip = %d, want 2", ExitSkip)
	}
	if ExitUserError != 3 {
		t.Errorf("ExitUserError = %d, want 3", ExitUserError)
	}
	if ExitConfigError != 4 {
		t.Errorf("ExitConfigError = %d, want 4", ExitConfigError)
	}
}

func TestExitCodeError_Error(t *testing.T) {
	e := &ExitCodeError{Code: 2}
	if e.Error() != "exit code 2" {
		t.Errorf("Error() = %q, want %q", e.Error(), "exit code 2")
	}
}

func TestExitCodeFrom_WithExitCodeError(t *testing.T) {
	err := &ExitCodeError{Code: ExitSkip}
	if got := ExitCodeFrom(err); got != ExitSkip {
		t.Errorf("ExitCodeFrom = %d, want %d", got, ExitSkip)
	}
}

func TestExitCodeFrom_Wrapped(t *testing.T) {
	inner := &ExitCodeError{Code: ExitUserError}
	wrapped := fmt.Errorf("something failed: %w", inner)
	if got := ExitCodeFrom(wrapped); got != ExitUserError {
		t.Errorf("ExitCodeFrom(wrapped) = %d, want %d", got, ExitUserError)
	}
}

func TestExitCodeFrom_NonExitCodeError(t *testing.T) {
	err := errors.New("generic error")
	if got := ExitCodeFrom(err); got != -1 {
		t.Errorf("ExitCodeFrom(generic) = %d, want -1", got)
	}
}

func TestExitCodeFrom_Nil(t *testing.T) {
	if got := ExitCodeFrom(nil); got != -1 {
		t.Errorf("ExitCodeFrom(nil) = %d, want -1", got)
	}
}
