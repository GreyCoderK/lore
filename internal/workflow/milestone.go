package workflow

import (
	"fmt"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/engagement"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
)

// showMilestone displays a milestone reinforcement message on stderr when the
// document count in docsDir hits an exact threshold (3, 10, 25, 50).
// When tty is false the call is a no-op (AC-5: non-TTY suppression).
//
// Error handling: ListDocs returns (partialDocs, joinedParseErrors) when some
// files fail to parse, or (nil, nil) when the directory does not exist. Both
// cases are acceptable — we use len(docs) regardless (len(nil) == 0 in Go)
// so that a malformed or missing docs dir silently suppresses milestones
// rather than crashing the workflow.
func showMilestone(streams domain.IOStreams, docsDir string, tty bool) {
	if !tty {
		return
	}
	store := &storage.CorpusStore{Dir: docsDir}
	docs, _ := store.ListDocs(domain.DocFilter{})
	msg, ok := engagement.GetMilestoneMessage(len(docs))
	if !ok {
		return
	}
	fmt.Fprintf(streams.Err, "%10s %s\n", "", ui.Dim(msg))
}
