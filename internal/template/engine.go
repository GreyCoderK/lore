package template

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed defaults/*.tmpl
var defaultTemplates embed.FS

// Engine wraps text/template with embedded defaults and custom functions.
type Engine struct {
	tmpl *template.Template
}

// New creates a template engine with embedded default templates.
func New() (*Engine, error) {
	// CRITICAL: Funcs() MUST be called BEFORE ParseFS()
	tmpl := template.New("").Funcs(template.FuncMap{
		"formatDate": formatDate,
		"slugify":    slugify,
	})
	tmpl, err := tmpl.ParseFS(defaultTemplates, "defaults/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("template: parse defaults: %w", err)
	}
	return &Engine{tmpl: tmpl}, nil
}

// Render executes a named template with the given data.
func (e *Engine) Render(name string, data any) (string, error) {
	var buf bytes.Buffer
	if err := e.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("template: render %s: %w", name, err)
	}
	return buf.String(), nil
}
