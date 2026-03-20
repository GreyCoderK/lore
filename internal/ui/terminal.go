// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"os"

	"github.com/greycoderk/lore/internal/domain"
	"golang.org/x/term"
)

func IsTerminal(streams domain.IOStreams) bool {
	inFile, ok := streams.In.(*os.File)
	if !ok {
		return false
	}
	errFile, ok := streams.Err.(*os.File)
	if !ok {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if os.Getenv("LORE_LINE_MODE") == "1" {
		return false
	}
	return term.IsTerminal(int(inFile.Fd())) && term.IsTerminal(int(errFile.Fd()))
}

func ColorEnabled(streams domain.IOStreams) bool {
	if !IsTerminal(streams) {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return true
}
