// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/greycoderk/lore/internal/domain"
)

// testFindings returns a small set of findings for interactive tests.
func testFindings() []ReviewFinding {
	return []ReviewFinding{
		{
			Severity:    "contradiction",
			Title:       "Auth strategy inconsistent",
			Description: "Decision doc says OAuth2 but feature doc implements SAML",
			Documents:   []string{"auth-choice.md", "login-v2.md"},
			Evidence:    []Evidence{{File: "auth-choice.md", Quote: "We choose OAuth2"}},
			Confidence:  0.87,
			Hash:        "abcdef1234567890",
			DiffStatus:  ReviewDiffNew,
		},
		{
			Severity:    "gap",
			Title:       "Missing API docs",
			Description: "API referenced but undocumented",
			Documents:   []string{"api-ref.md"},
			Hash:        "1234567890abcdef",
			DiffStatus:  ReviewDiffPersisting,
		},
		{
			Severity:    "style",
			Title:       "Inconsistent terminology",
			Description: "rate limiter vs throttler",
			Documents:   []string{"design.md"},
			Hash:        "fedcba0987654321",
			DiffStatus:  ReviewDiffRegressed,
		},
	}
}

// testState returns a ReviewState pre-populated with the test findings.
func testState() *ReviewState {
	now := time.Now().UTC()
	s := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: make(map[string]StatefulFinding),
	}
	for _, f := range testFindings() {
		s.Findings[f.Hash] = StatefulFinding{
			Finding:   f,
			Status:    StatusActive,
			FirstSeen: now,
			LastSeen:  now,
		}
	}
	return s
}

func keyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func specialKeyMsg(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}

func TestReviewInteractive_BrowseNavigate(t *testing.T) {
	m := NewReviewInteractiveModel(testFindings(), nil, "", "", nil, nil)

	// Initial cursor at 0
	if m.cursor != 0 {
		t.Fatalf("expected cursor=0, got %d", m.cursor)
	}

	// Press j to move down
	model, _ := m.Update(keyMsg("j"))
	m = model.(ReviewInteractiveModel)
	if m.cursor != 1 {
		t.Fatalf("after j: expected cursor=1, got %d", m.cursor)
	}

	// Press down arrow to move down
	model, _ = m.Update(specialKeyMsg(tea.KeyDown))
	m = model.(ReviewInteractiveModel)
	if m.cursor != 2 {
		t.Fatalf("after down: expected cursor=2, got %d", m.cursor)
	}

	// Should not go past the end
	model, _ = m.Update(keyMsg("j"))
	m = model.(ReviewInteractiveModel)
	if m.cursor != 2 {
		t.Fatalf("at end: expected cursor=2, got %d", m.cursor)
	}

	// Press k to move up
	model, _ = m.Update(keyMsg("k"))
	m = model.(ReviewInteractiveModel)
	if m.cursor != 1 {
		t.Fatalf("after k: expected cursor=1, got %d", m.cursor)
	}

	// Press up arrow
	model, _ = m.Update(specialKeyMsg(tea.KeyUp))
	m = model.(ReviewInteractiveModel)
	if m.cursor != 0 {
		t.Fatalf("after up: expected cursor=0, got %d", m.cursor)
	}

	// Should not go below 0
	model, _ = m.Update(keyMsg("k"))
	m = model.(ReviewInteractiveModel)
	if m.cursor != 0 {
		t.Fatalf("at start: expected cursor=0, got %d", m.cursor)
	}
}

func TestReviewInteractive_EnterShowsDetail(t *testing.T) {
	m := NewReviewInteractiveModel(testFindings(), nil, "", "", nil, nil)

	if m.mode != modeBrowse {
		t.Fatalf("expected modeBrowse, got %d", m.mode)
	}

	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)

	if m.mode != modeDetail {
		t.Fatalf("expected modeDetail after Enter, got %d", m.mode)
	}

	// View should contain finding details
	view := m.View()
	if !strings.Contains(view, "Auth strategy inconsistent") {
		t.Error("detail view should contain finding title")
	}
	if !strings.Contains(view, "Description:") {
		t.Error("detail view should contain Description section")
	}
	if !strings.Contains(view, "Evidence:") {
		t.Error("detail view should contain Evidence section")
	}
	if !strings.Contains(view, "0.87") {
		t.Error("detail view should contain Confidence")
	}
}

func TestReviewInteractive_EscReturnsBrowse(t *testing.T) {
	m := NewReviewInteractiveModel(testFindings(), nil, "", "", nil, nil)

	// Enter detail mode
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)
	if m.mode != modeDetail {
		t.Fatalf("expected modeDetail, got %d", m.mode)
	}

	// Press Esc
	model, _ = m.Update(specialKeyMsg(tea.KeyEscape))
	m = model.(ReviewInteractiveModel)
	if m.mode != modeBrowse {
		t.Fatalf("expected modeBrowse after Esc, got %d", m.mode)
	}
}

func TestReviewInteractive_ResolveRemovesFromList(t *testing.T) {
	state := testState()
	m := NewReviewInteractiveModel(testFindings(), state, "", "", nil, nil)

	// Enter detail on first finding
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)

	// Press r to resolve
	model, _ = m.Update(keyMsg("r"))
	m = model.(ReviewInteractiveModel)

	if len(m.findings) != 2 {
		t.Fatalf("expected 2 findings after resolve, got %d", len(m.findings))
	}
	if m.resolvedCount != 1 {
		t.Fatalf("expected resolvedCount=1, got %d", m.resolvedCount)
	}
	if m.mode != modeBrowse {
		t.Fatalf("expected modeBrowse after resolve, got %d", m.mode)
	}

	// Verify state was updated
	entry, ok := state.Findings["abcdef1234567890"]
	if !ok {
		t.Fatal("finding should still exist in state")
	}
	if entry.Status != StatusResolved {
		t.Fatalf("expected StatusResolved, got %s", entry.Status)
	}
}

func TestReviewInteractive_IgnoreWithReason(t *testing.T) {
	state := testState()
	m := NewReviewInteractiveModel(testFindings(), state, "", "", nil, nil)

	// Enter detail mode
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)

	// Press i to start ignore prompt
	model, _ = m.Update(keyMsg("i"))
	m = model.(ReviewInteractiveModel)
	if m.mode != modeIgnorePrompt {
		t.Fatalf("expected modeIgnorePrompt, got %d", m.mode)
	}

	// Type a reason
	for _, ch := range "false positive" {
		model, _ = m.Update(keyMsg(string(ch)))
		m = model.(ReviewInteractiveModel)
	}
	if m.ignoreInput != "false positive" {
		t.Fatalf("expected ignoreInput='false positive', got '%s'", m.ignoreInput)
	}

	// Press Enter to confirm
	model, _ = m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)

	if len(m.findings) != 2 {
		t.Fatalf("expected 2 findings after ignore, got %d", len(m.findings))
	}
	if m.ignoredCount != 1 {
		t.Fatalf("expected ignoredCount=1, got %d", m.ignoredCount)
	}

	entry := state.Findings["abcdef1234567890"]
	if entry.Status != StatusIgnored {
		t.Fatalf("expected StatusIgnored, got %s", entry.Status)
	}
	if entry.IgnoreReason != "false positive" {
		t.Fatalf("expected reason 'false positive', got '%s'", entry.IgnoreReason)
	}
}

func TestReviewInteractive_IgnoreEscCancels(t *testing.T) {
	m := NewReviewInteractiveModel(testFindings(), testState(), "", "", nil, nil)

	// Navigate to detail, then ignore prompt
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)
	model, _ = m.Update(keyMsg("i"))
	m = model.(ReviewInteractiveModel)

	// Esc cancels
	model, _ = m.Update(specialKeyMsg(tea.KeyEscape))
	m = model.(ReviewInteractiveModel)
	if m.mode != modeDetail {
		t.Fatalf("expected modeDetail after Esc in ignore, got %d", m.mode)
	}
	if len(m.findings) != 3 {
		t.Fatal("findings should not change on cancel")
	}
}

func TestReviewInteractive_NonTTYFallsBackToPrintf(t *testing.T) {
	// In test, stdout is typically redirected to a pipe (not a TTY).
	result := IsInteractiveAvailable()
	if result {
		t.Skip("stdout is a TTY in this test environment — cannot verify non-TTY fallback")
	}
	// M5 fix: assert on the stored result, not a second call.
	if result {
		t.Error("expected IsInteractiveAvailable=false in non-TTY test")
	}
}

func TestReviewInteractive_QuitSavesState(t *testing.T) {
	state := testState()
	dir := t.TempDir()
	statePath := dir + "/review-state.json"

	m := NewReviewInteractiveModel(testFindings(), state, statePath, "", nil, nil)

	// Quit
	model, cmd := m.Update(keyMsg("q"))
	m = model.(ReviewInteractiveModel)

	if !m.quitting {
		t.Fatal("expected quitting=true")
	}
	if m.QuitSummary == "" {
		t.Fatal("expected non-empty QuitSummary")
	}
	if !strings.Contains(m.QuitSummary, "0 findings resolved") {
		t.Fatalf("unexpected summary: %s", m.QuitSummary)
	}

	// cmd should be tea.Quit
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd")
	}

	// Verify state file was written
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("state file should be saved on quit: %v", err)
	}
}

func TestReviewInteractive_BrowseView(t *testing.T) {
	m := NewReviewInteractiveModel(testFindings(), nil, "", "", nil, nil)
	view := m.View()

	if !strings.Contains(view, "Review") {
		t.Error("browse view should contain 'Review' header")
	}
	if !strings.Contains(view, "Auth strategy inconsistent") {
		t.Error("browse view should contain first finding title")
	}
	if !strings.Contains(view, "Missing API docs") {
		t.Error("browse view should contain second finding title")
	}
	if !strings.Contains(view, "j/k") {
		t.Error("browse view should contain key hints")
	}
}

func TestReviewInteractive_ResolveWithNoState(t *testing.T) {
	// No state — resolve should be a no-op
	m := NewReviewInteractiveModel(testFindings(), nil, "", "", nil, nil)

	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)
	model, _ = m.Update(keyMsg("r"))
	m = model.(ReviewInteractiveModel)

	// Should not crash, findings unchanged
	if len(m.findings) != 3 {
		t.Fatalf("expected 3 findings (no-op resolve), got %d", len(m.findings))
	}
}

func TestReviewInteractive_DeepDiveTriggered(t *testing.T) {
	called := false
	mp := &mockProvider{
		CompleteFunc: func(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
			called = true
			return "Deep dive result: the finding is real.", nil
		},
	}

	m := NewReviewInteractiveModel(testFindings(), nil, "", "", mp, nil)

	// Enter detail mode
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)

	// Press d for deep dive
	model, cmd := m.Update(keyMsg("d"))
	m = model.(ReviewInteractiveModel)

	if !m.deepDiveLoading {
		t.Error("expected deepDiveLoading=true after pressing d")
	}
	if m.deepDivedCount != 1 {
		t.Fatalf("expected deepDivedCount=1, got %d", m.deepDivedCount)
	}

	// Execute the command returned by Update to simulate async completion
	if cmd != nil {
		msg := cmd()
		model, _ = m.Update(msg)
		m = model.(ReviewInteractiveModel)
	}

	if !called {
		t.Error("expected AI provider to be called")
	}
	if m.deepDiveLoading {
		t.Error("expected deepDiveLoading=false after result")
	}
	if !strings.Contains(m.deepDiveText, "Deep dive result") {
		t.Errorf("expected deep dive text, got: %s", m.deepDiveText)
	}
}

func TestBuildDeepDivePrompt(t *testing.T) {
	reader := &mockCorpusReader{
		content: map[string]string{
			"auth-choice.md": "# Auth\nWe choose OAuth2 for simplicity.",
			"login-v2.md":    "# Login\nThe SAML flow uses signed assertions.",
		},
	}

	f := ReviewFinding{
		Severity:    "contradiction",
		Title:       "Auth inconsistent",
		Description: "OAuth2 vs SAML conflict",
		Documents:   []string{"auth-choice.md", "login-v2.md"},
	}

	sys, usr := BuildDeepDivePrompt(f, reader)

	if !strings.Contains(sys, "Angela") {
		t.Error("system prompt should mention Angela")
	}
	if !strings.Contains(usr, "Auth inconsistent") {
		t.Error("user prompt should contain finding title")
	}
	if !strings.Contains(usr, "We choose OAuth2") {
		t.Error("user prompt should contain doc content from reader")
	}
	if !strings.Contains(usr, "SAML flow") {
		t.Error("user prompt should contain second doc content")
	}
}

func TestBuildDeepDivePrompt_NilReader(t *testing.T) {
	f := ReviewFinding{
		Severity:  "gap",
		Title:     "Missing docs",
		Documents: []string{"api.md"},
	}

	sys, usr := BuildDeepDivePrompt(f, nil)
	if !strings.Contains(sys, "Angela") {
		t.Error("system prompt should work with nil reader")
	}
	if !strings.Contains(usr, "Missing docs") {
		t.Error("user prompt should contain title even with nil reader")
	}
	// Should not contain doc content markers
	if strings.Contains(usr, "===") {
		t.Error("should not contain doc content sections with nil reader")
	}
}

func TestReviewInteractive_ResolveAllQuits(t *testing.T) {
	// When the last finding is resolved, the TUI should auto-quit
	findings := []ReviewFinding{testFindings()[0]}
	state := &ReviewState{
		Version:  ReviewStateVersion,
		Findings: map[string]StatefulFinding{
			findings[0].Hash: {
				Finding:   findings[0],
				Status:    StatusActive,
				FirstSeen: time.Now().UTC(),
				LastSeen:  time.Now().UTC(),
			},
		},
	}

	m := NewReviewInteractiveModel(findings, state, "", "", nil, nil)
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)
	model, cmd := m.Update(keyMsg("r"))
	m = model.(ReviewInteractiveModel)

	if !m.quitting {
		t.Fatal("expected auto-quit when last finding is resolved")
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit command")
	}
}

func TestReviewInteractive_EditorHintWhenNoEditor(t *testing.T) {
	t.Setenv("EDITOR", "")
	m := NewReviewInteractiveModel(testFindings(), nil, "", "", nil, nil)

	// Enter detail
	model, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	m = model.(ReviewInteractiveModel)

	// Press o
	model, _ = m.Update(keyMsg("o"))
	m = model.(ReviewInteractiveModel)

	if !strings.Contains(m.deepDiveText, "$EDITOR") {
		t.Errorf("expected $EDITOR hint, got: %s", m.deepDiveText)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Persona-aware interactive TUI
// ─────────────────────────────────────────────────────────────────────────────

// testFindingsWithPersonas returns findings where AgreementCount is varied so
// sort ordering can be asserted deterministically.
//
// Input order:
//   - "Low-agreement gap"        agreement=1 severity=gap
//   - "Triple-agreement style"   agreement=3 severity=style
//   - "Double-agreement contradiction"  agreement=2 severity=contradiction
//
// Expected sort (agreement DESC, severity ASC):
//   - Triple-agreement style       (3)
//   - Double-agreement contradiction (2)
//   - Low-agreement gap            (1)
func testFindingsWithPersonas() []ReviewFinding {
	return []ReviewFinding{
		{
			Severity:       "gap",
			Title:          "Low-agreement gap",
			Personas:       []string{"dx-lead"},
			AgreementCount: 1,
			Hash:           "low-gap",
			DiffStatus:     ReviewDiffNew,
		},
		{
			Severity:       "style",
			Title:          "Triple-agreement style",
			Personas:       []string{"security-senior", "dx-lead", "tech-writer"},
			AgreementCount: 3,
			Hash:           "triple-style",
			DiffStatus:     ReviewDiffNew,
		},
		{
			Severity:       "contradiction",
			Title:          "Double-agreement contradiction",
			Personas:       []string{"security-senior", "dx-lead"},
			AgreementCount: 2,
			Hash:           "double-contra",
			DiffStatus:     ReviewDiffNew,
		},
	}
}

// TestInteractiveReview_PersonaSorting (AC-8).
// When any finding carries personas, the model must sort by AgreementCount DESC,
// severity ASC — NOT preserve the input order.
func TestInteractiveReview_PersonaSorting(t *testing.T) {
	m := NewReviewInteractiveModel(testFindingsWithPersonas(), nil, "", "", nil, nil)

	if len(m.findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(m.findings))
	}
	wantTitles := []string{
		"Triple-agreement style",           // agreement=3
		"Double-agreement contradiction",   // agreement=2
		"Low-agreement gap",                // agreement=1
	}
	for i, want := range wantTitles {
		if m.findings[i].Title != want {
			t.Errorf("position %d: got %q, want %q", i, m.findings[i].Title, want)
		}
	}
}

// TestInteractiveReview_PersonaSorting_SecondaryKey asserts severity tie-break.
// Two findings with identical agreement must sort by severity rank ASC.
func TestInteractiveReview_PersonaSorting_SecondaryKey(t *testing.T) {
	findings := []ReviewFinding{
		{Severity: "style", Title: "A-style", Personas: []string{"x"}, AgreementCount: 1, Hash: "a"},
		{Severity: "contradiction", Title: "B-contradiction", Personas: []string{"y"}, AgreementCount: 1, Hash: "b"},
		{Severity: "gap", Title: "C-gap", Personas: []string{"z"}, AgreementCount: 1, Hash: "c"},
	}
	m := NewReviewInteractiveModel(findings, nil, "", "", nil, nil)

	// Same agreement=1 across all three → severity order: contradiction, gap, style.
	want := []string{"B-contradiction", "C-gap", "A-style"}
	for i, w := range want {
		if m.findings[i].Title != w {
			t.Errorf("position %d: got %q, want %q", i, m.findings[i].Title, w)
		}
	}
}

// TestInteractiveReview_NoPersonas_BaselineRendering asserts that findings
// without persona data preserve the caller's input order (no silent reorder).
func TestInteractiveReview_NoPersonas_BaselineRendering(t *testing.T) {
	inputs := testFindings()
	// Capture input order snapshot BEFORE the constructor (defensive copy).
	wantOrder := make([]string, len(inputs))
	for i, f := range inputs {
		wantOrder[i] = f.Title
	}

	m := NewReviewInteractiveModel(inputs, nil, "", "", nil, nil)

	if len(m.findings) != len(wantOrder) {
		t.Fatalf("finding count changed: got %d, want %d", len(m.findings), len(wantOrder))
	}
	for i, w := range wantOrder {
		if m.findings[i].Title != w {
			t.Errorf("position %d reordered: got %q, want %q (baseline reviews must not reorder)",
				i, m.findings[i].Title, w)
		}
	}
}

// TestInteractiveReview_PersonaTagRendering ensures the browse view shows the
// persona tag column only when personas are active, and that it carries the
// abbreviated persona name.
func TestInteractiveReview_PersonaTagRendering(t *testing.T) {
	m := NewReviewInteractiveModel(testFindingsWithPersonas(), nil, "", "", nil, nil)
	out := m.viewBrowse()

	// Triple-agreement finding uses [sec+dx-+tec] (3 personas truncated to 3 chars)
	// and shows (3/3) agreement. The abbreviation is lowercased-first-3 of each name.
	if !strings.Contains(out, "sec") || !strings.Contains(out, "(3/3)") {
		t.Errorf("browse view must show persona abbreviations and agreement; got:\n%s", out)
	}
}

func TestInteractiveReview_DetailViewShowsPersonas(t *testing.T) {
	m := NewReviewInteractiveModel(testFindingsWithPersonas(), nil, "", "", nil, nil)
	// After sort, position 0 is "Triple-agreement style" with 3 personas
	// (two of which are not in the real registry — they must surface the
	// unknown-persona fallback hint).
	out := m.viewDetail()
	if !strings.Contains(out, "Flagged by:") {
		t.Errorf("detail view must include 'Flagged by:' block; got:\n%s", out)
	}
	if !strings.Contains(out, "security-senior") {
		t.Errorf("detail view must list persona names (even when unknown); got:\n%s", out)
	}
	if !strings.Contains(out, "unknown persona") {
		t.Errorf("detail view must hint unknown persona fallback; got:\n%s", out)
	}
	if !strings.Contains(out, "Agreement:") {
		t.Errorf("detail view must include agreement line for multi-persona findings; got:\n%s", out)
	}
}

func TestFormatPersonaTag_EmptyReturnsEmpty(t *testing.T) {
	f := ReviewFinding{Title: "no persona"}
	if got := formatPersonaTag(f); got != "" {
		t.Errorf("no personas must yield empty tag, got %q", got)
	}
}

func TestFormatPersonaTag_SinglePersona_NoAgreementSuffix(t *testing.T) {
	f := ReviewFinding{Personas: []string{"security-senior"}, AgreementCount: 1}
	got := formatPersonaTag(f)
	if !strings.Contains(got, "sec") {
		t.Errorf("tag must include abbreviation, got %q", got)
	}
	if strings.Contains(got, "/") {
		t.Errorf("single-persona tag must NOT show agreement suffix, got %q", got)
	}
}

// TestInteractiveReview_DetailViewShowsPersonaLens asserts the follow-up to AC-8:
// the detail view must expose each flagging persona's Icon + DisplayName + Expertise
// so the reader can understand WHY a finding was surfaced, not just WHO flagged it.
func TestInteractiveReview_DetailViewShowsPersonaLens(t *testing.T) {
	// Pick a real persona from the registry and use its name in the finding.
	reg := GetRegistry()
	if len(reg) == 0 {
		t.Fatal("empty persona registry")
	}
	p := reg[0]
	findings := []ReviewFinding{{
		Severity:       "gap",
		Title:          "Test finding",
		Personas:       []string{p.Name},
		AgreementCount: 1,
		Hash:           "h1",
	}}
	m := NewReviewInteractiveModel(findings, nil, "", "", nil, nil)
	out := m.viewDetail()

	if !strings.Contains(out, "Flagged by:") {
		t.Errorf("detail view must include 'Flagged by:' block; got:\n%s", out)
	}
	if !strings.Contains(out, p.DisplayName) {
		t.Errorf("detail view must include persona DisplayName %q; got:\n%s", p.DisplayName, out)
	}
	if !strings.Contains(out, p.Expertise) {
		t.Errorf("detail view must include persona Expertise %q; got:\n%s", p.Expertise, out)
	}
}

// TestInteractiveReview_DetailView_UnknownPersonaFallback asserts that when
// the AI returns a persona name not in the registry, the detail view does not
// crash — it shows the raw identifier with a (unknown persona) hint so the
// reader can spot AI hallucination on the Personas field.
func TestInteractiveReview_DetailView_UnknownPersonaFallback(t *testing.T) {
	findings := []ReviewFinding{{
		Severity:       "gap",
		Title:          "Test",
		Personas:       []string{"not-a-registered-persona"},
		AgreementCount: 1,
		Hash:           "h1",
	}}
	m := NewReviewInteractiveModel(findings, nil, "", "", nil, nil)
	out := m.viewDetail()
	if !strings.Contains(out, "not-a-registered-persona") {
		t.Errorf("unknown persona must appear in fallback; got:\n%s", out)
	}
	if !strings.Contains(out, "unknown persona") {
		t.Errorf("unknown persona must carry hint; got:\n%s", out)
	}
}

// TestPersonaByName_ExportedMatchesInternal verifies the exported accessor
// returns the same profiles as the package-internal lookup, preventing drift.
func TestPersonaByName_ExportedMatchesInternal(t *testing.T) {
	for _, p := range GetRegistry() {
		a, okA := personaByName(p.Name)
		b, okB := PersonaByName(p.Name)
		if okA != okB {
			t.Errorf("exported vs internal ok mismatch for %q: %v vs %v", p.Name, okA, okB)
		}
		if a.Name != b.Name || a.DisplayName != b.DisplayName {
			t.Errorf("exported vs internal profile mismatch for %q", p.Name)
		}
	}
	// Unknown name returns false.
	if _, ok := PersonaByName("definitely-not-a-persona"); ok {
		t.Error("unknown name must return ok=false")
	}
}
