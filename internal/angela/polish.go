// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// bodyForPrompt returns the body portion of a document for inclusion
// in an AI prompt, stripping any leading frontmatter block. The AI
// never sees frontmatter (invariant I25): the caller holds fm_bytes
// verbatim and re-attaches them on write.
//
// If the input has no frontmatter (no `---\n` prefix) or the
// frontmatter is malformed, the input is returned unchanged — we
// refuse to block prompt construction on a YAML parse error. The
// caller is expected to have already validated the source via
// storage.ExtractFrontmatter during the pipeline's pre-AI guard
// (invariant I28).
func bodyForPrompt(doc string) string {
	if doc == "" {
		return ""
	}
	_, body, err := storage.ExtractFrontmatter([]byte(doc))
	if err != nil {
		return doc
	}
	return string(body)
}

// BuildPolishPrompt constructs the AI prompt for polishing a document.
// Returns (systemPrompt, userContent) where system is stable/cacheable and user varies per call.
// When personas is non-nil, persona directives are injected into the user content.
func BuildPolishPrompt(doc string, meta domain.DocMeta, styleGuide string, corpusSummary string, personas []PersonaProfile, audience ...string) (string, string) {
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

- Return ONLY the improved BODY content. Never emit ` + "`---`" + ` delimiters and never emit YAML frontmatter
  fields (type, date, commit, status, tags, related, generated_by). The frontmatter is managed
  by Lore and is NOT part of your input — you see only the body.
- If the body appears to start with a ` + "`---`" + ` block, that is body content (a Markdown horizontal
  rule inside the body), not a delimiter — leave it as body content
- NEVER add generic filler. Banned phrases: "Moreover", "It is worth noting", "Furthermore",
  "This approach", "In conclusion", "It should be noted", "As mentioned above",
  "This is important because", "improves the overall workflow", "ensures reliability"
- NEVER paraphrase the same idea twice. Say it once, precisely
- Stay grounded in the original topic. Expand with specifics from context, but do NOT invent
  unrelated content or hallucinate technical details not present in the original
- If a sentence is vague and you cannot make it concrete from context, DELETE it
- Every section must earn its place — empty or repetitive sections should be removed
- The ## Why section is sacred: it must answer "why this choice, not another?" with specifics

LANGUAGE RULE (CRITICAL):
- DETECT the document's language from its content (headings, body text)
- ALL new sections, headings, and text you add MUST be in the SAME language as the document
- If the document is in French: ## Why → ## Pourquoi, ## Alternatives Considered → ## Alternatives envisagées,
  ## How It Works → ## Fonctionnement, ## Impact → ## Impact, ## Fix → ## Correction,
  ## What Changed → ## Ce qui a changé, ## Context → ## Contexte, ## Changes → ## Changements
- If the document is in English: use English headings as specified in the structure rules above
- NEVER mix languages. A French document gets French headings and French prose. No exceptions.

PRESERVE ORIGINAL CONTENT:
- NEVER delete existing sections. You may restructure or enrich them, but all information must remain
- NEVER remove code blocks, SQL queries, configuration examples, or technical details from the original
- NEVER summarize or truncate detailed content — if the original has 30 lines of JPQL, keep them
- KEEP original section headings. Do NOT rename ## headings unless they are grammatically incorrect
  in the document's language. "Logique Métier" stays "Logique Métier", not "How It Works"
- You may ADD new sections but not at the cost of removing existing ones.
  New sections must be in the document's language (e.g., ## Pourquoi, not ## Why, for a French doc)
- Your job is to ENRICH: add diagrams, tables, and context alongside existing content — never replace it

WRITING STYLE:
- Direct, technical, opinionated. Write for senior developers
- Short paragraphs (2-4 lines). No wall of text
- Active voice. "Git closes stdin" not "stdin is closed by Git"
- Specific > generic. "Reduced latency from 200ms to 50ms" not "improved performance"
- If the document is already excellent, return it unchanged

═══════════════════════════════════════
QUALITY REFERENCE — What a polished document looks like
═══════════════════════════════════════

A well-polished document has ALL of these:
1. A ## Pourquoi / ## Why section that tells a STORY: the situation before, the pain point, and what this solves
2. At least ONE mermaid diagram (architecture, sequence, or flowchart) — placed near the relevant section
3. Tables for any structured data (endpoints, fields, comparison)
4. Code blocks with language tags for ALL code/config/SQL examples
5. Existing content PRESERVED and ENRICHED, not replaced or summarized
6. Section headings that match the document's ORIGINAL language
7. Specific numbers, names, and technical details — never vague
8. Short paragraphs — if a paragraph exceeds 4 lines, split it
`)


	// If an audience is specified, override PRESERVE rules with rewrite rules.
	// The document is being adapted for a different audience, not just enriched.
	//
	// The raw audience string is cached in `targetAudience` and sanitized at
	// every interpolation point — including the user-content block below,
	// which previously wrote the raw string and could escape the structural
	// separation.
	targetAudience := ""
	if len(audience) > 0 && audience[0] != "" {
		targetAudience = sanitizeShortField(audience[0])
	}
	if targetAudience != "" {
		sys.WriteString(`

═══════════════════════════════════════
AUDIENCE REWRITE MODE (overrides PRESERVE rules)
═══════════════════════════════════════

You are rewriting this document for a SPECIFIC audience: "` + targetAudience + `"

REWRITE RULES:
- ADAPT the content, structure, vocabulary, and level of detail for this audience
- You MAY remove sections that are irrelevant to this audience (e.g., remove JPQL queries for a commercial team)
- You MAY simplify technical jargon, add business context, or restructure entirely
- You MAY change section headings to match what this audience expects
- KEEP the core message and factual accuracy — never invent facts
- KEEP diagrams but simplify them if needed (fewer nodes, business-level labels)
- KEEP tables but adapt columns to what this audience cares about
- The output language must match the original document's language
- Add a note at the top: "> Adapted for: ` + targetAudience + `"
`)
	}

	// User content: varies per call
	var usr strings.Builder

	// Inject audience directive prominently
	if targetAudience != "" {
		usr.WriteString("TARGET AUDIENCE: " + targetAudience + "\n")
		usr.WriteString("Rewrite the document below for this audience. Adapt tone, depth, and structure.\n\n")
	}

	// Inject persona directives
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

	// Use unique delimiters to prevent prompt injection via triple backticks in document content.
	// The body is stripped of frontmatter before interpolation (invariant I25) — the AI never sees
	// the `---` YAML block. The caller re-attaches frontmatter bytes verbatim on write (invariant I24).
	usr.WriteString("DOCUMENT BODY TO IMPROVE (between <<<BODY>>> and <<<END_BODY>>> markers):\n")
	usr.WriteString("<<<BODY>>>\n")
	usr.WriteString(sanitizePromptContent(bodyForPrompt(doc)))
	usr.WriteString("\n<<<END_BODY>>>\n\n")
	usr.WriteString("Return ONLY the improved body. No explanations, no wrapping, no `---` delimiters, no <<<BODY>>> markers.")

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

// PolishOpts holds optional parameters for Polish.
type PolishOpts struct {
	Audience      string // target audience for rewrite mode (empty = standard polish)
	ConfigMaxToks int    // angela.max_tokens from .lorerc (0 = auto-compute)
}

// Polish sends a document to the AI provider for enhancement.
// Returns the improved document content. Exactly 1 API call.
func Polish(ctx context.Context, provider domain.AIProvider, doc string, meta domain.DocMeta, styleGuide string, corpusSummary string, personas []PersonaProfile, opts ...PolishOpts) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("angela: polish: no AI provider configured")
	}
	if len(doc) > maxAIInputSize {
		return "", fmt.Errorf("angela: document too large for AI processing (%d bytes, max %d)", len(doc), maxAIInputSize)
	}

	var audience string
	var cfgMax int
	if len(opts) > 0 {
		audience = opts[0].Audience
		cfgMax = opts[0].ConfigMaxToks
	}
	systemPrompt, userContent := BuildPolishPrompt(doc, meta, styleGuide, corpusSummary, personas, audience)
	docWordCount := len(strings.Fields(doc))
	maxTokens := ResolveMaxTokens("polish", docWordCount, cfgMax)
	result, err := provider.Complete(ctx, userContent, domain.WithSystem(systemPrompt), domain.WithMaxTokens(maxTokens))
	if err != nil {
		return "", fmt.Errorf("angela: polish: %w", err)
	}

	// Strip markdown fencing if AI wraps the response
	result = stripCodeFence(result)

	// Post-process: local quality improvements (no API)
	result = PostProcess(doc, result)

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
