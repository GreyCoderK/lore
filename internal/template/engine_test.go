package template

import (
	"strings"
	"testing"
)

func TestEngine_RenderDecisionTemplate(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	data := struct {
		What         string
		Why          string
		Alternatives string
		Impact       string
	}{
		What:         "Add JWT auth middleware",
		Why:          "Stateless authentication for microservices",
		Alternatives: "- Session-based auth with Redis\n- OAuth2 with external provider",
		Impact:       "- API routes now require Bearer token\n- Auth middleware added to router chain",
	}

	result, err := engine.Render("decision.md.tmpl", data)
	if err != nil {
		t.Fatalf("template: render: %v", err)
	}

	if !strings.Contains(result, "# Add JWT auth middleware") {
		t.Error("expected title in output")
	}
	if !strings.Contains(result, "Stateless authentication") {
		t.Error("expected why section")
	}
	if !strings.Contains(result, "Session-based auth") {
		t.Error("expected alternatives section")
	}
}

func TestEngine_RenderUnknownTemplate(t *testing.T) {
	engine, err := New()
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	_, err = engine.Render("nonexistent.tmpl", nil)
	if err == nil {
		t.Error("expected error for unknown template")
	}
}

func TestFuncs_Slugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Hello World", "hello-world"},
		{"JWT Auth Middleware", "jwt-auth-middleware"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
