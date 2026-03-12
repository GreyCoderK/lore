package generator

import (
	"context"
	"strings"
	"testing"
	"time"

	loretemplate "github.com/museigen/lore/internal/template"
)

func TestGenerate_ProducesContent(t *testing.T) {
	engine, err := loretemplate.New()
	if err != nil {
		t.Fatalf("template: new: %v", err)
	}

	input := DemoInput{
		Type:         "decision",
		What:         "Add JWT auth middleware",
		Why:          "Stateless authentication for microservices",
		Commit:       "abc1234",
		Date:         time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC),
		Status:       "demo",
		Tags:         []string{"auth", "jwt"},
		GeneratedBy:  "lore-demo",
		Alternatives: "- Session-based auth with Redis\n- OAuth2 with external provider",
		Impact:       "- API routes now require Bearer token",
	}

	body, err := Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("generator: generate: %v", err)
	}

	if !strings.Contains(body, "Add JWT auth middleware") {
		t.Error("expected 'what' in body")
	}
	if !strings.Contains(body, "Stateless authentication") {
		t.Error("expected 'why' in body")
	}
}

func TestGenerate_CancelledContext(t *testing.T) {
	engine, err := loretemplate.New()
	if err != nil {
		t.Fatalf("template: new: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = Generate(ctx, engine, DemoInput{})
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
