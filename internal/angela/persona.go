// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/i18n"
)

// PersonaProfile represents an expert lens that Angela can activate
// for document review. Personas are Go values — no external files.
type PersonaProfile struct {
	Name        string
	DisplayName string
	Icon        string
	Expertise   string
	Principles  []string
	DraftChecks []DraftCheck
	// PromptDirective is the AI instruction block used for DRAFT/POLISH.
	// It is written for the "one document, make it better" task.
	PromptDirective string
	// ReviewDirective is the AI instruction block used for REVIEW.
	// Review is corpus-wide coherence analysis — each persona must tell the
	// AI what THEIR lens looks for across the set of docs (contradictions,
	// gaps, stylistic drift, etc.), NOT how to fix a single doc. When empty,
	// BuildPersonaReviewPrompt falls back to PromptDirective with a warning
	// footer so the regression is surfaced in the prompt itself.
	ReviewDirective string
	DocTypes        []string // explicit type activation
	ContentSignals  []string // keyword content activation (EN + FR)
}

// DraftCheck is a persona-specific structural check run during draft analysis.
// Check returns a raw Suggestion (without persona prefix); the caller decorates.
type DraftCheck struct {
	Label string
	Check func(body string, sections map[string]string) *Suggestion
}

// ScoredPersona pairs a persona with its resolution score.
type ScoredPersona struct {
	Profile PersonaProfile
	Score   int
}

// registry is the ordered list of all native Angela personas.
// Use GetRegistry() to obtain an immutable copy.
var registry = []PersonaProfile{
	{
		Name:        "storyteller",
		DisplayName: "Affoue",
		Icon:        "📖",
		Expertise:   "Narrative clarity and authentic storytelling",
		Principles: []string{
			"Why is the protagonist of every document",
			"Move from abstract to concrete",
			"Use analogies to anchor understanding",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "Why too listy",
				Check: func(body string, sections map[string]string) *Suggestion {
					why := sections["## Why"]
					if why == "" {
						return nil
					}
					bullets := 0
					chars := 0
					for _, line := range strings.Split(why, "\n") {
						trimmed := strings.TrimSpace(line)
						if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
							bullets++
						}
						chars += utf8.RuneCountInString(trimmed)
					}
					if bullets > 3 && chars < 500 {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaWhyTooListy,
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `STORYTELLING LENS (Affoue):
- The ## Why section is the story's climax — it must answer "why THIS choice?" not just "what we did"
- Replace vague motivations with a concrete narrative: what was the pain? what broke? what did users experience?
- Convert bullet-list-only sections into 2-3 sentence narratives that flow. Lists are scaffolding, not the story
- Add a "before this change" vs "after this change" framing when relevant
- Use one concrete analogy if it genuinely clarifies (avoid forced analogies)`,
		ReviewDirective: `STORYTELLING LENS (Affoue) — REVIEW MODE:
- Flag NARRATIVE CONTRADICTIONS across documents: two decisions with incompatible "why" stories for the same scope
- Flag ORPHAN DECISIONS: feature docs whose causal story points to a decision doc that does not exist
- Flag TEMPORAL GAPS: decisions referenced by features but older than the feature implementation without a superseding record
- Do NOT flag a single doc's prose quality — that is polish's job, not review's
- Prefer one strong narrative-coherence finding over five micro-nits`,
		DocTypes:        []string{"decision", "note"},
		ContentSignals:  signalsStoryteller,
	},
	{
		Name:        "tech-writer",
		DisplayName: "Sialou",
		Icon:        "✏️",
		Expertise:   "Technical writing precision and clarity",
		Principles: []string{
			"Every word serves a purpose",
			"Bullets over paragraphs",
			"Diagrams over text when possible",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "Long paragraphs",
				Check: func(body string, sections map[string]string) *Suggestion {
					for _, paragraph := range strings.Split(body, "\n\n") {
						lines := strings.Split(strings.TrimSpace(paragraph), "\n")
						if len(lines) <= 5 {
							continue
						}
						allText := true
						for _, l := range lines {
							t := strings.TrimSpace(l)
							if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "```") {
								allText = false
								break
							}
						}
						if allText {
							return &Suggestion{
								Category: "persona",
								Severity: "info",
								Message:  i18n.T().Angela.PersonaLongParagraphs,
							}
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `TECHNICAL WRITING LENS (Sialou):
- Cut every sentence that doesn't add information. Target: 30-50% fewer words than the original
- Replace paragraphs > 4 lines with bullet lists or tables
- Add a mermaid diagram if the doc describes any process, flow, or architecture — this is mandatory, not optional
- Add code blocks with language tags for commands, config, or API examples
- Structure: scannable headers → key info in first sentence of each section → details after`,
		ReviewDirective: `TECHNICAL WRITING LENS (Sialou) — REVIEW MODE:
- Flag TERMINOLOGY DRIFT across documents: the same concept named differently (e.g. "rate limiter" in doc A, "throttler" in doc B)
- Flag STRUCTURAL INCONSISTENCY: docs of the same type with different required sections or header conventions
- Flag MISSING DIAGRAMS CROSS-CORPUS: a technology mentioned in 3+ docs but never diagrammed anywhere
- Do NOT flag a single doc's terseness — that's polish's domain
- Every finding must cite verbatim quotes from at least two docs`,
		DocTypes:        []string{"feature", "refactor", "release"},
		ContentSignals:  signalsTechWriter,
	},
	{
		Name:        "qa-reviewer",
		DisplayName: "Kouame",
		Icon:        "🔍",
		Expertise:   "Quality assurance and validation criteria",
		Principles: []string{
			"Every claim needs validation criteria",
			"Edge cases must be explicit",
			"Documentation has a shelf life",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "Missing verification criteria",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasVerification := strings.Contains(lower, "verify") ||
						strings.Contains(lower, "validate") ||
						strings.Contains(lower, "assert") ||
						strings.Contains(lower, "criteria") ||
						strings.Contains(lower, "vérifier") ||
						strings.Contains(lower, "valider") ||
						strings.Contains(lower, "critère") ||
						containsWord(lower, "test") ||
						containsWord(lower, "check")
					if !hasVerification {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaMissingVerify,
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `QA LENS (Kouame):
- Add a "## How to Verify" section with concrete steps (commands to run, expected output)
- List edge cases and failure modes explicitly — what happens when X fails? what about Y?
- If the doc claims something works, specify how to test it: exact command, expected result
- Flag any undocumented assumptions (e.g., "requires Redis" but Redis setup isn't mentioned)`,
		ReviewDirective: `QA LENS (Kouame) — REVIEW MODE:
- Flag UNVERIFIED CLAIMS across the corpus: docs that assert a behavior without any doc in the corpus describing how to verify it
- Flag CONTRADICTORY CLAIMS about the same behavior in different docs (doc A says "returns 200", doc B says "returns 204" for the same endpoint)
- Flag DOCUMENTATION/TEST DRIFT: a feature doc references a test file or command that no other doc in the corpus mentions
- Do NOT demand that every doc gain a verification section — that's polish's domain
- Evidence quotes must pair docs that genuinely contradict, not docs that merely differ in scope`,
		DocTypes:        []string{"bugfix", "feature"},
		ContentSignals:  signalsQAReviewer,
	},
	{
		Name:        "architect",
		DisplayName: "Doumbia",
		Icon:        "🏗️",
		Expertise:   "System design, trade-offs, and scalability",
		Principles: []string{
			"Trade-offs must be explicit",
			"User value over technical elegance",
			"Boring technology is a feature",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "Architecture without trade-offs",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasArchi := strings.Contains(lower, "architecture") ||
						strings.Contains(lower, "component") ||
						strings.Contains(lower, "composant") ||
						containsWord(lower, "design") ||
						containsWord(lower, "system") ||
						containsWord(lower, "système") ||
						containsWord(lower, "conception")
					hasTradeoff := strings.Contains(lower, "trade-off") ||
						strings.Contains(lower, "tradeoff") ||
						strings.Contains(lower, "compromis") ||
						strings.Contains(lower, "alternative") ||
						strings.Contains(lower, "drawback") ||
						strings.Contains(lower, "inconvénient")
					if hasArchi && !hasTradeoff {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaNoTradeoffs,
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `ARCHITECTURE LENS (Doumbia):
- Add or improve ## Alternatives Considered with a comparison table: Option | Pros | Cons | Verdict
- Make trade-offs explicit: what did we gain? what did we sacrifice? why is that acceptable?
- Add a mermaid architecture diagram showing component relationships or data flow
- Quantify impact when possible: latency, throughput, resource usage, complexity cost`,
		ReviewDirective: `ARCHITECTURE LENS (Doumbia) — REVIEW MODE:
- Flag ARCHITECTURAL CONTRADICTIONS across decisions: two decision docs that reach incompatible conclusions on the same layer or component
- Flag SCOPE CONFLICTS: feature docs that implement something outside the boundaries declared by the governing decision doc
- Flag OBSOLETE DECISIONS: a newer decision that supersedes an older one without the old doc being marked as such
- Flag MISSING DECISION RECORDS: technical patterns adopted in 3+ features but lacking an owning decision doc
- Do NOT suggest adding "Alternatives Considered" sections to individual docs — that's polish's job`,
		DocTypes:        []string{"decision", "refactor"},
		ContentSignals:  signalsArchitect,
	},
	{
		Name:        "ux-designer",
		DisplayName: "Gougou",
		Icon:        "🎨",
		Expertise:   "User empathy, mental models, and accessibility",
		Principles: []string{
			"Start from user empathy",
			"Respect the user's mental model",
			"Accessibility is not optional",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "User-facing change without UX impact",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasUserFacing := containsWord(lower, "user") ||
						strings.Contains(lower, "interface") ||
						strings.Contains(lower, "utilisateur") ||
						containsWord(lower, "ui") ||
						containsWord(lower, "ux")
					hasImpact := strings.Contains(lower, "impact") ||
						strings.Contains(lower, "experience") ||
						strings.Contains(lower, "expérience") ||
						strings.Contains(lower, "workflow") ||
						strings.Contains(lower, "accessib") || // matches accessibility/accessibilité
						strings.Contains(lower, "ergonomie")
					if hasUserFacing && !hasImpact {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaUxNoImpact,
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `UX LENS (Gougou):
- Add "## User Impact" if missing: who is affected? what changes in their workflow?
- Describe the before/after user experience concretely: "Before: user sees X. After: user sees Y"
- Flag accessibility concerns: does this work on all platforms? in CI? in non-TTY?
- If there's a UI change, describe exactly what the user sees (terminal output, dialog, etc.)`,
		ReviewDirective: `UX LENS (Gougou) — REVIEW MODE:
- Flag USER-JOURNEY GAPS across features: a flow that starts in one doc and ends without any doc describing the next step
- Flag INCOMPATIBLE UX PROMISES: two features that deliver conflicting experiences to the same user persona (e.g., one claims non-interactive CI support, another demands a TTY)
- Flag MISSING ACCESSIBILITY CROSS-REFERENCES: accessibility claims in one doc with no supporting design or test references anywhere else in the corpus
- Do NOT demand a "User Impact" section in every single doc — that's polish's responsibility
- Evidence quotes must prove the UX contradiction is real, not inferred from tone`,
		DocTypes:        []string{"feature"},
		ContentSignals:  signalsUXDesigner,
	},
	{
		Name:        "business-analyst",
		DisplayName: "Beda",
		Icon:        "📊",
		Expertise:   "Requirements traceability and business value",
		Principles: []string{
			"Requirements must be traceable",
			"Business value must be explicit",
			"Stakeholder alignment matters",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "Business content without explicit value",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasBusiness := strings.Contains(lower, "requirement") ||
						strings.Contains(lower, "stakeholder") ||
						strings.Contains(lower, "business") ||
						strings.Contains(lower, "customer") ||
						strings.Contains(lower, "exigence") ||
						strings.Contains(lower, "partie-prenante") ||
						containsWord(lower, "métier") ||
						containsWord(lower, "client") ||
						containsWord(lower, "besoin")
					hasValue := strings.Contains(lower, "value") ||
						strings.Contains(lower, "valeur") ||
						strings.Contains(lower, "benefit") ||
						strings.Contains(lower, "bénéfice") ||
						strings.Contains(lower, "outcome") ||
						strings.Contains(lower, "résultat") ||
						containsWord(lower, "roi") ||
						containsWord(lower, "goal") ||
						containsWord(lower, "objectif")
					if hasBusiness && !hasValue {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaBusinessNoValue,
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `BUSINESS LENS (Beda):
- Link the change to a concrete business outcome: what user problem does this solve?
- Quantify value when possible: time saved, errors prevented, users impacted
- If this was driven by a requirement, name it (compliance, SLA, customer request)
- Add ## Impact section if missing: who benefits and how?`,
		ReviewDirective: `BUSINESS LENS (Beda) — REVIEW MODE:
- Flag VALUE/SCOPE CONTRADICTIONS: two docs that promise incompatible outcomes to the same stakeholder
- Flag UNTRACEABLE REQUIREMENTS: features or releases that cite compliance/SLA/customer requirements with no requirements doc anywhere in the corpus
- Flag ORPHAN BUSINESS OUTCOMES: decision or feature docs with impact claims that never reconnect to a release or retrospective doc
- Do NOT ask for an Impact section in every doc — that's polish's beat
- Every finding must quote the contradictory or missing business claim verbatim`,
		DocTypes:       []string{"feature", "release"},
		ContentSignals: signalsBusinessAnalyst,
	},
	{
		// Ouattara is the persona of the Example Synthesizer family (story
		// 8-17/8-18). Today his draft checks cover the api-postman
		// domain (REST endpoints). When new synthesizers ship (sql-query,
		// env-vars, state-diagram), their domain-specific DraftChecks are
		// APPENDED here — the persona grows append-only with the family.
		//
		// Identifier stays "api-designer" for .lorerc backward compat. The
		// DisplayName "Ouattara" signals that the scope spans the whole
		// synthesizer family, not just Postman.
		Name:        "api-designer",
		DisplayName: "Ouattara",
		Icon:        "🔌",
		Expertise:   "API contracts, synthesizer-ready docs, HTTP semantics",
		Principles: []string{
			"A request example is worth a thousand words",
			"Every field must be typed, required-flagged, and described",
			"Errors are part of the contract, not an afterthought",
		},
		DraftChecks: []DraftCheck{
			{
				Label: "Endpoints without request examples",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasEndpoint := strings.Contains(lower, "post /") ||
						strings.Contains(lower, "get /") ||
						strings.Contains(lower, "put /") ||
						strings.Contains(lower, "patch /") ||
						strings.Contains(lower, "delete /") ||
						strings.Contains(lower, "endpoint") ||
						strings.Contains(lower, "route")
					hasExample := strings.Contains(body, "```http") ||
						strings.Contains(body, "```json") ||
						strings.Contains(lower, "curl") ||
						strings.Contains(lower, "postman") ||
						strings.Contains(lower, "example")
					if hasEndpoint && !hasExample {
						return &Suggestion{
							Category: "persona",
							Severity: "warning",
							Message:  i18n.T().Angela.PersonaAPINoExample,
						}
					}
					return nil
				},
			},
			{
				Label: "Missing error responses",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasEndpoint := strings.Contains(lower, "post /") ||
						strings.Contains(lower, "get /") ||
						strings.Contains(lower, "endpoint")
					hasErrors := strings.Contains(lower, "400") ||
						strings.Contains(lower, "401") ||
						strings.Contains(lower, "403") ||
						strings.Contains(lower, "404") ||
						strings.Contains(lower, "422") ||
						strings.Contains(lower, "500") ||
						strings.Contains(lower, "error") ||
						strings.Contains(lower, "erreur")
					if hasEndpoint && !hasErrors {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaAPIMissingErrors,
						}
					}
					return nil
				},
			},
			{
				Label: "DTO fields without required flag",
				Check: func(body string, sections map[string]string) *Suggestion {
					lower := strings.ToLower(body)
					hasDTO := strings.Contains(lower, "dto") ||
						strings.Contains(lower, "request body") ||
						strings.Contains(lower, "corps de la requête") ||
						strings.Contains(lower, "payload") ||
						strings.Contains(lower, "champ")
					hasRequiredCol := strings.Contains(lower, "requis") ||
						strings.Contains(lower, "required") ||
						strings.Contains(lower, "mandatory") ||
						strings.Contains(lower, "obligatoire") ||
						strings.Contains(body, "✅") ||
						strings.Contains(body, "✗") ||
						strings.Contains(body, "✓")
					if hasDTO && !hasRequiredCol {
						return &Suggestion{
							Category: "persona",
							Severity: "info",
							Message:  i18n.T().Angela.PersonaAPIMissingRequired,
						}
					}
					return nil
				},
			},
		},
		PromptDirective: `SYNTHESIZER-FAMILY LENS (Ouattara):
- Every endpoint MUST have a complete HTTP example block (method + URL + Authorization header + Content-Type + body):
  ` + "```http\nPOST /api/resource\nAuthorization: Bearer {{jwt}}\nContent-Type: application/json\n\n{ ... }\n```" + `
- DTO field tables MUST have columns: Field | Type | Required | Description
  Mark required fields ✅, optional fields —. Never leave the Required column empty.
- Add a dedicated "## Error Responses" section listing at minimum:
  400 Bad Request, 401 Unauthorized, 403 Forbidden, 404 Not Found, 500 Internal Server Error
  Each with a JSON body example showing the error shape your API actually returns.
- Server-injected fields (e.g. corporateCode from JWT) must be flagged explicitly:
  "Ignored — resolved server-side from the JWT principal"
- Authentication: describe the scheme (Bearer JWT, API key, Basic) and where the token comes from.
- If pagination is used: document page/size defaults, max allowed size, and the response envelope shape.
- Flag any field whose behavior differs between endpoints (e.g., a filter ignored during export).`,
		ReviewDirective: `SYNTHESIZER-FAMILY LENS (Ouattara) — REVIEW MODE:
- Flag CROSS-DOC API CONTRADICTIONS: the same endpoint documented with different methods, fields, or error codes in different docs
- Flag INCONSISTENT AUTH CLAIMS: endpoints documented with incompatible auth schemes in different docs (Bearer vs API key vs public)
- Flag SERVER-INJECTED-FIELD DIVERGENCE: a field declared server-injected in one doc but treated as client-provided elsewhere (this pairs with invariant I5)
- Flag MISSING OR DUPLICATE ERROR-RESPONSE TABLES across the API corpus
- Do NOT add HTTP blocks or error tables yourself — that's polish + synthesizer territory (stories 8-17/8-18)`,
		DocTypes:       []string{"feature", "reference", "api"},
		ContentSignals: signalsAPIDesigner,
	},
}

// GetRegistry returns a deep copy of the persona registry.
// Slices (Principles, DraftChecks, DocTypes, ContentSignals) are independently copied.
func GetRegistry() []PersonaProfile {
	out := make([]PersonaProfile, len(registry))
	for i, p := range registry {
		out[i] = p
		out[i].Principles = append([]string(nil), p.Principles...)
		out[i].DraftChecks = append([]DraftCheck(nil), p.DraftChecks...)
		out[i].DocTypes = append([]string(nil), p.DocTypes...)
		out[i].ContentSignals = append([]string(nil), p.ContentSignals...)
	}
	return out
}

// containsWord checks if word appears as a standalone token in text.
// Case-insensitive. Handles punctuation stripping and French elision
// (l'utilisateur → matches "utilisateur", d'architecture → matches "architecture").
func containsWord(text, word string) bool {
	target := strings.ToLower(word)
	for _, w := range strings.Fields(strings.ToLower(text)) {
		cleaned := strings.Trim(w, ".,;:!?()[]{}\"'`*_~<>/")
		if cleaned == target {
			return true
		}
		// Handle French elision: l'utilisateur, d'architecture, s'assurer, n'importe
		for _, sep := range []string{"'", "\u2019"} {
			for _, part := range strings.Split(cleaned, sep) {
				if part == target {
					return true
				}
			}
		}
	}
	return false
}

// extractAllSections parses a markdown body into a map of header → content.
func extractAllSections(body string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(body, "\n")
	var currentHeader string
	var currentContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			if currentHeader != "" {
				sections[currentHeader] = strings.Join(currentContent, "\n")
			}
			currentHeader = trimmed
			currentContent = nil
			continue
		}
		if currentHeader != "" {
			currentContent = append(currentContent, line)
		}
	}
	if currentHeader != "" {
		sections[currentHeader] = strings.Join(currentContent, "\n")
	}

	return sections
}

// ResolvePersonas selects up to 3 personas based on document type and content signals.
// Type match = +10 points, each content signal found = +2 points.
// Returns fallback [tech-writer] if no persona scores > 0.
func ResolvePersonas(docType, body string) []ScoredPersona {
	lower := strings.ToLower(body)
	lowerType := strings.ToLower(docType)
	var results []ScoredPersona

	for _, p := range registry {
		score := 0

		// Type match (case-insensitive)
		for _, dt := range p.DocTypes {
			if dt == lowerType {
				score += 10
				break
			}
		}

		// Content signals (word-boundary match)
		for _, signal := range p.ContentSignals {
			if containsWord(lower, signal) {
				score += 2
			}
		}

		if score > 0 {
			results = append(results, ScoredPersona{Profile: p, Score: score})
		}
	}

	// Fallback: tech-writer
	if len(results) == 0 {
		for _, p := range registry {
			if p.Name == "tech-writer" {
				return []ScoredPersona{{Profile: p, Score: 0}}
			}
		}
	}

	// Sort by score descending (insertion sort — max 6 elements)
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	// Top 3 max
	if len(results) > 3 {
		results = results[:3]
	}

	return results
}

// audiencePersonaBoosts maps audience keywords to persona names that should be prioritized.
var audiencePersonaBoosts = map[string][]string{
	// Commercial / sales / business
	"commercial":  {"business-analyst", "storyteller"},
	"commerciale": {"business-analyst", "storyteller"},
	"vente":       {"business-analyst", "storyteller"},
	"sales":       {"business-analyst", "storyteller"},
	"business":    {"business-analyst", "storyteller"},
	"marketing":   {"business-analyst", "storyteller", "ux-designer"},
	"client":      {"business-analyst", "ux-designer"},

	// Management / executive
	"cto":         {"architect", "business-analyst"},
	"ceo":         {"business-analyst", "storyteller"},
	"management":  {"business-analyst", "architect"},
	"direction":   {"business-analyst", "architect"},
	"executive":   {"business-analyst", "architect"},

	// Technical audiences
	"développeur":     {"tech-writer", "architect"},
	"developer":       {"tech-writer", "architect"},
	"nouveau":         {"tech-writer", "storyteller"},
	"junior":          {"tech-writer", "storyteller"},
	"onboarding":      {"tech-writer", "storyteller"},

	// Quality / audit
	"audit":      {"qa-reviewer", "business-analyst"},
	"compliance": {"qa-reviewer", "business-analyst"},
	"qualité":    {"qa-reviewer", "tech-writer"},
	"qa":         {"qa-reviewer", "tech-writer"},

	// Design
	"design":    {"ux-designer", "storyteller"},
	"ux":        {"ux-designer", "storyteller"},
	"ergonomie": {"ux-designer", "storyteller"},

	// API / integration — synthesizer family
	"api":         {"api-designer", "tech-writer"},
	"postman":     {"api-designer", "tech-writer"},
	"intégration": {"api-designer", "architect"},
	"integration": {"api-designer", "architect"},
	"swagger":     {"api-designer", "tech-writer"},
	"openapi":     {"api-designer", "tech-writer"},
	"webhook":     {"api-designer", "architect"},
	"rest":        {"api-designer", "tech-writer"},
	"http":        {"api-designer", "tech-writer"},
}

// ResolvePersonasForAudience selects personas optimized for a target audience.
// It boosts personas whose expertise matches the audience, then falls back to
// standard resolution for remaining slots.
func ResolvePersonasForAudience(docType, body, audience string) []ScoredPersona {
	if audience == "" {
		return ResolvePersonas(docType, body)
	}

	// Find matching boosts from audience keywords
	boosted := map[string]bool{}
	lowerAud := strings.ToLower(audience)
	for keyword, personaNames := range audiencePersonaBoosts {
		if strings.Contains(lowerAud, keyword) {
			for _, name := range personaNames {
				boosted[name] = true
			}
		}
	}

	// Start with standard resolution
	standard := ResolvePersonas(docType, body)

	if len(boosted) == 0 {
		return standard
	}

	// Re-score: boosted personas get +20 points
	var results []ScoredPersona
	for _, p := range registry {
		score := 0

		// Standard scoring
		lowerType := strings.ToLower(docType)
		for _, dt := range p.DocTypes {
			if dt == lowerType {
				score += 10
				break
			}
		}
		lower := strings.ToLower(body)
		for _, signal := range p.ContentSignals {
			if containsWord(lower, signal) {
				score += 2
			}
		}

		// Audience boost
		if boosted[p.Name] {
			score += 20
		}

		if score > 0 {
			results = append(results, ScoredPersona{Profile: p, Score: score})
		}
	}

	// Sort descending
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	if len(results) > 3 {
		results = results[:3]
	}

	return results
}

// DescribePersonas returns a human-readable string of active personas with scores.
func DescribePersonas(scored []ScoredPersona) string {
	if len(scored) == 0 {
		return "none"
	}
	var parts []string
	for _, sp := range scored {
		parts = append(parts, fmt.Sprintf("%s %s (score: %d)", sp.Profile.Icon, sp.Profile.DisplayName, sp.Score))
	}
	return strings.Join(parts, ", ")
}

// Profiles extracts PersonaProfile slice from scored results.
func Profiles(scored []ScoredPersona) []PersonaProfile {
	out := make([]PersonaProfile, len(scored))
	for i, sp := range scored {
		out[i] = sp.Profile
	}
	return out
}

// ─── Smart persona selection per document type ──────────────────

// defaultPersonaMapping maps document types to an ordered list of
// persona names. When `Selection == "auto"`, the first N entries
// (up to Max) are selected. The order is intentional: highest-value
// lens first so the Max cap keeps the best ones.
//
// The 12 mappings below are the MVP defaults. Post-MVP, users
// will be able to override via `cfg.Angela.Personas.TypeMapping`.
var defaultPersonaMapping = map[string][]string{
	"decision":  {"storyteller", "architect", "business-analyst"},
	"feature":   {"storyteller", "tech-writer", "qa-reviewer", "ux-designer"},
	"bugfix":    {"qa-reviewer", "tech-writer"},
	"refactor":  {"architect", "tech-writer"},
	"tutorial":  {"tech-writer", "storyteller"},
	"guide":     {"tech-writer", "ux-designer"},
	"howto":     {"tech-writer", "qa-reviewer"},
	"reference": {"tech-writer", "api-designer"},
	"landing":   {"tech-writer", "business-analyst"},
	"concept":   {"tech-writer", "storyteller"},
	"blog-post": {"storyteller", "tech-writer"},
	"api":       {"api-designer", "tech-writer", "qa-reviewer"},
}

// registryByName indexes the persona registry by name for O(1) lookup.
// Built via sync.Once to prevent data race under concurrent access.
// sync.Once is safe here because `registry` is a compile-time constant;
// no test needs to mutate it, so the index never needs to be rebuilt.
var (
	registryByName     map[string]PersonaProfile
	registryByNameOnce sync.Once
)

func personaByName(name string) (PersonaProfile, bool) {
	registryByNameOnce.Do(func() {
		registryByName = make(map[string]PersonaProfile, len(registry))
		for _, p := range registry {
			registryByName[p.Name] = p
		}
	})
	p, ok := registryByName[name]
	return p, ok
}

// PersonaByName is the exported variant of personaByName for callers outside
// the angela package (e.g. cmd formatters that need to resolve persona names
// carried on ReviewFinding.Personas to a full PersonaProfile for display).
func PersonaByName(name string) (PersonaProfile, bool) {
	return personaByName(name)
}

// SelectPersonasForDoc returns the persona profiles to activate for a
// given document, honoring the PersonasConfig selection mode and
// free-form mode switch.
//
// Takes docType string (not full DocMeta) by design — only the type
// field is needed for persona selection in the current scope. Expanding
// to full DocMeta is deferred to post-MVP.
//
func SelectPersonasForDoc(docType string, cfg config.PersonasConfig) []PersonaProfile {
	maxP := cfg.Max
	if maxP <= 0 {
		maxP = 3
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Selection)) {
	case "none":
		return nil
	case "all":
		all := GetRegistry()
		if len(all) > maxP {
			all = all[:maxP]
		}
		return all
	case "manual":
		return resolveManualPersonas(cfg.ManualList, maxP)
	default: // "auto" or empty
		return selectAutoPersonas(docType, cfg.FreeFormMode, maxP)
	}
}

// resolveManualPersonas returns profiles matching the names in list,
// capped at max. Unknown names are silently skipped.
func resolveManualPersonas(list []string, max int) []PersonaProfile {
	var out []PersonaProfile
	for _, name := range list {
		if len(out) >= max {
			break
		}
		if p, ok := personaByName(strings.TrimSpace(name)); ok {
			out = append(out, p)
		}
	}
	return out
}

// selectAutoPersonas implements the "auto" selection mode using the
// type→persona mapping and the free-form mode switch.
func selectAutoPersonas(docType, freeFormMode string, max int) []PersonaProfile {
	lower := strings.ToLower(strings.TrimSpace(docType))
	freeForm := isFreeFormType(lower)

	if freeForm {
		switch strings.ToLower(strings.TrimSpace(freeFormMode)) {
		case "none":
			// Free-form docs get zero personas.
			return nil
		case "minimal", "":
			// Free-form docs get only tech-writer.
			if p, ok := personaByName("tech-writer"); ok {
				return []PersonaProfile{p}
			}
			return nil
		// "full" falls through to the mapping below.
		}
	}

	// Look up the type mapping. For strict types this is always used;
	// for free-form types it's only reached when FreeFormMode == "full".
	names, ok := defaultPersonaMapping[lower]
	if !ok {
		// Unknown type → fallback to tech-writer only.
		if p, ok := personaByName("tech-writer"); ok {
			return []PersonaProfile{p}
		}
		return nil
	}
	return resolveManualPersonas(names, max)
}

// AverageScore returns the average resolution score of the given scored personas.
func AverageScore(scored []ScoredPersona) float64 {
	if len(scored) == 0 {
		return 0
	}
	total := 0
	for _, sp := range scored {
		total += sp.Score
	}
	return float64(total) / float64(len(scored))
}

// RunPersonaDraftChecks runs all draft checks from the given personas against the body.
// Suggestion messages are decorated with the persona's icon and display name.
func RunPersonaDraftChecks(body string, personas []PersonaProfile) []Suggestion {
	return runPersonaDraftChecksWithSections(body, personas, extractAllSections(body))
}

// runPersonaDraftChecksWithSections is the internal implementation using pre-parsed sections.
func runPersonaDraftChecksWithSections(body string, personas []PersonaProfile, sections map[string]string) []Suggestion {
	var suggestions []Suggestion

	for _, p := range personas {
		for _, dc := range p.DraftChecks {
			if s := dc.Check(body, sections); s != nil {
				decorated := *s
				decorated.Message = fmt.Sprintf("[%s %s] %s", p.Icon, p.DisplayName, s.Message)
				suggestions = append(suggestions, decorated)
			}
		}
	}

	return suggestions
}

// BuildPersonaPrompt constructs the persona section for the AI polish prompt.
func BuildPersonaPrompt(personas []PersonaProfile) string {
	if len(personas) == 0 {
		return ""
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "YOUR EXPERT TEAM FOR THIS REVIEW:\nAngela activates %d expert lens(es) for this document:\n\n", len(personas))

	// Persona fields are sanitized before injection. Today personas come
	// from a compile-time registry, but the cmd layer already accepts
	// persona names from `.lorerc` and future work may extend config-driven
	// persona content — close the asymmetry with `BuildPolishPrompt` now,
	// since audience/style-guide are already sanitized there and here.
	for _, p := range personas {
		fmt.Fprintf(&sb, "%s %s — %s\n",
			sanitizeShortField(p.Icon),
			sanitizeShortField(p.DisplayName),
			sanitizeShortField(p.Expertise),
		)
	}
	sb.WriteString("\n")

	for _, p := range personas {
		sb.WriteString(sanitizePromptContent(p.PromptDirective))
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// BuildPersonaReviewPrompt builds the persona-lens block for the REVIEW
// command. Each persona's directive MUST be review-specific: corpus-wide
// coherence concerns, not per-document polish fixes. When a persona has no
// ReviewDirective populated, we fall back to PromptDirective with an explicit
// WARNING line so the mis-targeted instructions do not silently degrade
// review quality.
func BuildPersonaReviewPrompt(personas []PersonaProfile) string {
	if len(personas) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "YOUR EXPERT REVIEW TEAM:\nAngela activates %d expert lens(es) for this corpus review:\n\n", len(personas))

	// Same sanitization policy as BuildPersonaPrompt.
	for _, p := range personas {
		fmt.Fprintf(&sb, "%s %s — %s\n",
			sanitizeShortField(p.Icon),
			sanitizeShortField(p.DisplayName),
			sanitizeShortField(p.Expertise),
		)
	}
	sb.WriteString("\n")

	for _, p := range personas {
		if strings.TrimSpace(p.ReviewDirective) != "" {
			sb.WriteString(sanitizePromptContent(p.ReviewDirective))
		} else {
			// Fallback with explicit warning so the AI doesn't silently apply
			// draft-oriented instructions to a corpus-review task.
			fmt.Fprintf(&sb, "WARNING: persona %q has no review-specific directive; using draft directive as fallback — findings may drift toward polish concerns.\n",
				sanitizeShortField(p.Name))
			sb.WriteString(sanitizePromptContent(p.PromptDirective))
		}
		sb.WriteString("\n\n")
	}
	return sb.String()
}
