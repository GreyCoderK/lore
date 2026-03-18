// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"

	"github.com/greycoderk/lore/internal/domain"
)

func ActionableError(streams domain.IOStreams, message string, command string) {
	_, _ = fmt.Fprintf(streams.Err, "%s %s\n", Error("Error:"), message)
	_, _ = fmt.Fprintf(streams.Err, "  %s %s\n", Dim("Run:"), command)
}
