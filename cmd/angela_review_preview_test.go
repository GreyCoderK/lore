// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

// streamsForPreview returns IOStreams whose Out/Err are inspectable bytes.Buffers.
func streamsForPreview() (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	return domain.IOStreams{In: &bytes.Buffer{}, Out: out, Err: errBuf}, out, errBuf
}

// baseReviewPreviewInputs returns a known-model fixture so Preflight populates
// EstimatedCost and expected time. Tests for unknown-model and warning paths
// override fields locally.
func baseReviewPreviewInputs() reviewPreviewInputs {
	return reviewPreviewInputs{
		CorpusBytes:    4200,
		CorpusDocCount: 3,
		Model:          "claude-sonnet-4-6",
		MaxTokens:      4000,
		Timeout:        60 * time.Second,
		Personas:       nil,
		Audience:       "",
		Format:         "",
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AC-2: text report content
// ─────────────────────────────────────────────────────────────────────────────

func TestReviewPreview_TextReport_BaselineFields(t *testing.T) {
	streams, out, _ := streamsForPreview()
	if err := runReviewPreview(streams, &config.Config{}, baseReviewPreviewInputs()); err != nil {
		t.Fatalf("runReviewPreview returned error: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"Review preview",
		"Corpus:",
		"3 documents",
		"Model:",
		"claude-sonnet-4-6",
		"baseline (no personas)",
		"Audience:         (none)",
		"Estimated tokens:",
		"Context window:",
		"Estimated cost:",
		"$",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preview text missing %q.\nGot:\n%s", want, got)
		}
	}
}

// AC-3: unknown model falls back to "unknown" cost line.
func TestReviewPreview_UnknownModelFallback(t *testing.T) {
	streams, out, _ := streamsForPreview()
	in := baseReviewPreviewInputs()
	in.Model = "not-a-registered-model"
	if err := runReviewPreview(streams, &config.Config{}, in); err != nil {
		t.Fatalf("runReviewPreview returned error: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "unknown (model pricing not registered)") {
		t.Errorf("unknown model must surface fallback cost line; got:\n%s", got)
	}
	if strings.Contains(got, "Estimated cost:   $") {
		t.Errorf("unknown model must NOT emit a $ cost; got:\n%s", got)
	}
}

// AC-4: Preflight warnings surface in the Warnings section. Engineering a
// scenario where the token budget lands in the 85%-of-context-window warning
// band gets us a deterministic warning.
func TestReviewPreview_WithWarnings(t *testing.T) {
	streams, out, _ := streamsForPreview()
	in := baseReviewPreviewInputs()
	// Sonnet 4.6 context is 200K. Push input+maxOutput over 85% to trip the warn.
	in.CorpusBytes = 500_000 // ~142K tokens
	in.MaxTokens = 60_000    // total ~202K > 200K → ShouldAbort+warning emitted early
	if err := runReviewPreview(streams, &config.Config{}, in); err != nil {
		t.Fatalf("runReviewPreview must return nil even with warnings/abort: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "ABORT") && !strings.Contains(got, "Warnings") {
		t.Errorf("heavy payload must surface warnings or abort line; got:\n%s", got)
	}
}

// AC-5: persona integration — tokens + cost reflect the persona prompt.
func TestReviewPreview_WithPersonas_ReflectsInflatedTokens(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, outWith, _ := streamsForPreview()
	streamsBase, outBase, _ := streamsForPreview()

	inBase := baseReviewPreviewInputs()
	inWith := inBase
	personas, _ := resolvePersonaNames([]string{name})
	inWith.Personas = personas

	if err := runReviewPreview(streamsBase, &config.Config{}, inBase); err != nil {
		t.Fatalf("baseline preview error: %v", err)
	}
	if err := runReviewPreview(streams, &config.Config{}, inWith); err != nil {
		t.Fatalf("with-personas preview error: %v", err)
	}

	// Persona names must appear in the with-personas report.
	if !strings.Contains(outWith.String(), name) {
		t.Errorf("with-personas preview must list persona names, got:\n%s", outWith.String())
	}
	if !strings.Contains(outWith.String(), "1 persona") {
		t.Errorf("with-personas preview must show persona count, got:\n%s", outWith.String())
	}
	// Token count with personas should be strictly greater than baseline.
	baseTokens := extractInputTokens(t, outBase.String())
	withTokens := extractInputTokens(t, outWith.String())
	if withTokens <= baseTokens {
		t.Errorf("persona-inflated tokens (%d) must exceed baseline (%d)", withTokens, baseTokens)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AC-6: --format=json schema
// ─────────────────────────────────────────────────────────────────────────────

func TestReviewPreview_JSONSchema(t *testing.T) {
	streams, out, _ := streamsForPreview()
	in := baseReviewPreviewInputs()
	in.Format = "json"

	if err := runReviewPreview(streams, &config.Config{}, in); err != nil {
		t.Fatalf("runReviewPreview(json) error: %v", err)
	}

	var report reviewPreviewReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, out.String())
	}
	if report.Mode != "preview" {
		t.Errorf("mode = %q, want 'preview'", report.Mode)
	}
	if report.Model != "claude-sonnet-4-6" {
		t.Errorf("model = %q, want claude-sonnet-4-6", report.Model)
	}
	if report.CorpusDocs != 3 {
		t.Errorf("corpus_documents = %d, want 3", report.CorpusDocs)
	}
	// Nil slices must be normalized to empty arrays for stable schema.
	if report.Personas == nil {
		t.Errorf("personas must be [] not null in JSON")
	}
	if report.Warnings == nil {
		t.Errorf("warnings must be [] not null in JSON")
	}
	if report.EstimatedCostUSD == nil {
		t.Errorf("known model must yield non-nil estimated_cost_usd pointer")
	} else if *report.EstimatedCostUSD < 0 {
		t.Errorf("known model must have non-negative cost, got %f", *report.EstimatedCostUSD)
	}
	if report.SchemaVersion == "" {
		t.Error("schema_version must be set so consumers can detect breaking changes")
	}
}

func TestReviewPreview_UnknownFormat_Rejected(t *testing.T) {
	streams, _, _ := streamsForPreview()
	in := baseReviewPreviewInputs()
	in.Format = "yaml"
	err := runReviewPreview(streams, &config.Config{}, in)
	if err == nil || !strings.Contains(err.Error(), "unknown --format") {
		t.Fatalf("unknown format must error with clear message, got: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AC-7: zero HTTP calls enforced via a counting RoundTripper.
// ─────────────────────────────────────────────────────────────────────────────

// countingTransport is a http.RoundTripper that records how many times it was
// called. Any non-zero count is treated as a contract violation for --preview.
type countingTransport struct {
	count int64
}

func (c *countingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	atomic.AddInt64(&c.count, 1)
	return nil, http.ErrUseLastResponse
}

func TestReviewPreview_ZeroHTTPCalls(t *testing.T) {
	// Swap http.DefaultTransport in case any code path sneaks an http.Get.
	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()
	counter := &countingTransport{}
	http.DefaultTransport = counter

	streams, _, _ := streamsForPreview()
	in := baseReviewPreviewInputs()
	in.Format = "json" // exercise both render paths

	if err := runReviewPreview(streams, &config.Config{}, in); err != nil {
		t.Fatalf("preview must not error on a well-formed input: %v", err)
	}
	if got := atomic.LoadInt64(&counter.count); got != 0 {
		t.Errorf("--preview made %d HTTP call(s); AC-7 requires zero", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AC-8: zero side effects — no write, no state change.
// We assert this structurally: runReviewPreview receives only streams + cfg +
// inputs. It has no handle to any writer other than streams.Out/Err which are
// in-memory buffers in the test. A separate grep-style guard is out of scope
// here; the zero-provider + zero-http guarantees give us the meaningful safety.
// ─────────────────────────────────────────────────────────────────────────────

func TestReviewPreview_NoPanicOnZeroConfig(t *testing.T) {
	streams, _, _ := streamsForPreview()
	// Entirely empty config — should not panic. Model empty means unknown model
	// → fallback cost line. Preflight tolerates empty model gracefully.
	in := baseReviewPreviewInputs()
	in.Model = ""
	if err := runReviewPreview(streams, &config.Config{}, in); err != nil {
		t.Fatalf("zero-config preview must not error: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AC-5 & flag resolution: resolveReviewPersonasForPreview behavior
// ─────────────────────────────────────────────────────────────────────────────

func TestResolveReviewPersonasForPreview_NoFlag_Baseline(t *testing.T) {
	p, err := resolveReviewPersonasForPreview(&config.Config{}, nil, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("no flag, no config must yield nil personas, got %v", p)
	}
}

func TestResolveReviewPersonasForPreview_PersonaFlag_Activates(t *testing.T) {
	name := firstRegistryPersonaName(t)
	p, err := resolveReviewPersonasForPreview(&config.Config{}, []string{name}, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p) != 1 || p[0].Name != name {
		t.Errorf("expected 1 persona %q, got %+v", name, p)
	}
}

func TestResolveReviewPersonasForPreview_NoPersonasFlag_Baseline(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	p, err := resolveReviewPersonasForPreview(cfg, nil, true, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("--no-personas must yield nil even with config, got %v", p)
	}
}

func TestResolveReviewPersonasForPreview_UseConfigured_NoPrompt(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	p, err := resolveReviewPersonasForPreview(cfg, nil, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p) != 1 {
		t.Errorf("--use-configured-personas with 1 configured must yield 1 persona, got %d", len(p))
	}
}

func TestResolveReviewPersonasForPreview_TTY_Configured_NoPrompt_Baseline(t *testing.T) {
	// Preview mode intentionally skips the TTY prompt path. Having configured
	// personas without any activation flag yields baseline — the preview does
	// NOT open a y/N dialog.
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	p, err := resolveReviewPersonasForPreview(cfg, nil, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("preview without flag must stay baseline (no prompt), got %v", p)
	}
}

func TestResolveReviewPersonasForPreview_MutuallyExclusive(t *testing.T) {
	name := firstRegistryPersonaName(t)
	_, err := resolveReviewPersonasForPreview(&config.Config{}, []string{name}, true, false)
	if err == nil {
		t.Fatal("expected mutual exclusion error for --persona + --no-personas")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// AC-9: --preview + --interactive rejected with a clear error.
// ─────────────────────────────────────────────────────────────────────────────

// TestReviewCmd_PreviewInteractiveRejected exercises the cobra command to
// confirm the AC-9 mutual-exclusion guard fires BEFORE any docs/provider setup.
func TestReviewCmd_PreviewInteractiveRejected(t *testing.T) {
	streams, _, _ := streamsForPreview()
	cfg := &config.Config{}
	path := ""
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--preview", "--interactive"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --preview and --interactive are set together")
	}
	// Cobra's MarkFlagsMutuallyExclusive emits a stable message listing both
	// flag names. Assert on the names rather than the full text so the test
	// survives cobra version bumps that tweak the exact wording.
	msg := err.Error()
	if !strings.Contains(msg, "preview") || !strings.Contains(msg, "interactive") {
		t.Errorf("error must name both conflicting flags, got: %v", err)
	}
	if !strings.Contains(msg, "mutually") && !strings.Contains(msg, "none of the others") {
		t.Errorf("error must express mutual exclusion, got: %v", err)
	}
}

// writeFiveDocs populates a temp dir with 5 minimal markdown docs — the
// minimum PrepareDocSummaries accepts — so a cobra-level preview can run
// end-to-end with a real corpus reader.
func writeFiveDocs(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, fmt.Sprintf("doc%d.md", i))
		body := fmt.Sprintf("# Doc %d\n\nA short paragraph about topic %d.\n", i, i)
		if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// End-to-end cobra test for zero-HTTP preview contract.
// The existing unit test TestReviewPreview_ZeroHTTPCalls calls runReviewPreview
// directly and so cannot catch a regression where RunE instantiates the AI
// provider BEFORE the preview short-circuit. This test runs the full cobra
// command path and asserts a counting http.RoundTripper stays at zero hits.
func TestReviewCmd_Preview_E2E_ZeroHTTPCalls(t *testing.T) {
	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()
	counter := &countingTransport{}
	http.DefaultTransport = counter

	docsDir := writeFiveDocs(t)
	streams, _, _ := streamsForPreview()
	cfg := &config.Config{}
	path := docsDir
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--preview"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("preview cmd must succeed, got: %v", err)
	}
	if got := atomic.LoadInt64(&counter.count); got != 0 {
		t.Errorf("cobra --preview made %d HTTP call(s); zero-HTTP contract violated", got)
	}
}

// End-to-end cobra test for no-side-effect preview contract.
// Enables review.differential so the non-preview path would write a state
// file; asserts --preview short-circuits BEFORE any state write, proving the
// no-side-effect contract holds end-to-end (not just at the runReviewPreview
// unit layer).
func TestReviewCmd_Preview_E2E_NoStateWrite(t *testing.T) {
	docsDir := writeFiveDocs(t)
	stateDir := t.TempDir()

	cfg := &config.Config{}
	cfg.Angela.StateDir = stateDir
	cfg.Angela.Review.Differential.Enabled = true
	cfg.Angela.Review.Differential.StateFile = "review-state.json"

	streams, _, _ := streamsForPreview()
	path := docsDir
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--preview"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("preview cmd must succeed, got: %v", err)
	}

	entries, rerr := os.ReadDir(stateDir)
	if rerr != nil {
		t.Fatalf("readdir: %v", rerr)
	}
	if len(entries) != 0 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("--preview must not write any state file; found: %v", names)
	}
}

// End-to-end cobra test for --preview --format=json.
// Complements the direct-runReviewPreview JSON schema test by covering the
// full flag-parse → RunE → render path through cobra.
func TestReviewCmd_Preview_E2E_FormatJSON(t *testing.T) {
	docsDir := writeFiveDocs(t)
	streams, out, _ := streamsForPreview()
	cfg := &config.Config{}
	path := docsDir
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--preview", "--format", "json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("preview --format=json must succeed, got: %v", err)
	}

	var report reviewPreviewReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("cobra --preview --format=json output must be valid JSON: %v\nraw: %s", err, out.String())
	}
	if report.SchemaVersion == "" {
		t.Error("schema_version missing from e2e JSON output")
	}
	if report.Mode != "preview" {
		t.Errorf("mode = %q, want 'preview'", report.Mode)
	}
	if report.CorpusDocs != 5 {
		t.Errorf("corpus_documents = %d, want 5 (five .md files in temp dir)", report.CorpusDocs)
	}
}

// Broken-pipe errors from closed downstream readers must be swallowed so
// piping `--preview | head -1` doesn't spam stderr with a command error or
// change the exit code. Any other write error must still propagate so real
// failures (full disk, closed fd) are not hidden.
func TestSwallowBrokenPipe(t *testing.T) {
	if swallowBrokenPipe(nil) != nil {
		t.Error("nil input must return nil")
	}
	// Direct syscall.EPIPE.
	if err := swallowBrokenPipe(syscall.EPIPE); err != nil {
		t.Errorf("syscall.EPIPE must be swallowed, got %v", err)
	}
	// Wrapped EPIPE (os.PathError is what os.File.Write returns on EPIPE).
	wrapped := &os.PathError{Op: "write", Path: "/dev/stdout", Err: syscall.EPIPE}
	if err := swallowBrokenPipe(wrapped); err != nil {
		t.Errorf("wrapped EPIPE must be swallowed, got %v", err)
	}
	// String-matched "broken pipe" from a non-syscall source.
	if err := swallowBrokenPipe(errors.New("write |1: broken pipe")); err != nil {
		t.Errorf("broken pipe string match must be swallowed, got %v", err)
	}
	// Unrelated errors must propagate.
	other := errors.New("disk full")
	if err := swallowBrokenPipe(other); err == nil {
		t.Error("unrelated error must propagate, got nil")
	}
}

// --format without --preview must error out.
// Silently ignoring --format would hide CI mistakes where the operator
// expected JSON for downstream parsing and got the text report instead.
func TestReviewCmd_Format_WithoutPreview_Rejected(t *testing.T) {
	streams, _, _ := streamsForPreview()
	cfg := &config.Config{}
	path := ""
	cmd := newAngelaReviewCmd(cfg, streams, &path)
	cmd.SetArgs([]string{"--format", "json"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --format is used without --preview")
	}
	if !strings.Contains(err.Error(), "--format") || !strings.Contains(err.Error(), "--preview") {
		t.Errorf("error must explain the flag dependency, got: %v", err)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

// extractInputTokens finds the integer after "input" in a line like
// "Estimated tokens: 1,234 input → 4,000 output max" and returns it.
// Commas in the number are stripped.
func extractInputTokens(t *testing.T, report string) int {
	t.Helper()
	for _, line := range strings.Split(report, "\n") {
		if !strings.Contains(line, "Estimated tokens:") {
			continue
		}
		// Extract the digits+commas before "input".
		i := strings.Index(line, "input")
		if i < 0 {
			continue
		}
		seg := strings.TrimSpace(line[:i])
		// take the last whitespace-separated field
		fields := strings.Fields(seg)
		if len(fields) == 0 {
			continue
		}
		raw := strings.ReplaceAll(fields[len(fields)-1], ",", "")
		var n int
		_, err := fmtSscanf(raw, &n)
		if err != nil {
			continue
		}
		return n
	}
	t.Fatalf("could not find 'Estimated tokens:' line in report:\n%s", report)
	return 0
}

// fmtSscanf is a tiny indirection so the test file does not need to import
// "fmt" for a single call (we already import the ones we use).
func fmtSscanf(s string, n *int) (int, error) {
	var v int
	consumed := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		v = v*10 + int(r-'0')
		consumed++
	}
	if consumed == 0 {
		return 0, errBadInt
	}
	*n = v
	return consumed, nil
}

var errBadInt = bytesErr("bad int")

type bytesErr string

func (b bytesErr) Error() string { return string(b) }
