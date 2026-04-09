// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// DocSummary holds extracted metadata and content snippets for a single document.
type DocSummary struct {
	Filename string
	Type     string
	Date     string
	Tags     []string
	Branch   string // branch at capture time; "" for legacy docs
	Scope    string // scope from conventional commit; "" if none
	Summary  string // adaptive: top sections by content length (max 450 runes total)
}

// ReviewFinding represents a single issue found during corpus review.
type ReviewFinding struct {
	Severity    string   `json:"severity"`    // "contradiction", "gap", "style", "obsolete"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Documents   []string `json:"documents"`   // filenames concerned
	Relevance   string   `json:"relevance,omitempty"` // "high", "medium", "low" — set when --for is used
}

// ReviewReport holds the complete result of a corpus review.
type ReviewReport struct {
	Findings []ReviewFinding
	DocCount int
}

// severityOrder defines the sort priority for findings.
var severityOrder = map[string]int{
	"contradiction": 0,
	"gap":           1,
	"obsolete":      2,
	"style":         3,
}

// sanitizePromptContent provides defense-in-depth against prompt injection.
// Primary defense: structural separation of system prompt and user content.
// Secondary defense: delimiter replacement prevents content from mimicking prompt boundaries.
// Note: determined adversaries with document write access may still craft injection attempts.
// This is acceptable because document authors are trusted users of the repository.
func sanitizePromptContent(s string) string {
	r := strings.NewReplacer(
		// Delimiter markers — prevent structural confusion
		"<<<CORPUS>>>", "[CORPUS]",
		"<<<END_CORPUS>>>", "[END_CORPUS]",
		"<<<STYLE_GUIDE>>>", "[STYLE_GUIDE]",
		"<<<END_STYLE_GUIDE>>>", "[END_STYLE_GUIDE]",
		"<<<DOCUMENT>>>", "[DOCUMENT]",
		"<<<END_DOCUMENT>>>", "[END_DOCUMENT]",
	)
	return r.Replace(s)
}

// BuildReviewPrompt constructs the AI prompt for corpus-wide review.
// Returns (systemPrompt, userContent) where system is stable/cacheable and user varies per call.
// signals may be nil (no pre-analysis). Corpus is serialized in TOON format.
// ReviewOpts holds optional parameters for Review.
type ReviewOpts struct {
	Audience  string      // target audience — findings will be framed for this audience
	VHSSignals *VHSSignals // VHS cross-reference signals (nil if no tape dir found)
}

func BuildReviewPrompt(docs []DocSummary, styleGuide string, signals *CorpusSignals, audience ...string) (string, string) {
	return BuildReviewPromptWithVHS(docs, styleGuide, signals, nil, audience...)
}

// BuildReviewPromptWithVHS constructs the AI prompt including VHS cross-reference signals.
func BuildReviewPromptWithVHS(docs []DocSummary, styleGuide string, signals *CorpusSignals, vhs *VHSSignals, audience ...string) (string, string) {
	// System prompt: stable across calls (cacheable)
	var sys strings.Builder
	sys.WriteString(`You are Angela, a senior technical editor reviewing a Lore documentation corpus.
Lore captures the "why" behind code changes. Your task: find coherence issues across the corpus.

ANALYSIS STRATEGY:
1. Group documents by Type (decision, feature, bugfix, etc.)
2. Compare WITHIN each group first — same-type contradictions are critical
3. Cross-reference by scope and branch — same scope = same effort
4. Check temporal coherence — newer docs may supersede older ones

WHAT TO FIND:

  "contradiction" (CRITICAL):
  - Two decision docs that reach opposite conclusions on the same topic
  - A feature doc that contradicts a decision doc (e.g., decided JWT but built sessions)
  - Same-scope documents with conflicting technical details
  - ONLY flag contradictions when the conflict is concrete and specific

  "gap" (IMPORTANT):
  - A technology or component mentioned in 2+ docs but never documented itself
  - A scope with 2+ docs but no summary document
  - A decision referenced by feature docs but the decision doc is missing

  "obsolete" (MODERATE):
  - Decision docs that a later decision explicitly supersedes
  - Docs referencing technologies/patterns the corpus shows were replaced
  - ONLY flag as obsolete when there is concrete evidence of replacement

  "style" (LOW):
  - Inconsistent terminology for the same concept across docs
  - Naming inconsistencies (e.g., "rate limiter" vs "throttler" for the same thing)

  VHS DEMO COHERENCE (if vhs_signals section present):
  - Flag orphan tapes (demo GIFs not referenced in any doc) as "gap"
  - Flag orphan GIF references (docs referencing GIFs with no tape source) as "gap"
  - Flag command mismatches (tape commands that don't match known CLI commands) as "obsolete"

QUALITY RULES:
- Be specific: name the exact documents and the exact conflict/gap
- Do NOT flag vague or speculative issues — every finding must be backed by evidence from the corpus
- Do NOT flag documents for being short or lacking sections — that is polish's job, not review's
- Do NOT suggest removing, renaming, or restructuring existing sections or headings
- Do NOT flag detailed or lengthy documents as a problem — detail is a feature, not a bug
- RESPECT each document's language. If a doc is in French, findings about it must be written in French.
  Do NOT flag French headings or terminology as style issues. Mixed-language corpora are normal
- Aim for 3-8 high-quality findings, not 20 weak ones
- If no real issues found, return: {"findings": []}

OUTPUT FORMAT:
- Return a JSON object: {"findings": [{severity, title, description, documents}]}
- Valid severities: "contradiction", "gap", "obsolete", "style"
- documents: array of filenames involved
- Return ONLY the JSON. No markdown, no explanation, no wrapping.
`)

	// If an audience is specified, adapt findings for that audience
	if len(audience) > 0 && audience[0] != "" {
		sys.WriteString(`

═══════════════════════════════════════
AUDIENCE-ADAPTED REVIEW
═══════════════════════════════════════

You are reviewing this corpus for a SPECIFIC audience: "` + sanitizePromptContent(audience[0]) + `"

ADDITIONAL RULES FOR AUDIENCE REVIEW:
- Frame findings in terms that matter to this audience
- For a commercial team: focus on gaps in business value documentation, missing client-facing explanations
- For a CTO: focus on architectural contradictions, missing risk assessments, technical debt signals
- For new developers: focus on missing onboarding context, unexplained jargon, missing "getting started" docs
- For audit/compliance: focus on missing traceability, undocumented security decisions, missing approval records
- Add a "relevance" field (high/medium/low) to each finding indicating how much it matters to this audience
- The output format becomes: {"findings": [{severity, title, description, documents, relevance}]}
`)
	}

	// User content: varies per call
	var usr strings.Builder

	if styleGuide != "" {
		usr.WriteString("PROJECT STYLE GUIDE (between markers):\n")
		usr.WriteString("<<<STYLE_GUIDE>>>\n")
		usr.WriteString(sanitizePromptContent(styleGuide))
		usr.WriteString("\n<<<END_STYLE_GUIDE>>>\n\n")
	}

	// TOON format preamble
	usr.WriteString("The corpus below uses TOON (pipe-separated) format. Each section starts with a header row defining field names. Data rows follow with values separated by |. Pipes in values are escaped as \\|. Backslashes are escaped as \\\\.\n\n")

	// Serialize corpus + signals in TOON format (with VHS signals if available)
	if vhs != nil {
		usr.WriteString(SerializeTOONWithVHS(docs, signals, vhs))
	} else {
		usr.WriteString(SerializeTOON(docs, signals))
	}
	usr.WriteString("\n")

	usr.WriteString("Return ONLY a JSON object with a \"findings\" array. No markdown, no explanation.")

	return sys.String(), usr.String()
}

// Review performs a corpus-wide analysis using the AI provider.
// Exactly 1 API call. Returns the review report sorted by severity.
func Review(ctx context.Context, provider domain.AIProvider, docs []DocSummary, styleGuide string, opts ...ReviewOpts) (*ReviewReport, error) {
	if provider == nil {
		return nil, fmt.Errorf("angela: review: no AI provider configured")
	}

	var aud string
	var vhs *VHSSignals
	if len(opts) > 0 {
		aud = opts[0].Audience
		vhs = opts[0].VHSSignals
	}
	signals := AnalyzeCorpusSignals(docs)
	systemPrompt, userContent := BuildReviewPromptWithVHS(docs, styleGuide, signals, vhs, aud)
	if len(userContent) > maxAIInputSize {
		return nil, fmt.Errorf("angela: review corpus too large for AI processing (%d bytes, max %d)", len(userContent), maxAIInputSize)
	}
	maxTokens := ResolveMaxTokens("review", 0)
	result, err := provider.Complete(ctx, userContent, domain.WithSystem(systemPrompt), domain.WithMaxTokens(maxTokens))
	if err != nil {
		return nil, fmt.Errorf("angela: review: %w", err)
	}

	findings, err := parseReviewResponse(result)
	if err != nil {
		return nil, err
	}

	sortFindings(findings)

	return &ReviewReport{
		Findings: findings,
		DocCount: len(docs),
	}, nil
}

// jsonResponseWrapper matches the expected AI response format.
type jsonResponseWrapper struct {
	Findings []ReviewFinding `json:"findings"`
}

// codeBlockRe matches ```json ... ``` blocks.
var codeBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\n?\\s*```")

// parseReviewResponse attempts to parse the AI response as JSON.
// Strategy: 1) try full response, 2) try ```json``` block, 3) error.
func parseReviewResponse(response string) ([]ReviewFinding, error) {
	response = strings.TrimSpace(response)

	// Strategy 1: try full response
	var wrapper jsonResponseWrapper
	if err := json.Unmarshal([]byte(response), &wrapper); err == nil {
		return normalizeFindings(wrapper.Findings), nil
	}

	// Strategy 2: try ```json ... ``` block
	matches := codeBlockRe.FindStringSubmatch(response)
	if len(matches) >= 2 {
		if err := json.Unmarshal([]byte(strings.TrimSpace(matches[1])), &wrapper); err == nil {
			return normalizeFindings(wrapper.Findings), nil
		}
	}

	// Strategy 3: error
	return nil, fmt.Errorf("angela: review: %s", i18n.T().Cmd.AngelaReviewParseErr)
}

// normalizeFindings ensures a nil slice becomes an empty slice (JSON [] instead of null).
// validSeverities is the set of recognized finding severities.
var validSeverities = map[string]bool{
	"contradiction": true,
	"gap":           true,
	"obsolete":      true,
	"style":         true,
}

// normalizeFindings ensures a nil slice becomes an empty slice and
// normalizes unknown severities to "style" (lowest priority).
func normalizeFindings(f []ReviewFinding) []ReviewFinding {
	if f == nil {
		return []ReviewFinding{}
	}
	for i := range f {
		f[i].Severity = strings.ToLower(strings.TrimSpace(f[i].Severity))
		if !validSeverities[f[i].Severity] {
			f[i].Severity = "style"
		}
	}
	return f
}

// sortFindings sorts findings by severity priority: contradiction > gap > obsolete > style.
// Unknown severities sort last (after style).
func sortFindings(findings []ReviewFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		oi := severityRank(findings[i].Severity)
		oj := severityRank(findings[j].Severity)
		return oi < oj
	})
}

// severityRank returns the sort priority for a severity string.
// Unknown severities return a high value to sort last.
func severityRank(sev string) int {
	if rank, ok := severityOrder[sev]; ok {
		return rank
	}
	return len(severityOrder) // sort after all known severities
}

// PrepareDocSummaries reads documents from the corpus and builds summaries.
// Returns error if fewer than 5 documents exist (AC-2).
// Limits to 50 docs: 25 most recent + 25 oldest when corpus exceeds 50.
// ReviewFilter controls which documents are included in a review.
type ReviewFilter struct {
	Pattern *regexp.Regexp // if non-nil, only include files matching this pattern
	All     bool           // if true, include all docs (no 25+25 sampling)
}

func PrepareDocSummaries(reader domain.CorpusReader, filters ...ReviewFilter) ([]DocSummary, int, error) {
	allDocs, err := reader.ListDocs(domain.DocFilter{})
	if err != nil {
		return nil, 0, fmt.Errorf("angela: review: list docs: %w", err)
	}

	// Apply regex filter if provided
	var filter ReviewFilter
	if len(filters) > 0 {
		filter = filters[0]
	}
	if filter.Pattern != nil {
		filtered := allDocs[:0]
		for _, d := range allDocs {
			if filter.Pattern.MatchString(d.Filename) {
				filtered = append(filtered, d)
			}
		}
		allDocs = filtered
	}

	totalCount := len(allDocs)
	minRequired := 5
	if filter.Pattern != nil {
		minRequired = 2 // lower threshold when filtering
	}
	if totalCount < minRequired {
		return nil, totalCount, fmt.Errorf(i18n.T().Cmd.AngelaReviewMinDocs, minRequired, totalCount)
	}

	// Sort by date descending (most recent first)
	sort.Slice(allDocs, func(i, j int) bool {
		return allDocs[i].Date > allDocs[j].Date
	})

	// Select docs: all if --all or <= 50, else 25 newest + 25 oldest
	var selected []domain.DocMeta
	if filter.All || totalCount <= 50 {
		selected = allDocs
	} else {
		selected = append(selected, allDocs[:25]...)
		selected = append(selected, allDocs[totalCount-25:]...)
	}

	// Build summaries
	summaries := make([]DocSummary, 0, len(selected))
	for _, meta := range selected {
		content, readErr := reader.ReadDoc(meta.Filename)
		if readErr != nil {
			continue // unreadable docs silently skipped — acceptable for MVP
		}

		summary := ExtractAdaptiveSummary(content, 450)
		summaries = append(summaries, DocSummary{
			Filename: meta.Filename,
			Type:     meta.Type,
			Date:     meta.Date,
			Tags:     meta.Tags,
			Branch:   meta.Branch,
			Scope:    meta.Scope,
			Summary:  summary,
		})
	}

	return summaries, totalCount, nil
}

// sectionEntry holds a parsed ## heading and its body content.
type sectionEntry struct {
	heading string
	body    string
}

// parseAllSections extracts all ## level headings and their body content from markdown.
// Skips front matter (--- delimited).
func parseAllSections(content string) []sectionEntry {
	lines := strings.Split(content, "\n")
	var sections []sectionEntry
	var currentHeading string
	var currentBody strings.Builder
	inFrontMatter := false
	frontMatterSeen := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip YAML front matter
		if trimmed == "---" {
			if !frontMatterSeen {
				inFrontMatter = true
				frontMatterSeen = true
				continue
			}
			if inFrontMatter {
				inFrontMatter = false
				continue
			}
		}
		if inFrontMatter {
			continue
		}

		// New ## heading
		if strings.HasPrefix(trimmed, "## ") {
			// Save previous section if any
			if currentHeading != "" {
				body := strings.TrimSpace(currentBody.String())
				if body != "" {
					sections = append(sections, sectionEntry{heading: currentHeading, body: body})
				}
			}
			currentHeading = strings.TrimSpace(trimmed[3:])
			currentBody.Reset()
			continue
		}

		// # heading (h1) ends current section but doesn't start one we track
		if strings.HasPrefix(trimmed, "# ") {
			if currentHeading != "" {
				body := strings.TrimSpace(currentBody.String())
				if body != "" {
					sections = append(sections, sectionEntry{heading: currentHeading, body: body})
				}
				currentHeading = ""
				currentBody.Reset()
			}
			continue
		}

		// Accumulate body
		if currentHeading != "" {
			if currentBody.Len() > 0 || trimmed != "" {
				if currentBody.Len() > 0 {
					currentBody.WriteString(" ")
				}
				currentBody.WriteString(trimmed)
			}
		}
	}

	// Don't forget last section
	if currentHeading != "" {
		body := strings.TrimSpace(currentBody.String())
		if body != "" {
			sections = append(sections, sectionEntry{heading: currentHeading, body: body})
		}
	}

	return sections
}

// truncateRunes truncates s to maxRunes runes.
func truncateRunes(s string, maxRunes int) string {
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes])
}
