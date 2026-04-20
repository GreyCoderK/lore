// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bytes"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// SanitizeReport describes the structural events encountered while
// processing an AI polish response: leaked frontmatter stripped, and
// duplicate sections detected + arbitrated. Callers wire this into a
// LogEntry for audit (invariant I30) and into stderr messages when
// --verbose is set.
type SanitizeReport struct {
	// LeakedFM records whether an AI-emitted `---\n...\n---\n` block
	// was stripped from the head of the body. Zero value means no
	// strip happened.
	LeakedFM StripInfo

	// DupGroups holds the duplicate-section groups that were detected
	// (before any arbitration). Empty when no duplicates.
	DupGroups []DupGroup

	// Resolutions holds the per-group resolutions that were applied.
	// Same length and order as DupGroups on success. Populated when
	// arbitration succeeded; empty on arbitration abort/refuse.
	Resolutions []Resolution

	// Source records where the arbitration decision came from:
	// "user" (TTY prompt), "rule" (--arbitrate-rule flag), or "" when
	// no duplicates were present.
	Source string
}

// SanitizeAIOutput runs the sanitize + arbitrate pipeline on the raw
// AI response. This is the single entry point the polish command
// calls after a provider response lands and before the diff/write
// stage.
//
// Responsibilities, in order:
//   1. If the AI cheated and emitted a full document (leading `---\n`
//      block), extract the body via storage.ExtractFrontmatter — the
//      leaked front matter bytes are discarded. Invariant I26.
//   2. Defensively run stripLeakedFrontmatter one more time in case
//      the AI wrote an unparseable `---` prefix that ExtractFrontmatter
//      rejected but that stripLeakedFrontmatter's simpler scan can
//      still match.
//   3. Detect duplicate sections via detectDuplicateSections. I27.
//   4. Arbitrate per the given rule/TTY/streams. User abort or non-TTY
//      refusal surface as typed errors (ErrArbitrateAbort /
//      ErrArbitrateRefused).
//   5. Apply resolutions to produce a cleaned body.
//
// The returned SanitizeReport is always populated with the findings
// that were observed, including Source (pre-populated before
// arbitration runs) so callers can log the outcome regardless of
// whether arbitration succeeded, aborted, or was refused.
func SanitizeAIOutput(
	rawAIOutput []byte,
	rule ArbitrationRule,
	isTTY bool,
	streams domain.IOStreams,
	opts ArbitrateOptions,
) (cleanedBody []byte, report SanitizeReport, err error) {
	body, report := detectAIStructuralIssues(rawAIOutput)

	// Arbitrate (only if there are duplicates to resolve).
	if len(report.DupGroups) == 0 {
		return body, report, nil
	}
	// Pre-populate Source so callers can include it in a log entry
	// even if arbitration returns ErrArbitrateAbort/ErrArbitrateRefused.
	if rule != RuleNone {
		report.Source = "rule"
	} else {
		report.Source = "user"
	}
	resolutions, arbErr := arbitrateDuplicates(report.DupGroups, body, rule, isTTY, streams, opts)
	if arbErr != nil {
		return nil, report, arbErr
	}
	report.Resolutions = resolutions

	// Apply resolutions.
	cleaned := applyDuplicateResolutions(body, report.DupGroups, resolutions)
	return cleaned, report, nil
}

// DetectStructuralIssues returns the findings (leaked FM, duplicate
// sections) in an AI polish response WITHOUT running arbitration.
// Used by the dry-run path to report findings on stderr while
// preserving its zero-side-effect contract (AC-14): no prompt, no
// write, no polish.log entry.
//
// The returned body is the input with any leaked `---\n...\n---\n`
// block stripped from the head — useful for dry-run stdout so that
// piped tools see a clean body rather than a mixed full-doc payload.
func DetectStructuralIssues(rawAIOutput []byte) ([]byte, SanitizeReport) {
	return detectAIStructuralIssues(rawAIOutput)
}

// detectAIStructuralIssues runs the detection pipeline (strip leaked
// FM + find duplicate sections). Shared by SanitizeAIOutput (full
// pipeline) and DetectStructuralIssues (dry-run detection only).
func detectAIStructuralIssues(rawAIOutput []byte) ([]byte, SanitizeReport) {
	var report SanitizeReport
	body := rawAIOutput

	// AI emitted a full doc — extract body and discard any leaked
	// frontmatter bytes.
	if bytes.HasPrefix(body, []byte("---\n")) {
		_, bodyBytes, xerr := storage.ExtractFrontmatter(body)
		if xerr == nil {
			stripped := len(body) - len(bodyBytes)
			body = bodyBytes
			report.LeakedFM = StripInfo{Stripped: true, Bytes: stripped, Line: 1}
		}
	}

	// Defensive re-strip — covers partial leaks that ExtractFrontmatter
	// rejects (malformed YAML between delimiters) but stripLeakedFrontmatter
	// still matches because it only requires well-formed delimiters.
	stripped, stripInfo := stripLeakedFrontmatter(body)
	if stripInfo.Stripped {
		body = stripped
		report.LeakedFM.Stripped = true
		report.LeakedFM.Bytes += stripInfo.Bytes
		if report.LeakedFM.Line == 0 {
			report.LeakedFM.Line = stripInfo.Line
		}
	}

	report.DupGroups = detectDuplicateSections(body)
	return body, report
}

// StripInfo reports whether a leaked frontmatter block was stripped from
// an AI-produced body, and if so how many bytes were removed. When
// Stripped is false, the remaining fields carry no meaning.
//
// Used by the polish pipeline to honor invariant I26 (leaked `---`
// blocks stripped from AI body before write) — silent by default,
// visible under --verbose (see story 8-21 AC-4).
type StripInfo struct {
	Stripped bool
	Bytes    int // number of bytes removed (the leaked block length)
	Line     int // 1-based starting line of the stripped block (always 1 for prefix strips)
}

// stripLeakedFrontmatter removes a leading `---\nYAML\n---\n` block
// from the AI body output if one is present. This is a defensive pass:
// the AI is instructed NOT to emit frontmatter (invariant I25), but
// some providers occasionally re-echo it regardless. A clean strip at
// the exact start of the body is deterministic and always safe.
//
// Only a block that begins at byte offset 0 is stripped. An unclosed
// leaked block (opening `---\n` without a matching closing `\n---\n`)
// is NOT stripped — it is left as-is so subsequent layers can surface
// it if it matters. An empty `---\n---\n` sentinel at the start is
// also not stripped (it is either a user artifact or something the
// AI would rarely produce).
//
// Code fences are not consulted here: the strip only applies when the
// body begins immediately with `---\n`, so in-body `---` sequences
// (e.g. Markdown horizontal rules, YAML inside fenced blocks) are
// never affected.
func stripLeakedFrontmatter(body []byte) ([]byte, StripInfo) {
	const open = "---\n"
	const close = "\n---\n"

	if !bytes.HasPrefix(body, []byte(open)) {
		return body, StripInfo{}
	}
	rest := body[len(open):]

	// "---\n---\n..." at the exact start — empty FM sentinel. Do not
	// touch: this is either a user artifact (rare) or noise we don't
	// want to auto-fix silently.
	if bytes.HasPrefix(rest, []byte(open)) {
		return body, StripInfo{}
	}

	idx := bytes.Index(rest, []byte(close))
	if idx < 0 {
		// Opening delimiter but no close — unclosed leak. Leave alone.
		return body, StripInfo{}
	}

	end := len(open) + idx + len(close)
	return body[end:], StripInfo{
		Stripped: true,
		Bytes:    end,
		Line:     1,
	}
}

// SectionLocation describes where a `## Heading` section sits in a body.
//
// ByteStart / ByteEnd form a half-open range [ByteStart, ByteEnd) over
// the original body bytes: the range begins at the heading line and
// ends at the start of the next heading (or end of body for the last
// section). applyDuplicateResolutions uses these offsets for in-place
// removal without reparsing.
type SectionLocation struct {
	Heading   string // trimmed heading text including the "## " prefix
	Line      int    // 1-based line number of the heading line
	ByteStart int    // inclusive offset in body where the heading line begins
	ByteEnd   int    // exclusive offset where the section ends
	Words     int    // word count of the section body (after the heading line) — for UI preview
}

// DupGroup collects all occurrences of a single heading that appears
// more than once in an AI body. Occurrences are ordered by source
// appearance (first hit first).
//
// Used to drive invariant I27 (duplicate sections trigger arbitration,
// never silent de-dup) in the polish pipeline.
type DupGroup struct {
	Heading     string
	Occurrences []SectionLocation
}

// detectDuplicateSections returns one DupGroup per heading that appears
// two or more times in body. Groups are ordered by the first occurrence
// of each heading in source order.
//
// Code fences are respected: a line like `## Subheading` inside a
// triple-backtick fenced block is NOT counted as a section. This
// mirrors SplitSections' semantics.
func detectDuplicateSections(body []byte) []DupGroup {
	locs := locateHeadings(body)
	byHeading := make(map[string][]SectionLocation, len(locs))
	order := make([]string, 0, len(locs))
	for _, loc := range locs {
		if _, seen := byHeading[loc.Heading]; !seen {
			order = append(order, loc.Heading)
		}
		byHeading[loc.Heading] = append(byHeading[loc.Heading], loc)
	}
	var groups []DupGroup
	for _, h := range order {
		occ := byHeading[h]
		if len(occ) >= 2 {
			groups = append(groups, DupGroup{Heading: h, Occurrences: occ})
		}
	}
	return groups
}

// locateHeadings walks the body once and emits a SectionLocation for
// every `## ` heading it finds outside of fenced code blocks. It does
// not split the body into sections — it only reports where each
// heading sits so that detection and arbitration can work on byte
// offsets rather than reparsed strings.
func locateHeadings(body []byte) []SectionLocation {
	var out []SectionLocation
	inFence := false

	// Track cumulative byte offset as we walk lines. We use a manual
	// scan rather than strings.Split so that offsets line up with the
	// original body bytes including trailing newlines.
	offset := 0
	lineNum := 1

	// Index of the currently-open section entry in `out` whose ByteEnd
	// needs to be set when the next heading (or EOF) is seen.
	openIdx := -1

	closeCurrent := func(endOffset int) {
		if openIdx >= 0 {
			out[openIdx].ByteEnd = endOffset
			openIdx = -1
		}
	}

	for offset <= len(body) {
		// Find end of current line.
		lineEnd := bytes.IndexByte(body[offset:], '\n')
		var line []byte
		var advance int
		if lineEnd < 0 {
			line = body[offset:]
			advance = len(line)
			if advance == 0 {
				break
			}
		} else {
			line = body[offset : offset+lineEnd]
			advance = lineEnd + 1
		}

		trimmed := bytes.TrimSpace(line)

		// Toggle fence state on any line starting with ``` (after
		// trimming indent). Matches SplitSections' convention.
		if bytes.HasPrefix(trimmed, []byte("```")) {
			inFence = !inFence
		} else if !inFence && bytes.HasPrefix(trimmed, []byte("## ")) {
			// New heading encountered — close the previous section at
			// the start of this line.
			closeCurrent(offset)
			out = append(out, SectionLocation{
				Heading:   string(trimmed),
				Line:      lineNum,
				ByteStart: offset,
			})
			openIdx = len(out) - 1
		}

		offset += advance
		lineNum++
		if lineEnd < 0 {
			break
		}
	}

	// Close the last open section at EOF.
	closeCurrent(len(body))

	// Second pass: compute word counts on the body portion of each
	// section (everything after the heading line, before ByteEnd).
	for i := range out {
		bodyStart := out[i].ByteStart
		// Advance past the heading line (up to and including its '\n').
		if nl := bytes.IndexByte(body[bodyStart:], '\n'); nl >= 0 {
			bodyStart += nl + 1
		} else {
			bodyStart = out[i].ByteEnd
		}
		if bodyStart > out[i].ByteEnd {
			bodyStart = out[i].ByteEnd
		}
		out[i].Words = len(strings.Fields(string(body[bodyStart:out[i].ByteEnd])))
	}

	return out
}
