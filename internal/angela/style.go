// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
)

// StyleGuide holds parsed style rules for Angela draft analysis.
type StyleGuide struct {
	RequireWhy          bool
	RequireAlternatives bool
	MaxBodyLength       int // in runes, 0 = no limit
	MinTags             int

	// Warnings collects any parse-time issues (e.g. unknown rules).
	Warnings []Suggestion
}

// knownRules is the set of recognized style guide keys.
var knownRules = map[string]bool{
	"require_why":          true,
	"require_alternatives": true,
	"max_body_length":      true,
	"min_tags":             true,
}

// ParseStyleGuide parses style guide rules from config.
// Returns default rules if input is nil.
// Unknown keys produce warning suggestions (catches typos in .lorerc).
func ParseStyleGuide(rules map[string]interface{}) *StyleGuide {
	guide := &StyleGuide{
		RequireWhy:          true,
		RequireAlternatives: false,
		MaxBodyLength:       0,
		MinTags:             0,
	}

	if rules == nil {
		return guide
	}

	for key := range rules {
		if !knownRules[key] {
			guide.Warnings = append(guide.Warnings, Suggestion{
				Category: "style",
				Severity: "info",
				Message:  fmt.Sprintf("style guide: unknown rule %q (ignored)", key),
			})
		}
	}

	if v, ok := rules["require_why"]; ok {
		if b, ok := v.(bool); ok {
			guide.RequireWhy = b
		}
	}
	if v, ok := rules["require_alternatives"]; ok {
		if b, ok := v.(bool); ok {
			guide.RequireAlternatives = b
		}
	}
	if v, ok := rules["max_body_length"]; ok {
		switch n := v.(type) {
		case int:
			guide.MaxBodyLength = n
		case float64:
			guide.MaxBodyLength = int(n)
		}
	}
	if v, ok := rules["min_tags"]; ok {
		switch n := v.(type) {
		case int:
			guide.MinTags = n
		case float64:
			guide.MinTags = int(n)
		}
	}

	return guide
}

// FormatStyleGuideRules returns a prompt-ready string describing the active rules.
// Returns empty string if no rules are active.
func FormatStyleGuideRules(guide *StyleGuide) string {
	if guide == nil {
		return ""
	}
	var b strings.Builder
	if guide.RequireWhy {
		b.WriteString("- Section '## Why' is required\n")
	}
	if guide.RequireAlternatives {
		b.WriteString("- Section '## Alternatives' is required\n")
	}
	if guide.MaxBodyLength > 0 {
		fmt.Fprintf(&b, "- Maximum body length: %d characters\n", guide.MaxBodyLength)
	}
	if guide.MinTags > 0 {
		fmt.Fprintf(&b, "- Minimum %d tags required per document\n", guide.MinTags)
	}
	return b.String()
}
