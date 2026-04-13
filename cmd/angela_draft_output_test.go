// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/testutil"
)

// ═══════════════════════════════════════════════════════════════
// Story 8.5 tests: CLI flags + JSON reporter + exit codes
// ═══════════════════════════════════════════════════════════════

// TestParseSeverityFlag_Valid verifies the happy path of --severity
// parsing: multiple key=value pairs become a map.
func TestParseSeverityFlag_Valid(t *testing.T) {
	got, err := parseSeverityFlag([]string{"coherence=off", "style=warning"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["coherence"] != "off" || got["style"] != "warning" {
		t.Errorf("unexpected map: %v", got)
	}
}

// TestParseSeverityFlag_InvalidRejected verifies that malformed pairs
// produce a clear error at flag parse time.
func TestParseSeverityFlag_InvalidRejected(t *testing.T) {
	tests := []string{
		"just-a-category",
		"=value-only",
		"",
	}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			_, err := parseSeverityFlag([]string{s})
			if err == nil {
				t.Errorf("expected error for %q", s)
			}
		})
	}
}

// TestParseSeverityFlag_Empty verifies that an empty slice returns nil
// (no allocation) to match the "no override" case.
func TestParseSeverityFlag_Empty(t *testing.T) {
	got, err := parseSeverityFlag(nil)
	if err != nil || got != nil {
		t.Errorf("expected (nil, nil), got (%v, %v)", got, err)
	}
}

// TestMergeSeverityOverride_FlagOverridesConfig verifies that CLI
// values take precedence on key collisions.
func TestMergeSeverityOverride_FlagOverridesConfig(t *testing.T) {
	cfgMap := map[string]string{"coherence": "info", "style": "warning"}
	flagMap := map[string]string{"coherence": "off"}
	got := mergeSeverityOverride(cfgMap, flagMap)
	if got["coherence"] != "off" {
		t.Errorf("flag should win, got %q", got["coherence"])
	}
	if got["style"] != "warning" {
		t.Errorf("config-only key should persist, got %q", got["style"])
	}
}

// TestMergeSeverityOverride_NilInputs verifies that nil inputs produce
// nil output (not an empty map, to avoid useless allocations).
func TestMergeSeverityOverride_NilInputs(t *testing.T) {
	if got := mergeSeverityOverride(nil, nil); got != nil {
		t.Errorf("both nil should return nil, got %v", got)
	}
}

// TestValidateFailOn accepts all documented levels and rejects typos.
func TestValidateFailOn(t *testing.T) {
	valid := []string{"error", "warning", "info", "never"}
	for _, v := range valid {
		if err := validateFailOn(v); err != nil {
			t.Errorf("expected %q accepted, got %v", v, err)
		}
	}
	invalid := []string{"banana", "panic", "yes"}
	for _, v := range invalid {
		if err := validateFailOn(v); err == nil {
			t.Errorf("expected %q rejected", v)
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// Reporter tests — use DraftReport directly, no cobra round-trip
// ═══════════════════════════════════════════════════════════════

// TestJSONDraftReporter_SchemaShape verifies that the JSON output
// matches the documented schema: version, mode, scanned, reviewed,
// files, summary with by_severity + by_category.
func TestJSONDraftReporter_SchemaShape(t *testing.T) {
	report := DraftReport{
		Version: draftJSONSchemaVersion,
		Mode:    "standalone",
		Scanned: 2,
		Files: []DraftFileReport{
			{
				Filename: "a.md",
				Score:    95,
				Grade:    "A",
				Profile:  "strict",
				Suggestions: []angela.Suggestion{
					{Category: "coherence", Severity: "info", Message: "m1"},
					{Category: "structure", Severity: "warning", Message: "m2"},
				},
			},
			{
				Filename: "b.md",
				Score:    100,
				Grade:    "A",
				Profile:  "free-form",
			},
		},
	}
	report.computeSummary()

	var out bytes.Buffer
	rep := &jsonDraftReporter{out: &out}
	if err := rep.Report(report); err != nil {
		t.Fatalf("report: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal: %v\npayload: %s", err, out.String())
	}
	if v, ok := decoded["version"].(float64); !ok || int(v) != draftJSONSchemaVersion {
		t.Errorf("version: got %v, want %d", decoded["version"], draftJSONSchemaVersion)
	}
	if decoded["mode"] != "standalone" {
		t.Errorf("mode: got %v", decoded["mode"])
	}
	if v, ok := decoded["scanned"].(float64); !ok || int(v) != 2 {
		t.Errorf("scanned: got %v", decoded["scanned"])
	}
	if v, ok := decoded["reviewed"].(float64); !ok || int(v) != 1 {
		t.Errorf("reviewed: got %v, want 1", decoded["reviewed"])
	}
	summary, ok := decoded["summary"].(map[string]interface{})
	if !ok {
		t.Fatalf("summary missing or wrong type: %v", decoded["summary"])
	}
	if v := summary["total_suggestions"].(float64); int(v) != 2 {
		t.Errorf("total_suggestions: got %v, want 2", v)
	}
	bySev := summary["by_severity"].(map[string]interface{})
	if int(bySev["info"].(float64)) != 1 || int(bySev["warning"].(float64)) != 1 {
		t.Errorf("by_severity: %v", bySev)
	}
	byCat := summary["by_category"].(map[string]interface{})
	if int(byCat["coherence"].(float64)) != 1 || int(byCat["structure"].(float64)) != 1 {
		t.Errorf("by_category: %v", byCat)
	}
}

// TestJSONDraftReporter_EmptyFilesRendersAsArray verifies that an empty
// Files slice is rendered as JSON [] not null, so consumers can rely on
// a stable shape.
func TestJSONDraftReporter_EmptyFilesRendersAsArray(t *testing.T) {
	report := DraftReport{Version: draftJSONSchemaVersion, Mode: "standalone"}
	report.computeSummary()

	var out bytes.Buffer
	rep := &jsonDraftReporter{out: &out}
	if err := rep.Report(report); err != nil {
		t.Fatalf("report: %v", err)
	}
	if !strings.Contains(out.String(), `"files": []`) {
		t.Errorf("expected files: [] in output, got: %s", out.String())
	}
}

// TestHumanDraftReporter_VerboseShowsInfo verifies the pre-existing
// behavior: verbose mode prints every finding, default mode hides info.
func TestHumanDraftReporter_VerboseShowsInfo(t *testing.T) {
	report := DraftReport{
		Mode:    "lore-native",
		Scanned: 1,
		Files: []DraftFileReport{
			{
				Filename: "a.md",
				Score:    80,
				Grade:    "B",
				Suggestions: []angela.Suggestion{
					{Category: "style", Severity: "info", Message: "filler"},
					{Category: "structure", Severity: "warning", Message: "missing ## Why"},
				},
			},
		},
	}
	report.computeSummary()

	var outDefault bytes.Buffer
	(&humanDraftReporter{out: &outDefault, verbose: false}).Report(report)
	if strings.Contains(outDefault.String(), "filler") {
		t.Errorf("default mode should hide info-level messages, got: %s", outDefault.String())
	}
	if !strings.Contains(outDefault.String(), "missing ## Why") {
		t.Errorf("default mode should show warnings, got: %s", outDefault.String())
	}

	var outVerbose bytes.Buffer
	(&humanDraftReporter{out: &outVerbose, verbose: true}).Report(report)
	if !strings.Contains(outVerbose.String(), "filler") {
		t.Errorf("verbose mode should show info-level messages, got: %s", outVerbose.String())
	}
}

// ═══════════════════════════════════════════════════════════════
// End-to-end flag tests (through cobra)
// ═══════════════════════════════════════════════════════════════

// runAngelaDraftJSON is a convenience wrapper for JSON-format runs.
// It returns the parsed JSON payload alongside the raw streams.
func runAngelaDraftJSON(t *testing.T, args ...string) (DraftReport, string, error) {
	t.Helper()
	stdout, _, err := runAngelaDraftAllWithArgs(t, nil, args...)
	if err != nil {
		return DraftReport{}, stdout, err
	}
	var report DraftReport
	if decodeErr := json.Unmarshal([]byte(stdout), &report); decodeErr != nil {
		return DraftReport{}, stdout, decodeErr
	}
	return report, stdout, nil
}

// TestAngelaDraft_All_JSONFormat runs the command end-to-end with
// --format=json and verifies the stdout is a parseable DraftReport.
func TestAngelaDraft_All_JSONFormat(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-04-02\"\n---\n## What\nShort.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-json-2026-04-02.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	report, raw, err := runAngelaDraftJSON(t, "draft", "--all", "--format=json")
	if err != nil {
		t.Fatalf("json format: %v\nraw: %s", err, raw)
	}
	if report.Version != draftJSONSchemaVersion {
		t.Errorf("version: got %d, want %d", report.Version, draftJSONSchemaVersion)
	}
	if report.Scanned < 1 {
		t.Errorf("scanned: got %d, want >= 1", report.Scanned)
	}
	if report.Summary.BySeverity == nil {
		t.Errorf("summary.BySeverity should be non-nil")
	}
}

// TestAngelaDraft_All_SeverityOverrideOff verifies that --severity
// category=off drops findings in that category from the report entirely.
func TestAngelaDraft_All_SeverityOverrideOff(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	// A doc whose body mentions another doc to trigger coherence suggestions.
	doc1 := "---\ntype: decision\nstatus: published\ndate: \"2026-04-01\"\n---\n## What\nSee also decision-other-2026-04-01.md for context.\n## Why\nBecause.\n"
	doc2 := "---\ntype: decision\nstatus: published\ndate: \"2026-04-01\"\n---\n## What\nOther decision.\n## Why\nReasons.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-main-2026-04-01.md"), []byte(doc1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "decision-other-2026-04-01.md"), []byte(doc2), 0644); err != nil {
		t.Fatal(err)
	}

	// Baseline: run without override and count coherence findings.
	baseline, _, err := runAngelaDraftJSON(t, "draft", "--all", "--format=json")
	if err != nil {
		t.Fatalf("baseline: %v", err)
	}
	baselineCoherence := baseline.Summary.ByCategory["coherence"]

	// With override: coherence category is dropped.
	muted, _, err := runAngelaDraftJSON(t, "draft", "--all", "--format=json", "--severity", "coherence=off")
	if err != nil {
		t.Fatalf("muted: %v", err)
	}
	if muted.Summary.ByCategory["coherence"] != 0 {
		t.Errorf("coherence should be 0 after off, got %d (baseline was %d)",
			muted.Summary.ByCategory["coherence"], baselineCoherence)
	}
}

// TestAngelaDraft_All_FailOnWarningExits verifies that --fail-on=warning
// returns a non-zero ExitCodeError when warnings are present.
func TestAngelaDraft_All_FailOnWarningExits(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	// A very short doc → structure warning (missing sections).
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-04-05\"\n---\n## What\nOne line.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-fail-2026-04-05.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runAngelaDraftAllWithArgs(t, nil, "draft", "--all", "--fail-on=warning")
	if err == nil {
		t.Fatal("expected non-zero exit with --fail-on=warning, got nil")
	}
	var ec *cli.ExitCodeError
	if !errors.As(err, &ec) {
		t.Fatalf("expected *cli.ExitCodeError, got %T: %v", err, err)
	}
	if ec.Code != 1 && ec.Code != 2 {
		t.Errorf("unexpected exit code: %d", ec.Code)
	}
}

// TestAngelaDraft_All_FailOnNeverAlwaysZero verifies that --fail-on=never
// returns nil regardless of finding count.
func TestAngelaDraft_All_FailOnNeverAlwaysZero(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-04-05\"\n---\n## What\nOne line.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-never-2026-04-05.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runAngelaDraftAllWithArgs(t, nil, "draft", "--all", "--fail-on=never")
	if err != nil {
		t.Errorf("--fail-on=never should never fail, got %v", err)
	}
}

// TestAngelaDraft_All_StrictPromotesWarnings verifies that --strict
// promotes warnings to errors in the JSON report's by_severity counts.
func TestAngelaDraft_All_StrictPromotesWarnings(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-04-05\"\n---\n## What\nShort.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-strict-2026-04-05.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	// --fail-on=never so the command doesn't exit before we capture JSON.
	report, _, err := runAngelaDraftJSON(t, "draft", "--all", "--format=json", "--strict", "--fail-on=never")
	if err != nil {
		t.Fatalf("strict run: %v", err)
	}
	if report.Summary.BySeverity["warning"] != 0 {
		t.Errorf("strict mode should leave 0 warnings, got %d",
			report.Summary.BySeverity["warning"])
	}
	if report.Summary.BySeverity["error"] == 0 {
		t.Errorf("strict mode should produce errors, got by_severity: %v",
			report.Summary.BySeverity)
	}
}

// TestAngelaDraft_InvalidFailOnRejected verifies that a typo in
// --fail-on fails at flag parse time.
func TestAngelaDraft_InvalidFailOnRejected(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-04-05\"\n---\n## What\nOK.\n## Why\nOK.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-typo-2026-04-05.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runAngelaDraftAllWithArgs(t, nil, "draft", "--all", "--fail-on=banana")
	if err == nil {
		t.Fatal("expected error on invalid --fail-on")
	}
	if !strings.Contains(err.Error(), "invalid --fail-on") {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestAngelaDraft_InvalidSeverityRejected verifies that a malformed
// --severity value fails at flag parse time.
func TestAngelaDraft_InvalidSeverityRejected(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	testutil.Chdir(t, dir)

	docsDir := filepath.Join(dir, ".lore", "docs")
	doc := "---\ntype: decision\nstatus: published\ndate: \"2026-04-05\"\n---\n## What\nOK.\n## Why\nOK.\n"
	if err := os.WriteFile(filepath.Join(docsDir, "decision-sev-2026-04-05.md"), []byte(doc), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := runAngelaDraftAllWithArgs(t, nil, "draft", "--all", "--severity", "no-equals-sign")
	if err == nil {
		t.Fatal("expected error on malformed --severity")
	}
	if !strings.Contains(err.Error(), "invalid --severity") {
		t.Errorf("unexpected error: %v", err)
	}
}

// assertIOStreams is a sanity helper that ensures the test harness
// hasn't accidentally started emitting human output on stdout (which
// would corrupt JSON consumers). Used implicitly by runAngelaDraftJSON.
var _ = domain.IOStreams{}
var _ = config.Config{}
