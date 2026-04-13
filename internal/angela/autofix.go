// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — autofix.go
//
// Draft autofix — safe mechanical fixes for common findings. No AI calls.
// All fixers are deterministic, offline, and non-destructive. Two modes:
// safe (minimal, additive) and aggressive (stubs + tags).

package angela

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
)

// AutofixMode controls which fixers are applied.
type AutofixMode int

const (
	AutofixSafe       AutofixMode = iota // additive only, no content deletion
	AutofixAggressive                     // stubs + tags + related
)

// AutofixFileResult is the outcome for a single file.
type AutofixFileResult struct {
	Filename      string
	Fixed         []string // descriptions of fixes applied
	Skipped       []string // findings that need manual fix
	Error         error
	StillPresent  int // findings remaining after re-analysis
}

// AutofixReport summarizes the full autofix run.
type AutofixReport struct {
	FilesModified int
	FindingsFixed int
	FilesSkipped  int
	Errors        int
	Files         []AutofixFileResult
}

// RunAutofix applies mechanical fixes to a single file. Returns the fixed
// content and what was changed. Does NOT write to disk (caller handles that).
func RunAutofix(content string, meta domain.DocMeta, mode AutofixMode, corpus []domain.DocMeta) (string, AutofixFileResult) {
	result := AutofixFileResult{Filename: meta.Filename}
	current := content

	// Safe fixers (always applied)
	safeFixes := []struct {
		name string
		fn   func(string, domain.DocMeta) (string, []string)
	}{
		{"date", fixMissingDate},
		{"type", fixMissingType},
		{"code-fences", fixCodeFences},
		{"date-format", fixMalformedDate},
	}

	for _, f := range safeFixes {
		updated, fixed := f.fn(current, meta)
		if len(fixed) > 0 {
			current = updated
			result.Fixed = append(result.Fixed, fixed...)
		}
	}

	// Aggressive fixers (only in aggressive mode)
	if mode == AutofixAggressive {
		aggressiveFixes := []struct {
			name string
			fn   func(string, domain.DocMeta, []domain.DocMeta) (string, []string)
		}{
			{"tags", fixMissingTags},
			{"stubs", fixMissingSections},
			{"related", fixMissingRelated},
		}
		for _, f := range aggressiveFixes {
			updated, fixed := f.fn(current, meta, corpus)
			if len(fixed) > 0 {
				current = updated
				result.Fixed = append(result.Fixed, fixed...)
			}
		}
	}

	// Non-destructive guard: content must not shrink unexpectedly.
	// Allow up to 10% shrinkage for reformatting (date normalization, etc.).
	// Threshold is 10% (not a fixed byte count), and Fixed is cleared on trigger.
	maxShrink := len(content) / 10
	if maxShrink < 20 {
		maxShrink = 20
	}
	if len(current) < len(content)-maxShrink {
		result.Fixed = nil // clear misleading fixed list
		result.Error = fmt.Errorf("autofix: content shrunk by %d bytes — refusing to write (non-destructive guard)", len(content)-len(current))
		return content, result
	}

	return current, result
}

// --- Safe fixers ---

// fixMissingDate adds `date: YYYY-MM-DD` to front matter if absent.
func fixMissingDate(content string, meta domain.DocMeta) (string, []string) {
	if meta.Date != "" {
		return content, nil
	}
	if !strings.HasPrefix(content, "---\n") {
		return content, nil
	}
	today := time.Now().UTC().Format("2006-01-02")
	return insertFrontMatterField(content, "date", today), []string{"added date: " + today}
}

// fixMissingType infers type from filename or defaults to "note".
func fixMissingType(content string, meta domain.DocMeta) (string, []string) {
	if meta.Type != "" {
		return content, nil
	}
	if !strings.HasPrefix(content, "---\n") {
		return content, nil
	}
	inferredType := inferTypeFromFilename(meta.Filename)
	return insertFrontMatterField(content, "type", inferredType), []string{"added type: " + inferredType}
}

// fixCodeFences tags untagged code fences with detected language.
func fixCodeFences(content string, _ domain.DocMeta) (string, []string) {
	lines := strings.Split(content, "\n")
	var fixed []string
	inFence := false
	fenceStart := -1
	var fenceLines []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if !inFence {
				// Opening fence
				tag := strings.TrimPrefix(trimmed, "```")
				tag = strings.TrimSpace(tag)
				if tag == "" {
					// Untagged fence — collect lines until closing
					inFence = true
					fenceStart = i
					fenceLines = nil
				} else {
					// Already-tagged fence — track state so the
					// closing ``` is matched, but don't record
					// fenceStart (we won't re-tag it).
					inFence = true
				}
			} else {
				// Closing fence — detect language
				inFence = false
				if fenceStart >= 0 && len(fenceLines) > 0 {
					lang := DetectLanguageMultiLine(fenceLines)
					if lang != "" {
						lines[fenceStart] = "```" + lang
						fixed = append(fixed, fmt.Sprintf("tagged code fence line %d as %s", fenceStart+1, lang))
					}
				}
				fenceStart = -1
				fenceLines = nil
			}
		} else if inFence {
			fenceLines = append(fenceLines, line)
		}
	}

	if len(fixed) == 0 {
		return content, nil
	}
	return strings.Join(lines, "\n"), fixed
}

// malformedDateRe matches dates like 2026/04/10 or 2026.04.10.
var malformedDateRe = regexp.MustCompile(`^(\d{4})[/.](\d{2})[/.](\d{2})$`)

// fixMalformedDate reformats non-ISO dates in front matter.
func fixMalformedDate(content string, meta domain.DocMeta) (string, []string) {
	if meta.Date == "" {
		return content, nil
	}
	m := malformedDateRe.FindStringSubmatch(meta.Date)
	if m == nil {
		return content, nil
	}
	isoDate := m[1] + "-" + m[2] + "-" + m[3]
	updated := strings.Replace(content, "date: "+meta.Date, "date: "+isoDate, 1)
	if updated == content {
		// Try with quotes
		updated = strings.Replace(content, "date: \""+meta.Date+"\"", "date: "+isoDate, 1)
	}
	if updated == content {
		return content, nil
	}
	return updated, []string{fmt.Sprintf("reformatted date %s → %s", meta.Date, isoDate)}
}

// --- Aggressive fixers ---

// fixMissingTags generates tags from content using simple TF-IDF.
func fixMissingTags(content string, meta domain.DocMeta, _ []domain.DocMeta) (string, []string) {
	if len(meta.Tags) > 0 {
		return content, nil
	}
	if !strings.HasPrefix(content, "---\n") {
		return content, nil
	}
	// Refuse to insert if `tags:` already exists in front matter
	// (handles edge case where parser returned empty slice from `tags: []`).
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx >= 0 && strings.Contains(content[4:4+endIdx], "\ntags:") {
		return content, nil
	}

	tags := generateTags(content, 3)
	if len(tags) < 3 {
		return content, nil // not enough meaningful tokens
	}

	// M2 fix: emit YAML block sequence to avoid bracket-syntax issues
	// with special chars in tags.
	var tagLines strings.Builder
	tagLines.WriteString("\n")
	for _, t := range tags {
		tagLines.WriteString("  - " + t + "\n")
	}
	return insertFrontMatterField(content, "tags", tagLines.String()), []string{"generated tags: " + strings.Join(tags, ", ")}
}

// fixMissingSections inserts missing ## stubs (aggressive only).
func fixMissingSections(content string, meta domain.DocMeta, _ []domain.DocMeta) (string, []string) {
	if isFreeFormType(meta.Type) {
		return content, nil
	}

	body := stripFrontMatter(content)
	var fixed []string
	result := content

	for _, section := range []string{"What", "Why"} {
		if !hasSection(body, "## "+section) {
			stub := fmt.Sprintf("\n## %s\n\n<!-- TODO: describe %s -->\n", section, strings.ToLower(section))
			result += stub
			body += stub
			fixed = append(fixed, "added ## "+section+" stub")
		}
	}
	return result, fixed
}

// fixMissingRelated adds corpus references to front matter (aggressive only).
func fixMissingRelated(content string, meta domain.DocMeta, corpus []domain.DocMeta) (string, []string) {
	if len(meta.Related) > 0 || len(corpus) == 0 {
		return content, nil
	}
	if !strings.HasPrefix(content, "---\n") {
		return content, nil
	}

	body := stripFrontMatter(content)
	bodyLower := strings.ToLower(body)
	var found []string
	for _, other := range corpus {
		if other.Filename == meta.Filename {
			continue
		}
		slug := strings.TrimSuffix(other.Filename, ".md")
		if strings.Contains(bodyLower, strings.ToLower(slug)) {
			found = append(found, slug)
		}
	}
	if len(found) == 0 {
		return content, nil
	}

	// Add related field
	updated := content
	for _, slug := range found {
		updated = addRelatedToFrontMatter(updated, slug+".md")
	}
	if updated == content {
		return content, nil
	}
	return updated, []string{"added related: " + strings.Join(found, ", ")}
}

// --- Tag generation ---

// English + French stopwords for tag generation.
var stopwords = map[string]bool{
	// English
	"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
	"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
	"with": true, "by": true, "from": true, "is": true, "are": true, "was": true,
	"were": true, "be": true, "been": true, "being": true, "have": true, "has": true,
	"had": true, "do": true, "does": true, "did": true, "will": true, "would": true,
	"could": true, "should": true, "may": true, "might": true, "must": true,
	"shall": true, "can": true, "this": true, "that": true, "these": true,
	"those": true, "it": true, "its": true, "not": true, "no": true, "if": true,
	"then": true, "than": true, "so": true, "as": true, "up": true, "out": true,
	"about": true, "into": true, "over": true, "after": true, "before": true,
	"between": true, "each": true, "every": true, "all": true, "any": true,
	"some": true, "more": true, "most": true, "other": true, "new": true,
	"just": true, "also": true, "when": true, "how": true, "what": true,
	"which": true, "who": true, "where": true, "why": true, "we": true,
	"you": true, "they": true, "our": true, "your": true, "their": true,
	// French
	"le": true, "la": true, "les": true, "un": true, "une": true, "des": true,
	"de": true, "du": true, "et": true, "ou": true, "en": true, "est": true,
	"sont": true, "dans": true, "sur": true, "par": true, "pour": true,
	"avec": true, "ce": true, "cette": true, "ces": true, "il": true,
	"elle": true, "nous": true, "vous": true, "ils": true, "ne": true,
	"pas": true, "plus": true, "qui": true, "que": true, "se": true,
	"son": true, "sa": true, "ses": true, "au": true, "aux": true,
}

// generateTags extracts top-N meaningful keywords from document content.
func generateTags(content string, n int) []string {
	body := stripFrontMatter(content)

	// Use title + first paragraph for tag extraction
	lines := strings.Split(body, "\n")
	var text strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" && text.Len() > 200 {
			break // enough context after first paragraph
		}
		text.WriteString(trimmed + " ")
	}

	// Tokenize
	words := strings.FieldsFunc(text.String(), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	// Count term frequency
	tf := make(map[string]int)
	for _, w := range words {
		w = strings.ToLower(w)
		if len(w) < 3 || stopwords[w] {
			continue
		}
		tf[w]++
	}

	// Sort by frequency descending
	type termFreq struct {
		term string
		freq int
	}
	var ranked []termFreq
	for t, f := range tf {
		ranked = append(ranked, termFreq{t, f})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].freq != ranked[j].freq {
			return ranked[i].freq > ranked[j].freq
		}
		return ranked[i].term < ranked[j].term
	})

	var tags []string
	for _, r := range ranked {
		if len(tags) >= n {
			break
		}
		tags = append(tags, r.term)
	}
	return tags
}

// --- Helpers ---

// insertFrontMatterField adds a field to existing YAML front matter.
// If value starts with a newline (multiline block, e.g. YAML sequence),
// the key is emitted on its own line without a trailing space before the value.
func insertFrontMatterField(content, key, value string) string {
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	endIdx := strings.Index(content[4:], "\n---\n")
	if endIdx < 0 {
		return content
	}
	insertPos := 4 + endIdx
	var newLine string
	if strings.HasPrefix(value, "\n") {
		newLine = fmt.Sprintf("%s:%s", key, value)
	} else {
		newLine = fmt.Sprintf("%s: %s\n", key, value)
	}
	return content[:insertPos] + "\n" + newLine + content[insertPos:]
}

// inferTypeFromFilename maps directory patterns to document types.
func inferTypeFromFilename(filename string) string {
	lower := strings.ToLower(filename)
	dir := strings.ToLower(filepath.Dir(filename))

	patterns := map[string]string{
		"decision": "decision",
		"feature":  "feature",
		"bugfix":   "bugfix",
		"refactor": "refactor",
		"guide":    "guide",
		"tutorial": "tutorial",
	}
	for pattern, docType := range patterns {
		if strings.Contains(dir, pattern) || strings.Contains(lower, pattern) {
			return docType
		}
	}
	return "note"
}

// AutofixDryRun computes what would change without writing. Returns a
// unified diff string.
func AutofixDryRun(original, fixed, filename string) string {
	if original == fixed {
		return ""
	}
	diff, err := UnifiedDiffString(original, fixed, UnifiedDiffOptions{
		FromFile: filename + " (original)",
		ToFile:   filename + " (autofixed)",
		Context:  3,
	})
	if err != nil {
		return fmt.Sprintf("(diff error: %v)", err)
	}
	return diff
}

// AutofixWriteWithBackup writes the fixed content with optional backup.
func AutofixWriteWithBackup(docPath, fixed string, backupEnabled bool, workDir, stateDir string) (string, error) {
	if backupEnabled {
		backupPath, err := WriteBackup(workDir, stateDir, "draft-backups", docPath)
		if err != nil {
			return "", fmt.Errorf("autofix: backup: %w", err)
		}
		_ = backupPath
	}
	if err := fileutil.AtomicWrite(docPath, []byte(fixed), 0o644); err != nil {
		return "", fmt.Errorf("autofix: write: %w", err)
	}
	return docPath, nil
}

// ReanalyzeAfterFix runs AnalyzeDraft on the fixed content and returns
// the count of remaining findings.
func ReanalyzeAfterFix(fixed string, meta domain.DocMeta, guide *StyleGuide, corpus []domain.DocMeta) int {
	suggestions := AnalyzeDraft(fixed, meta, guide, corpus, nil)
	return len(suggestions)
}

// FormatAutofixReport produces the human-readable summary.
func FormatAutofixReport(report AutofixReport) string {
	var b strings.Builder
	b.WriteString("Autofix summary:\n")
	fmt.Fprintf(&b, "  %d files modified\n", report.FilesModified)
	fmt.Fprintf(&b, "  %d findings fixed\n", report.FindingsFixed)
	fmt.Fprintf(&b, "  %d files skipped (no fixable findings)\n", report.FilesSkipped)
	b.WriteString(fmt.Sprintf("  %d errors\n", report.Errors))

	if report.FilesModified > 0 {
		b.WriteString("\n  Fixed:\n")
		for _, f := range report.Files {
			if len(f.Fixed) > 0 {
				b.WriteString(fmt.Sprintf("    %s: %s\n", f.Filename, strings.Join(f.Fixed, ", ")))
			}
		}
	}
	return b.String()
}

// ParseAutofixMode converts a string flag value to AutofixMode.
func ParseAutofixMode(s string) (AutofixMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "safe":
		return AutofixSafe, nil
	case "aggressive":
		return AutofixAggressive, nil
	default:
		return 0, fmt.Errorf("autofix: unknown mode %q (expected safe or aggressive)", s)
	}
}

