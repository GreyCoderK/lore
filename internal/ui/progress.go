package ui

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// Progress displays "[##·] 3/5 label" on stderr.
func Progress(streams domain.IOStreams, current, total int, label string) {
	if current < 0 {
		current = 0
	}
	if total < 0 {
		total = 0
	}
	if current > total {
		current = total
	}
	bar := strings.Repeat("#", current) + strings.Repeat("·", total-current)
	fmt.Fprintf(streams.Err, "[%s] %d/%d %s\n", bar, current, total, label)
}
