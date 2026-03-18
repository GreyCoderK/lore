// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

var logoUnicode = `╦  ╔═╗ ╦═╗ ╔═╗
║  ║ ║ ╠╦╝ ╠═╣
╩═╝╚═╝ ╩╚═ ╚═╝`

var logoASCII = `+---+ +---+ +--+ +---+
|   | | | | |++| |
+---+ +---+ +  + +---+`

func supportsUnicode() bool {
	for _, env := range []string{"LANG", "LC_CTYPE", "LC_ALL"} {
		if strings.Contains(os.Getenv(env), "UTF-8") {
			return true
		}
	}
	return false
}

// PrintLogo displays the ASCII wordmark on stderr.
func PrintLogo(streams domain.IOStreams) {
	logo := logoASCII
	if supportsUnicode() {
		logo = logoUnicode
	}
	fmt.Fprintf(streams.Err, "\n%s\n\n", logo)
}
