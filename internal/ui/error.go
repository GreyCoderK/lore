// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

func ActionableError(streams domain.IOStreams, message string, command string) {
	_, _ = fmt.Fprintf(streams.Err, "%s %s\n", Error(i18n.T().UI.ErrorPrefix), message)
	_, _ = fmt.Fprintf(streams.Err, "  %s %s\n", Dim(i18n.T().UI.RunPrefix), command)
}
