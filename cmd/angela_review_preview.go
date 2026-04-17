// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// previewFormat enumerates the selector values accepted by --format under
// --preview. Keeping the enum here keeps the cmd dispatcher switchable and
// makes the JSON schema test (AC-6) trivial to assert against.
const (
	previewFormatText = "text"
	previewFormatJSON = "json"
)

// reviewPreviewInputs groups the minimum data runReviewPreview needs. Built
// inside RunE (summaries already computed, personas already resolved).
//
// The Summaries + StyleGuide + VHS fields carry the real prompt inputs so
// Preflight accounts for the FULL AI payload (system prompt + audience
// directive + style guide + TOON corpus + VHS signals + persona block).
// Without these, the preview cost systematically under-estimates the real
// run's cost, which breaks the story's "zero financial surprise" promise.
type reviewPreviewInputs struct {
	Summaries      []angela.DocSummary
	StyleGuide     string
	VHSSignals     *angela.VHSSignals
	CorpusBytes    int // retained as a lightweight summary byte count for the report
	CorpusDocCount int
	Model          string
	MaxTokens      int
	Timeout        time.Duration
	Personas       []angela.PersonaProfile // nil if baseline
	PersonaNames   []string                // for reporting, even before resolution
	Audience       string
	Format         string
}

// previewSchemaVersion is the stable schema tag for the --format=json output.
// Increment whenever a field is renamed, removed, or changes semantics.
// Adding a new optional field does not require a bump.
const previewSchemaVersion = "1"

// reviewPreviewReport is the shape both the text and JSON renderers consume.
// Kept flat so the JSON schema is stable and easy to version.
//
// Pointer types carry "unknown" as JSON `null` instead of a `-1` sentinel
// that naive scripts (`jq '.estimated_cost_usd' | awk '{sum+=$1}'`) would
// silently subtract from aggregates. Consumers that want strict presence
// should check `schema_version` first, then for null.
type reviewPreviewReport struct {
	SchemaVersion        string   `json:"schema_version"`
	Mode                 string   `json:"mode"`
	CorpusDocs           int      `json:"corpus_documents"`
	CorpusBytes          int      `json:"corpus_bytes"`
	Model                string   `json:"model"`
	Personas             []string `json:"personas"`
	Audience             string   `json:"audience"`
	EstimatedInputTokens int      `json:"estimated_input_tokens"`
	MaxOutputTokens      int      `json:"max_output_tokens"`
	ContextWindowUsedPct float64  `json:"context_window_used_pct"`
	EstimatedCostUSD     *float64 `json:"estimated_cost_usd"` // null when unknown
	ExpectedSeconds      *int     `json:"expected_seconds"`   // null when unknown
	Warnings             []string `json:"warnings"`
	ShouldAbort          bool     `json:"should_abort"`
	AbortReason          string   `json:"abort_reason,omitempty"`
}

// resolveReviewPersonasForPreview is the preview-mode counterpart to
// decideReviewPersonas. It NEVER prompts the user (preview is non-interactive
// by definition) and NEVER emits an info log.
//
// To avoid drift, it delegates to decideReviewPersonas with isTTY=false so:
//
//   - the "TTY + configured but no flag" branch collapses to personaNonTTYInfo,
//     which this function flattens to baseline (preview never logs the info
//     banner — quick answer, no side output on stderr by design)
//   - every other branch is handled by the shared decision logic
//
// Users who want the cost with config personas activated must pass
// --use-configured-personas (AC-5 + 8-19 AC-12).
func resolveReviewPersonasForPreview(
	cfg *config.Config,
	flagPersonaNames []string,
	flagNoPersonas bool,
	flagUseConfigured bool,
) ([]angela.PersonaProfile, error) {
	decision, err := decideReviewPersonas(cfg, flagPersonaNames, flagNoPersonas, flagUseConfigured, false)
	if err != nil {
		return nil, err
	}
	switch decision.Resolution {
	case personaFromFlag, personaFromConfig:
		return decision.Personas, nil
	case personaBaseline, personaNonTTYInfo:
		// Baseline by design: no prompt, no info log on preview path.
		return nil, nil
	case personaPromptRequired:
		// Unreachable when isTTY=false, but we treat it defensively as baseline
		// so a future regression in decideReviewPersonas cannot leak personas
		// into preview output without an explicit flag.
		return nil, nil
	}
	return nil, nil
}

// runReviewPreview executes the 8-20 preview path. It runs Preflight locally
// (no provider instantiation, no AI call, no side effect) and emits the report
// to stdout in the requested format. Returns an error only for flag-level
// problems (unknown format); a Preflight ShouldAbort is surfaced via the
// report, never via a non-zero exit (preview is informational, see AC-4).
func runReviewPreview(streams domain.IOStreams, cfg *config.Config, in reviewPreviewInputs) error {
	// Resolve the personas into a list of display names for the report. When
	// Personas was already resolved (flag or config), prefer names from
	// PersonaProfile.Name for stability; otherwise fall back to PersonaNames.
	names := in.PersonaNames
	if len(in.Personas) > 0 {
		names = make([]string, 0, len(in.Personas))
		for _, p := range in.Personas {
			names = append(names, p.Name)
		}
	}

	// Build the REAL prompt that would be sent to the AI and pass it to
	// Preflight. This accounts for the system prompt, audience directive,
	// style guide, TOON corpus, VHS signals, and persona block — everything
	// the real run will spend tokens on. Falling back to a corpus-bytes
	// approximation (as the initial implementation did) systematically
	// under-counted the cost by 30-100% on non-trivial reviews.
	var systemPrompt, userContent string
	if len(in.Summaries) > 0 {
		audience := []string{}
		if in.Audience != "" {
			audience = append(audience, in.Audience)
		}
		systemPrompt, userContent = angela.BuildReviewPromptWithVHS(
			in.Summaries, in.StyleGuide, nil, in.VHSSignals, in.Personas, audience...,
		)
	} else {
		// Degenerate case: no corpus available (rare — guarded in RunE).
		// Fall back to the approximation so preview still returns something.
		bytesIn := in.CorpusBytes
		if len(in.Personas) > 0 {
			bytesIn += len(angela.BuildPersonaPrompt(in.Personas))
		}
		userContent = strings.Repeat("x", bytesIn)
	}

	pf := angela.Preflight(userContent, systemPrompt, in.Model, in.MaxTokens, in.Timeout)

	// Wrap unknown values as nil pointers so JSON emits `null` instead of a
	// numeric sentinel (-1 / 0) that downstream consumers would otherwise
	// silently aggregate into their totals.
	var costPtr *float64
	if pf.EstimatedCost >= 0 {
		v := pf.EstimatedCost
		costPtr = &v
	}
	var secondsPtr *int
	if s := expectedSecondsFromPreflight(pf, in.Model); s > 0 {
		secondsPtr = &s
	}

	report := reviewPreviewReport{
		SchemaVersion:        previewSchemaVersion,
		Mode:                 "preview",
		CorpusDocs:           in.CorpusDocCount,
		CorpusBytes:          in.CorpusBytes,
		Model:                in.Model,
		Personas:             names,
		Audience:             in.Audience,
		EstimatedInputTokens: pf.EstimatedInputTokens,
		MaxOutputTokens:      pf.MaxOutputTokens,
		ContextWindowUsedPct: contextWindowPct(pf.EstimatedInputTokens, pf.MaxOutputTokens, in.Model),
		EstimatedCostUSD:     costPtr,
		ExpectedSeconds:      secondsPtr,
		Warnings:             pf.Warnings,
		ShouldAbort:          pf.ShouldAbort,
		AbortReason:          pf.AbortReason,
	}

	switch in.Format {
	case previewFormatJSON:
		return renderPreviewJSON(streams, report)
	case "", previewFormatText:
		return renderPreviewText(streams, report)
	default:
		return fmt.Errorf(i18n.T().Cmd.AngelaReviewErrUnknownFormat, in.Format)
	}
}

// renderPreviewText writes the human-readable report to stdout (AC-2).
func renderPreviewText(streams domain.IOStreams, r reviewPreviewReport) error {
	ta := i18n.T().Angela
	var b strings.Builder
	b.WriteString(ta.UIReviewPreviewHeader)

	fmt.Fprintf(&b, ta.UIReviewPreviewCorpus, r.CorpusDocs, formatBytes(r.CorpusBytes))
	fmt.Fprintf(&b, ta.UIReviewPreviewModel, r.Model)

	if len(r.Personas) == 0 {
		b.WriteString(ta.UIReviewPreviewPersonasBaseline)
	} else {
		fmt.Fprintf(&b, ta.UIReviewPreviewPersonasList,
			len(r.Personas), pluralS(len(r.Personas)), strings.Join(r.Personas, ", "))
	}

	if r.Audience == "" {
		b.WriteString(ta.UIReviewPreviewAudienceNone)
	} else {
		fmt.Fprintf(&b, ta.UIReviewPreviewAudience, r.Audience)
	}

	fmt.Fprintf(&b, ta.UIReviewPreviewTokens,
		formatTokens(r.EstimatedInputTokens), formatTokens(r.MaxOutputTokens))
	fmt.Fprintf(&b, ta.UIReviewPreviewContextWindow, r.ContextWindowUsedPct)

	if r.EstimatedCostUSD != nil {
		fmt.Fprintf(&b, ta.UIReviewPreviewCost, formatUSD(*r.EstimatedCostUSD))
	} else {
		b.WriteString(ta.UIReviewPreviewCostUnknown)
	}

	if r.ExpectedSeconds != nil && *r.ExpectedSeconds > 0 {
		fmt.Fprintf(&b, ta.UIReviewPreviewExpectedTime, *r.ExpectedSeconds)
	}

	if len(r.Warnings) > 0 {
		b.WriteString(ta.UIReviewPreviewWarningsHeader)
		for _, w := range r.Warnings {
			fmt.Fprintf(&b, "  - %s\n", w)
		}
	}
	if r.ShouldAbort {
		fmt.Fprintf(&b, ta.UIReviewPreviewAbort, r.AbortReason)
	}

	_, err := fmt.Fprint(streams.Out, b.String())
	return swallowBrokenPipe(err)
}

// renderPreviewJSON writes the machine-readable report to stdout (AC-6).
func renderPreviewJSON(streams domain.IOStreams, r reviewPreviewReport) error {
	// Normalize nil slices to empty so the JSON schema is stable.
	if r.Personas == nil {
		r.Personas = []string{}
	}
	if r.Warnings == nil {
		r.Warnings = []string{}
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("angela: review: preview json: %w", err)
	}
	_, err = fmt.Fprintln(streams.Out, string(data))
	return swallowBrokenPipe(err)
}

// swallowBrokenPipe returns nil when err is a broken-pipe signal from a closed
// downstream reader. Typical case: `lore angela review --preview | head -1`
// — head closes its stdin after the first line, and the next write on our
// stdout returns EPIPE. Propagating that as a command error adds "Error: ..."
// noise on stderr and a non-zero exit — both user-hostile for a read-only
// preview. Any other write error is returned unchanged so real failures
// (closed fd, full disk) still surface.
func swallowBrokenPipe(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.EPIPE) {
		return nil
	}
	if strings.Contains(err.Error(), "broken pipe") {
		return nil
	}
	return err
}

// contextWindowPct computes (inputTokens + maxOutputTokens) / ctxLimit as a
// percentage. Returns 0 when the model is unknown so the JSON schema stays
// deterministic (rather than emitting NaN or -1).
func contextWindowPct(inputTokens, maxOutput int, model string) float64 {
	limit, ok := angela.ModelContextLimit(model)
	if !ok || limit == 0 {
		return 0
	}
	return float64(inputTokens+maxOutput) * 100.0 / float64(limit)
}

// expectedSecondsFromPreflight infers wall-clock seconds from provider speed
// and projected output tokens. Returns 0 for unknown models — caller must
// decide whether to display "unknown" or omit the line.
func expectedSecondsFromPreflight(pf *angela.PreflightResult, model string) int {
	speed, ok := angela.ModelOutputSpeed(model)
	if !ok || speed <= 0 {
		return 0
	}
	expectedOutput := angela.ExpectedOutputTokens(pf.EstimatedInputTokens, pf.MaxOutputTokens)
	s := float64(expectedOutput) / speed
	if s < 1 {
		return 1
	}
	return int(s)
}

// formatBytes renders a byte count with a K/M suffix for readability.
func formatBytes(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1f MB", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0f KB", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
