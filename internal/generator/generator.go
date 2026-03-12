package generator

import (
	"context"
	"fmt"
	"time"

	loretemplate "github.com/museigen/lore/internal/template"
)

// DemoInput is the minimal struct for Story 1.4 (demo).
// Story 2.6 will replace it with a full GenerateInput/GenerateResult pattern.
type DemoInput struct {
	Type         string
	What         string
	Why          string
	Commit       string
	Date         time.Time
	Status       string
	Tags         []string
	GeneratedBy  string
	Alternatives string
	Impact       string
}

// Generate renders the template and returns generated content (string).
// The caller (cmd/demo.go) is responsible for calling storage.WriteDoc().
func Generate(ctx context.Context, engine *loretemplate.Engine, input DemoInput) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	body, err := engine.Render("decision.md.tmpl", input)
	if err != nil {
		return "", fmt.Errorf("generator: render: %w", err)
	}
	return body, nil
}
