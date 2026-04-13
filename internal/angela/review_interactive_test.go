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
