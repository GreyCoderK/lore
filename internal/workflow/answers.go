package workflow

import (
	"github.com/museigen/lore/internal/domain"
	"github.com/museigen/lore/internal/generator"
)

// Answers holds the user's responses to the interactive question flow.
type Answers struct {
	Type         string
	What         string
	Why          string
	Alternatives string // empty if express mode skipped
	Impact       string // empty if express mode skipped
}

// ToGenerateInput converts Answers + CommitInfo into a generator.GenerateInput.
// The conversion happens in workflow/ to avoid circular deps (generator → workflow).
// generatedBy distinguishes hook-triggered ("hook") from manual ("manual") flows so that
// the front-matter field is correct for both reactive.go and the future proactive.go (Story 2.7).
// CONSOLIDATE: Story 2.7 — extraire helper toAnswersInput() si duplication significative avec proactive.go.
func (a Answers) ToGenerateInput(commit *domain.CommitInfo, generatedBy string) generator.GenerateInput {
	return generator.GenerateInput{
		DocType:      a.Type,
		What:         a.What,
		Why:          a.Why,
		Alternatives: a.Alternatives,
		Impact:       a.Impact,
		CommitInfo:   commit,
		GeneratedBy:  generatedBy,
	}
}
