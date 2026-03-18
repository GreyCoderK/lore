// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrCorrupted      = errors.New("corrupted")
	ErrAlreadyExists  = errors.New("already exists")
	ErrNotInitialized = errors.New("lore not initialized")
	ErrNotGitRepo     = errors.New("not a git repository")
	ErrNotInteractive = errors.New("not interactive")
)