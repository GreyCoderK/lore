package ui

import (
	"fmt"

	"github.com/museigen/lore/internal/domain"
)

func Verb(streams domain.IOStreams, verb string, message string) {
	formatted := fmt.Sprintf("%10s", verb)
	if colorEnabled {
		fmt.Fprintf(streams.Err, "%s %s\n", Success(formatted), message)
	} else {
		fmt.Fprintf(streams.Err, "%s %s\n", formatted, message)
	}
}
