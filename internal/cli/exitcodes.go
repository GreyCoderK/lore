// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cli

import (
	"errors"
	"fmt"
)

// Exit codes for the lore CLI, following Unix conventions.
// Used by cmd/root.go and commands that need non-zero exits.
const (
	ExitOK    = 0
	ExitError = 1
	ExitSkip  = 2 // no match found (e.g. lore show with zero results)
)

// ExitCodeError is returned by commands that need a specific exit code
// without calling os.Exit directly (enables testability).
type ExitCodeError struct {
	Code int
}

func (e *ExitCodeError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}

// ExitCodeFrom returns the exit code if err is (or wraps) an *ExitCodeError, or -1 otherwise.
func ExitCodeFrom(err error) int {
	var e *ExitCodeError
	if errors.As(err, &e) {
		return e.Code
	}
	return -1
}
