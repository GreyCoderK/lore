// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// ArbitrationRule encodes the non-interactive policy for resolving
// duplicate sections in AI body output. Set from the --arbitrate-rule
// CLI flag.
type ArbitrationRule string

const (
	// RuleNone means the flag was not set. In TTY the pipeline will
	// prompt per group; in non-TTY the pipeline refuses.
	RuleNone ArbitrationRule = ""

	RuleFirst  ArbitrationRule = "first"
	RuleSecond ArbitrationRule = "second"
	RuleBoth   ArbitrationRule = "both"
	RuleAbort  ArbitrationRule = "abort"
)

// ValidArbitrationRule reports whether a user-supplied string is a
// recognized rule value. Used by the CLI flag validator.
func ValidArbitrationRule(s string) bool {
	switch ArbitrationRule(s) {
	case RuleNone, RuleFirst, RuleSecond, RuleBoth, RuleAbort:
		return true
	}
	return false
}

// ArbitrateChoice is the per-group resolution a user or rule settled on.
type ArbitrateChoice int

const (
	// ChoiceFirst keeps only the earliest occurrence of the heading.
	ChoiceFirst ArbitrateChoice = iota
	// ChoiceSecond keeps only the second occurrence; falls back to
	// ChoiceFirst if the group has a single occurrence.
	ChoiceSecond
	// ChoiceBoth keeps all occurrences in source order. No merging,
	// no combining — each block is preserved verbatim.
	ChoiceBoth
	// ChoiceAbort is a sentinel returned by promptPerGroup when the
	// user selects [a]. arbitrateDuplicates converts it to
	// ErrArbitrateAbort before returning.
	ChoiceAbort
)

// Resolution pairs a heading with the chosen action. The slice returned
// by arbitrateDuplicates has one entry per input DupGroup, in the same
// order.
type Resolution struct {
	Heading string
	Choice  ArbitrateChoice
}

// ArbitrateOptions carries optional UI settings for the interactive
// prompt. Zero value is safe.
type ArbitrateOptions struct {
	// Verbose: when true, preview 8 lines per occurrence instead of 3.
	Verbose bool
}

// Typed errors returned by arbitrateDuplicates. Callers use errors.Is
// to distinguish.
var (
	// ErrArbitrateAbort is returned when the user selected [a] in TTY
	// or when --arbitrate-rule=abort was supplied.
	ErrArbitrateAbort = errors.New("polish: arbitration aborted")

	// ErrArbitrateRefused is returned when duplicates are present but
	// the caller is non-interactive AND no --arbitrate-rule was set.
	// The pipeline surfaces this as a neutral stderr message pointing
	// at the flag (invariant I27).
	ErrArbitrateRefused = errors.New("polish: duplicate sections need TTY or --arbitrate-rule")
)

// arbitrateDuplicates routes each DupGroup to the right resolution
// source:
//
//   - rule == RuleAbort           → ErrArbitrateAbort (no prompt, no apply)
//   - rule in {first,second,both} → deterministic per-group resolution
//   - rule == RuleNone, isTTY     → interactive prompt per group
//   - rule == RuleNone, !isTTY    → ErrArbitrateRefused
//
// The function does NOT mutate the body; it only returns resolutions.
// Callers pass the result to applyDuplicateResolutions to produce the
// post-arbitration body.
func arbitrateDuplicates(
	groups []DupGroup,
	body []byte,
	rule ArbitrationRule,
	isTTY bool,
	streams domain.IOStreams,
	opts ArbitrateOptions,
) ([]Resolution, error) {
	if len(groups) == 0 {
		return nil, nil
	}
	if rule == RuleAbort {
		return nil, ErrArbitrateAbort
	}
	if rule != RuleNone {
		return applyRule(rule, groups), nil
	}
	if !isTTY {
		return nil, ErrArbitrateRefused
	}
	return promptPerGroup(groups, body, streams, opts)
}

// applyRule maps a deterministic rule onto every duplicate group.
// RuleAbort is handled by the caller — it does not reach here.
func applyRule(rule ArbitrationRule, groups []DupGroup) []Resolution {
	var choice ArbitrateChoice
	switch rule {
	case RuleFirst:
		choice = ChoiceFirst
	case RuleSecond:
		choice = ChoiceSecond
	case RuleBoth:
		choice = ChoiceBoth
	default:
		// Should not happen — caller filters RuleNone/RuleAbort.
		choice = ChoiceFirst
	}
	out := make([]Resolution, len(groups))
	for i, g := range groups {
		out[i] = Resolution{Heading: g.Heading, Choice: choice}
	}
	return out
}

// promptPerGroup runs the interactive prompt loop, one group at a time.
// Each group blocks until the user picks a valid option; [d] re-shows
// full content then re-prompts.
func promptPerGroup(
	groups []DupGroup,
	body []byte,
	streams domain.IOStreams,
	opts ArbitrateOptions,
) ([]Resolution, error) {
	reader := bufio.NewReader(streams.In)
	out := make([]Resolution, 0, len(groups))

	for gi, g := range groups {
		// Header.
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Angela.ArbitrateGroupHeader,
			gi+1, len(groups), g.Heading, len(g.Occurrences))

		// Preview each occurrence.
		previewLines := 3
		if opts.Verbose {
			previewLines = 8
		}
		renderOccurrencePreviews(streams.Err, body, g.Occurrences, previewLines)

		// Options line.
		renderOptions(streams.Err, len(g.Occurrences))

		// Prompt loop: re-prompt on invalid input, loop through [d].
		for {
			_, _ = fmt.Fprint(streams.Err, i18n.T().Angela.ArbitratePrompt)
			input, err := reader.ReadString('\n')
			if err != nil {
				// EOF on non-TTY-like input — treat as abort for safety.
				return nil, ErrArbitrateAbort
			}
			choice, ok := parsePromptInput(strings.TrimSpace(input), len(g.Occurrences))
			if !ok {
				_, _ = fmt.Fprintln(streams.Err, i18n.T().Angela.ArbitrateInvalidChoice)
				continue
			}
			switch choice {
			case ChoiceAbort:
				return nil, ErrArbitrateAbort
			case -1: // [d] — show full content, re-prompt
				renderFullContent(streams.Err, body, g.Occurrences)
				renderOptions(streams.Err, len(g.Occurrences))
				continue
			default:
				out = append(out, Resolution{Heading: g.Heading, Choice: choice})
				goto nextGroup
			}
		}
	nextGroup:
	}
	return out, nil
}

// parsePromptInput decodes the trimmed user input. It returns the
// ArbitrateChoice and true on a valid selection, or -1 and false
// otherwise. "d" is returned as -1 so the caller re-shows full content.
func parsePromptInput(in string, numOccurrences int) (ArbitrateChoice, bool) {
	switch strings.ToLower(in) {
	case "1":
		return ChoiceFirst, true
	case "2":
		// Fall back to ChoiceFirst if only one occurrence (defensive —
		// should not happen since we only prompt when len >= 2).
		if numOccurrences < 2 {
			return ChoiceFirst, true
		}
		return ChoiceSecond, true
	case "b":
		return ChoiceBoth, true
	case "a":
		return ChoiceAbort, true
	case "d":
		return -1, true
	}
	return 0, false
}

// renderOccurrencePreviews prints a compact summary of each occurrence:
// one indented block per occurrence showing the first N body lines plus
// a word count and "lines shown / total" suffix.
func renderOccurrencePreviews(w interface {
	Write(p []byte) (int, error)
}, body []byte, occs []SectionLocation, previewLines int) {
	for i, occ := range occs {
		totalLines, preview := extractOccurrencePreview(body, occ, previewLines)
		_, _ = fmt.Fprintf(w, i18n.T().Angela.ArbitratePreviewLine, i+1, occ.Line, occ.Words)
		for _, line := range preview {
			_, _ = fmt.Fprintf(w, "      %s\n", line)
		}
		if totalLines > previewLines {
			_, _ = fmt.Fprintf(w, i18n.T().Angela.ArbitratePreviewTruncated, previewLines, totalLines)
		}
		_, _ = fmt.Fprintln(w)
	}
}

// extractOccurrencePreview returns the total body-line count for the
// occurrence and the first previewLines non-empty lines (skipping the
// heading line itself).
func extractOccurrencePreview(body []byte, occ SectionLocation, previewLines int) (int, []string) {
	// Slice the occurrence body (excluding the heading line).
	region := body[occ.ByteStart:occ.ByteEnd]
	nl := bytes.IndexByte(region, '\n')
	var after []byte
	if nl < 0 {
		after = nil
	} else {
		after = region[nl+1:]
	}
	allLines := strings.Split(strings.TrimRight(string(after), "\n"), "\n")
	// Pick first previewLines non-empty lines (keeps preview useful
	// when the body starts with blank lines).
	preview := make([]string, 0, previewLines)
	for _, line := range allLines {
		if len(preview) >= previewLines {
			break
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		preview = append(preview, line)
	}
	return len(allLines), preview
}

// renderOptions prints the option line appropriate for the group size.
// For 2 occurrences: "keep first / keep second / keep both".
// For 3+ occurrences: "keep first / keep second / keep all".
func renderOptions(w interface {
	Write(p []byte) (int, error)
}, numOccurrences int) {
	bothLabel := i18n.T().Angela.ArbitrateOptKeepBoth
	if numOccurrences > 2 {
		bothLabel = fmt.Sprintf(i18n.T().Angela.ArbitrateOptKeepAll, numOccurrences)
	}
	_, _ = fmt.Fprintf(w, i18n.T().Angela.ArbitrateOptsLine, bothLabel)
}

// renderFullContent dumps every occurrence of the group verbatim,
// separated by a short banner. Used when the user selects [d].
func renderFullContent(w interface {
	Write(p []byte) (int, error)
}, body []byte, occs []SectionLocation) {
	for i, occ := range occs {
		_, _ = fmt.Fprintf(w, i18n.T().Angela.ArbitrateOccurrenceBanner, i+1, len(occs), occ.Line)
		_, _ = fmt.Fprint(w, string(body[occ.ByteStart:occ.ByteEnd]))
	}
	_, _ = fmt.Fprintln(w, "---")
}

// applyDuplicateResolutions produces a new body with the resolutions
// applied. For every group, occurrences not kept by the chosen
// ArbitrateChoice are spliced out at their byte ranges; kept
// occurrences remain at their original positions in source order.
//
// Sections outside any DupGroup are untouched — the function only
// splices out specifically-targeted byte ranges.
//
// Preconditions:
//   - len(resolutions) == len(groups)
//   - resolutions[i].Heading == groups[i].Heading
//   - No Resolution has Choice == ChoiceAbort (caller filters those)
func applyDuplicateResolutions(body []byte, groups []DupGroup, resolutions []Resolution) []byte {
	if len(groups) == 0 || len(resolutions) == 0 {
		return body
	}
	type dropRange struct{ start, end int }
	var drops []dropRange

	for i, g := range groups {
		r := resolutions[i]
		switch r.Choice {
		case ChoiceFirst:
			for _, occ := range g.Occurrences[1:] {
				drops = append(drops, dropRange{occ.ByteStart, occ.ByteEnd})
			}
		case ChoiceSecond:
			if len(g.Occurrences) >= 2 {
				drops = append(drops, dropRange{g.Occurrences[0].ByteStart, g.Occurrences[0].ByteEnd})
				for _, occ := range g.Occurrences[2:] {
					drops = append(drops, dropRange{occ.ByteStart, occ.ByteEnd})
				}
			}
			// len == 1 (shouldn't happen for a DupGroup): nothing to drop.
		case ChoiceBoth:
			// Keep all — no drops.
		case ChoiceAbort:
			// Caller is expected to convert ChoiceAbort into an early
			// exit before we get here. Treat as no-op defensively.
		}
	}

	if len(drops) == 0 {
		return body
	}

	// Sort by start offset, merge overlaps defensively, then splice.
	sort.Slice(drops, func(i, j int) bool { return drops[i].start < drops[j].start })

	var buf bytes.Buffer
	cursor := 0
	for _, d := range drops {
		if d.start < cursor {
			// Overlapping range — shouldn't happen with our data, but
			// guard against it rather than producing a corrupt body.
			if d.end > cursor {
				cursor = d.end
			}
			continue
		}
		buf.Write(body[cursor:d.start])
		cursor = d.end
	}
	buf.Write(body[cursor:])
	return buf.Bytes()
}
