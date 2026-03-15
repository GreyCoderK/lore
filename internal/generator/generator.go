package generator

import (
	"context"
	"fmt"
	"time"

	"github.com/museigen/lore/internal/domain"
	loretemplate "github.com/museigen/lore/internal/template"
)

// GenerateInput holds all inputs for document generation.
// The caller (workflow/) converts Answers + CommitInfo into this flat struct.
// generator/ does NOT import workflow/ or storage/ (CORRECTION C1).
type GenerateInput struct {
	DocType      string
	What         string
	Why          string
	Alternatives string // empty if express mode
	Impact       string // empty if express mode
	CommitInfo   *domain.CommitInfo
	GeneratedBy  string // "hook" or "manual"
}

// Date returns the commit date formatted as YYYY-MM-DD, or today if CommitInfo is nil.
func (g GenerateInput) Date() string {
	if g.CommitInfo != nil && !g.CommitInfo.Date.IsZero() {
		return g.CommitInfo.Date.Format("2006-01-02")
	}
	return time.Now().Format("2006-01-02")
}

// GenerateResult holds the rendered Markdown body and the document metadata
// built from the input. The caller (workflow/) passes Body + Meta to storage.WriteDoc().
// Filename and Path are NOT included — those are determined by storage/ (CORRECTION C1).
// date is captured once inside Generate() so TemplateContext and Meta are always consistent.
type GenerateResult struct {
	Body string         // Markdown content rendered from the template
	Meta domain.DocMeta // Document metadata derived from GenerateInput
}

// Generate renders a document template from the given input and returns a
// GenerateResult containing the body and pre-built metadata. The caller is
// responsible for persisting the result via storage.WriteDoc() — generator/
// never touches the filesystem (CORRECTION C1).
func Generate(ctx context.Context, engine *loretemplate.Engine, input GenerateInput) (GenerateResult, error) {
	select {
	case <-ctx.Done():
		return GenerateResult{}, ctx.Err()
	default:
	}

	commit := ""
	author := ""
	if input.CommitInfo != nil {
		commit = input.CommitInfo.Hash
		author = input.CommitInfo.Author
	}

	// Capture date once — guarantees TemplateContext and DocMeta are always consistent
	// even if time.Now() crosses midnight between two calls (M8 fix).
	date := input.Date()

	tc := loretemplate.TemplateContext{
		Type:         input.DocType,
		Title:        input.What,
		Date:         date,
		Commit:       commit,
		Author:       author,
		Why:          input.Why,
		Impact:       input.Impact,
		Alternatives: input.Alternatives,
		GeneratedBy:  input.GeneratedBy,
	}

	tmplName := input.DocType + ".md.tmpl"
	body, err := engine.Render(tmplName, tc)
	if err != nil {
		return GenerateResult{}, fmt.Errorf("generator: render: %w", err)
	}

	meta := domain.DocMeta{
		Type:        input.DocType,
		Date:        date,
		Commit:      commit,
		Status:      "draft",
		GeneratedBy: input.GeneratedBy,
	}

	return GenerateResult{Body: body, Meta: meta}, nil
}
