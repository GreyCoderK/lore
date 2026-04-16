// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package apipostman

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
)

const (
	synthesizerName    = "api-postman"
	synthesizerVersion = "1.0.0"

	// doctTypeWhitelist is the frontmatter type values this synthesizer
	// considers for activation. Not exhaustive; extended via per-synthesizer
	// config when projects want custom doc types.
	docTypeFeature   = "feature"
	docTypeAPI       = "api"
	docTypeReference = "reference"
)

// endpointsHeadingCandidates lists the fuzzy patterns used by
// FuzzyFindSection to locate the endpoints section. Bilingual by design:
// we support French and English headings without requiring template
// conformance (tolerant parsing, decision Q1).
var endpointsHeadingCandidates = []string{
	"(?i)^endpoints?$",
	"(?i)^routes?$",
	"(?i)^apis?$",
	"(?i)^rest$",
	"(?i)^api$",
	"(?i)^http routes?$",
}

var filtersHeadingCandidates = []string{
	"(?i)^filtres?(?:\\s+accept[eé]s?)?.*$",
	"(?i)^filters?$",
	"(?i)^params?$",
	"(?i)^body$",
	"(?i)^payload$",
	"(?i)^request$",
}

var securityHeadingCandidates = []string{
	"(?i)^s[eé]curit[eé]$",
	"(?i)^security$",
	"(?i)^auth(orisation|entication|z|n)?$",
}

// endpointPattern matches a "POST /api/..." token anywhere in a line. The
// gap between method and path is permissive (backticks, pipes, spaces) to
// support markdown tables, list items, and prose; the path terminator
// excludes whitespace, backticks, pipes, and parens so the captured path
// stays clean regardless of the surrounding decoration.
var endpointPattern = regexp.MustCompile(`(?i)\b(POST|GET|PUT|PATCH|DELETE)\b[^\n]*?(/api/[^\s\`+"`"+`\|)]+)`)

// backtickField matches an inline-code span (` `name` `) with a valid
// identifier-like name. Used to extract fields from prose enumerations and
// security bullets.
var backtickField = regexp.MustCompile("`([A-Za-z_][A-Za-z0-9_]*)`")

// minMaxTrigger matches "(+Min/Max)" (case-insensitive, spaces tolerated)
// immediately after a field backtick. Its presence expands the preceding
// field into 3 fields: base, baseMin, baseMax.
var minMaxTrigger = regexp.MustCompile(`(?i)\(\s*\+\s*min\s*/\s*max\s*\)`)

// requiredInlineMarker detects "requis" or "required" tokens within a field's
// parenthetical qualifier (or as **required** bold markers on the same line).
var requiredInlineMarker = regexp.MustCompile(`(?i)\*?\*?\b(requis|required)\b\*?\*?`)

// Synthesizer is the api-postman concrete implementation.
type Synthesizer struct{}

// init registers this synthesizer with the framework's process-wide
// registry. Consumers enable it via cfg.Angela.Synthesizers.Enabled or
// the --synthesizers flag.
func init() {
	synthesizer.DefaultRegistry.Register(&Synthesizer{})
}

// Name implements synthesizer.Synthesizer.
func (*Synthesizer) Name() string { return synthesizerName }

// Applies gates the synthesizer to feature/api/reference docs that expose
// at least one REST endpoint under an Endpoints-like heading.
func (*Synthesizer) Applies(doc *synthesizer.Doc) bool {
	if doc == nil {
		return false
	}
	switch doc.Meta.Type {
	case docTypeFeature, docTypeAPI, docTypeReference:
	default:
		return false
	}
	if !endpointPattern.MatchString(doc.Body) {
		return false
	}
	if _, _, ok := synthesizer.FuzzyFindSection(doc, endpointsHeadingCandidates); !ok {
		return false
	}
	return true
}

// endpointHit is an internal extraction row produced by detectEndpoints.
type endpointHit struct {
	Method      string
	Path        string
	Description string
	Line        int
	MethodSpan  [2]int // [colStart, colEnd] of the method token in the source line
	PathSpan    [2]int
}

// detectEndpoints scans the endpoints section for every (method, path)
// tuple recognized by endpointPattern. Returns hits in source order.
func detectEndpoints(doc *synthesizer.Doc, section *synthesizer.Section) []endpointHit {
	if section == nil {
		return nil
	}
	var hits []endpointHit
	for lineNo := section.StartLine + 1; lineNo <= section.EndLine && lineNo < len(doc.Lines); lineNo++ {
		line := doc.Lines[lineNo]
		matches := endpointPattern.FindAllStringSubmatchIndex(line, -1)
		for _, m := range matches {
			// m = [full_start, full_end, method_start, method_end, path_start, path_end]
			if len(m) < 6 {
				continue
			}
			method := strings.ToUpper(line[m[2]:m[3]])
			path := line[m[4]:m[5]]
			hits = append(hits, endpointHit{
				Method:      method,
				Path:        path,
				Description: extractTableDescription(line),
				Line:        lineNo,
				MethodSpan:  [2]int{m[2], m[3]},
				PathSpan:    [2]int{m[4], m[5]},
			})
		}
	}
	return hits
}

// extractTableDescription returns the last pipe-delimited cell of a
// markdown table row, or "" when the line is not a table row. Used to
// salvage the human-readable description column.
func extractTableDescription(line string) string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return ""
	}
	parts := strings.Split(strings.TrimSuffix(strings.TrimPrefix(trimmed, "|"), "|"), "|")
	if len(parts) < 3 {
		return ""
	}
	return strings.TrimSpace(parts[len(parts)-1])
}

// fieldHit is one extracted field with enough context to emit evidence.
//
// Required and RequiredSpan are intentionally set together via
// markRequired(): a field whose Required flag is true MUST also carry a
// non-zero RequiredSpan pointing at the literal marker in the source.
// This pairing is enforced by markRequired and validated at evidence
// emission time (code review finding #6).
type fieldHit struct {
	Name     string
	Line     int
	ColStart int
	ColEnd   int
	Required bool

	// RequiredSpan points at the "requis"/"required"/etc. marker when
	// Required is true. Kept to emit a dedicated evidence for the
	// requiredness claim (I4 - every decision traces to a literal span).
	// When Required is true, RequiredSpan[0] MUST be > 0.
	RequiredSpan   [3]int // [line, colStart, colEnd]
	RequiredToken  string
	IsMinMaxBase   bool    // this field came from a (+Min/Max) expansion's base
	ExpandedParent string  // for Min/Max-expanded fields, the base name
	TriggerSpan    [3]int  // [line, colStart, colEnd] of (+Min/Max) trigger for expanded fields
	TriggerToken   string  // literal trigger text, e.g. "(+Min/Max)"
}

// markRequired sets Required=true and records the marker span atomically.
// Callers MUST use this helper rather than setting Required and
// RequiredSpan separately - the pairing invariant is what lets evidence
// collection trust the span without extra defensive branches.
func (f *fieldHit) markRequired(line int, colStart, colEnd int, token string) {
	if line <= 0 || colEnd <= colStart {
		// Invalid span - refuse to mark required. Failing silently here
		// would violate I4 at evidence emission time; callers pass
		// regex-derived spans which are always positive on a real match,
		// so hitting this branch indicates a bug in the extraction
		// strategy, not a legitimate runtime state.
		return
	}
	f.Required = true
	f.RequiredSpan = [3]int{line, colStart, colEnd}
	f.RequiredToken = token
}

// detectFields walks the filters section extracting fields via the three
// strategies declared in AC-4: table column, inline-code enumeration,
// bullet list. Prose-only sections emit zero hits (caller adds a warning
// downstream if needed).
//
// When section is nil (no filters heading found), the function falls back
// to a document-wide table scan: any table whose first-column header is
// "champ" / "field" / "param" / "filter" is consumed. This handles docs
// where the field reference lives under a shared "DTO" heading rather
// than under the endpoint's Filters heading.
func detectFields(doc *synthesizer.Doc, section *synthesizer.Section) []fieldHit {
	if section != nil {
		if hits := fieldsFromTable(doc, section); len(hits) > 0 {
			return hits
		}
		if hits := fieldsFromInlineEnumeration(doc, section); len(hits) > 0 {
			return hits
		}
		if hits := fieldsFromBulletList(doc, section); len(hits) > 0 {
			return hits
		}
	}
	// Section-less fallback: scan the whole doc for any eligible table.
	return fieldsFromTable(doc, nil)
}

// tableFieldColumnHeaders lists the column titles (normalized) that
// identify a table's "field name" column. Case- and accent-tolerant via
// explicit listing of French and English variants.
var tableFieldColumnHeaders = map[string]struct{}{
	"champ":     {},
	"champs":    {},
	"field":     {},
	"fields":    {},
	"parameter": {},
	"param":     {},
	"params":    {},
	"filtre":    {},
	"filter":    {},
	"filters":   {},
	"attribut":  {},
	"attribute": {},
}

// fieldsFromTable extracts fields from a markdown table inside section
// (or anywhere in the doc when section is nil). The table must have a
// header row whose first column matches tableFieldColumnHeaders. Each
// subsequent row contributes one field; backticked names are preferred
// over bare names. When a column labeled "requis" / "required" exists,
// a truthy cell (✅, ✓, yes, oui, true) flips the Required flag.
func fieldsFromTable(doc *synthesizer.Doc, section *synthesizer.Section) []fieldHit {
	startLn, endLn := 1, len(doc.Lines)-1
	if section != nil {
		startLn = section.StartLine + 1
		if section.EndLine > 0 && section.EndLine < len(doc.Lines) {
			endLn = section.EndLine
		}
	}

	var hits []fieldHit
	for ln := startLn; ln <= endLn && ln < len(doc.Lines); ln++ {
		header := doc.Lines[ln]
		cells := splitMarkdownTableRow(header)
		if len(cells) < 2 {
			continue
		}
		firstCol := strings.ToLower(strings.TrimSpace(stripBackticks(cells[0])))
		if _, ok := tableFieldColumnHeaders[firstCol]; !ok {
			continue
		}
		// Locate the "required" column, if any.
		requiredCol := -1
		for i, h := range cells {
			norm := strings.ToLower(strings.TrimSpace(stripBackticks(h)))
			if norm == "requis" || norm == "required" {
				requiredCol = i
				break
			}
		}
		// Skip the separator row "|---|---|".
		sepLn := ln + 1
		if sepLn >= len(doc.Lines) || !isMarkdownTableSeparator(doc.Lines[sepLn]) {
			continue
		}
		// Consume rows until a blank line or non-table row.
		for rowLn := sepLn + 1; rowLn <= endLn && rowLn < len(doc.Lines); rowLn++ {
			row := doc.Lines[rowLn]
			if strings.TrimSpace(row) == "" {
				break
			}
			rowCells := splitMarkdownTableRow(row)
			if len(rowCells) < 1 {
				break
			}
			name, colStart, colEnd := extractFieldNameFromCell(row, rowCells[0])
			if name == "" {
				continue
			}
			hit := fieldHit{
				Name:     name,
				Line:     rowLn,
				ColStart: colStart,
				ColEnd:   colEnd,
			}
			if requiredCol >= 0 && requiredCol < len(rowCells) {
				if markerIsTruthy(rowCells[requiredCol]) {
					span := locateRequiredMarker(row, rowCells[requiredCol])
					if span[1] > span[0] {
						hit.markRequired(rowLn, span[0], span[1], row[span[0]:span[1]])
					}
				}
			}
			hits = append(hits, hit)
			// Min/Max expansion for table-derived fields: the name itself
			// carries "Min/Max" as a slash-separated token ("creditAmountMin/Max").
			hits = expandSlashMinMax(hits, row, rowLn)
		}
		ln = endLn // don't look for another table in the same section
	}
	return hits
}

// splitMarkdownTableRow splits a "| a | b | c |" row into ["a", "b", "c"].
// Returns nil for non-table lines.
func splitMarkdownTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil
	}
	inner := trimmed[1 : len(trimmed)-1]
	parts := strings.Split(inner, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.TrimSpace(p))
	}
	return out
}

// isMarkdownTableSeparator reports whether line is the "|---|---|" divider.
func isMarkdownTableSeparator(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") {
		return false
	}
	for _, r := range trimmed {
		switch r {
		case '|', '-', ':', ' ':
			continue
		default:
			return false
		}
	}
	return strings.Contains(trimmed, "-")
}

// stripBackticks removes leading/trailing backticks from a cell token.
func stripBackticks(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "`")
	s = strings.TrimSuffix(s, "`")
	return s
}

// extractFieldNameFromCell finds the field name within a table cell on the
// declared row. Prefers an inline-code span; falls back to the whole cell
// content. Returns (name, colStart, colEnd) where col* are byte offsets
// into the raw line.
func extractFieldNameFromCell(line, cell string) (string, int, int) {
	// Try backticked span first.
	if m := backtickField.FindStringSubmatchIndex(cell); m != nil {
		name := cell[m[2]:m[3]]
		// Locate absolute position of the name in the raw line.
		absIdx := strings.Index(line, name)
		if absIdx < 0 {
			return "", 0, 0
		}
		return name, absIdx, absIdx + len(name)
	}
	bare := strings.TrimSpace(cell)
	bare = stripBackticks(bare)
	if bare == "" {
		return "", 0, 0
	}
	// Only accept identifier-like names to avoid picking up prose.
	valid := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(?:/[A-Za-z]+)?$`)
	if !valid.MatchString(bare) {
		return "", 0, 0
	}
	absIdx := strings.Index(line, bare)
	if absIdx < 0 {
		return "", 0, 0
	}
	return bare, absIdx, absIdx + len(bare)
}

// markerIsTruthy reports whether a required-column cell signals required.
func markerIsTruthy(cell string) bool {
	norm := strings.ToLower(strings.TrimSpace(cell))
	norm = strings.TrimSpace(stripBackticks(norm))
	switch norm {
	case "✅", "✓", "✔", "✔️", "yes", "y", "oui", "o", "true", "requis", "required", "x":
		return true
	}
	return false
}

// locateRequiredMarker returns the absolute [colStart, colEnd] of the
// truthy marker within the raw line.
func locateRequiredMarker(line, cell string) [2]int {
	trimmed := strings.TrimSpace(cell)
	idx := strings.Index(line, trimmed)
	if idx < 0 {
		return [2]int{0, 0}
	}
	return [2]int{idx, idx + len(trimmed)}
}

// expandSlashMinMax handles the "foo/Bar" naming convention in tables
// (e.g. "creditAmountMin/Max" -> creditAmountMin + creditAmountMax).
// The slash is a literal character in the source cell, so the expansion
// conforms to I4 (every emitted field has a literal anchor).
func expandSlashMinMax(hits []fieldHit, line string, lineNo int) []fieldHit {
	if len(hits) == 0 {
		return hits
	}
	last := hits[len(hits)-1]
	if !strings.Contains(last.Name, "/") {
		return hits
	}
	slash := strings.Index(last.Name, "/")
	base := last.Name[:slash]
	suffix := last.Name[slash+1:]
	if base == "" || suffix == "" {
		return hits
	}
	// Replace the raw "<base>/<suffix>" hit with two separate fields:
	// <base> with the suffix stripped off, then <base stem>+<suffix>.
	// The base stem is base with its trailing suffix (e.g. "Min") removed
	// when base ends with a known affix; otherwise we just use base.
	stem := base
	trim := map[string]struct{}{"Min": {}, "Max": {}, "min": {}, "max": {}}
	for affix := range trim {
		if strings.HasSuffix(base, affix) {
			stem = strings.TrimSuffix(base, affix)
			break
		}
	}
	_ = stem // stem currently unused; future use: emit the base field too

	// Rewrite the last hit to <base>.
	hits[len(hits)-1] = fieldHit{
		Name:     base,
		Line:     last.Line,
		ColStart: last.ColStart,
		ColEnd:   last.ColEnd,
		Required: last.Required,
	}
	// Append <stem><suffix> as a second field using the same source span.
	var secondName string
	if strings.HasSuffix(base, "Min") {
		secondName = strings.TrimSuffix(base, "Min") + suffix
	} else if strings.HasSuffix(base, "Max") {
		secondName = strings.TrimSuffix(base, "Max") + suffix
	} else {
		secondName = base + suffix
	}
	hits = append(hits, fieldHit{
		Name:         secondName,
		Line:         last.Line,
		ColStart:     last.ColStart,
		ColEnd:       last.ColEnd,
		Required:     last.Required,
		IsMinMaxBase: true,
		TriggerSpan:  [3]int{lineNo, last.ColStart, last.ColEnd},
		TriggerToken: line[last.ColStart:last.ColEnd],
	})
	return hits
}

func fieldsFromInlineEnumeration(doc *synthesizer.Doc, section *synthesizer.Section) []fieldHit {
	var hits []fieldHit
	for lineNo := section.StartLine + 1; lineNo <= section.EndLine && lineNo < len(doc.Lines); lineNo++ {
		line := doc.Lines[lineNo]
		if strings.HasPrefix(strings.TrimSpace(line), "-") {
			continue // bullets are strategy 3
		}
		matches := backtickField.FindAllStringSubmatchIndex(line, -1)
		if len(matches) == 0 {
			continue
		}
		for _, m := range matches {
			name := line[m[2]:m[3]]
			hit := fieldHit{
				Name:     name,
				Line:     lineNo,
				ColStart: m[2],
				ColEnd:   m[3],
			}

			// Look for a nearby parenthetical qualifier after the name.
			tail := line[m[1]:]
			tail = strings.TrimLeft(tail, " \t")
			if strings.HasPrefix(tail, "(") {
				if close := strings.Index(tail, ")"); close > 0 {
					qual := tail[:close+1]
					if reqMatch := requiredInlineMarker.FindStringIndex(qual); reqMatch != nil {
						absStart := m[1] + reqMatch[0]
						absEnd := m[1] + reqMatch[1]
						hit.markRequired(lineNo, absStart, absEnd, line[absStart:absEnd])
					}
				}
			}
			hits = append(hits, hit)
		}
		hits = expandMinMaxOnLine(hits, line, lineNo)
	}
	return hits
}

// expandMinMaxOnLine scans the provided hits and, for each field whose
// trailing context contains "(+Min/Max)", appends two additional synthetic
// fields (<name>Min, <name>Max) carrying evidence pointing at the trigger
// tokens.
//
// Regex-returned indices are bounds-checked defensively before slicing -
// Go's regexp package guarantees valid indices on successful matches, but
// encoding future changes (e.g., switching to a different regex engine)
// could invalidate that guarantee. The explicit guard keeps the contract
// tests deterministic (code review finding #4).
func expandMinMaxOnLine(hits []fieldHit, line string, lineNo int) []fieldHit {
	triggerMatches := minMaxTrigger.FindAllStringIndex(line, -1)
	if len(triggerMatches) == 0 {
		return hits
	}
	lineLen := len(line)
	expanded := make([]fieldHit, 0, len(hits))
	consumedTriggers := make(map[int]bool) // triggerMatch start idx
	for i := range hits {
		h := hits[i]
		// find the earliest trigger that appears after this hit's end
		// AND before the next hit on the same line.
		nextColStart := lineLen
		for _, other := range hits {
			if other.Line == lineNo && other.ColStart > h.ColEnd && other.ColStart < nextColStart {
				nextColStart = other.ColStart
			}
		}
		triggerIdx := -1
		for j, tm := range triggerMatches {
			if len(tm) != 2 || tm[0] < 0 || tm[1] > lineLen || tm[0] > tm[1] {
				continue // malformed match, skip defensively
			}
			if consumedTriggers[tm[0]] {
				continue
			}
			if tm[0] >= h.ColEnd && tm[1] <= nextColStart {
				triggerIdx = j
				break
			}
		}
		expanded = append(expanded, h)
		if triggerIdx >= 0 {
			tm := triggerMatches[triggerIdx]
			consumedTriggers[tm[0]] = true
			for _, suffix := range []string{"Min", "Max"} {
				expanded = append(expanded, fieldHit{
					Name:           h.Name + suffix,
					Line:           h.Line,
					ColStart:       h.ColStart,
					ColEnd:         h.ColEnd,
					Required:       false,
					IsMinMaxBase:   true,
					ExpandedParent: h.Name,
					TriggerSpan:    [3]int{lineNo, tm[0], tm[1]},
					TriggerToken:   line[tm[0]:tm[1]],
				})
			}
		}
	}
	return expanded
}

func fieldsFromBulletList(doc *synthesizer.Doc, section *synthesizer.Section) []fieldHit {
	var hits []fieldHit
	for lineNo := section.StartLine + 1; lineNo <= section.EndLine && lineNo < len(doc.Lines); lineNo++ {
		line := doc.Lines[lineNo]
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "-") {
			continue
		}
		match := backtickField.FindStringSubmatchIndex(line)
		if match == nil {
			continue
		}
		name := line[match[2]:match[3]]
		hit := fieldHit{
			Name:     name,
			Line:     lineNo,
			ColStart: match[2],
			ColEnd:   match[3],
		}
		if reqMatch := requiredInlineMarker.FindStringIndex(line); reqMatch != nil {
			hit.markRequired(lineNo, reqMatch[0], reqMatch[1], line[reqMatch[0]:reqMatch[1]])
		}
		hits = append(hits, hit)
	}
	return hits
}

// detectServerInjected parses the Security section for fields explicitly
// declared as server-injected. Returns (fields, hasSection).
//
// When hasSection is false, the caller applies degraded mode (well-known
// filter + missing-security-section warning).
func detectServerInjected(doc *synthesizer.Doc) (map[string]struct{}, bool) {
	section, _, ok := synthesizer.FuzzyFindSection(doc, securityHeadingCandidates)
	if !ok {
		return nil, false
	}
	names := make(map[string]struct{})
	for lineNo := section.StartLine + 1; lineNo <= section.EndLine && lineNo < len(doc.Lines); lineNo++ {
		line := doc.Lines[lineNo]
		if !containsInjectedKeyword(line) {
			continue
		}
		for _, m := range backtickField.FindAllStringSubmatch(line, -1) {
			if len(m) >= 2 {
				names[m[1]] = struct{}{}
			}
		}
	}
	return names, true
}

func containsInjectedKeyword(line string) bool {
	lower := strings.ToLower(line)
	keywords := []string{
		"injecté", "injecte", "injected", "server-side", "côté serveur", "cote serveur",
		"jamais depuis le client", "never from the client",
	}
	for _, k := range keywords {
		if strings.Contains(lower, strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// Detect implements synthesizer.Synthesizer. Returns one Candidate per
// detected endpoint - Synthesize later emits TWO Blocks per Candidate
// (minimal + full variants) per AC-11.
func (*Synthesizer) Detect(doc *synthesizer.Doc) ([]synthesizer.Candidate, error) {
	endpointsSection, _, _ := synthesizer.FuzzyFindSection(doc, endpointsHeadingCandidates)
	endpoints := detectEndpoints(doc, endpointsSection)
	if len(endpoints) == 0 {
		return nil, nil
	}

	filtersSection, _, _ := synthesizer.FuzzyFindSection(doc, filtersHeadingCandidates)
	fields := detectFields(doc, filtersSection)

	serverInjected, hasSecurity := detectServerInjected(doc)

	candidates := make([]synthesizer.Candidate, 0, len(endpoints))
	for _, ep := range endpoints {
		candidates = append(candidates, synthesizer.Candidate{
			Key: fmt.Sprintf("%s %s", ep.Method, ep.Path),
			Anchor: synthesizer.Evidence{
				Field:    synthesizer.MetaFieldEndpoint,
				File:     doc.Path,
				Line:     ep.Line,
				ColStart: ep.MethodSpan[0],
				ColEnd:   ep.PathSpan[1],
				Snippet:  doc.Lines[ep.Line][ep.MethodSpan[0]:ep.PathSpan[1]],
				Rule:     "literal",
			},
			Extra: map[string]any{
				"endpoint":         ep,
				"fields":           fields,
				"endpointsHeading": headingOf(endpointsSection),
				"serverInjected":   serverInjected,
				"hasSecurity":      hasSecurity,
			},
		})
	}
	return candidates, nil
}

func headingOf(section *synthesizer.Section) string {
	if section == nil {
		return ""
	}
	return section.Heading
}

// Synthesize emits the minimal + full variant Blocks for a single endpoint
// candidate. See synthesize.go for the heavy lifting.
func (s *Synthesizer) Synthesize(c synthesizer.Candidate, cfg synthesizer.Config) (synthesizer.Block, []synthesizer.Evidence, []synthesizer.Warning, error) {
	return buildBlocksForCandidate(c, cfg)
}

// sortedFieldNames is a helper used by tests and internal audits.
func sortedFieldNames(fields []fieldHit) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.Name
	}
	sort.Strings(names)
	return names
}
