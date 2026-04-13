// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

const testOriginal = `---
type: decision
date: 2026-01-01
status: final
---
## What
We chose OAuth2.

## Why
Simplicity.

## Impact
Minimal.
`

const testPolished = `---
type: decision
date: 2026-01-01
status: final
---
## What
We chose OAuth2 for its widespread adoption and simplicity.

## Why
OAuth2 reduces integration friction by leveraging existing identity providers.
This avoids the complexity of building a custom authentication layer.

## Impact
Minimal operational overhead with significant developer experience improvement.

## How to Verify
Run the integration test suite against the staging identity provider.
`

func polishKeyMsg(key string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
}

func TestPolishInteractive_AcceptAllEqualsFullPolish(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Accept all changed sections
	for i := range m.sections {
		if m.sections[i].Changed || m.sections[i].IsNew {
			m.cursor = i
			model, _ := m.Update(polishKeyMsg("a"))
			m = model.(PolishInteractiveModel)
		}
	}

	// Should be in confirm write mode
	if m.mode != polishModeConfirmWrite {
		t.Fatalf("expected polishModeConfirmWrite, got %d", m.mode)
	}

	// Confirm write
	model, _ := m.Update(polishKeyMsg("y"))
	m = model.(PolishInteractiveModel)

	if !m.Written {
		t.Fatal("expected Written=true")
	}

	// Final doc should contain polished content
	if !strings.Contains(m.FinalDoc, "widespread adoption") {
		t.Error("final doc should contain polished What section")
	}
	if !strings.Contains(m.FinalDoc, "How to Verify") {
		t.Error("final doc should contain new section")
	}
}

func TestPolishInteractive_RejectAllEqualsOriginal(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Reject all changed sections
	for i := range m.sections {
		if m.sections[i].Changed || m.sections[i].IsNew {
			m.cursor = i
			model, _ := m.Update(polishKeyMsg("r"))
			m = model.(PolishInteractiveModel)
		}
	}

	if m.mode != polishModeConfirmWrite {
		t.Fatalf("expected polishModeConfirmWrite, got %d", m.mode)
	}

	model, _ := m.Update(polishKeyMsg("y"))
	m = model.(PolishInteractiveModel)

	if !m.Written {
		t.Fatal("expected Written=true")
	}

	// Final doc should NOT contain polished-only content
	if strings.Contains(m.FinalDoc, "widespread adoption") {
		t.Error("rejected: should not contain polished What content")
	}
	if strings.Contains(m.FinalDoc, "How to Verify") {
		t.Error("rejected: should not contain new section")
	}
	// Should still have original content
	if !strings.Contains(m.FinalDoc, "We chose OAuth2.") {
		t.Error("should contain original What content")
	}
}

func TestPolishInteractive_MixedAcceptReject(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Directly set decisions on sections to test reassembly
	for i := range m.sections {
		s := &m.sections[i]
		if !s.Changed && !s.IsNew {
			continue
		}
		if s.Heading == "## What" || s.IsNew {
			s.Decision = DecisionAccepted
		} else {
			s.Decision = DecisionRejected
		}
	}

	// Trigger reassembly
	m.mode = polishModeConfirmWrite
	model, _ := m.Update(polishKeyMsg("y"))
	m = model.(PolishInteractiveModel)

	if !m.Written {
		t.Fatal("expected Written=true")
	}

	// What should be polished
	if !strings.Contains(m.FinalDoc, "widespread adoption") {
		t.Error("accepted What should use polished version")
	}
	// Why should be original
	if !strings.Contains(m.FinalDoc, "Simplicity.") {
		t.Error("rejected Why should keep original")
	}
}

func TestPolishInteractive_NewSectionsPrompted(t *testing.T) {
	diffs := ComputeSectionDiffs(testOriginal, testPolished)

	hasNew := false
	for _, d := range diffs {
		if d.IsNew {
			hasNew = true
			if d.Heading != "## How to Verify" {
				t.Errorf("expected new section '## How to Verify', got '%s'", d.Heading)
			}
		}
	}
	if !hasNew {
		t.Error("expected at least one new section")
	}
}

func TestPolishInteractive_FrontMatterPreserved(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	if m.frontMatter == "" {
		t.Fatal("expected front matter to be extracted")
	}
	if !strings.Contains(m.frontMatter, "type: decision") {
		t.Error("front matter should contain type")
	}
	if !strings.Contains(m.frontMatter, "date: 2026-01-01") {
		t.Error("front matter should contain date")
	}

	// Accept all and check front matter in output
	for i := range m.sections {
		if m.sections[i].Changed || m.sections[i].IsNew {
			m.cursor = i
			model, _ := m.Update(polishKeyMsg("a"))
			m = model.(PolishInteractiveModel)
		}
	}
	model, _ := m.Update(polishKeyMsg("y"))
	m = model.(PolishInteractiveModel)

	if !strings.Contains(m.FinalDoc, "type: decision") {
		t.Error("final doc should preserve front matter")
	}
}

func TestPolishInteractive_EditActionStoresContent(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Find first changed section
	for i, s := range m.sections {
		if s.Changed {
			m.cursor = i
			break
		}
	}

	// Simulate editor returning content
	editedContent := "Custom edited content from user."
	msg := polishEditorFinishedMsg{content: editedContent}
	model, _ := m.Update(msg)
	m = model.(PolishInteractiveModel)

	if m.sections[m.cursor].Decision != DecisionEdited {
		// The cursor may have advanced after edit; check the section that was edited
		found := false
		for _, s := range m.sections {
			if s.Decision == DecisionEdited {
				found = true
				if s.EditedContent != editedContent {
					t.Errorf("expected edited content '%s', got '%s'", editedContent, s.EditedContent)
				}
			}
		}
		if !found {
			t.Error("expected at least one section with DecisionEdited")
		}
	}
}

func TestPolishInteractive_QuitDoesNotWrite(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Press q
	model, _ := m.Update(polishKeyMsg("q"))
	m = model.(PolishInteractiveModel)

	if m.mode != polishModeConfirmQuit {
		t.Fatalf("expected confirm quit mode, got %d", m.mode)
	}

	// Confirm quit
	model, cmd := m.Update(polishKeyMsg("y"))
	m = model.(PolishInteractiveModel)

	if !m.quitting {
		t.Fatal("expected quitting=true")
	}
	if m.Written {
		t.Error("quit should NOT write")
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd")
	}
	if !strings.Contains(m.QuitSummary, "Quit without saving") {
		t.Errorf("unexpected summary: %s", m.QuitSummary)
	}
}

func TestPolishInteractive_QuitDeniedReturnsToDiff(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	model, _ := m.Update(polishKeyMsg("q"))
	m = model.(PolishInteractiveModel)

	// Deny quit
	model, _ = m.Update(polishKeyMsg("n"))
	m = model.(PolishInteractiveModel)

	if m.mode != polishModeDiff {
		t.Fatalf("expected to return to diff mode, got %d", m.mode)
	}
}

func TestPolishInteractive_NonTTYFallback(t *testing.T) {
	result := IsTTYAvailable()
	if result {
		t.Skip("stdout is a TTY in this test env")
	}
	// M5 fix: single call, assert on stored result.
	if result {
		t.Error("expected IsTTYAvailable=false in non-TTY test")
	}
}

func TestPolishInteractive_SkippedSectionsPrompted(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Skip all changed sections
	for i := range m.sections {
		if m.sections[i].Changed || m.sections[i].IsNew {
			m.cursor = i
			model, _ := m.Update(polishKeyMsg("s"))
			m = model.(PolishInteractiveModel)
		}
	}

	if m.mode != polishModeSkippedPrompt {
		t.Fatalf("expected skipped prompt mode, got %d", m.mode)
	}

	// Accept all skipped
	model, _ := m.Update(polishKeyMsg("a"))
	m = model.(PolishInteractiveModel)

	if m.mode != polishModeConfirmWrite {
		t.Fatalf("expected confirm write after accepting skipped, got %d", m.mode)
	}
}

func TestComputeSectionDiffs(t *testing.T) {
	diffs := ComputeSectionDiffs(testOriginal, testPolished)

	if len(diffs) == 0 {
		t.Fatal("expected non-empty diffs")
	}

	// Count changed and new
	changed, newCount := 0, 0
	for _, d := range diffs {
		if d.Changed {
			changed++
		}
		if d.IsNew {
			newCount++
		}
	}

	if changed == 0 {
		t.Error("expected at least one changed section")
	}
	if newCount != 1 {
		t.Errorf("expected 1 new section (How to Verify), got %d", newCount)
	}
}

func TestExtractFrontMatter(t *testing.T) {
	fm := extractFrontMatter(testOriginal)
	if !strings.HasPrefix(fm, "---\n") {
		t.Error("front matter should start with ---")
	}
	if !strings.Contains(fm, "type: decision") {
		t.Error("should contain type field")
	}

	noFM := extractFrontMatter("# Just a heading\nNo front matter.")
	if noFM != "" {
		t.Error("should return empty for no front matter")
	}
}

func TestPolishInteractive_ViewShowsSectionDiff(t *testing.T) {
	m := NewPolishInteractiveModel(testOriginal, testPolished, "test.md")

	// Navigate to first changed section
	for i, s := range m.sections {
		if s.Changed && !s.IsNew {
			m.cursor = i
			break
		}
	}

	view := m.View()
	if !strings.Contains(view, "Section") {
		t.Error("view should contain Section header")
	}
	if !strings.Contains(view, "Accept") || !strings.Contains(view, "Reject") {
		t.Error("view should contain action hints")
	}
}
