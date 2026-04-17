// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
)

// ═══════════════════════════════════════════════════════════════
// Draft CI output & gating
//
// This file is the single source of truth for how `lore angela draft`
// renders its results. The reporter abstraction lets human and JSON
// modes share the same input data (DraftReport) so they never drift.
// ═══════════════════════════════════════════════════════════════

// draftJSONSchemaVersion is the version of the JSON output schema.
// Bump this on any breaking change to DraftJSONReport so CI pipelines
// that parse the output can detect incompatibilities.
const draftJSONSchemaVersion = 1

// DraftFileReport captures the result of analyzing a single document.
// Used by both human and JSON reporters to avoid a second data pass.
type DraftFileReport struct {
	Filename    string               `json:"file"`
	Score       int                  `json:"score"`
	Grade       string               `json:"grade"`
	Profile     string               `json:"profile"` // "strict" | "free-form"
	Suggestions []angela.Suggestion  `json:"suggestions"`
}

// DraftReport is the complete output of a draft run (single file or --all).
// It captures everything the reporters need: mode, counts, per-file details
// and aggregated summary.
//
// Diff + Resolved (omitempty): when differential mode is
// active, Diff carries the aggregate new/persisting/resolved counts and
// Resolved lists the findings that disappeared since the previous run
// (including findings whose source file was deleted entirely).
type DraftReport struct {
	Version  int                        `json:"version"`
	Mode     string                     `json:"mode"`    // "lore-native"|"hybrid"|"standalone"
	Scanned  int                        `json:"scanned"` // total docs analyzed
	Reviewed int                        `json:"reviewed"` // docs with at least one finding
	Files    []DraftFileReport          `json:"files"`
	Summary  DraftSummary               `json:"summary"`
	Diff     *angela.DraftDiff          `json:"diff_summary,omitempty"`
	Resolved []angela.ResolvedSuggestion `json:"resolved,omitempty"`
}

// DraftSummary holds aggregated counts for the report footer and for
// structured CI consumption.
type DraftSummary struct {
	TotalSuggestions int            `json:"total_suggestions"`
	BySeverity       map[string]int `json:"by_severity"`
	ByCategory       map[string]int `json:"by_category"`
}

// computeSummary walks a report's per-file slices and populates the
// summary counts. Called once at report assembly time.
func (r *DraftReport) computeSummary() {
	r.Summary.BySeverity = map[string]int{}
	r.Summary.ByCategory = map[string]int{}
	r.Reviewed = 0
	r.Summary.TotalSuggestions = 0
	for _, f := range r.Files {
		if len(f.Suggestions) > 0 {
			r.Reviewed++
		}
		for _, s := range f.Suggestions {
			r.Summary.TotalSuggestions++
			r.Summary.BySeverity[s.Severity]++
			r.Summary.ByCategory[s.Category]++
		}
	}
}

// allSuggestions returns a flat slice of every finding in the report.
// Used by the exit-code resolver which does not care about per-file
// grouping.
func (r *DraftReport) allSuggestions() []angela.Suggestion {
	var out []angela.Suggestion
	for _, f := range r.Files {
		out = append(out, f.Suggestions...)
	}
	return out
}

// ═══════════════════════════════════════════════════════════════
// Reporter interface + implementations
// ═══════════════════════════════════════════════════════════════

// DraftReporter renders a DraftReport to an output stream. Implementations
// decide the format (human-readable or machine-parseable JSON).
//
// Report() is called once with the complete, finalized report. This is
// the primary rendering path for JSON and tests.
//
// ReportFile() is an optional streaming hook: the runner calls it after
// each per-file analysis to let the human reporter show the row inline
// with the progress bar. JSON reporters implement it as a no-op and
// emit the full payload only from Report().
type DraftReporter interface {
	Report(r DraftReport) error
	ReportFile(f DraftFileReport)
}

// newDraftReporter returns the reporter matching the requested format.
// Unknown formats fall back to "human" with a warning on stderr.
//
// Human output goes to streams.Err (matches existing behavior so
// progress bars and findings share one channel). JSON output goes to
// stdout so pipes like `| jq` work naturally.
//
// verbose controls whether the human reporter shows info-level findings
// inline (verbose=true) or only warnings/errors (verbose=false). When
// format=json, --verbose has no effect: JSON output always includes all
// findings regardless of verbosity. This is intentional — JSON consumers
// should filter client-side.
func newDraftReporter(format string, streams domain.IOStreams, verbose bool) DraftReporter {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		return &jsonDraftReporter{out: streams.Out}
	case "", "human":
		return &humanDraftReporter{out: streams.Err, verbose: verbose}
	default:
		fmt.Fprintf(streams.Err, "warning: unknown --format %q, falling back to human\n", format)
		return &humanDraftReporter{out: streams.Err, verbose: verbose}
	}
}

// jsonDraftReporter marshals the report to indented JSON. One call per
// run. All counts and summaries are precomputed before the reporter
// runs so that JSON consumers can rely on a stable schema.
type jsonDraftReporter struct {
	out io.Writer
}

func (j *jsonDraftReporter) Report(r DraftReport) error {
	// Ensure nil slices become [] in JSON (nicer for consumers).
	for i := range r.Files {
		if r.Files[i].Suggestions == nil {
			r.Files[i].Suggestions = []angela.Suggestion{}
		}
	}
	if r.Files == nil {
		r.Files = []DraftFileReport{}
	}
	enc := json.NewEncoder(j.out)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// ReportFile is a no-op for JSON output: the per-file data is included
// in the final payload emitted by Report(), and streaming rows would
// corrupt the single JSON document on stdout.
func (j *jsonDraftReporter) ReportFile(_ DraftFileReport) {}

// humanDraftReporter writes the existing colored terminal output. It
// preserves the per-file progress rows that users already know. This
// is the default when stdout is a TTY.
//
// verbose controls whether info-level suggestions are printed inline.
// In non-verbose mode only warnings and errors are shown; info findings
// are counted in the summary but not listed per-file (matches the
// original behavior so existing users see no regression).
//
// streamed tracks whether ReportFile() has already printed rows so
// Report() can skip them and only emit the summary footer, preserving
// the pre-refactor "per-file row immediately follows its progress bar"
// behavior.
//
// diffOnly asks the human reporter to hide findings tagged
// DiffStatus="persisting" in the per-file rows. NEW and RESOLVED always
// show through so the user sees the delta without the noise.
type humanDraftReporter struct {
	out      io.Writer
	verbose  bool
	streamed bool
	diffOnly bool
}

// ReportFile renders a single file row immediately after its analysis
// completes. Called from the runner loop so rows interleave with the
// progress bar the way the original output did.
func (h *humanDraftReporter) ReportFile(f DraftFileReport) {
	h.streamed = true
	h.writeFileRow(f)
}

// writeFileRow is the shared rendering primitive for both streamed and
// batched modes. It prints the grade line plus any inline findings.
//
// When h.diffOnly is set, findings tagged PERSISTING are
// filtered out of the inline detail list. The row header (grade +
// filename) is still shown so the user can see which file is in which
// state. Prefix markers (+ / =) tag NEW and PERSISTING visually.
func (h *humanDraftReporter) writeFileRow(f DraftFileReport) {
	t := i18n.T().Cmd
	grade := fmt.Sprintf("%d/100 (%s)", f.Score, f.Grade)
	if len(f.Suggestions) == 0 {
		fmt.Fprintf(h.out, "  %-10s %-8s %s\n", grade, "ok", f.Filename)
		return
	}

	warnings := 0
	for _, s := range f.Suggestions {
		if s.Severity == angela.SeverityWarning || s.Severity == angela.SeverityError {
			warnings++
		}
	}
	// i18n format strings are validated at compile time by TestAngelaSuggestions_FormatSpecifierParity.
	label := fmt.Sprintf(t.AngelaDraftAllSugg, len(f.Suggestions))
	if warnings > 0 {
		label = fmt.Sprintf(t.AngelaDraftAllSuggWarn, len(f.Suggestions), warnings)
	}
	fmt.Fprintf(h.out, "  %-10s %-8s %-40s %s\n", grade, "review", f.Filename, label)

	// Inline details:
	//   - verbose: print every suggestion
	//   - default: print only warnings/errors (info findings are
	//     summarized but not listed, to keep the default output
	//     scannable)
	//   - diffOnly: skip suggestions tagged PERSISTING so
	//     the user sees only what changed since the previous run
	for _, s := range f.Suggestions {
		if h.diffOnly && s.DiffStatus == angela.DiffStatusPersisting {
			continue
		}
		if !h.verbose && s.Severity != angela.SeverityWarning && s.Severity != angela.SeverityError {
			continue
		}
		// Prefix with a diff marker when set so the user
		// can spot new findings at a glance. Empty status → space so
		// column widths stay aligned.
		marker := " "
		switch s.DiffStatus {
		case angela.DiffStatusNew:
			marker = "+"
		case angela.DiffStatusPersisting:
			marker = "="
		}
		// i18n format strings are validated at compile time by TestAngelaSuggestions_FormatSpecifierParity.
		fmt.Fprintf(h.out, "       %s %-8s %-14s %s\n", marker, s.Severity, s.Category, s.Message)
	}
}

func (h *humanDraftReporter) Report(r DraftReport) error {
	t := i18n.T().Cmd

	// When rows were already streamed via ReportFile, skip the batch
	// rendering. Otherwise print per-file rows (e.g. single-file mode
	// or tests that call Report() directly without streaming).
	if !h.streamed {
		for _, f := range r.Files {
			h.writeFileRow(f)
		}
	}

	// When there are RESOLVED findings, list them before
	// the summary so the user sees what cleared up. The list is
	// already sorted deterministically by AnnotateAndDiff.
	if len(r.Resolved) > 0 {
		fmt.Fprintf(h.out, "\n%s\n", t.AngelaDraftResolvedHeader)
		for _, rs := range r.Resolved {
			fmt.Fprintf(h.out, "       - %-8s %-14s %s  (%s)\n",
				rs.Suggestion.Severity, rs.Suggestion.Category, rs.Suggestion.Message, rs.File)
		}
	}

	// Summary footer. Kept in sync with the existing i18n catalog entry
	// so French and English users see the same wording.
	// i18n format strings are validated at compile time by TestAngelaSuggestions_FormatSpecifierParity.
	fmt.Fprintf(h.out, "\n"+t.AngelaDraftAllSummary+"\n",
		r.Reviewed, r.Scanned, r.Summary.TotalSuggestions)

	// Differential summary line. Only printed when the
	// runner populated r.Diff (differential mode active).
	if r.Diff != nil {
		fmt.Fprintf(h.out, t.AngelaDraftDiffSummary+"\n",
			r.Diff.New, r.Diff.Persisting, r.Diff.Resolved)
	}

	// Badge hint when everything is clean — nudge the user to advertise it.
	if r.Summary.TotalSuggestions == 0 {
		fmt.Fprintf(h.out, "\n%s\n", ui.Dim(i18n.T().Cmd.BadgeHintDraftClean))
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════
// CLI flag parsing helpers
// ═══════════════════════════════════════════════════════════════

// parseSeverityFlag parses repeated `--severity category=level` flags
// into a map. Empty input → nil map. Invalid entries (missing `=`) are
// returned as an error so cobra fails flag parsing cleanly.
//
// This merges with the config-level override map: CLI values take
// precedence over .lorerc values for the current run.
func parseSeverityFlag(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 || parts[0] == "" {
			return nil, fmt.Errorf("invalid --severity value %q (want category=level)", pair)
		}
		out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	return out, nil
}

// mergeSeverityOverride combines the config-level override map with a
// CLI override map. CLI wins on key collisions. Either input may be nil.
func mergeSeverityOverride(fromConfig, fromFlag map[string]string) map[string]string {
	if len(fromConfig) == 0 && len(fromFlag) == 0 {
		return nil
	}
	out := make(map[string]string, len(fromConfig)+len(fromFlag))
	for k, v := range fromConfig {
		out[k] = v
	}
	for k, v := range fromFlag {
		out[k] = v
	}
	return out
}

// validateFailOn checks that a fail_on value is one of the accepted
// levels. Returns a helpful error for typos. Called at flag parse time.
func validateFailOn(value string) error {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "error", "warning", "info", "never":
		return nil
	default:
		return fmt.Errorf("invalid --fail-on %q (want error|warning|info|never)", value)
	}
}

// drawProgress is a tiny wrapper that only draws a progress bar when
// the format is human and stderr is visible. Keeps the --all loop
// tidy when JSON output is requested.
func drawProgress(format string, streams domain.IOStreams, current, total int, label string) {
	if strings.EqualFold(strings.TrimSpace(format), "json") {
		return
	}
	ui.Progress(streams, current, total, label)
}
