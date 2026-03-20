// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// BuildPolishPrompt constructs the AI prompt for polishing a document.
// The AI is asked to return the COMPLETE document (not a diff).
// When personas is non-nil, persona directives are injected into the prompt (AC-4).
func BuildPolishPrompt(doc string, meta domain.DocMeta, styleGuide string, corpusSummary string, personas []PersonaProfile) string {
	var sb strings.Builder

	sb.WriteString("You are Angela, an expert documentation reviewer for the Lore project.\n")
	sb.WriteString("Your task: improve the clarity, structure, and completeness of the following document.\n\n")

	// Inject persona directives (AC-4)
	if len(personas) > 0 {
		sb.WriteString(BuildPersonaPrompt(personas))
	}

	sb.WriteString("RULES:\n")
	sb.WriteString("- Return the COMPLETE modified document (front matter + body)\n")
	sb.WriteString("- Do NOT modify front matter fields (type, date, commit, status, tags, related, generated_by)\n")
	sb.WriteString("- Improve clarity, structure, grammar, and completeness of the body\n")
	sb.WriteString("- Do NOT invent new content — only reformulate and restructure what exists\n")
	sb.WriteString("- Preserve all Markdown formatting (headers, lists, code blocks)\n")
	sb.WriteString("- If the document is already good, return it unchanged\n\n")

	if styleGuide != "" {
		sb.WriteString("PROJECT STYLE GUIDE (between markers):\n")
		sb.WriteString("<<<STYLE_GUIDE>>>\n")
		sb.WriteString(styleGuide)
		sb.WriteString("\n<<<END_STYLE_GUIDE>>>\n\n")
	}

	if corpusSummary != "" {
		sb.WriteString("EXISTING CORPUS (for context only, between markers):\n")
		sb.WriteString("<<<CORPUS>>>\n")
		sb.WriteString(corpusSummary)
		sb.WriteString("\n<<<END_CORPUS>>>\n\n")
	}

	// Use unique delimiters to prevent prompt injection via triple backticks in document content
	sb.WriteString("DOCUMENT TO IMPROVE (between <<<DOCUMENT>>> and <<<END_DOCUMENT>>> markers):\n")
	sb.WriteString("<<<DOCUMENT>>>\n")
	sb.WriteString(doc)
	sb.WriteString("\n<<<END_DOCUMENT>>>\n\n")
	sb.WriteString("Return ONLY the improved document content. No explanations, no wrapping, no <<<DOCUMENT>>> markers.")

	return sb.String()
}

// BuildCorpusSummary creates a compact summary of corpus metadata for the prompt.
// Limited to 20 entries to respect token limits.
func BuildCorpusSummary(corpus []domain.DocMeta) string {
	limit := 20
	if len(corpus) < limit {
		limit = len(corpus)
	}

	var sb strings.Builder
	for i := 0; i < limit; i++ {
		meta := corpus[i]
		sb.WriteString(fmt.Sprintf("- [%s] %s", meta.Type, meta.Filename))
		if len(meta.Tags) > 0 {
			sb.WriteString(fmt.Sprintf(" (tags: %s)", strings.Join(meta.Tags, ", ")))
		}
		sb.WriteString("\n")
	}
	if len(corpus) > 20 {
		sb.WriteString(fmt.Sprintf("... and %d more documents\n", len(corpus)-20))
	}
	return sb.String()
}

// Polish sends a document to the AI provider for enhancement.
// Returns the improved document content. Exactly 1 API call.
func Polish(ctx context.Context, provider domain.AIProvider, doc string, meta domain.DocMeta, styleGuide string, corpusSummary string, personas []PersonaProfile) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("angela: polish: no AI provider configured")
	}

	prompt := BuildPolishPrompt(doc, meta, styleGuide, corpusSummary, personas)
	result, err := provider.Complete(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("angela: polish: %w", err)
	}

	// Strip markdown fencing if AI wraps the response
	result = stripCodeFence(result)

	return result, nil
}

// stripCodeFence removes ```...``` wrapping if the AI added it.
// Only strips if BOTH opening and closing fences are present and balanced.
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)

	// Check for opening fence on first line
	if !strings.HasPrefix(s, "```") {
		return s
	}

	// Find end of first line (the opening fence line)
	idx := strings.Index(s, "\n")
	if idx < 0 {
		return s // single line starting with ``` — don't strip
	}

	// Check for closing fence on last line
	if !strings.HasSuffix(s, "\n```") && !strings.HasSuffix(s, "\n``` ") {
		// Closing fence must be on its own line
		if !strings.HasSuffix(strings.TrimSpace(s), "```") || strings.Count(s, "```") < 2 {
			return s // no balanced closing — don't strip
		}
	}

	// Strip opening fence line and closing ```
	inner := s[idx+1:]
	lastFence := strings.LastIndex(inner, "\n```")
	if lastFence >= 0 {
		inner = inner[:lastFence]
	}

	return strings.TrimSpace(inner)
}
