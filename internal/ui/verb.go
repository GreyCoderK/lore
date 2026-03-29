// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

func Verb(streams domain.IOStreams, verb string, message string) {
	formatted := fmt.Sprintf("%10s", verb)
	if isColorEnabled() {
		fmt.Fprintf(streams.Err, "%s %s\n", Success(formatted), message)
	} else {
		fmt.Fprintf(streams.Err, "%s %s\n", formatted, message)
	}
}

func VerbDelete(streams domain.IOStreams, message string) {
	formatted := fmt.Sprintf("%10s", i18n.T().UI.VerbDeleted)
	if isColorEnabled() {
		fmt.Fprintf(streams.Err, "%s %s\n", Error(formatted), message)
	} else {
		fmt.Fprintf(streams.Err, "%s %s\n", formatted, message)
	}
}
