// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package template

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed defaults/*.tmpl
var defaultTemplates embed.FS

// funcMap returns the shared template function map.
func funcMap() template.FuncMap {
	return template.FuncMap{
		"formatDate": formatDate,
		"slugify":    slugify,
		"commitLink": commitLink,
	}
}

// GlobalDir returns the user-global template directory (~/.config/lore/templates).
// Returns empty string if the home directory cannot be determined.
func GlobalDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".config", "lore", "templates")
}

// Engine wraps text/template with hierarchical resolution and custom functions.
type Engine struct {
	localDir  string             // .lore/templates/ (project-local)
	globalDir string             // ~/.config/lore/templates/ (user-global)
	defaults  *template.Template // embedded defaults parsed from embed.FS
	funcs     template.FuncMap
}

// New creates a template engine with hierarchical resolution.
// localDir is the project-local template directory (e.g. ".lore/templates").
// globalDir is the user-global template directory (e.g. "~/.config/lore/templates").
func New(localDir, globalDir string) (*Engine, error) {
	fmap := funcMap()
	// Parse embedded defaults with custom functions registered.
	tmpl := template.New("").Funcs(fmap)
	tmpl, err := tmpl.ParseFS(defaultTemplates, "defaults/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("template: parse defaults: %w", err)
	}
	return &Engine{
		localDir:  localDir,
		globalDir: globalDir,
		defaults:  tmpl,
		funcs:     fmap,
	}, nil
}

// resolve finds a template by name using the hierarchy: local > global > embedded defaults.
func (e *Engine) resolve(name string) (*template.Template, error) {
	// 1. Local: project .lore/templates/
	if e.localDir != "" {
		tmpl, found, err := e.tryReadTemplate(filepath.Join(e.localDir, name), name)
		if err != nil {
			return nil, fmt.Errorf("template: resolve %s: local: %w", name, err)
		}
		if found {
			return tmpl, nil
		}
	}

	// 2. Global: ~/.config/lore/templates/
	if e.globalDir != "" {
		tmpl, found, err := e.tryReadTemplate(filepath.Join(e.globalDir, name), name)
		if err != nil {
			return nil, fmt.Errorf("template: resolve %s: global: %w", name, err)
		}
		if found {
			return tmpl, nil
		}
	}

	// 3. Default: embed.FS
	tmpl := e.defaults.Lookup(name)
	if tmpl == nil {
		return nil, fmt.Errorf("template: resolve %s: not found in local, global, or defaults", name)
	}
	return tmpl, nil
}

// tryReadTemplate attempts to read and parse a template file.
// Returns (template, true, nil) if found and parsed,
// (nil, false, nil) if file does not exist,
// (nil, false, err) if file exists but cannot be read or parsed.
func (e *Engine) tryReadTemplate(path, name string) (*template.Template, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	tmpl, parseErr := template.New(name).Funcs(e.funcs).Parse(string(data))
	if parseErr != nil {
		return nil, false, parseErr
	}
	return tmpl, true, nil
}

// Render resolves a named template via the hierarchy and executes it with the given data.
func (e *Engine) Render(name string, data TemplateContext) (string, error) {
	tmpl, err := e.resolve(name)
	if err != nil {
		return "", fmt.Errorf("template: render %s: %w", name, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("template: render %s: %w", name, err)
	}
	return buf.String(), nil
}

// TemplateExists checks whether a template with the given name is available
// in any level of the hierarchy (local, global, or embedded defaults).
func (e *Engine) TemplateExists(name string) bool {
	_, err := e.resolve(name)
	return err == nil
}
