// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// ParseDoc builds a Doc from raw markdown bytes. It separates frontmatter
// from body using storage.UnmarshalPermissive (synthesizers must work on
// docs that lack a fully-validated lore frontmatter, e.g. mkdocs/hugo
// corpora in standalone mode), splits the body into lines, and parses out
// the section tree.
//
// path is recorded as Doc.Path verbatim (relative or absolute, caller's
// choice). data is the raw file content.
func ParseDoc(path string, data []byte) (*Doc, error) {
	meta, body, err := storage.UnmarshalPermissive(data)
	if err != nil {
		return nil, fmt.Errorf("synthesizer: parse %s: %w", path, err)
	}

	doc := &Doc{
		Path:       path,
		Meta:       meta,
		Body:       body,
		Lines:      splitLines1Indexed(body),
		Sections:   ParseSections(body),
		Signatures: SignaturesFromMeta(meta),
	}
	return doc, nil
}

// splitLines1Indexed returns Body split on "\n" prefixed with an empty
// element so Lines[N] addresses line N (1-based). This matches Evidence.Line
// semantics and makes range-loop indices align with line numbers in error
// messages.
//
// CRLF defense (code review finding #3): even though storage.UnmarshalPermissive
// normalizes CRLF→LF before handing us the body, we strip stray trailing \r
// here as belt-and-suspenders. A CR inside a line would shift byte offsets
// and break the I4 snippet comparison at contract-check time. Keeping the
// normalization local keeps the framework correct for any future caller
// that builds a Doc without going through ParseDoc.
func splitLines1Indexed(body string) []string {
	raw := strings.Split(body, "\n")
	out := make([]string, 0, len(raw)+1)
	out = append(out, "") // index 0 placeholder
	for _, line := range raw {
		out = append(out, strings.TrimRight(line, "\r"))
	}
	return out
}

// ParseSections walks body and returns a flat list of headings in source
// order. EndLine of each section points at the last line BEFORE the next
// heading at the same or shallower level, so Content captures the entire
// subtree under the heading (including deeper sub-headings).
//
// Lines are 1-indexed (consistent with Evidence.Line).
//
// Recognized headings: ATX style only ("#" through "######"). Setext
// headings (underlines) are not supported in MVP - feature docs use ATX.
//
// Fenced code blocks (```lang ... ```) are skipped so that lines like
// "# Full" inside an http/json fence do not become phantom headings and
// prematurely close the enclosing section. Both "```" and "~~~" fences
// are recognized; tildes are accepted for compatibility with CommonMark
// even though lore generators emit backticks.
func ParseSections(body string) []Section {
	lines := strings.Split(body, "\n")
	type pending struct {
		idx     int // index in result slice
		level   int
		startLn int
	}
	var result []Section
	var stack []pending

	closeAtAndAbove := func(level, beforeLine int) {
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			result[top.idx].EndLine = beforeLine - 1
			result[top.idx].Content = sliceLinesContent(lines, top.startLn, beforeLine-1)
		}
	}

	inFence := false
	fenceChar := byte(0)

	for i, raw := range lines {
		ln := i + 1
		if marker, ok := fenceMarker(raw); ok {
			if !inFence {
				inFence = true
				fenceChar = marker
			} else if marker == fenceChar {
				inFence = false
				fenceChar = 0
			}
			continue
		}
		if inFence {
			continue
		}
		level, title := parseATXHeading(raw)
		if level == 0 {
			continue
		}
		closeAtAndAbove(level, ln)
		result = append(result, Section{
			Heading:   strings.TrimSpace(raw),
			Level:     level,
			Title:     title,
			StartLine: ln,
		})
		stack = append(stack, pending{
			idx:     len(result) - 1,
			level:   level,
			startLn: ln,
		})
	}
	closeAtAndAbove(0, len(lines)+1)
	return result
}

// fenceMarker reports whether line opens or closes a fenced code block and
// returns the fence character ('`' or '~') on match. CommonMark requires
// at least 3 consecutive fence chars; leading whitespace up to 3 spaces
// is tolerated. The closing fence must use the same character as the
// opening one (tracked by the caller), which is why we return the char.
func fenceMarker(line string) (byte, bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 {
		return 0, false
	}
	if len(trimmed) < 3 {
		return 0, false
	}
	ch := trimmed[0]
	if ch != '`' && ch != '~' {
		return 0, false
	}
	run := 0
	for run < len(trimmed) && trimmed[run] == ch {
		run++
	}
	if run < 3 {
		return 0, false
	}
	return ch, true
}

// parseATXHeading returns (level, title) for a line that is an ATX heading,
// or (0, "") otherwise. Leading whitespace is tolerated; trailing "#"
// closing characters are stripped.
func parseATXHeading(line string) (int, string) {
	trimmed := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(trimmed, "#") {
		return 0, ""
	}
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level > 6 {
		return 0, ""
	}
	if level == len(trimmed) || (trimmed[level] != ' ' && trimmed[level] != '\t') {
		return 0, ""
	}
	title := strings.TrimSpace(trimmed[level:])
	title = strings.TrimRight(title, "#")
	title = strings.TrimSpace(title)
	return level, title
}

// sliceLinesContent extracts content from lines[start:end] inclusive,
// EXCLUDING the first line (the heading itself). Returns "" when the range
// is empty or out of bounds.
func sliceLinesContent(lines []string, headingLine, endLine int) string {
	if headingLine < 1 || headingLine > len(lines) {
		return ""
	}
	contentStart := headingLine // 1-based -> next line is index headingLine in 0-based
	if contentStart >= len(lines) {
		return ""
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	if endLine < contentStart {
		return ""
	}
	return strings.Join(lines[contentStart:endLine], "\n")
}

// FuzzyFindSection scans doc.Sections for a heading whose normalized title
// matches one of the provided candidate patterns. Returns the best match,
// a confidence in [0.0, 1.0], and whether a match was found.
//
// Confidence model:
//
//   - 1.0 when the normalized heading equals the canonical form of a candidate.
//   - 0.85 when the heading's normalized title contains a candidate as a
//     whole-word substring (e.g. "API endpoints" matches "endpoints").
//   - 0.7 when a candidate regex matches anywhere in the heading.
//   - Below 0.7, no match is returned.
//
// Candidates may be plain words ("endpoints", "security") or regex
// alternatives ("(?i)endpoints?|routes?"). Plain words are interpreted
// case-insensitively with optional plural tolerance handled by the caller's
// regex (or by listing both forms).
func FuzzyFindSection(doc *Doc, candidates []string) (*Section, float64, bool) {
	if doc == nil || len(doc.Sections) == 0 || len(candidates) == 0 {
		return nil, 0, false
	}

	var best *Section
	var bestScore float64

	patterns := compileCandidates(candidates)

	for i := range doc.Sections {
		sec := &doc.Sections[i]
		score := scoreHeading(sec.Title, candidates, patterns)
		if score > bestScore {
			bestScore = score
			best = sec
		}
	}

	if bestScore < 0.7 {
		return nil, 0, false
	}
	return best, bestScore, true
}

// scoreHeading returns the highest confidence in [0, 1] that a heading
// title matches one of the candidates.
//
// Regex candidates are matched against BOTH the raw title and the
// normalized title (leading numbering / emphasis stripped). This lets
// "(?i)^sécurité$" match "3. Sécurité" in a numbered TOC.
func scoreHeading(title string, candidates []string, patterns []*regexp.Regexp) float64 {
	normTitle := normalize(title)
	var best float64

	for _, cand := range candidates {
		// Skip patterns that look like regexes (compiled separately).
		if isRegexPattern(cand) {
			continue
		}
		normCand := normalize(cand)
		if normTitle == normCand {
			return 1.0
		}
		if containsWholeWord(normTitle, normCand) {
			if 0.85 > best {
				best = 0.85
			}
		}
	}

	for _, re := range patterns {
		if re == nil {
			continue
		}
		if re.MatchString(title) || re.MatchString(normTitle) {
			if 0.7 > best {
				best = 0.7
			}
		}
	}

	return best
}

// compileCandidates turns each regex-looking candidate into a *Regexp. Bad
// patterns are logged to os.Stderr and skipped (nil slot) so the matcher
// still runs on the remaining good patterns. Silent failure was flagged by
// the 2026-04-15 code review (finding #11) as a config-debugging hazard.
func compileCandidates(candidates []string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(candidates))
	for _, c := range candidates {
		if !isRegexPattern(c) {
			continue
		}
		re, err := regexp.Compile(c)
		if err != nil {
			fmt.Fprintf(os.Stderr, "synthesizer: fuzzy-heading pattern %q failed to compile: %v\n", c, err)
			out = append(out, nil)
			continue
		}
		out = append(out, re)
	}
	return out
}

func isRegexPattern(s string) bool {
	// Heuristic: contains regex metacharacters.
	for _, r := range s {
		switch r {
		case '?', '|', '(', ')', '[', ']', '+', '*', '.', '\\', '^', '$':
			return true
		}
	}
	return false
}

// normalize lowercases, trims, collapses whitespace, and strips leading
// markdown emphasis or numbering ("1. ", "**", "_") so heading variants
// don't defeat the match. Accents are preserved (French headings stay as
// "sécurité" instead of becoming "securite") so callers must list both
// language variants when needed.
func normalize(s string) string {
	s = strings.TrimSpace(s)
	// Strip leading numbering "1. " / "1) " / "- ".
	for {
		trimmed := strings.TrimLeft(s, "0123456789")
		if len(trimmed) < len(s) && len(trimmed) > 0 {
			r := trimmed[0]
			if r == '.' || r == ')' || r == ' ' {
				s = strings.TrimLeft(trimmed, ".) ")
				continue
			}
		}
		break
	}
	s = strings.TrimLeft(s, "-*_ \t")
	s = strings.ToLower(s)
	// Collapse internal runs of whitespace.
	var b strings.Builder
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimSpace(b.String())
}

func containsWholeWord(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	idx := strings.Index(haystack, needle)
	for idx >= 0 {
		left := idx == 0 || !isWordChar(rune(haystack[idx-1]))
		right := idx+len(needle) == len(haystack) || !isWordChar(rune(haystack[idx+len(needle)]))
		if left && right {
			return true
		}
		next := strings.Index(haystack[idx+1:], needle)
		if next < 0 {
			return false
		}
		idx = idx + 1 + next
	}
	return false
}

func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// MetaFromDoc returns a domain.DocMeta populated with the given doc's
// metadata plus an updated Synthesized map. Used by the polish hook to
// produce the new frontmatter before the storage layer writes it.
func MetaFromDoc(doc *Doc, signatures map[string]Signature) domain.DocMeta {
	out := doc.Meta
	SignaturesToMeta(&out, signatures)
	return out
}
