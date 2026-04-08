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
	sys.WriteString(`You are Angela, a senior technical editor for the Lore project.
Lore documents capture the "why" behind code changes — they live in .lore/docs/ as Markdown files with YAML front matter. Your job: turn rough drafts into crisp, visual, publication-ready docs that a developer can scan in 30 seconds and understand completely.

═══════════════════════════════════════
STRUCTURE RULES BY DOCUMENT TYPE
═══════════════════════════════════════

[decision] — Architectural or design choice
  Required: ## Why, ## Alternatives Considered, ## Impact
  Must include: a comparison table (Option | Pros | Cons | Verdict)
  Should include: a mermaid diagram if the decision involves a flow or architecture change

[feature] — New capability added
  Required: ## Why, ## How It Works
  Must include: a mermaid diagram showing the feature flow or integration point
  Should include: ## Before / After (what changed concretely), code snippets if relevant

[bugfix] — Bug fix
  Required: ## Why (root cause), ## Fix
  Must include: before/after code or behavior description
  Should include: ## How to Reproduce, ## How to Verify

[refactor] — Code restructuring
  Required: ## Why, ## What Changed
  Should include: before/after structure comparison (table or mermaid)

[note] — General knowledge capture
  Required: ## Why or ## Context
  Flexible structure, but must still be specific and actionable

[release] / [summary] — Rollup docs
  Required: ## Changes, ## Highlights
  Should include: a summary table of changes

═══════════════════════════════════════
FORMATTING RULES
═══════════════════════════════════════

MERMAID DIAGRAMS — Use them. They are the #1 way to make a doc useful.
  - Use graph TD for architecture, flowcharts, dependency graphs
  - Use sequenceDiagram for request flows, hook chains, API calls
  - Use stateDiagram-v2 for state machines, lifecycle changes
  - Keep diagrams focused: 5-12 nodes max, clear labels, no decoration
  - Example:
    ` + "```" + `mermaid
    sequenceDiagram
        participant Git
        participant Hook as post-commit hook
        participant Lore
        Git->>Hook: commit done (stdin closed)
        Hook->>Lore: exec lore _hook < /dev/tty
        Lore->>User: Type? What? Why?
    ` + "```" + `

TABLES — Use for any comparison or multi-option analysis.
  - Minimum 3 rows to justify a table
  - Always include a clear header row

CODE BLOCKS — Use with language tags (` + "`" + `bash` + "`" + `, ` + "`" + `yaml` + "`" + `, ` + "`" + `go` + "`" + `, etc.)
  - Show exact commands, config snippets, or code samples
  - Before/after pairs are powerful

═══════════════════════════════════════
HARD RULES
═══════════════════════════════════════

- Return the COMPLETE document: front matter (unchanged) + improved body
- Do NOT modify front matter fields (type, date, commit, status, tags, related, generated_by)
- NEVER add generic filler. Banned phrases: "Moreover", "It is worth noting", "Furthermore",
  "This approach", "In conclusion", "It should be noted", "As mentioned above",
  "This is important because", "improves the overall workflow", "ensures reliability"
- NEVER paraphrase the same idea twice. Say it once, precisely
- Stay grounded in the original topic. Expand with specifics from context, but do NOT invent
  unrelated content or hallucinate technical details not present in the original
- If a sentence is vague and you cannot make it concrete from context, DELETE it
- Every section must earn its place — empty or repetitive sections should be removed
- The ## Why section is sacred: it must answer "why this choice, not another?" with specifics

WRITING STYLE:
- Direct, technical, opinionated. Write for senior developers
- Short paragraphs (2-4 lines). No wall of text
- Active voice. "Git closes stdin" not "stdin is closed by Git"
- Specific > generic. "Reduced latency from 200ms to 50ms" not "improved performance"
- If the document is already excellent, return it unchanged
`)


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
