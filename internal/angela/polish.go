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
// Returns (systemPrompt, userContent) where system is stable/cacheable and user varies per call.
// When personas is non-nil, persona directives are injected into the user content (AC-4).
func BuildPolishPrompt(doc string, meta domain.DocMeta, styleGuide string, corpusSummary string, personas []PersonaProfile) (string, string) {
	// System prompt: stable across calls (cacheable)
	var sys strings.Builder
	sys.WriteString("You are Angela, an expert documentation reviewer for the Lore project.\n")
	sys.WriteString("Your task: improve the clarity, structure, and completeness of the following document.\n\n")
	sys.WriteString("RULES:\n")
	sys.WriteString("- Return the COMPLETE modified document (front matter + body)\n")
	sys.WriteString("- Do NOT modify front matter fields (type, date, commit, status, tags, related, generated_by)\n")
	sys.WriteString("- Improve clarity, structure, grammar, and completeness of the body\n")
	sys.WriteString("- Do NOT invent new content — only reformulate and restructure what exists\n")
	sys.WriteString("- Preserve all Markdown formatting (headers, lists, code blocks)\n")
	sys.WriteString("- If the document is already good, return it unchanged\n")

	// User content: varies per call
	var usr strings.Builder

	// Inject persona directives (AC-4)
	if len(personas) > 0 {
		usr.WriteString(BuildPersonaPrompt(personas))
	}

	if styleGuide != "" {
		usr.WriteString("PROJECT STYLE GUIDE (between markers):\n")
		usr.WriteString("<<<STYLE_GUIDE>>>\n")
		usr.WriteString(sanitizePromptContent(styleGuide))
		usr.WriteString("\n<<<END_STYLE_GUIDE>>>\n\n")
	}

	if corpusSummary != "" {
		usr.WriteString("EXISTING CORPUS (for context only, between markers):\n")
		usr.WriteString("<<<CORPUS>>>\n")
		usr.WriteString(sanitizePromptContent(corpusSummary))
		usr.WriteString("\n<<<END_CORPUS>>>\n\n")
	}

	// Use unique delimiters to prevent prompt injection via triple backticks in document content
	usr.WriteString("DOCUMENT TO IMPROVE (between <<<DOCUMENT>>> and <<<END_DOCUMENT>>> markers):\n")
	usr.WriteString("<<<DOCUMENT>>>\n")
	usr.WriteString(sanitizePromptContent(doc))
	usr.WriteString("\n<<<END_DOCUMENT>>>\n\n")
	usr.WriteString("Return ONLY the improved document content. No explanations, no wrapping, no <<<DOCUMENT>>> markers.")

	return sys.String(), usr.String()
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
		fmt.Fprintf(&sb, "- [%s] %s", meta.Type, meta.Filename)
		var parts []string
		if meta.Scope != "" {
			parts = append(parts, "scope: "+meta.Scope)
		}
		if meta.Branch != "" && meta.Branch != "main" {
			parts = append(parts, "branch: "+meta.Branch)
		}
		if len(meta.Tags) > 0 {
			parts = append(parts, "tags: "+strings.Join(meta.Tags, ", "))
		}
		if len(parts) > 0 {
			fmt.Fprintf(&sb, " (%s)", strings.Join(parts, ", "))
		}
		sb.WriteString("\n")
	}
	if len(corpus) > 20 {
		fmt.Fprintf(&sb, "... and %d more documents\n", len(corpus)-20)
	}
	return sb.String()
}

// maxAIInputSize is the maximum document size accepted for AI processing (~100KB, roughly 25K tokens).
const maxAIInputSize = 100_000

// Polish sends a document to the AI provider for enhancement.
// Returns the improved document content. Exactly 1 API call.
func Polish(ctx context.Context, provider domain.AIProvider, doc string, meta domain.DocMeta, styleGuide string, corpusSummary string, personas []PersonaProfile) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("angela: polish: no AI provider configured")
	}
	if len(doc) > maxAIInputSize {
		return "", fmt.Errorf("angela: document too large for AI processing (%d bytes, max %d)", len(doc), maxAIInputSize)
	}

	systemPrompt, userContent := BuildPolishPrompt(doc, meta, styleGuide, corpusSummary, personas)
	docWordCount := len(strings.Fields(doc))
	maxTokens := ResolveMaxTokens("polish", docWordCount)
	result, err := provider.Complete(ctx, userContent, domain.WithSystem(systemPrompt), domain.WithMaxTokens(maxTokens))
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
