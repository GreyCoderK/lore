package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngine_RenderDecisionTemplate(t *testing.T) {
	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	data := TemplateContext{
		Title:        "Add JWT auth middleware",
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
	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	_, err = engine.Render("nonexistent.tmpl", TemplateContext{})
	if err == nil {
		t.Error("expected error for unknown template")
	}
}

func TestEngine_TemplateExists(t *testing.T) {
	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	if !engine.TemplateExists("decision.md.tmpl") {
		t.Error("expected decision.md.tmpl to exist")
	}
	if engine.TemplateExists("nonexistent.tmpl") {
		t.Error("expected nonexistent.tmpl to not exist")
	}
}

func TestEngine_AllDefaultTemplatesExist(t *testing.T) {
	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	templates := []string{
		"decision.md.tmpl",
		"feature.md.tmpl",
		"bugfix.md.tmpl",
		"refactor.md.tmpl",
		"release.md.tmpl",
		"note.md.tmpl",
	}

	for _, name := range templates {
		if !engine.TemplateExists(name) {
			t.Errorf("expected default template %s to exist", name)
		}
	}
}

func TestEngine_RenderAllTemplates(t *testing.T) {
	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	data := TemplateContext{
		Type:         "feature",
		Title:        "Test Feature",
		Date:         "2026-03-12",
		Commit:       "abc1234def5678",
		Author:       "test-author",
		Tags:         []string{"test", "feature"},
		Why:          "Because testing is important",
		Impact:       "All tests pass",
		Alternatives: "Do nothing",
		GeneratedBy:  "manual",
		AngelaMode:   "",
	}

	templates := []string{
		"decision.md.tmpl",
		"feature.md.tmpl",
		"bugfix.md.tmpl",
		"refactor.md.tmpl",
		"release.md.tmpl",
		"note.md.tmpl",
	}

	for _, name := range templates {
		result, err := engine.Render(name, data)
		if err != nil {
			t.Errorf("render %s: %v", name, err)
			continue
		}

		// AC-3: Markdown standard, no custom syntax
		if !strings.Contains(result, "# Test Feature") {
			t.Errorf("%s: expected title heading in output", name)
		}
		if strings.Contains(result, "{{") {
			t.Errorf("%s: output contains unresolved template syntax", name)
		}
	}
}

func TestEngine_TemplateContextAllFields(t *testing.T) {
	// AC-4: ALL 11 variables must be accessible to templates.
	// Use a custom template that renders every field to prove availability.
	localDir := t.TempDir()

	allFieldsTmpl := `Type={{.Type}}
Title={{.Title}}
Date={{.Date}}
Commit={{.Commit}}
Author={{.Author}}
Tags={{range $i, $t := .Tags}}{{if $i}},{{end}}{{$t}}{{end}}
Why={{.Why}}
Impact={{.Impact}}
Alternatives={{.Alternatives}}
GeneratedBy={{.GeneratedBy}}
AngelaMode={{.AngelaMode}}`

	if err := os.WriteFile(filepath.Join(localDir, "allfields.md.tmpl"), []byte(allFieldsTmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := New(localDir, "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	data := TemplateContext{
		Type:         "decision",
		Title:        "Context Test",
		Date:         "2026-01-15",
		Commit:       "deadbeef",
		Author:       "tester",
		Tags:         []string{"tag1", "tag2"},
		Why:          "Why value here",
		Impact:       "Impact value here",
		Alternatives: "Alt value here",
		GeneratedBy:  "hook",
		AngelaMode:   "draft",
	}

	result, err := engine.Render("allfields.md.tmpl", data)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	checks := map[string]string{
		"Type":         "Type=decision",
		"Title":        "Title=Context Test",
		"Date":         "Date=2026-01-15",
		"Commit":       "Commit=deadbeef",
		"Author":       "Author=tester",
		"Tags":         "Tags=tag1,tag2",
		"Why":          "Why=Why value here",
		"Impact":       "Impact=Impact value here",
		"Alternatives": "Alternatives=Alt value here",
		"GeneratedBy":  "GeneratedBy=hook",
		"AngelaMode":   "AngelaMode=draft",
	}

	for field, expected := range checks {
		if !strings.Contains(result, expected) {
			t.Errorf("AC-4: %s not accessible in template. Expected %q in output:\n%s", field, expected, result)
		}
	}
}

// TestEngine_HierarchyLocalOverridesDefault verifies AC-1: local > global > defaults.
func TestEngine_HierarchyLocalOverridesDefault(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	// Create local override for decision.md.tmpl
	localContent := "# LOCAL: {{.Title}}\nLocal override body"
	if err := os.WriteFile(filepath.Join(localDir, "decision.md.tmpl"), []byte(localContent), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := New(localDir, globalDir)
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	result, err := engine.Render("decision.md.tmpl", TemplateContext{Title: "Override Test"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "# LOCAL: Override Test") {
		t.Errorf("expected local override, got: %s", result)
	}
}

// TestEngine_HierarchyGlobalOverridesDefault verifies global > defaults.
func TestEngine_HierarchyGlobalOverridesDefault(t *testing.T) {
	localDir := t.TempDir() // empty, no override
	globalDir := t.TempDir()

	// Create global override for decision.md.tmpl
	globalContent := "# GLOBAL: {{.Title}}\nGlobal override body"
	if err := os.WriteFile(filepath.Join(globalDir, "decision.md.tmpl"), []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := New(localDir, globalDir)
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	result, err := engine.Render("decision.md.tmpl", TemplateContext{Title: "Global Test"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "# GLOBAL: Global Test") {
		t.Errorf("expected global override, got: %s", result)
	}
}

// TestEngine_HierarchyLocalTakesPriority verifies local > global when both exist.
func TestEngine_HierarchyLocalTakesPriority(t *testing.T) {
	localDir := t.TempDir()
	globalDir := t.TempDir()

	localContent := "# LOCAL: {{.Title}}"
	globalContent := "# GLOBAL: {{.Title}}"

	if err := os.WriteFile(filepath.Join(localDir, "decision.md.tmpl"), []byte(localContent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "decision.md.tmpl"), []byte(globalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := New(localDir, globalDir)
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	result, err := engine.Render("decision.md.tmpl", TemplateContext{Title: "Priority Test"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "# LOCAL: Priority Test") {
		t.Errorf("expected local to win over global, got: %s", result)
	}
}

// TestEngine_HierarchyFallbackToDefaults verifies fallback to embed.FS when no overrides.
func TestEngine_HierarchyFallbackToDefaults(t *testing.T) {
	localDir := t.TempDir()  // empty
	globalDir := t.TempDir() // empty

	engine, err := New(localDir, globalDir)
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	result, err := engine.Render("decision.md.tmpl", TemplateContext{Title: "Fallback Test", Why: "test"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "# Fallback Test") {
		t.Errorf("expected default template output, got: %s", result)
	}
}

// TestEngine_OverrideCanUseCustomFuncs verifies that overrides have access to custom funcs.
func TestEngine_OverrideCanUseCustomFuncs(t *testing.T) {
	localDir := t.TempDir()

	content := `# {{.Title}}
Commit: {{commitLink .Commit}}
Slug: {{slugify .Title}}`
	if err := os.WriteFile(filepath.Join(localDir, "decision.md.tmpl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	engine, err := New(localDir, "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	result, err := engine.Render("decision.md.tmpl", TemplateContext{Title: "Func Test", Commit: "abc1234"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.Contains(result, "[`abc1234`]") {
		t.Errorf("expected commitLink in output, got: %s", result)
	}
	if !strings.Contains(result, "func-test") {
		t.Errorf("expected slugify in output, got: %s", result)
	}
}

// TestEngine_GoldenFiles compares rendered output against golden files for each template type.
func TestEngine_GoldenFiles(t *testing.T) {
	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	data := TemplateContext{
		Title:        "Add JWT auth middleware",
		Why:          "Stateless authentication for microservices",
		Alternatives: "- Session-based auth with Redis\n- OAuth2 with external provider",
		Impact:       "- API routes now require Bearer token",
	}

	types := []string{"decision", "feature", "bugfix", "refactor", "release", "note"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			result, err := engine.Render(typ+".md.tmpl", data)
			if err != nil {
				t.Fatalf("render %s: %v", typ, err)
			}

			goldenPath := filepath.Join("testdata", typ+".md.golden")
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden file %s: %v", goldenPath, err)
			}

			if result != string(want) {
				t.Errorf("%s output mismatch.\n--- got ---\n%s\n--- want ---\n%s", typ, result, string(want))
			}
		})
	}
}

// TestEngine_Integration_AllTypes is an integration test verifying all 6 template types
// produce valid Markdown with a full TemplateContext.
func TestEngine_Integration_AllTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	engine, err := New("", "")
	if err != nil {
		t.Fatalf("template: new engine: %v", err)
	}

	data := TemplateContext{
		Type:         "feature",
		Title:        "Integration Test Feature",
		Date:         "2026-03-12",
		Commit:       "abc1234def5678",
		Author:       "integration-tester",
		Tags:         []string{"integration", "test"},
		Why:          "Verify all templates render correctly",
		Impact:       "Ensures template engine quality",
		Alternatives: "Manual testing",
		GeneratedBy:  "manual",
		AngelaMode:   "draft",
	}

	types := []string{"decision", "feature", "bugfix", "refactor", "release", "note"}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			result, err := engine.Render(typ+".md.tmpl", data)
			if err != nil {
				t.Fatalf("render %s: %v", typ, err)
			}

			// Must contain a Markdown heading
			if !strings.Contains(result, "# Integration Test Feature") {
				t.Errorf("%s: missing title heading", typ)
			}

			// Must be standard Markdown — no template syntax
			if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
				t.Errorf("%s: contains unresolved template syntax", typ)
			}

			// Must not be empty
			if len(strings.TrimSpace(result)) == 0 {
				t.Errorf("%s: empty output", typ)
			}
		})
	}
}
