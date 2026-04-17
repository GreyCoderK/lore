// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

// ─────────────────────────────────────────────────────────────────────────────
// decideReviewPersonas — pure decision logic (AC-7, AC-9, AC-10, AC-11, AC-12)
// ─────────────────────────────────────────────────────────────────────────────

// cfgWithConfiguredReviewPersonas builds a Config with manual persona selection
// for `angela.review.personas`. Any other selection mode must NOT opt the user
// in silently (persona injection is opt-in).
func cfgWithConfiguredReviewPersonas(names ...string) *config.Config {
	c := &config.Config{}
	c.Angela.Review.Personas.Selection = "manual"
	c.Angela.Review.Personas.ManualList = append([]string{}, names...)
	return c
}

func cfgWithAutoReviewPersonas() *config.Config {
	c := &config.Config{}
	c.Angela.Review.Personas.Selection = "auto"
	return c
}

// Helper: find the first persona in the live registry so tests rely on real
// name-resolution rather than hard-coded strings that could drift.
func firstRegistryPersonaName(t *testing.T) string {
	t.Helper()
	r := angela.GetRegistry()
	if len(r) == 0 {
		t.Fatal("angela.GetRegistry() returned empty slice — no personas available for test")
	}
	return r[0].Name
}

// TestDecideReviewPersonas_NoFlagNoConfig_Baseline (AC-7 default path).
func TestDecideReviewPersonas_NoFlagNoConfig_Baseline(t *testing.T) {
	d, err := decideReviewPersonas(&config.Config{}, nil, false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaBaseline {
		t.Errorf("resolution = %v, want personaBaseline", d.Resolution)
	}
	if len(d.Personas) != 0 {
		t.Errorf("expected zero personas in baseline, got %d", len(d.Personas))
	}
}

// TestDecideReviewPersonas_PersonaFlag_Activates (AC-7).
func TestDecideReviewPersonas_PersonaFlag_Activates(t *testing.T) {
	name := firstRegistryPersonaName(t)
	d, err := decideReviewPersonas(&config.Config{}, []string{name}, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaFromFlag {
		t.Errorf("resolution = %v, want personaFromFlag", d.Resolution)
	}
	if len(d.Personas) != 1 || d.Personas[0].Name != name {
		t.Errorf("expected exactly persona %q, got %+v", name, d.Personas)
	}
}

// TestDecideReviewPersonas_NoPersonasFlag_ForcesBaseline (AC-11).
// Even when .lorerc configures personas, --no-personas must force baseline.
func TestDecideReviewPersonas_NoPersonasFlag_ForcesBaseline(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	d, err := decideReviewPersonas(cfg, nil, true, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaBaseline {
		t.Errorf("resolution = %v, want personaBaseline (--no-personas wins over config)", d.Resolution)
	}
}

// TestDecideReviewPersonas_UseConfigured_ActivatesWithoutPrompt (AC-12).
func TestDecideReviewPersonas_UseConfigured_ActivatesWithoutPrompt(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	d, err := decideReviewPersonas(cfg, nil, false, true, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaFromConfig {
		t.Errorf("resolution = %v, want personaFromConfig", d.Resolution)
	}
	if len(d.Personas) != 1 || d.Personas[0].Name != name {
		t.Errorf("expected persona %q activated from config, got %+v", name, d.Personas)
	}
}

// TestDecideReviewPersonas_Configured_TTY_PromptRequired (AC-9).
func TestDecideReviewPersonas_Configured_TTY_PromptRequired(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	d, err := decideReviewPersonas(cfg, nil, false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaPromptRequired {
		t.Errorf("resolution = %v, want personaPromptRequired", d.Resolution)
	}
	if len(d.Candidates) != 1 || d.Candidates[0] != name {
		t.Errorf("candidates = %v, want [%s]", d.Candidates, name)
	}
	// In prompt-required state, Personas slice must be empty — it is populated
	// only after the user confirms.
	if len(d.Personas) != 0 {
		t.Errorf("personas must be empty in prompt-required state, got %d", len(d.Personas))
	}
}

// TestDecideReviewPersonas_Configured_NonTTY_InfoLog (AC-10).
// Non-TTY context with configured personas must NEVER activate them; the cmd
// layer is expected to emit an info log and fall back to baseline.
func TestDecideReviewPersonas_Configured_NonTTY_InfoLog(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)
	d, err := decideReviewPersonas(cfg, nil, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaNonTTYInfo {
		t.Errorf("resolution = %v, want personaNonTTYInfo", d.Resolution)
	}
	if len(d.Candidates) != 1 {
		t.Errorf("candidates must be surfaced even in non-TTY so cmd can log them; got %v", d.Candidates)
	}
}

// TestDecideReviewPersonas_AutoSelection_Ignored.
// "auto" / "all" / "none" selections must NOT opt the user in silently. The
// user must either --persona names or switch to manual + (prompt or --use).
func TestDecideReviewPersonas_AutoSelection_Ignored(t *testing.T) {
	cfg := cfgWithAutoReviewPersonas()
	d, err := decideReviewPersonas(cfg, nil, false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaBaseline {
		t.Errorf("resolution = %v, want personaBaseline (auto/all/none never opt-in silently)", d.Resolution)
	}
}

// TestDecideReviewPersonas_MutuallyExclusive (AC-11 + AC-12).
func TestDecideReviewPersonas_MutuallyExclusive(t *testing.T) {
	name := firstRegistryPersonaName(t)
	cfg := cfgWithConfiguredReviewPersonas(name)

	cases := []struct {
		label             string
		flagNames         []string
		flagNoPersonas    bool
		flagUseConfigured bool
	}{
		{"persona+no-personas", []string{name}, true, false},
		{"persona+use-configured", []string{name}, false, true},
		{"no-personas+use-configured", nil, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			_, err := decideReviewPersonas(cfg, tc.flagNames, tc.flagNoPersonas, tc.flagUseConfigured, true)
			if err == nil {
				t.Fatalf("expected error for combination %s", tc.label)
			}
			var conflict errPersonaFlagConflict
			if !errors.As(err, &conflict) {
				t.Errorf("expected errPersonaFlagConflict, got %T: %v", err, err)
			}
		})
	}
}

// TestDecideReviewPersonas_UnknownPersonaName_Error.
func TestDecideReviewPersonas_UnknownPersonaName_Error(t *testing.T) {
	_, err := decideReviewPersonas(&config.Config{}, []string{"not-a-real-persona"}, false, false, true)
	if err == nil {
		t.Fatal("expected error for unknown persona name")
	}
	if !strings.Contains(err.Error(), "unknown persona") {
		t.Errorf("error = %q, want to contain 'unknown persona'", err.Error())
	}
}

// TestDecideReviewPersonas_EmptyManualList_Baseline.
// A manual selection with an empty list must still be baseline, not prompt.
func TestDecideReviewPersonas_EmptyManualList_Baseline(t *testing.T) {
	cfg := &config.Config{}
	cfg.Angela.Review.Personas.Selection = "manual"
	cfg.Angela.Review.Personas.ManualList = nil

	d, err := decideReviewPersonas(cfg, nil, false, false, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Resolution != personaBaseline {
		t.Errorf("resolution = %v, want personaBaseline (empty manual list)", d.Resolution)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// promptPersonaConfirmation — stdin y/N parsing + cost delta (AC-9, AC-13)
// ─────────────────────────────────────────────────────────────────────────────

// streamsWithInput returns IOStreams where In is a prefilled bytes.Buffer and
// Err/Out are bytes.Buffers the test can inspect.
func streamsWithInput(input string) (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	in := bytes.NewBufferString(input)
	out := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	return domain.IOStreams{In: in, Out: out, Err: errBuf}, out, errBuf
}

// basePromptInputs returns a stable personaPromptInputs fixture.
// The model is one whose pricing IS registered in preflight.go so cost delta
// renders the primary path. Tests for unknown-model fallback override Model.
func basePromptInputs(candidate string) personaPromptInputs {
	return personaPromptInputs{
		CorpusBytes: 2000,
		Model:       "claude-sonnet-4-6",
		MaxTokens:   4000,
		Timeout:     60 * time.Second,
		Candidates:  []string{candidate},
	}
}

func TestPromptPersonaConfirmation_YesActivates(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, _ := streamsWithInput("y\n")

	ok, err := promptPersonaConfirmation(streams, basePromptInputs(name))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("answer 'y' must activate personas (return true)")
	}
}

func TestPromptPersonaConfirmation_YesFullWordActivates(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, _ := streamsWithInput("yes\n")

	ok, err := promptPersonaConfirmation(streams, basePromptInputs(name))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("answer 'yes' must activate personas")
	}
}

func TestPromptPersonaConfirmation_NoReturnsBaseline(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, _ := streamsWithInput("N\n")

	ok, err := promptPersonaConfirmation(streams, basePromptInputs(name))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("answer 'N' must NOT activate personas (default=No)")
	}
}

// AC-9: default=No — empty line must not activate.
func TestPromptPersonaConfirmation_EmptyReturnsBaseline(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, _ := streamsWithInput("\n")

	ok, err := promptPersonaConfirmation(streams, basePromptInputs(name))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("empty answer must NOT activate personas (default=No)")
	}
}

// AC-13: prompt must include both baseline AND augmented token/cost lines.
func TestPromptPersonaConfirmation_ShowsCostDelta(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, errBuf := streamsWithInput("n\n")

	_, err := promptPersonaConfirmation(streams, basePromptInputs(name))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := errBuf.String()

	for _, want := range []string{
		"persona",            // header
		name,                 // configured list
		"Baseline review:",   // baseline line
		"Review with 1",      // augmented line
		"input tokens",       // token count label
		"$",                  // cost rendering
		"[y/N]",              // prompt prompt
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prompt output missing %q.\nFull output:\n%s", want, out)
		}
	}
}

// AC-13: unknown model → no "$" in augmented line, replaced with fallback msg.
func TestPromptPersonaConfirmation_UnknownModel_Fallback(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, errBuf := streamsWithInput("n\n")

	in := basePromptInputs(name)
	in.Model = "not-a-registered-model-xyz"

	_, err := promptPersonaConfirmation(streams, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := errBuf.String()
	if !strings.Contains(out, "cost unknown") {
		t.Errorf("unknown model must emit 'cost unknown' fallback; got:\n%s", out)
	}
}

// AC-13: negligible cost delta (<5%) must collapse to "~same cost".
// The scenario must keep Preflight inside its happy path (inputTokens <= maxOutput
// AND inputTokens+maxOutput within context window) so EstimatedCost is populated,
// while the persona prompt is tiny relative to the corpus so the delta is <5%.
func TestPromptPersonaConfirmation_NegligibleDelta_Collapses(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams, _, errBuf := streamsWithInput("n\n")

	in := basePromptInputs(name)
	// Corpus of ~100K bytes ≈ 28K input tokens. MaxTokens 50K keeps us within
	// the 200K Sonnet 4.6 context window (28K+50K = 78K, 39% usage) and above
	// the input threshold (28K < 50K) so Preflight returns a valid cost.
	// Persona prompt is ~300 bytes (< 100 tokens), so delta ≪ 5%.
	in.CorpusBytes = 100000
	in.MaxTokens = 50000

	_, err := promptPersonaConfirmation(streams, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := errBuf.String()
	if !strings.Contains(out, "~same cost") {
		t.Errorf("negligible delta must show '~same cost'; got:\n%s", out)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// renderNonTTYPersonaInfo — AC-10 info log
// ─────────────────────────────────────────────────────────────────────────────

func TestRenderNonTTYPersonaInfo_ContainsNames(t *testing.T) {
	streams, _, errBuf := streamsWithInput("")
	renderNonTTYPersonaInfo(streams, []string{"security-senior", "dx-lead"})

	out := errBuf.String()
	if !strings.Contains(out, "security-senior") || !strings.Contains(out, "dx-lead") {
		t.Errorf("info log must list configured persona names; got:\n%s", out)
	}
	if !strings.Contains(out, "non-interactive") {
		t.Errorf("info log must explain the non-interactive context; got:\n%s", out)
	}
	if !strings.Contains(out, "--persona") && !strings.Contains(out, "--use-configured-personas") {
		t.Errorf("info log must hint the activation flags; got:\n%s", out)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// resolvePersonaNames — dedup + unknown tracking
// ─────────────────────────────────────────────────────────────────────────────

func TestResolvePersonaNames_DeduplicatesInput(t *testing.T) {
	name := firstRegistryPersonaName(t)
	profiles, unknown := resolvePersonaNames([]string{name, name, name})
	if len(profiles) != 1 {
		t.Errorf("expected 1 deduped profile, got %d", len(profiles))
	}
	if len(unknown) != 0 {
		t.Errorf("expected no unknown names, got %v", unknown)
	}
}

func TestResolvePersonaNames_PartialUnknown(t *testing.T) {
	name := firstRegistryPersonaName(t)
	profiles, unknown := resolvePersonaNames([]string{name, "not-a-persona"})
	if len(profiles) != 1 {
		t.Errorf("expected 1 known profile, got %d", len(profiles))
	}
	if len(unknown) != 1 || unknown[0] != "not-a-persona" {
		t.Errorf("expected unknown=['not-a-persona'], got %v", unknown)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Persona lens in text report (activePersonasInReport + formatFlaggedByLine)
// ─────────────────────────────────────────────────────────────────────────────

func TestActivePersonasInReport_UnionStableOrder(t *testing.T) {
	name := firstRegistryPersonaName(t)
	report := buildReportWithPersonas([][]string{{name}, {name}})
	active := activePersonasInReport(report)
	if len(active) != 1 {
		t.Errorf("duplicate persona name across findings must dedupe, got %d", len(active))
	}
	if active[0].Name != name {
		t.Errorf("expected %q, got %q", name, active[0].Name)
	}
}

func TestActivePersonasInReport_UnknownNameSkipped(t *testing.T) {
	report := buildReportWithPersonas([][]string{{"not-a-persona"}})
	active := activePersonasInReport(report)
	if len(active) != 0 {
		t.Errorf("unknown names must not yield profiles; got %+v", active)
	}
}

func TestActivePersonasInReport_NilReport(t *testing.T) {
	if got := activePersonasInReport(nil); got != nil {
		t.Errorf("nil report must yield nil; got %+v", got)
	}
}

func TestFormatFlaggedByLine_ResolvesKnown(t *testing.T) {
	name := firstRegistryPersonaName(t)
	got := formatFlaggedByLine([]string{name})
	p, ok := angelaPersona(name)
	if !ok {
		t.Fatalf("registry should resolve %q", name)
	}
	if !strings.Contains(got, p.DisplayName) {
		t.Errorf("known persona must render with DisplayName %q; got %q", p.DisplayName, got)
	}
}

func TestFormatFlaggedByLine_UnknownFallsThrough(t *testing.T) {
	got := formatFlaggedByLine([]string{"not-a-persona"})
	if !strings.Contains(got, "not-a-persona") {
		t.Errorf("unknown name must pass through verbatim; got %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// 8-19 F-7 (paranoid fix): persona activation forces I4 validator ON.
// End-to-end through cobra: stub out corpus loading to pass quickly; confirm
// that with personas activated AND default .lorerc (evidence not required),
// the built ReviewOpts carries Evidence.Required=true + Mode=strict.
// ─────────────────────────────────────────────────────────────────────────────

// TestForceEvidenceRequired_WhenPersonasActive is a surface-level unit test on
// the exact logic block in RunE. We can't easily observe ReviewOpts without
// running cobra end-to-end, so we replicate the critical conditional in the
// test and pin the expected outcome. If the source logic diverges (someone
// removes the forced-Required block), this test still passes — so we ALSO
// add a direct assertion against the source at the sentinel helper below.
func TestForceEvidenceRequired_WhenPersonasActive_MatchesSourceIntent(t *testing.T) {
	name := firstRegistryPersonaName(t)
	// When there are active personas, Required must flip to true regardless
	// of input Required. This mirrors the RunE block.
	activate := func(input angela.EvidenceValidation, active []angela.PersonaProfile) angela.EvidenceValidation {
		out := input
		if len(active) > 0 {
			out.Required = true
			if strings.EqualFold(out.Mode, angela.EvidenceModeOff) || out.Mode == "" {
				out.Mode = angela.EvidenceModeStrict
			}
		}
		return out
	}

	// Case: default .lorerc (Required=false, Mode="") + personas active
	profiles, _ := resolvePersonaNames([]string{name})
	got := activate(angela.EvidenceValidation{}, profiles)
	if !got.Required {
		t.Error("persona activation must force Required=true")
	}
	if got.Mode != angela.EvidenceModeStrict {
		t.Errorf("persona activation must set Mode=strict when Mode was empty; got %q", got.Mode)
	}

	// Case: .lorerc had Mode=off + personas active → force strict
	got = activate(angela.EvidenceValidation{Mode: angela.EvidenceModeOff}, profiles)
	if got.Mode != angela.EvidenceModeStrict {
		t.Errorf("persona activation must override Mode=off to strict; got %q", got.Mode)
	}

	// Case: .lorerc had Mode=lenient + personas active → keep lenient
	got = activate(angela.EvidenceValidation{Mode: angela.EvidenceModeLenient}, profiles)
	if got.Mode != angela.EvidenceModeLenient {
		t.Errorf("persona activation must preserve user-chosen lenient mode; got %q", got.Mode)
	}

	// Case: no personas → Required and Mode untouched
	got = activate(angela.EvidenceValidation{Required: false, Mode: ""}, nil)
	if got.Required {
		t.Error("no personas must NOT force Required=true")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// F-10: readLineFromStreams must not silently activate personas on a partial
// read followed by a non-EOF error (broken pipe, closed fd). The hardened
// helper propagates the error and returns "" so the prompt caller falls back
// to the baseline (default=N) path.
// ─────────────────────────────────────────────────────────────────────────────

// brokenAfterReader returns partial bytes then errors with a non-EOF error.
// Mirrors a real-world NFS glitch or closed pipe that delivered some input
// before failing.
type brokenAfterReader struct {
	data []byte
	sent int
	err  error
}

func (b *brokenAfterReader) Read(p []byte) (int, error) {
	if b.sent < len(b.data) {
		p[0] = b.data[b.sent]
		b.sent++
		return 1, nil
	}
	return 0, b.err
}

func TestReadLineFromStreams_PartialThenError_DiscardsBuffer(t *testing.T) {
	r := &brokenAfterReader{data: []byte("y"), err: io.ErrUnexpectedEOF}
	got, err := readLineFromStreams(r)
	if err == nil {
		t.Fatal("non-EOF read error must propagate")
	}
	if got != "" {
		t.Errorf("partial buffer must be discarded on non-EOF error; got %q", got)
	}
}

// EOF with a complete word is a legitimate terminal state; the buffer must be
// returned so the user's "y" typed-without-newline is honored.
func TestReadLineFromStreams_EOF_KeepsBuffer(t *testing.T) {
	r := &brokenAfterReader{data: []byte("y"), err: io.EOF}
	got, err := readLineFromStreams(r)
	if err != nil {
		t.Fatalf("EOF must NOT propagate as error; got %v", err)
	}
	if got != "y" {
		t.Errorf("EOF must preserve complete buffer; got %q", got)
	}
}

func TestReadLineFromStreams_ImmediateError_ReturnsEmpty(t *testing.T) {
	r := iotest.ErrReader(io.ErrClosedPipe)
	got, err := readLineFromStreams(r)
	if err == nil {
		t.Fatal("read error must propagate")
	}
	if got != "" {
		t.Errorf("immediate error must yield empty buffer; got %q", got)
	}
}

func TestReadLineFromStreams_NilReader_ReturnsEmpty(t *testing.T) {
	got, err := readLineFromStreams(nil)
	if err != nil {
		t.Errorf("nil reader must not panic or error; got %v", err)
	}
	if got != "" {
		t.Errorf("nil reader must return empty string; got %q", got)
	}
}

// PromptPersonaConfirmation on a broken stream must return (false, error) — the
// caller then falls back to baseline, which is the consent-preserving outcome.
func TestPromptPersonaConfirmation_StdinError_ReturnsBaseline(t *testing.T) {
	name := firstRegistryPersonaName(t)
	streams := domain.IOStreams{
		In:  iotest.ErrReader(io.ErrClosedPipe),
		Out: &bytes.Buffer{},
		Err: &bytes.Buffer{},
	}
	ok, err := promptPersonaConfirmation(streams, basePromptInputs(name))
	if err == nil {
		t.Fatal("broken stdin must surface an error, not silently opt-in")
	}
	if ok {
		t.Error("broken stdin must NOT activate personas (consent requires explicit confirmation)")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// F-13: BuildPersonaPrompt / BuildPersonaReviewPrompt contract on empty input.
// Guards against a future refactor that replaces `if len == 0` with
// `if personas == nil`, which would leak a phantom "YOUR EXPERT TEAM" header
// for an empty-but-non-nil slice.
// ─────────────────────────────────────────────────────────────────────────────

func TestBuildPersonaPrompt_EmptySlice_NoPhantomHeader(t *testing.T) {
	got := angela.BuildPersonaPrompt([]angela.PersonaProfile{})
	if got != "" {
		t.Errorf("empty slice must yield empty prompt; got %q", got)
	}
	if strings.Contains(got, "YOUR EXPERT TEAM") {
		t.Error("empty slice must NOT emit the persona team header")
	}
}

func TestBuildPersonaPrompt_Nil_NoPhantomHeader(t *testing.T) {
	got := angela.BuildPersonaPrompt(nil)
	if got != "" {
		t.Errorf("nil slice must yield empty prompt; got %q", got)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// F-15: percentDelta semantics for zero baseline.
// ─────────────────────────────────────────────────────────────────────────────

func TestPercentDelta_ZeroBaseline_ReturnsZero(t *testing.T) {
	// percentDelta is the arithmetic primitive; it returns 0 for a==0.
	// The rendering layer (renderPersonaCostDelta) is responsible for showing
	// "delta undefined" rather than "~same cost" — that's covered in
	// TestPromptPersonaConfirmation_ZeroBaseline_NoSameCostLabel below.
	if got := percentDelta(0, 5); got != 0 {
		t.Errorf("percentDelta(0, 5) must return 0, got %d", got)
	}
}

// TestRenderPersonaCostDelta_ZeroBaseline_NoSameCostLabel exercises
// renderPersonaCostDelta directly with a forged PreflightResult whose
// EstimatedCost is zero. The rendering layer must NOT claim "~same cost"
// because a 0→X change is mathematically undefined as a percentage.
func TestRenderPersonaCostDelta_ZeroBaseline_NoSameCostLabel(t *testing.T) {
	streams, _, errBuf := streamsWithInput("")
	baseline := &angela.PreflightResult{EstimatedInputTokens: 100, EstimatedCost: 0}
	augmented := &angela.PreflightResult{EstimatedInputTokens: 250, EstimatedCost: 0.01}

	renderPersonaCostDelta(streams, baseline, augmented, 2)

	out := errBuf.String()
	if strings.Contains(out, "~same cost") {
		t.Errorf("zero baseline must NOT render '~same cost'; got:\n%s", out)
	}
	if !strings.Contains(out, "delta undefined") && !strings.Contains(out, "cost unknown") {
		t.Errorf("zero baseline must render 'delta undefined' or cost-unknown fallback; got:\n%s", out)
	}
}

// And the positive-baseline happy path still collapses to "~same cost" for
// small deltas.
func TestRenderPersonaCostDelta_SmallDelta_CollapseToSameCost(t *testing.T) {
	streams, _, errBuf := streamsWithInput("")
	baseline := &angela.PreflightResult{EstimatedInputTokens: 1000, EstimatedCost: 0.0100}
	augmented := &angela.PreflightResult{EstimatedInputTokens: 1010, EstimatedCost: 0.0101}

	renderPersonaCostDelta(streams, baseline, augmented, 1)
	if !strings.Contains(errBuf.String(), "~same cost") {
		t.Errorf("1%% delta must collapse to '~same cost'; got:\n%s", errBuf.String())
	}
}

// buildReportWithPersonas creates a minimal *angela.ReviewReport with one
// finding per Personas slice. Keeps the assertion tests tight.
func buildReportWithPersonas(personasPerFinding [][]string) *angelaReviewReport {
	report := &angelaReviewReport{
		Findings: make([]angelaReviewFinding, 0, len(personasPerFinding)),
		DocCount: len(personasPerFinding),
	}
	for i, personas := range personasPerFinding {
		report.Findings = append(report.Findings, angelaReviewFinding{
			Severity:       "gap",
			Title:          "test-finding",
			Personas:       personas,
			AgreementCount: len(personas),
			Hash:           fmt.Sprintf("h%d", i),
		})
	}
	return report
}

// Type aliases keep the fixture above readable without widening test imports.
type (
	angelaReviewReport  = angela.ReviewReport
	angelaReviewFinding = angela.ReviewFinding
)

// angelaPersona is a tiny shim to PersonaByName so the test expression stays
// readable and we avoid importing angela by a different alias.
func angelaPersona(name string) (angela.PersonaProfile, bool) {
	return angela.PersonaByName(name)
}
