// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// CheckCoherence validates a document against the existing corpus.
// Works on metadata only (no file reads) for performance.
func CheckCoherence(doc string, meta domain.DocMeta, corpus []domain.DocMeta) []Suggestion {
	if len(corpus) == 0 {
		return nil
	}

	var suggestions []Suggestion

	// Skip tag-based duplicate detection for free-form types: in standalone
	// mode (mkdocs sites, generic Markdown directories) the tags for these
	// types are inferred from the filename, which creates massive
	// false-positive noise (e.g. EN/FR translation pairs flagged as dupes,
	// or every "index" page marked duplicate of every other).
	dupeCheckEnabled := !isFreeFormType(meta.Type)

	// Check for similar documents (same type + shared tag)
	if dupeCheckEnabled {
		for _, other := range corpus {
			if other.Filename == meta.Filename {
				continue // skip self
			}
			if other.Type == meta.Type && hasSharedTag(meta.Tags, other.Tags) {
				// Ignore translation pairs (foo.md vs foo.fr.md, foo.en.md, etc.)
				if isTranslationPair(meta.Filename, other.Filename) {
					continue
				}
				suggestions = append(suggestions, Suggestion{
					Category: "coherence",
					Severity: "warning",
					Message:  fmt.Sprintf("Possible duplicate: %s (same type, shared tags: %s)", other.Filename, sharedTags(meta.Tags, other.Tags)),
				})
			}
		}
	}

	// Suggest cross-references for docs with shared tags
	if dupeCheckEnabled {
		for _, other := range corpus {
			if other.Filename == meta.Filename {
				continue
			}
			if isAlreadyRelated(meta.Related, other.Filename) {
				continue
			}
			if isTranslationPair(meta.Filename, other.Filename) {
				continue
			}
			if hasSharedTag(meta.Tags, other.Tags) && other.Type != meta.Type {
				suggestions = append(suggestions, Suggestion{
					Category: "coherence",
					Severity: "info",
					Message:  fmt.Sprintf("Related document found: %s (shared tags: %s)", other.Filename, sharedTags(meta.Tags, other.Tags)),
				})
			}
		}
	}

	// Scope-based coherence: same scope = stronger signal than shared tags.
	if meta.Scope != "" {
		for _, other := range corpus {
			if other.Filename == meta.Filename {
				continue
			}
			if other.Scope == meta.Scope {
				if other.Type == meta.Type {
					suggestions = append(suggestions, Suggestion{
						Category: "coherence",
						Severity: "warning",
						Message:  fmt.Sprintf("Same scope %q and type: %s — potential overlap, consider consolidating", meta.Scope, other.Filename),
					})
				} else if !isAlreadyRelated(meta.Related, other.Filename) {
					suggestions = append(suggestions, Suggestion{
						Category: "coherence",
						Severity: "info",
						Message:  fmt.Sprintf("Same scope %q: %s [%s] — consider adding to related", meta.Scope, other.Filename, other.Type),
					})
				}
			}
		}
	}

	// Detect corpus doc titles mentioned in body
	body := stripFrontMatter(doc)
	bodyLower := strings.ToLower(body)
	for _, other := range corpus {
		if other.Filename == meta.Filename {
			continue
		}
		if isTranslationPair(meta.Filename, other.Filename) {
			continue
		}
		slug := strings.TrimSuffix(other.Filename, ".md")
		if strings.Contains(bodyLower, strings.ToLower(slug)) {
			if !isAlreadyRelated(meta.Related, other.Filename) {
				suggestions = append(suggestions, Suggestion{
					Category: "coherence",
					Severity: "info",
					Message:  fmt.Sprintf("Document %q mentioned in body — consider adding to related", other.Filename),
				})
			}
		}
	}

	return suggestions
}

// isTranslationPair reports whether two filenames are translations of each
// other (e.g. "installation.md" vs "installation.fr.md"). Translation pairs
// share the same base name and differ only by a language code segment before
// the .md extension, or one has no language code and the other has exactly one.
// Recognised codes: fr, en, es, de, it, pt, zh, ja, ko, ru, ar, nl, pl.
func isTranslationPair(a, b string) bool {
	baseA, langA := splitLangFromFilename(a)
	baseB, langB := splitLangFromFilename(b)
	if baseA != baseB {
		return false
	}
	// Same base; pair iff exactly one is in a different language from the other.
	return langA != langB
}

// splitLangFromFilename returns (baseName, langCode) for "foo.fr.md" → ("foo", "fr").
// For "foo.md" returns ("foo", ""). Language code must be in a known list to
// avoid matching unrelated dotted filenames like "api.v2.md".
func splitLangFromFilename(filename string) (base, lang string) {
	name := strings.TrimSuffix(filename, ".md")
	// Check for trailing ".<lang>" segment.
	idx := strings.LastIndex(name, ".")
	if idx < 0 {
		return name, ""
	}
	candidate := name[idx+1:]
	switch candidate {
	case "fr", "en", "es", "de", "it", "pt", "zh", "ja", "ko", "ru", "ar", "nl", "pl":
		return name[:idx], candidate
	}
	return name, ""
}

func hasSharedTag(a, b []string) bool {
	for _, tagA := range a {
		for _, tagB := range b {
			if tagA == tagB {
				return true
			}
		}
	}
	return false
}

func sharedTags(a, b []string) string {
	return strings.Join(sharedTagList(a, b), ", ")
}

func isAlreadyRelated(related []string, filename string) bool {
	slug := strings.TrimSuffix(filename, ".md")
	for _, r := range related {
		if r == filename || r == slug {
			return true
		}
	}
	return false
}
