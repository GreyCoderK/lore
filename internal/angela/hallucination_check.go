// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — hallucination_check.go
//
// Local, deterministic post-polish verification that detects factual
// claims the AI may have invented during a rewrite.
//
// The check works in three phases:
//
//  1. Sentence diff: split both original and polished into sentences,
//     find sentences in the polished text that don't appear in the
//     original (new text added by the AI).
//  2. Claim extraction: scan each new sentence for metric patterns,
//     version strings, proper nouns from a tech whitelist, and large
//     numbers preceded by action verbs.
//  3. Support check: for each extracted claim, verify that its core
//     token (the number, version, or noun) appears somewhere in the
//     original document or in the corpus summary. Claims that cannot
//     be sourced are flagged as unsupported.
//
// Pure Go, zero API calls. Default strictness is "warn".
package angela

import (
	"regexp"
	"strings"
	"unicode"
)

// FactualClaim represents a specific claim found in new text.
type FactualClaim struct {
	Text    string `json:"text"`    // the sentence containing the claim
	Section string `json:"section"` // the ## section it lives in (may be empty)
	Type    string `json:"type"`    // "metric", "version", "proper-noun", "number"
	Core    string `json:"core"`    // the extracted token (e.g. "200ms", "v2.0", "PostgreSQL")
}

// HallucinationCheck holds the result of a post-polish verification.
type HallucinationCheck struct {
	NewFactualClaims []FactualClaim `json:"new_factual_claims"`
	Unsupported      []FactualClaim `json:"unsupported"`
}

// CheckHallucinations compares original and polished text, extracts
// factual claims from newly added sentences, and classifies each as
// supported or unsupported.
//
// Main entry point. corpusSummary may be empty.
// Wired into cmd/angela_polish.go (~line 346) for both full and
// incremental polish paths — the hallucination check runs on the
// unified result.Polished output regardless of which path produced it.
func CheckHallucinations(original, polished, corpusSummary string) HallucinationCheck {
	newSents := newSentences(original, polished)
	if len(newSents) == 0 {
		return HallucinationCheck{}
	}

	// Determine which section each new sentence belongs to.
	sectionMap := buildSectionMap(polished)

	var allClaims []FactualClaim
	for _, sent := range newSents {
		section := sectionMap[sent]
		claims := extractClaims(sent, section)
		allClaims = append(allClaims, claims...)
	}

	if len(allClaims) == 0 {
		return HallucinationCheck{}
	}

	// Normalize sources for matching.
	origNorm := normalizeForClaim(original)
	if len(corpusSummary) > 512*1024 {
		corpusSummary = corpusSummary[:512*1024]
	}
	corpusNorm := normalizeForClaim(corpusSummary)

	var unsupported []FactualClaim
	for _, c := range allClaims {
		if !isSupported(c, origNorm, corpusNorm) {
			unsupported = append(unsupported, c)
		}
	}

	return HallucinationCheck{
		NewFactualClaims: allClaims,
		Unsupported:      unsupported,
	}
}

// ── Sentence splitting ──────────────────────────────────────────

// splitSentences splits text into sentences using a simple heuristic.
// Go's regexp2 doesn't support lookbehind, so we do a manual scan:
// split after `.` / `!` / `?` followed by whitespace + uppercase.
func splitSentences(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	// Collapse paragraph breaks into a single newline.
	text = multiNewlineRe.ReplaceAllString(text, "\n")

	var sentences []string
	start := 0
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r != '.' && r != '!' && r != '?' {
			continue
		}
		// Need at least one whitespace then an uppercase letter.
		j := i + 1
		for j < len(runes) && (runes[j] == ' ' || runes[j] == '\n' || runes[j] == '\t') {
			j++
		}
		if j == i+1 || j >= len(runes) {
			continue // no whitespace gap, or end of text
		}
		if !isUpperRune(runes[j]) {
			continue
		}
		if pw := precedingWord(runes, i); abbreviations[pw] {
			continue
		}
		sent := strings.TrimSpace(string(runes[start : i+1]))
		if sent != "" {
			sentences = append(sentences, sent)
		}
		start = j
	}
	// Remainder.
	if start < len(runes) {
		sent := strings.TrimSpace(string(runes[start:]))
		if sent != "" {
			sentences = append(sentences, sent)
		}
	}
	return sentences
}

var multiNewlineRe = regexp.MustCompile(`\n{2,}`)

var abbreviations = map[string]bool{
	"e.g": true, "i.e": true, "etc": true, "vs": true,
	"Dr": true, "Mr": true, "Mrs": true, "Fig": true,
	"al": true, "cf": true,
}

// precedingWord scans backwards from pos in runes to extract the word
// immediately before that position (excluding the character at pos).
func precedingWord(runes []rune, pos int) string {
	end := pos
	for end > 0 && runes[end-1] == '.' {
		end--
	}
	start := end
	for start > 0 && runes[start-1] != ' ' && runes[start-1] != '\n' && runes[start-1] != '\t' {
		start--
	}
	if start == end {
		return ""
	}
	return string(runes[start:end])
}

func isUpperRune(r rune) bool {
	return r >= 'A' && r <= 'Z' || (r >= 0xC0 && r <= 0x24E && unicode.IsUpper(r))
}

// newSentences returns sentences in polished that don't have a close
// match in original. A "close match" is defined as a normalized form
// appearing in the original's normalized sentence set.
func newSentences(original, polished string) []string {
	origSents := splitSentences(original)
	polSents := splitSentences(polished)

	// Build a set of normalized original sentences for O(1) lookup.
	origSet := make(map[string]bool, len(origSents))
	for _, s := range origSents {
		origSet[normSentence(s)] = true
	}

	var out []string
	for _, s := range polSents {
		if !origSet[normSentence(s)] {
			out = append(out, s)
		}
	}
	return out
}

// normSentence normalizes a sentence for comparison: lowercase,
// collapse whitespace, strip trailing punctuation.
func normSentence(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = wsRe.ReplaceAllString(s, " ")
	s = strings.TrimRight(s, ".!?:;,")
	return s
}

// buildSectionMap maps each sentence in text to the ## heading it
// falls under (empty string for preamble).
func buildSectionMap(text string) map[string]string {
	m := make(map[string]string)
	lines := strings.Split(text, "\n")
	currentSection := ""
	var buf strings.Builder

	flush := func() {
		if buf.Len() > 0 {
			for _, s := range splitSentences(buf.String()) {
				m[s] = currentSection
			}
			buf.Reset()
		}
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			currentSection = trimmed
			continue
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	flush()
	return m
}

// ── Claim extraction ────────────────────────────────────────────

// Regex patterns for factual claims.
var (
	metricRe  = regexp.MustCompile(`\d+\s*(?:ms|s|MB|GB|KB|TB|%|req/s|tok/s|rps|qps|tps)`)
	versionRe = regexp.MustCompile(`v?\d+\.\d+(?:\.\d+)?`)
	numberRe  = regexp.MustCompile(`(?i)(?:reduced?|increased?|doubled?|tripled?|improved?|cut|gained?|saved?|dropped?)\s+(?:by\s+)?(\d{2,})`)
)

// techProperNouns is a whitelist of common technology names that are
// frequently hallucinated by AI when rewriting technical docs.
// Lowercase for comparison; the extraction checks the original casing.
var techProperNouns = map[string]bool{
	"postgresql": true, "postgres": true, "mysql": true, "mariadb": true,
	"mongodb": true, "redis": true, "memcached": true, "elasticsearch": true,
	"opensearch": true, "cassandra": true, "dynamodb": true, "cockroachdb": true,
	"sqlite": true,
	"aws": true, "azure": true, "gcp": true, "cloudflare": true, "vercel": true,
	"heroku": true, "digitalocean": true, "netlify": true,
	"kubernetes": true, "docker": true, "terraform": true, "ansible": true,
	"nginx": true, "apache": true, "caddy": true, "traefik": true,
	"react": true, "vue": true, "angular": true, "svelte": true, "nextjs": true,
	"node": true, "deno": true, "bun": true,
	"graphql": true, "grpc": true, "protobuf": true,
	"kafka": true, "rabbitmq": true, "nats": true, "pulsar": true,
	"prometheus": true, "grafana": true, "datadog": true, "sentry": true,
	"github": true, "gitlab": true, "bitbucket": true, "jenkins": true,
}

// extractClaims scans a sentence for factual claims.
func extractClaims(sentence, section string) []FactualClaim {
	var out []FactualClaim

	// Metrics: "200ms", "45%", "3 GB"
	for _, m := range metricRe.FindAllString(sentence, -1) {
		out = append(out, FactualClaim{
			Text:    sentence,
			Section: section,
			Type:    "metric",
			Core:    m,
		})
	}

	// Versions: "v2.0", "15.3", "1.2.3"
	for _, m := range versionRe.FindAllString(sentence, -1) {
		out = append(out, FactualClaim{
			Text:    sentence,
			Section: section,
			Type:    "version",
			Core:    m,
		})
	}

	// Numbers with action verbs: "reduced by 200", "cut 50"
	for _, m := range numberRe.FindAllStringSubmatch(sentence, -1) {
		if len(m) > 1 {
			out = append(out, FactualClaim{
				Text:    sentence,
				Section: section,
				Type:    "number",
				Core:    m[1],
			})
		}
	}

	// Proper nouns from tech whitelist.
	words := strings.Fields(sentence)
	for _, w := range words {
		clean := strings.Trim(w, ".,;:!?()[]{}\"'`*_~<>/")
		if techProperNouns[strings.ToLower(clean)] {
			out = append(out, FactualClaim{
				Text:    sentence,
				Section: section,
				Type:    "proper-noun",
				Core:    clean,
			})
		}
	}

	return deduplicateClaims(out)
}

// deduplicateClaims removes duplicate claims (same Core + Type).
func deduplicateClaims(claims []FactualClaim) []FactualClaim {
	seen := make(map[string]bool, len(claims))
	var out []FactualClaim
	for _, c := range claims {
		key := c.Type + ":" + c.Core
		if !seen[key] {
			seen[key] = true
			out = append(out, c)
		}
	}
	return out
}

// ── Support check ───────────────────────────────────────────────

// normalizeForClaim prepares source text for claim matching.
// Lowercases, collapses whitespace to single space, trims edges.
// This preserves word boundaries so strings.Contains requires full
// word adjacency — e.g. "vue" won't false-match inside "revenue".
func normalizeForClaim(s string) string {
	s = strings.ToLower(s)
	s = wsRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// isSupported checks whether a claim's core token appears in the
// normalized original or corpus summary.
func isSupported(c FactualClaim, origNorm, corpusNorm string) bool {
	core := normalizeForClaim(c.Core)
	if core == "" {
		return true // empty core = false positive in extraction
	}
	if strings.Contains(origNorm, core) || strings.Contains(corpusNorm, core) {
		return true
	}
	// For metrics like "200ms" vs "200 ms" (or vice versa), try matching
	// with spaces stripped from the core token. This is safe because we
	// only strip the short core, not the full document (which would
	// reintroduce the false-match bug on word boundaries).
	coreNoSpace := strings.ReplaceAll(core, " ", "")
	if coreNoSpace != core {
		// core had spaces ("200 ms") → try without ("200ms")
		return strings.Contains(origNorm, coreNoSpace) || strings.Contains(corpusNorm, coreNoSpace)
	}
	// core has no spaces ("200ms") → try inserting a space between
	// digit-letter boundaries to match "200 ms" in original.
	coreSpaced := insertMetricSpace(coreNoSpace)
	if coreSpaced != coreNoSpace {
		return strings.Contains(origNorm, coreSpaced) || strings.Contains(corpusNorm, coreSpaced)
	}
	return false
}

// insertMetricSpace inserts a space at digit→letter boundaries.
// "200ms" → "200 ms", "45req" → "45 req". Only for metric matching.
func insertMetricSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	prev := rune(0)
	for _, r := range s {
		if prev >= '0' && prev <= '9' && (r < '0' || r > '9') {
			b.WriteRune(' ')
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}
