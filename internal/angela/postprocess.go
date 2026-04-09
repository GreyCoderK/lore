// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
)

// PostProcess applies local text transformations to improve AI output quality.
// No API calls — pure string processing. Applied after AI response, before diff.
// Each transformation is idempotent and independent.
func PostProcess(original, polished string) string {
	polished = restoreHeadingNumbers(original, polished)
	polished = normalizeCodeFenceLanguages(polished)
	polished = normalizeMermaidIndent(polished)
	return polished
}

// restoreHeadingNumbers re-applies section numbers from the original if the AI stripped them.
// Example: original "## 4. Logique Métier", polished "## Logique Métier" → "## 4. Logique Métier"
func restoreHeadingNumbers(original, polished string) string {
	origLines := strings.Split(original, "\n")
	polLines := strings.Split(polished, "\n")

	// Build map: heading text without number → full heading with number
	numbered := make(map[string]string) // "Logique Métier" → "## 4. Logique Métier"
	for _, line := range origLines {
		trimmed := strings.TrimSpace(line)
		prefix, text := parseHeading(trimmed)
		if prefix == "" {
			continue
		}
		stripped := stripNumber(text)
		if stripped != text {
			// Original has a number, store mapping
			numbered[strings.ToLower(prefix+stripped)] = trimmed
		}
	}

	if len(numbered) == 0 {
		return polished
	}

	// Apply to polished lines
	for i, line := range polLines {
		trimmed := strings.TrimSpace(line)
		prefix, text := parseHeading(trimmed)
		if prefix == "" {
			continue
		}
		key := strings.ToLower(prefix + text)
		if original, ok := numbered[key]; ok {
			polLines[i] = original
		}
	}

	return strings.Join(polLines, "\n")
}

// parseHeading returns ("## ", "Title") for "## Title", or ("", "") if not a heading.
func parseHeading(line string) (string, string) {
	if strings.HasPrefix(line, "### ") {
		return "### ", strings.TrimSpace(line[4:])
	}
	if strings.HasPrefix(line, "## ") {
		return "## ", strings.TrimSpace(line[3:])
	}
	return "", ""
}

// stripNumber removes leading "1. ", "4.2 ", etc. from a heading text.
func stripNumber(text string) string {
	i := 0
	for i < len(text) && (text[i] >= '0' && text[i] <= '9' || text[i] == '.') {
		i++
	}
	if i > 0 && i < len(text) && text[i] == ' ' {
		return strings.TrimSpace(text[i+1:])
	}
	return text
}

// normalizeCodeFenceLanguages adds language tags to bare code fences by detecting content.
func normalizeCodeFenceLanguages(content string) string {
	lines := strings.Split(content, "\n")
	inFence := false
	fenceStart := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "```") && !inFence {
			inFence = true
			fenceStart = i
			continue
		}
		if trimmed == "```" && inFence {
			inFence = false
			fenceStart = -1
			continue
		}

		// If we just entered a bare fence (```\n), detect language from first content line
		if inFence && fenceStart == i-1 && strings.TrimSpace(lines[fenceStart]) == "```" {
			lang := DetectLanguage(trimmed)
			if lang != "" {
				lines[fenceStart] = "```" + lang
			}
		}
	}

	return strings.Join(lines, "\n")
}


// normalizeMermaidIndent ensures mermaid diagram content is indented with 4 spaces.
func normalizeMermaidIndent(content string) string {
	lines := strings.Split(content, "\n")
	inMermaid := false
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "```mermaid" {
			inMermaid = true
			result = append(result, line)
			continue
		}

		if inMermaid && trimmed == "```" {
			inMermaid = false
			result = append(result, line)
			continue
		}

		if inMermaid && trimmed != "" {
			// First line after ```mermaid is the diagram type (graph TD, sequenceDiagram, etc.)
			// Don't indent diagram type declarations
			if strings.HasPrefix(trimmed, "graph ") || strings.HasPrefix(trimmed, "sequenceDiagram") ||
				strings.HasPrefix(trimmed, "stateDiagram") || strings.HasPrefix(trimmed, "flowchart ") ||
				strings.HasPrefix(trimmed, "classDiagram") || strings.HasPrefix(trimmed, "gantt") ||
				strings.HasPrefix(trimmed, "pie") || strings.HasPrefix(trimmed, "erDiagram") {
				result = append(result, fmt.Sprintf("    %s", trimmed))
			} else if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
				// Already indented, keep as-is
				result = append(result, line)
			} else {
				// Indent content lines
				result = append(result, fmt.Sprintf("        %s", trimmed))
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
