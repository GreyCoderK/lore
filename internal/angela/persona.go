// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// PersonaProfile represents an expert lens that Angela can activate
// for document review. Personas are Go values — no external files.
type PersonaProfile struct {
	Name            string
	DisplayName     string
	Icon            string
	Expertise       string
	Principles      []string
	DraftChecks     []DraftCheck
	PromptDirective string
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
							Message:  "Section '## Why' reads like a list — consider a narrative",
						}
					}
					return nil
				},
			},
		},
		PromptDirective: "STORYTELLING LENS (Affoue):\n- The Why is the protagonist — make it compelling\n- Move from abstract to concrete with examples\n- Use analogies to anchor understanding\n- Lists are scaffolding, not the final form",
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
								Message:  "Found paragraph(s) > 5 lines — consider breaking into bullets",
							}
						}
					}
					return nil
				},
			},
		},
		PromptDirective: "TECHNICAL WRITING LENS (Sialou):\n- Every word must serve a purpose — cut filler\n- Prefer bullets over long paragraphs\n- Use diagrams or code blocks when they communicate better than prose",
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
							Message:  "No verification criteria found — how will you know this works?",
						}
					}
					return nil
				},
			},
		},
		PromptDirective: "QA LENS (Kouame):\n- Every claim needs testable validation criteria\n- Make edge cases and failure modes explicit\n- Consider the shelf life of this documentation",
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
							Message:  "Architecture content without explicit trade-offs — what was considered and rejected?",
						}
					}
					return nil
				},
			},
		},
		PromptDirective: "ARCHITECTURE LENS (Doumbia):\n- Make trade-offs explicit — what was considered and rejected?\n- Prioritize user value over technical elegance\n- Boring technology is a feature, not a compromise",
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
							Message:  "User-facing change detected without UX impact discussion",
						}
					}
					return nil
				},
			},
		},
		PromptDirective: "UX LENS (Gougou):\n- Start from user empathy — who is affected and how?\n- Respect the user's mental model\n- Accessibility is not optional",
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
							Message:  "Business content without explicit value statement — what's the outcome?",
						}
					}
					return nil
				},
			},
		},
		PromptDirective: "BUSINESS LENS (Beda):\n- Requirements must be traceable to business goals\n- Make business value explicit — not assumed\n- Ensure stakeholder alignment is addressed",
		DocTypes:        []string{"feature", "release"},
		ContentSignals:  signalsBusinessAnalyst,
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
// Returns fallback [tech-writer] if no persona scores > 0 (AC-6).
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

	// Fallback: tech-writer (AC-6)
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

// Profiles extracts PersonaProfile slice from scored results.
func Profiles(scored []ScoredPersona) []PersonaProfile {
	out := make([]PersonaProfile, len(scored))
	for i, sp := range scored {
		out[i] = sp.Profile
	}
	return out
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

	for _, p := range personas {
		fmt.Fprintf(&sb, "%s %s — %s\n", p.Icon, p.DisplayName, p.Expertise)
	}
	sb.WriteString("\n")

	for _, p := range personas {
		sb.WriteString(p.PromptDirective)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
