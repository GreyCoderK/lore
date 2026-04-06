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

	// Check for similar documents (same type + shared tag)
	for _, other := range corpus {
		if other.Filename == meta.Filename {
			continue // skip self
		}
		if other.Type == meta.Type && hasSharedTag(meta.Tags, other.Tags) {
			suggestions = append(suggestions, Suggestion{
				Category: "coherence",
				Severity: "warning",
				Message:  fmt.Sprintf("Possible duplicate: %s (same type, shared tags: %s)", other.Filename, sharedTags(meta.Tags, other.Tags)),
			})
		}
	}

	// Suggest cross-references for docs with shared tags
	for _, other := range corpus {
		if other.Filename == meta.Filename {
			continue
		}
		if isAlreadyRelated(meta.Related, other.Filename) {
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
