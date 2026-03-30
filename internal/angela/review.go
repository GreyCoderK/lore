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
	Summary  string // adaptive: top sections by content length (max 450 runes total)
}

// ReviewFinding represents a single issue found during corpus review.
type ReviewFinding struct {
	Severity    string   `json:"severity"`    // "contradiction", "gap", "style", "obsolete"
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Documents   []string `json:"documents"`   // filenames concerned
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
func BuildReviewPrompt(docs []DocSummary, styleGuide string, signals *CorpusSignals) (string, string) {
	// System prompt: stable across calls (cacheable)
	var sys strings.Builder
	sys.WriteString("You are Angela, an expert documentation reviewer for the Lore project.\n")
	sys.WriteString("Your task: analyze the coherence of the following documentation corpus.\n\n")
	sys.WriteString("RULES:\n")
	sys.WriteString("- Group documents by Type (decision, feature, bugfix, etc.) and compare WITHIN each group first\n")
	sys.WriteString("- Contradictions between documents of the SAME type are higher severity than cross-type\n")
	sys.WriteString("- Identify contradictions between documents (especially same-type documents with conflicting conclusions)\n")
	sys.WriteString("- Identify gaps (topics referenced but not documented)\n")
	sys.WriteString("- Identify obsolete decisions that may need updating (compare dates — older docs may be superseded)\n")
	sys.WriteString("- Identify inconsistent terminology or style across the corpus\n")
	sys.WriteString("- Return your analysis as a JSON object with a \"findings\" array\n")
	sys.WriteString("- Each finding must have: severity, title, description, documents (array of filenames)\n")
	sys.WriteString("- Valid severities: \"contradiction\", \"gap\", \"obsolete\", \"style\"\n")
	sys.WriteString("- If no issues found, return: {\"findings\": []}\n")
	sys.WriteString("- Return ONLY the JSON object. No explanations, no wrapping.\n")

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

	// Serialize corpus + signals in TOON format
	usr.WriteString(SerializeTOON(docs, signals))
	usr.WriteString("\n")

	usr.WriteString("Return ONLY a JSON object with a \"findings\" array. No markdown, no explanation.")

	return sys.String(), usr.String()
}

// Review performs a corpus-wide analysis using the AI provider.
// Exactly 1 API call. Returns the review report sorted by severity.
func Review(ctx context.Context, provider domain.AIProvider, docs []DocSummary, styleGuide string) (*ReviewReport, error) {
	if provider == nil {
		return nil, fmt.Errorf("angela: review: no AI provider configured")
	}

	signals := AnalyzeCorpusSignals(docs)
	systemPrompt, userContent := BuildReviewPrompt(docs, styleGuide, signals)
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
func PrepareDocSummaries(reader domain.CorpusReader) ([]DocSummary, int, error) {
	allDocs, err := reader.ListDocs(domain.DocFilter{})
	if err != nil {
		return nil, 0, fmt.Errorf("angela: review: list docs: %w", err)
	}

	totalCount := len(allDocs)
	if totalCount < 5 {
		return nil, totalCount, fmt.Errorf(i18n.T().Cmd.AngelaReviewMinDocs, 5, totalCount)
	}

	// Sort by date descending (most recent first)
	sort.Slice(allDocs, func(i, j int) bool {
		return allDocs[i].Date > allDocs[j].Date
	})

	// Select docs: all if <= 50, else 25 newest + 25 oldest
	var selected []domain.DocMeta
	if totalCount <= 50 {
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
