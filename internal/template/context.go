// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package template

// TemplateContext holds all variables passed to templates during rendering.
// Built by the caller (workflow/ or cmd/) from domain.GenerateInput and domain.DocMeta.
// The template package never constructs TemplateContext itself.
type TemplateContext struct {
	Type         string
	Title        string
	Date         string // YYYY-MM-DD
	Commit       string
	Author       string
	Tags         []string
	Why          string
	Impact       string
	Alternatives string
	GeneratedBy  string // "hook" or "manual"
	AngelaMode   string
}
