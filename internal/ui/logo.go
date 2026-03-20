// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

var logoUnicode = ` ██╗      ██████╗ ██████╗ ███████╗
 ██║     ██╔═══██╗██╔══██╗██╔════╝
 ██║     ██║   ██║██████╔╝█████╗
 ██║     ██║   ██║██╔══██╗██╔══╝
 ███████╗╚██████╔╝██║  ██║███████╗
 ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚══════╝`

var logoCompact = `╦  ╔═╗ ╦═╗ ╔═╗
║  ║ ║ ╠╦╝ ╠═╣
╩═╝╚═╝ ╩╚═ ╚═╝`

var logoASCII = `+---+ +---+ +--+ +---+
|   | | | | |++| |
+---+ +---+ +  + +---+`

const tagline = "your code knows what. lore knows why."

func supportsUnicode() bool {
	for _, env := range []string{"LANG", "LC_CTYPE", "LC_ALL"} {
		if strings.Contains(os.Getenv(env), "UTF-8") {
			return true
		}
	}
	return false
}

func termWidth() int {
	// Default width; if detection is needed later, this is the hook point.
	return 80
}

// pickLogo selects the best logo variant for the current terminal.
func pickLogo() string {
	if !supportsUnicode() {
		return logoASCII
	}
	if termWidth() >= 40 {
		return logoUnicode
	}
	return logoCompact
}

// colorizeLogo applies a cyan color to the logo text when color is enabled.
func colorizeLogo(logo string) string {
	if !isColorEnabled() {
		return logo
	}
	return fmt.Sprintf("\033[1;36m%s\033[0m", logo)
}

// PrintLogo displays the ASCII wordmark on stderr.
func PrintLogo(streams domain.IOStreams) {
	logo := pickLogo()
	colored := colorizeLogo(logo)
	tag := Dim(tagline)
	fmt.Fprintf(streams.Err, "\n%s\n  %s\n\n", colored, tag)
}
