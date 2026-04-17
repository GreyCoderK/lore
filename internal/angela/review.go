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
//
// Evidence and Confidence: the AI must back every finding with a
// verbatim quote from a specific document, and the evidence validator
// rejects findings whose quotes cannot be found in the actual corpus.
// This is the project's primary defense against AI hallucinations in
// the review output.
//
// Hash and DiffStatus: when differential review is enabled, the runner
// computes a stable hash from severity + sorted documents + normalized
// title and tracks the finding's lifecycle across runs in a JSON state
// file. DiffStatus tags each finding with its position in that
// lifecycle for the current run.
type ReviewFinding struct {
	Severity    string     `json:"severity"`    // "contradiction", "gap", "style", "obsolete"
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Documents   []string   `json:"documents"`   // filenames concerned
	Relevance   string     `json:"relevance,omitempty"` // "high", "medium", "low" — set when --for is used
	Evidence    []Evidence `json:"evidence,omitempty"`  // verifiable quotes
	Confidence  float64    `json:"confidence,omitempty"` // AI self-assessment 0.0-1.0
	Hash        string     `json:"hash,omitempty"`       // stable identity for differential tracking
	DiffStatus  string     `json:"diff_status,omitempty"` // "new" | "persisting" | "regressed" | "resolved"

	// Personas lists the persona names that flagged this finding. Populated
	// only when the review was run with persona injection.
	// When multiple personas flag the same issue, the AI emits a single
	// finding with all names here. Empty in baseline (no-persona) reviews.
	Personas []string `json:"personas,omitempty"`

	// AgreementCount is len(Personas) when personas are active. Kept as a
	// distinct field so JSON consumers (CI scripts, dashboards) can filter
	// by agreement strength without parsing the array. Zero in baseline reviews.
	AgreementCount int `json:"agreement_count,omitempty"`
}

// Evidence is a verbatim citation from a specific corpus document used to
// justify a ReviewFinding. The AI populates these; the validator checks
// that each File exists and each Quote literally appears in its File
// (after whitespace normalization) before the finding is kept.
type Evidence struct {
	File  string `json:"file"`
	Quote string `json:"quote"`
	Line  int    `json:"line,omitempty"`
}

// ReviewReport holds the complete result of a corpus review.
//
// Diff (omitempty): when differential review is enabled, the cmd layer
// attaches the per-run lifecycle classification (NEW / PERSISTING /
// REGRESSED / RESOLVED) here so the reporter can surface the delta
// without re-doing the diff.
type ReviewReport struct {
	Findings []ReviewFinding   `json:"findings"`
	DocCount int               `json:"doc_count"`
	Rejected []RejectedFinding `json:"rejected,omitempty"` // evidence-validator drops
	Diff     *ReviewDiff       `json:"diff,omitempty"`     // differential lifecycle
}

// severityOrder defines the sort priority for findings.
var severityOrder = map[string]int{
	"contradiction": 0,
	"gap":           1,
	"obsolete":      2,
	"style":         3,
}

// promptMarkerRe matches every prompt delimiter we use anywhere in the
// codebase, case-insensitively, with tolerance for whitespace inside the
// brackets. Pinned as a regex so a new marker only needs to be added in
// one place.
//
// The regex-based approach (rather than an exact-match blacklist)
// ensures all case/whitespace variants of the markers are caught,
// including section/style markers added by multi-pass polish.
var promptMarkerRe = regexp.MustCompile(
	`(?i)<<<\s*/?\s*(END_)?(CORPUS|STYLE_GUIDE|DOCUMENT|SECTION|STYLE)\s*>>>`,
)

// sanitizePromptContent provides defense-in-depth against prompt injection.
// Primary defense: structural separation of system prompt and user content.
// Secondary defense: delimiter replacement prevents content from mimicking
// prompt boundaries.
//
// Also strips ASCII control characters (except TAB/LF/CR) so adversarial
// content cannot embed ANSI escape sequences or sneak NUL bytes past a
// downstream consumer. Note: determined adversaries with document write
// access may still craft injection attempts; this is acceptable because
// document authors are trusted users of the repository.
func sanitizePromptContent(s string) string {
	s = promptMarkerRe.ReplaceAllString(s, "[marker]")
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		// Allow common whitespace; drop other C0/C1 controls.
		if r == '\t' || r == '\n' || r == '\r' {
			b.WriteRune(r)
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// sanitizeShortField is the same as sanitizePromptContent plus an
// explicit length cap. Used for audience/persona/filename strings that
// are interpolated *inline* into prompts where even a short injection
// is dangerous. 200 matches the existing caps at the CLI layer.
func sanitizeShortField(s string) string {
	s = sanitizePromptContent(s)
	// Collapse newlines to spaces so a pasted multi-line string cannot
	// visually separate itself from surrounding prompt structure.
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if len(s) > 200 {
		s = s[:200]
	}
	return strings.TrimSpace(s)
}

// BuildReviewPrompt constructs the AI prompt for corpus-wide review.
// Returns (systemPrompt, userContent) where system is stable/cacheable and user varies per call.
// signals may be nil (no pre-analysis). Corpus is serialized in TOON format.
// ReviewOpts holds optional parameters for Review.
type ReviewOpts struct {
	Audience   string      // target audience — findings will be framed for this audience
	VHSSignals *VHSSignals // VHS cross-reference signals (nil if no tape dir found)

	// Evidence validation. When Evidence.Required is true and Reader
	// is non-nil, Review() runs ValidateFindings on the parsed
	// findings before sorting and returning. Reader is needed because
	// the validator reads full document content (DocSummary only
	// carries a truncated snippet) to check that each quoted passage
	// literally exists. Leaving Reader nil while Required is true is
	// treated as "no corpus available" and every evidence file-check
	// fails, which is the correct default-deny stance.
	Evidence EvidenceValidation
	Reader   domain.CorpusReader

	// ConfigMaxTokens lets Review() honor the user's `angela.max_tokens`
	// config instead of hard-coding the default. When zero, the package
	// default is used.
	ConfigMaxTokens int

	// Personas, when non-empty, activates persona-aware review.
	// The prompt injects BuildPersonaPrompt(Personas) and instructs the AI
	// to attribute each finding to the persona(s) that flagged it. Activation
	// is strictly opt-in: the cmd layer populates this only when the user
	// explicitly opted in (--persona flag, --use-configured-personas, or
	// interactive confirmation). nil/empty = baseline review.
	Personas []PersonaProfile
}

// BuildReviewPrompt is retained for test compatibility; production uses BuildReviewPromptWithVHS.
// This wrapper always passes nil personas (baseline behavior).
func BuildReviewPrompt(docs []DocSummary, styleGuide string, signals *CorpusSignals, audience ...string) (string, string) {
	return BuildReviewPromptWithVHS(docs, styleGuide, signals, nil, nil, audience...)
}

// BuildReviewPromptWithVHS constructs the AI prompt including VHS cross-reference signals.
// When personas is non-empty, persona directives are injected into the user content and the
// AI is instructed to attribute each finding to the persona(s) that flagged it.
func BuildReviewPromptWithVHS(docs []DocSummary, styleGuide string, signals *CorpusSignals, vhs *VHSSignals, personas []PersonaProfile, audience ...string) (string, string) {
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

EVIDENCE RULE (MANDATORY):
- Every finding MUST include an "evidence" array with at least one citation.
- Each evidence entry: { "file": "exact-filename.md", "quote": "EXACT TEXT COPIED FROM THE DOCUMENT" }
- Quotes must be a verbatim substring of the document. Do NOT paraphrase, summarise, or reword.
- Copy the shortest distinctive passage that proves your point (one sentence is usually enough).
- Add a "confidence" field (float 0.0 - 1.0) self-assessing how sure you are about the finding.
- Do NOT emit findings whose confidence is below 0.4.
- Findings without verifiable quotes are rejected automatically by a post-processing validator,
  so a hallucinated or reworded quote wastes a finding slot without reaching the user.

OUTPUT FORMAT:
- Return a JSON object: {"findings": [{severity, title, description, documents, evidence, confidence}]}
- Valid severities: "contradiction", "gap", "obsolete", "style"
- documents: array of filenames involved
- evidence: array of { file, quote } objects — at least one entry per finding
- confidence: float in [0.0, 1.0]
- Example:
  {"findings": [{
    "severity": "contradiction",
    "title": "Auth strategy conflicts",
    "description": "decision-auth-2026-01-10.md picks JWT but feature-session-2026-02-05.md ships cookies.",
    "documents": ["decision-auth-2026-01-10.md", "feature-session-2026-02-05.md"],
    "evidence": [
      {"file": "decision-auth-2026-01-10.md", "quote": "we will authenticate all API calls with stateless JWT tokens"},
      {"file": "feature-session-2026-02-05.md", "quote": "the new endpoint issues an HTTP-only session cookie on login"}
    ],
    "confidence": 0.9
  }]}
- Return ONLY the JSON. No markdown, no explanation, no wrapping.
`)

	// If personas are active, add persona-aware directives to the system prompt.
	// The directives instruct the AI to attribute each finding to the persona(s)
	// that flagged it, and to aggregate findings when multiple personas concur.
	// Single-pass persona injection (parity with BuildPolishPrompt) — no
	// fan-out × N API calls.
	if len(personas) > 0 {
		sys.WriteString(`

═══════════════════════════════════════
PERSONA-AWARE REVIEW
═══════════════════════════════════════

The user activated persona lenses for this review. For each persona, surface
findings that matter TO THAT PERSONA SPECIFICALLY, based on the persona's
principles and expertise.

ADDITIONAL RULES FOR PERSONA-AWARE REVIEW:
- Attribute each finding to the persona(s) whose expertise flagged it
- Add a "personas" field (array of persona names) to every finding
- Add an "agreement_count" integer field equal to len(personas)
- When MULTIPLE personas would flag the same issue, emit ONE finding listing
  all of them in "personas" — this is a signal of cross-lens agreement
- Each persona must use only its OWN principles — do not fabricate a persona's
  concern to inflate agreement. Agreement must be earned.
- The EVIDENCE RULE above still applies fully: every persona-attributed finding
  must carry a verifiable quote. A persona's expertise does NOT excuse missing
  evidence (invariant I4 — zero hallucination).

OUTPUT FORMAT UPDATE: findings now include "personas" and "agreement_count":
  {"findings": [{severity, title, description, documents, evidence, confidence,
                 personas, agreement_count}]}
`)
	}

	// If an audience is specified, adapt findings for that audience
	if len(audience) > 0 && audience[0] != "" {
		sys.WriteString(`

═══════════════════════════════════════
AUDIENCE-ADAPTED REVIEW
═══════════════════════════════════════

You are reviewing this corpus for a SPECIFIC audience: "` + sanitizeShortField(audience[0]) + `"

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

	// Inject persona directives. Placed FIRST so the AI sees the lens before
	// it ingests the corpus, and so the personas section short-circuits any
	// attempt by corpus content to redefine the lens.
	// BuildPersonaReviewPrompt (vs BuildPersonaPrompt) uses each persona's
	// ReviewDirective, which is review-specific (corpus coherence) rather
	// than the draft/polish-oriented PromptDirective. Follow-up 2026-04-17.
	if len(personas) > 0 {
		usr.WriteString(BuildPersonaReviewPrompt(personas))
	}

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
//
// When opts[0].Evidence.Required is true, the parsed findings are
// passed through ValidateFindings BEFORE sorting. In strict mode
// failing findings are moved to ReviewReport.Rejected. In lenient mode
// they stay in Findings but also show up in Rejected with their reason,
// so the CLI can surface both. In off mode (or Required=false) the
// validator is bypassed entirely.
func Review(ctx context.Context, provider domain.AIProvider, docs []DocSummary, styleGuide string, opts ...ReviewOpts) (*ReviewReport, error) {
	if provider == nil {
		return nil, fmt.Errorf("angela: review: no AI provider configured")
	}

	var o ReviewOpts
	if len(opts) > 0 {
		o = opts[0]
	}
	signals := AnalyzeCorpusSignals(docs)
	systemPrompt, userContent := BuildReviewPromptWithVHS(docs, styleGuide, signals, o.VHSSignals, o.Personas, o.Audience)
	if len(userContent) > maxAIInputSize {
		return nil, fmt.Errorf("angela: review corpus too large for AI processing (%d bytes, max %d)", len(userContent), maxAIInputSize)
	}
	maxTokens := ResolveMaxTokens("review", 0, o.ConfigMaxTokens)
	result, err := provider.Complete(ctx, userContent, domain.WithSystem(systemPrompt), domain.WithMaxTokens(maxTokens))
	if err != nil {
		return nil, fmt.Errorf("angela: review: %w", err)
	}

	findings, err := parseReviewResponse(result)
	if err != nil {
		return nil, err
	}

	// Validate evidence BEFORE sorting so the kept set is what gets
	// ranked by severity. The returned Rejected slice is attached to
	// the report so the CLI can surface drop reasons.
	validation := ValidateFindings(findings, o.Reader, o.Evidence)
	kept := validation.Valid
	sortFindings(kept)

	return &ReviewReport{
		Findings: kept,
		DocCount: len(docs),
		Rejected: validation.Rejected,
	}, nil
}

// jsonResponseWrapper matches the expected AI response format.
type jsonResponseWrapper struct {
	Findings []ReviewFinding `json:"findings"`
}

// codeBlockRe matches ```json ... ``` blocks.
var codeBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\n?\\s*```")

// parseReviewResponse attempts to parse the AI response as JSON.
//
// AI responses are messy in practice: the model sometimes wraps the
// JSON in a ```json fence, sometimes in a bare fence, sometimes emits
// a conversational preamble ("Sure, here are the findings:"), and
// sometimes appends a trailing paragraph. The strategies below are
// tried in order, from most-strict to most-permissive, and the first
// one that yields a valid wrapper or finding list wins.
//
// Includes bare-array, object-scan, and array-scan strategies so the
// review loop does not hard-fail on a perfectly usable response just
// because the AI strayed from the exact schema.
func parseReviewResponse(response string) ([]ReviewFinding, error) {
	response = strings.TrimSpace(response)

	// Strategy 1: entire response is the wrapper object.
	if findings, ok := tryParseWrapper(response); ok {
		return findings, nil
	}
	// Strategy 2: entire response is a bare findings array. Some AI
	// responses drop the `{"findings":...}` envelope and just emit the
	// inner array; unmarshaling into a []ReviewFinding directly works.
	if findings, ok := tryParseBareArray(response); ok {
		return findings, nil
	}
	// Strategy 3: fenced code block containing either a wrapper object
	// or a bare array. Covers ```json ... ``` and bare ``` ... ```.
	if matches := codeBlockRe.FindStringSubmatch(response); len(matches) >= 2 {
		inner := strings.TrimSpace(matches[1])
		if findings, ok := tryParseWrapper(inner); ok {
			return findings, nil
		}
		if findings, ok := tryParseBareArray(inner); ok {
			return findings, nil
		}
	}
	// Strategy 4: object scan — scan for the outermost `{ ... }` block
	// anywhere in the response. Handles "Here are the findings: { ... }"
	// and "{ ... } Let me know if you'd like more detail.".
	if start, end := findOutermost(response, '{', '}'); start >= 0 {
		if findings, ok := tryParseWrapper(response[start : end+1]); ok {
			return findings, nil
		}
	}
	// Strategy 5: array scan — same idea for a bare `[ ... ]` block.
	if start, end := findOutermost(response, '[', ']'); start >= 0 {
		if findings, ok := tryParseBareArray(response[start : end+1]); ok {
			return findings, nil
		}
	}
	return nil, fmt.Errorf("angela: review: %s", i18n.T().Cmd.AngelaReviewParseErr)
}

// tryParseWrapper attempts to unmarshal s as {"findings":[...]}.
// Returns the normalized findings and true on success.
func tryParseWrapper(s string) ([]ReviewFinding, bool) {
	var wrapper jsonResponseWrapper
	if err := json.Unmarshal([]byte(s), &wrapper); err == nil && wrapper.Findings != nil {
		return normalizeFindings(wrapper.Findings), true
	}
	return nil, false
}

// tryParseBareArray attempts to unmarshal s as a bare [ReviewFinding]
// slice (no wrapping object). Returns normalized findings and true on
// success.
func tryParseBareArray(s string) ([]ReviewFinding, bool) {
	var arr []ReviewFinding
	if err := json.Unmarshal([]byte(s), &arr); err == nil {
		return normalizeFindings(arr), true
	}
	return nil, false
}

// findOutermost returns the byte offsets of the outermost balanced
// pair `openCh`/`closeCh` inside s, or (-1, -1) if no balanced pair
// exists. String literals (delimited by `"` with `\"` escaping) are
// skipped so a `{` or `[` inside a quoted value does not confuse the
// balance counter. This is not a full JSON parser — it is a cheap
// recovery heuristic used only when the simpler strategies have
// already failed.
func findOutermost(s string, openCh, closeCh byte) (int, int) {
	start := -1
	depth := 0
	inStr := false
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inStr {
			if esc {
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				continue
			}
			if c == '"' {
				inStr = false
			}
			continue
		}
		if c == '"' {
			inStr = true
			continue
		}
		switch c {
		case openCh:
			if depth == 0 {
				start = i
			}
			depth++
		case closeCh:
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 {
				return start, i
			}
		}
	}
	return -1, -1
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
//
// Also strips ANSI escape sequences and C0/C1 control chars from
// Title and Description before storing them. The AI response goes
// straight to a terminal in the human reporter; without sanitization
// an adversarial AI (or a prompt-injected corpus) could embed
// cursor-move, erase-line, or clickable-link-hijack escapes in a
// finding title. Quote sanitization is intentionally skipped — the
// evidence validator needs verbatim match against the source content,
// so Quote must stay raw.
func normalizeFindings(f []ReviewFinding) []ReviewFinding {
	if f == nil {
		return []ReviewFinding{}
	}
	for i := range f {
		f[i].Severity = strings.ToLower(strings.TrimSpace(f[i].Severity))
		if !validSeverities[f[i].Severity] {
			f[i].Severity = "style"
		}
		// Title: short field — sanitize + length-cap.
		f[i].Title = sanitizeShortField(f[i].Title)
		// Description: longer free-form text — sanitize controls but
		// keep newlines/tabs so multi-line descriptions still render.
		f[i].Description = sanitizePromptContent(f[i].Description)
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
// Returns error if fewer than 5 documents exist.
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
