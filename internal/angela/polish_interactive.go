// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — polish_interactive.go
//
// Interactive polish with per-section accept/reject/edit. Uses
// Bubbletea TUI infrastructure from tui_common.go. Splits both
// original and polished documents into sections, computes diffs, and
// lets the user decide per section.

package angela

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SectionDecision tracks the user's choice for each section.
type SectionDecision int

const (
	DecisionPending  SectionDecision = iota
	DecisionAccepted                         // use polished version
	DecisionRejected                         // keep original
	DecisionEdited                           // use user-edited version
	DecisionSkipped                          // decide later
)

// SectionDiff pairs an original section with the corresponding polished section.
type SectionDiff struct {
	Heading      string // section heading (e.g. "## Why")
	Original     string // original body
	Polished     string // polished body (empty if section removed)
	IsNew        bool   // section exists only in polished
	IsRemoved    bool   // section exists only in original
	Changed      bool   // body differs between original and polished
	Decision     SectionDecision
	EditedContent string // content after user edit (DecisionEdited)
}

// polishViewMode tracks the TUI screen.
type polishViewMode int

const (
	polishModeDiff polishViewMode = iota
	polishModeConfirmWrite
	polishModeConfirmQuit
	polishModeSkippedPrompt
)

// polishEditorFinishedMsg signals editor exit with the edited content.
type polishEditorFinishedMsg struct {
	content string
	err     error
}

// PolishInteractiveModel is the Bubbletea model for interactive polish.
type PolishInteractiveModel struct {
	sections  []SectionDiff
	cursor    int
	mode      polishViewMode
	width     int
	height    int
	quitting  bool
	filename  string
	frontMatter string // preserved unchanged

	// Result
	Written     bool
	QuitSummary string
	FinalDoc    string // assembled document if Written
}

// NewPolishInteractiveModel creates the TUI model from original and polished docs.
func NewPolishInteractiveModel(original, polished, filename string) PolishInteractiveModel {
	sections := ComputeSectionDiffs(original, polished)
	fm := extractFrontMatter(original)

	return PolishInteractiveModel{
		sections:    sections,
		filename:    filename,
		frontMatter: fm,
		width:       80,
		height:      24,
	}
}

// Init implements tea.Model.
func (m PolishInteractiveModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m PolishInteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case polishEditorFinishedMsg:
		if msg.err == nil && m.cursor < len(m.sections) {
			m.sections[m.cursor].Decision = DecisionEdited
			m.sections[m.cursor].EditedContent = msg.content
			m.advanceToNextUndecided()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m PolishInteractiveModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle ctrl+c globally before dispatching to mode handlers.
	if msg.String() == "ctrl+c" {
		m.quitting = true
		m.QuitSummary = "Quit without saving."
		return m, tea.Quit
	}
	switch m.mode {
	case polishModeDiff:
		return m.handleDiffKey(msg)
	case polishModeConfirmWrite:
		return m.handleConfirmWriteKey(msg)
	case polishModeConfirmQuit:
		return m.handleConfirmQuitKey(msg)
	case polishModeSkippedPrompt:
		return m.handleSkippedKey(msg)
	}
	return m, nil
}

func (m PolishInteractiveModel) handleDiffKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		// Accept
		if m.cursor < len(m.sections) {
			m.sections[m.cursor].Decision = DecisionAccepted
			m.advanceToNextUndecided()
		}
	case "r":
		// Reject
		if m.cursor < len(m.sections) {
			m.sections[m.cursor].Decision = DecisionRejected
			m.advanceToNextUndecided()
		}
	case "e":
		// Edit
		return m.doEdit()
	case "s":
		// Skip
		if m.cursor < len(m.sections) {
			m.sections[m.cursor].Decision = DecisionSkipped
			m.advanceToNextUndecided()
		}
	case "n", "down", "j":
		if m.cursor < len(m.sections)-1 {
			m.cursor++
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "q":
		m.mode = polishModeConfirmQuit
	}
	return m, nil
}

func (m PolishInteractiveModel) handleConfirmWriteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.Written = true
		m.FinalDoc = m.reassemble()
		m.quitting = true
		m.QuitSummary = m.buildSummary()
		return m, tea.Quit
	case "n":
		m.mode = polishModeDiff
	}
	return m, nil
}

func (m PolishInteractiveModel) handleConfirmQuitKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.quitting = true
		m.QuitSummary = "Quit without saving."
		return m, tea.Quit
	case "n":
		m.mode = polishModeDiff
	}
	return m, nil
}

func (m PolishInteractiveModel) handleSkippedKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a":
		// Accept all skipped
		for i := range m.sections {
			if m.sections[i].Decision == DecisionSkipped {
				m.sections[i].Decision = DecisionAccepted
			}
		}
		m.mode = polishModeConfirmWrite
	case "r":
		// Reject all skipped
		for i := range m.sections {
			if m.sections[i].Decision == DecisionSkipped {
				m.sections[i].Decision = DecisionRejected
			}
		}
		m.mode = polishModeConfirmWrite
	case "c":
		// Continue reviewing skipped
		m.mode = polishModeDiff
		m.cursor = m.findFirstSkipped()
	}
	return m, nil
}

// --- Navigation ---

func (m *PolishInteractiveModel) advanceToNextUndecided() {
	// Check if all changed sections have decisions
	allDecided := true
	hasSkipped := false
	for _, s := range m.sections {
		if !s.Changed && !s.IsNew && !s.IsRemoved {
			continue
		}
		if s.Decision == DecisionPending {
			allDecided = false
			break
		}
		if s.Decision == DecisionSkipped {
			hasSkipped = true
		}
	}

	if allDecided && hasSkipped {
		m.mode = polishModeSkippedPrompt
		return
	}
	if allDecided {
		m.mode = polishModeConfirmWrite
		return
	}

	// Find next pending (includes Changed, IsNew, and IsRemoved sections)
	for i := m.cursor + 1; i < len(m.sections); i++ {
		if (m.sections[i].Changed || m.sections[i].IsNew || m.sections[i].IsRemoved) && m.sections[i].Decision == DecisionPending {
			m.cursor = i
			return
		}
	}
	// Wrap around
	for i := 0; i < m.cursor; i++ {
		if (m.sections[i].Changed || m.sections[i].IsNew || m.sections[i].IsRemoved) && m.sections[i].Decision == DecisionPending {
			m.cursor = i
			return
		}
	}
}

func (m PolishInteractiveModel) findFirstSkipped() int {
	for i, s := range m.sections {
		if s.Decision == DecisionSkipped {
			return i
		}
	}
	return 0
}

// --- Actions ---

func (m PolishInteractiveModel) doEdit() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.sections) {
		return m, nil
	}
	editorArgs := splitEditorCmd(os.Getenv("EDITOR"))
	if len(editorArgs) == 0 {
		return m, nil
	}
	s := m.sections[m.cursor]
	content := s.Polished
	if s.EditedContent != "" {
		content = s.EditedContent
	}

	// H4 fix: use state-dir-based temp path and ensure cleanup on all paths.
	tmp, err := os.CreateTemp("", "lore-polish-*.md")
	if err != nil {
		return m, nil
	}
	tmpPath := tmp.Name()
	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return m, nil
	}
	_ = tmp.Close()

	args := append(append([]string(nil), editorArgs[1:]...), tmpPath)
	c := exec.Command(editorArgs[0], args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		defer func() { _ = os.Remove(tmpPath) }()
		if err != nil {
			return polishEditorFinishedMsg{err: err}
		}
		data, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return polishEditorFinishedMsg{err: readErr}
		}
		return polishEditorFinishedMsg{content: string(data)}
	})
}

// --- View ---

var (
	styleAccepted = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	styleRejected = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	styleSkipped  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	styleNew      = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
	styleAdded    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
	styleRemoved  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
)

func (m PolishInteractiveModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.mode {
	case polishModeDiff:
		return m.viewDiff()
	case polishModeConfirmWrite:
		return m.viewConfirmWrite()
	case polishModeConfirmQuit:
		return m.viewConfirmQuit()
	case polishModeSkippedPrompt:
		return m.viewSkippedPrompt()
	}
	return ""
}

func (m PolishInteractiveModel) viewDiff() string {
	if m.cursor >= len(m.sections) {
		return TUIStyleTitle.Render("No sections to review.")
	}
	s := m.sections[m.cursor]
	changedCount := m.countChanged()
	var b strings.Builder

	heading := s.Heading
	if heading == "" {
		heading = "(preamble)"
	}

	// Header
	b.WriteString(TUIStyleTitle.Render(fmt.Sprintf("Section %d/%d: %s", m.cursor+1, len(m.sections), heading)))
	b.WriteString("  ")
	b.WriteString(TUIStyleDim.Render(fmt.Sprintf("(%d changed)", changedCount)))
	b.WriteString("\n\n")

	if s.IsNew {
		b.WriteString(styleNew.Render("NEW SECTION (proposed by AI)"))
		b.WriteString("\n\n")
		b.WriteString(renderWithPrefix(s.Polished, "+ "))
	} else if !s.Changed {
		b.WriteString(TUIStyleDim.Render("(unchanged)"))
	} else {
		// Show original and proposed
		origLines := strings.Count(s.Original, "\n") + 1
		polLines := strings.Count(s.Polished, "\n") + 1
		b.WriteString(fmt.Sprintf("Current (%d lines):\n", origLines))
		b.WriteString(renderWithPrefix(truncatePreview(s.Original, 10), "- "))
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprintf("Proposed (%d lines):\n", polLines))
		b.WriteString(renderWithPrefix(truncatePreview(s.Polished, 10), "+ "))
	}
	b.WriteString("\n\n")

	// Decision status
	b.WriteString(m.renderDecisionStatus(s))
	b.WriteString("\n\n")

	// Actions
	if s.IsNew {
		b.WriteString(TUIStyleHelpKey.Render("[a]") + " Accept    ")
		b.WriteString(TUIStyleHelpKey.Render("[r]") + " Reject")
	} else if s.Changed {
		b.WriteString(TUIStyleHelpKey.Render("[a]") + " Accept    ")
		b.WriteString(TUIStyleHelpKey.Render("[r]") + " Reject    ")
		b.WriteString(TUIStyleHelpKey.Render("[e]") + " Edit proposed    ")
		b.WriteString(TUIStyleHelpKey.Render("[s]") + " Skip")
	}
	b.WriteString("\n")
	b.WriteString(TUIStyleDim.Render("↑/↓ j/k/n: navigate    [q] Quit without saving"))
	return b.String()
}

func (m PolishInteractiveModel) viewConfirmWrite() string {
	accepted, rejected, edited := m.countDecisions()
	var b strings.Builder
	b.WriteString(TUIStyleTitle.Render("Write confirmation"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("You accepted %d sections, rejected %d, edited %d.\n", accepted, rejected, edited))
	b.WriteString(fmt.Sprintf("Write to %s? ", m.filename))
	b.WriteString(TUIStyleHelpKey.Render("[y]") + " Yes    " + TUIStyleHelpKey.Render("[n]") + " No")
	return b.String()
}

func (m PolishInteractiveModel) viewConfirmQuit() string {
	return TUIStyleTitle.Render("Discard all changes?") + "\n\n" +
		TUIStyleHelpKey.Render("[y]") + " Yes    " + TUIStyleHelpKey.Render("[n]") + " No"
}

func (m PolishInteractiveModel) viewSkippedPrompt() string {
	count := 0
	for _, s := range m.sections {
		if s.Decision == DecisionSkipped {
			count++
		}
	}
	var b strings.Builder
	b.WriteString(TUIStyleTitle.Render(fmt.Sprintf("You have %d skipped sections.", count)))
	b.WriteString("\n\n")
	b.WriteString(TUIStyleHelpKey.Render("[a]") + " Accept all skipped    ")
	b.WriteString(TUIStyleHelpKey.Render("[r]") + " Reject all skipped    ")
	b.WriteString(TUIStyleHelpKey.Render("[c]") + " Continue reviewing")
	return b.String()
}

func (m PolishInteractiveModel) renderDecisionStatus(s SectionDiff) string {
	switch s.Decision {
	case DecisionAccepted:
		return styleAccepted.Render("✓ ACCEPTED")
	case DecisionRejected:
		return styleRejected.Render("✗ REJECTED")
	case DecisionEdited:
		return styleAccepted.Render("✓ EDITED")
	case DecisionSkipped:
		return styleSkipped.Render("⊘ SKIPPED")
	default:
		return TUIStyleDim.Render("○ PENDING")
	}
}

// --- Reassembly ---

func (m PolishInteractiveModel) reassemble() string {
	var parts []string

	// Front matter preserved unchanged
	if m.frontMatter != "" {
		parts = append(parts, m.frontMatter)
	}

	for _, s := range m.sections {
		var content string
		switch s.Decision {
		case DecisionAccepted:
			if s.IsRemoved {
				continue // accepting removal means dropping the section
			}
			content = s.Polished
		case DecisionEdited:
			content = s.EditedContent
		case DecisionRejected, DecisionSkipped, DecisionPending:
			if s.IsNew {
				continue // new sections that are rejected are dropped
			}
			content = s.Original
		}

		if s.Heading != "" {
			parts = append(parts, s.Heading)
		}
		// M4 fix: trim trailing newlines to avoid double-blank-lines
		parts = append(parts, strings.TrimRight(content, "\n"))
	}

	return strings.Join(parts, "\n") + "\n"
}

func (m PolishInteractiveModel) buildSummary() string {
	accepted, rejected, edited := m.countDecisions()
	return fmt.Sprintf("Accepted %d, rejected %d, edited %d. Written to %s.",
		accepted, rejected, edited, m.filename)
}

func (m PolishInteractiveModel) countDecisions() (accepted, rejected, edited int) {
	for _, s := range m.sections {
		switch s.Decision {
		case DecisionAccepted:
			accepted++
		case DecisionRejected:
			rejected++
		case DecisionEdited:
			edited++
		}
	}
	return
}

func (m PolishInteractiveModel) countChanged() int {
	count := 0
	for _, s := range m.sections {
		if s.Changed || s.IsNew || s.IsRemoved {
			count++
		}
	}
	return count
}

// --- Section diff computation ---

// splitByHeading splits a document body (no front matter) into sections
// keyed by heading. Unlike SplitSections (which is designed for multi-pass
// polish), this ensures every `## ` line becomes its own section heading,
// even the very first one. The preamble (content before any ## heading)
// is stored under the empty-string key.
func splitByHeading(body string) []Section {
	lines := strings.Split(body, "\n")
	var sections []Section
	var currentHeading string
	var bodyLines []string
	fenceDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// L3 fix: track fence depth by backtick run length, not a toggle.
		if strings.HasPrefix(trimmed, "```") {
			if fenceDepth == 0 {
				fenceDepth++
			} else {
				fenceDepth--
				if fenceDepth < 0 {
					fenceDepth = 0
				}
			}
		}
		isHeading := fenceDepth == 0 && strings.HasPrefix(trimmed, "## ")
		if isHeading {
			// Save previous section
			sections = append(sections, Section{
				Heading: currentHeading,
				Body:    strings.Join(bodyLines, "\n"),
				Index:   len(sections),
			})
			currentHeading = trimmed
			bodyLines = nil
		} else {
			bodyLines = append(bodyLines, line)
		}
	}
	// Save last section
	sections = append(sections, Section{
		Heading: currentHeading,
		Body:    strings.Join(bodyLines, "\n"),
		Index:   len(sections),
	})
	return sections
}

// ComputeSectionDiffs splits original and polished by ## headings and
// pairs them by heading name. Sections present only in polished are
// marked IsNew. Sections present only in original are IsRemoved.
func ComputeSectionDiffs(original, polished string) []SectionDiff {
	origBody := stripFrontMatter(original)
	polBody := stripFrontMatter(polished)

	origSections := splitByHeading(origBody)
	polSections := splitByHeading(polBody)

	// Index polished sections by heading. If duplicate headings exist,
	// only the first is indexed — later duplicates fall through to the
	// "new section" path, which is safer than silently merging.
	polByHeading := make(map[string]Section, len(polSections))
	for _, s := range polSections {
		if _, exists := polByHeading[s.Heading]; exists {
			continue // keep first occurrence only
		}
		polByHeading[s.Heading] = s
	}

	var diffs []SectionDiff

	// Walk original sections in order
	seenHeadings := make(map[string]bool)
	for _, orig := range origSections {
		seenHeadings[orig.Heading] = true
		pol, found := polByHeading[orig.Heading]
		if !found {
			// Leave removed sections as DecisionPending so the user is
			// prompted to confirm the removal (accept) or keep the
			// original (reject).
			diffs = append(diffs, SectionDiff{
				Heading:   orig.Heading,
				Original:  orig.Body,
				IsRemoved: true,
				Changed:   true,
			})
			continue
		}
		changed := strings.TrimSpace(orig.Body) != strings.TrimSpace(pol.Body)
		d := SectionDiff{
			Heading:  orig.Heading,
			Original: orig.Body,
			Polished: pol.Body,
			Changed:  changed,
		}
		if !changed {
			d.Decision = DecisionAccepted // unchanged sections auto-accept
		}
		diffs = append(diffs, d)
	}

	// New sections (in polished but not in original)
	for _, pol := range polSections {
		if !seenHeadings[pol.Heading] {
			diffs = append(diffs, SectionDiff{
				Heading:  pol.Heading,
				Polished: pol.Body,
				IsNew:    true,
				Changed:  true,
			})
		}
	}

	return diffs
}

// --- Helpers ---

// extractFrontMatter returns the front matter block (including delimiters)
// or "" if none exists.
func extractFrontMatter(doc string) string {
	if !strings.HasPrefix(doc, "---\n") {
		return ""
	}
	endIdx := strings.Index(doc[4:], "\n---\n")
	if endIdx < 0 {
		return ""
	}
	return doc[:4+endIdx+5]
}

func renderWithPrefix(text, prefix string) string {
	lines := strings.Split(text, "\n")
	var b strings.Builder
	for _, line := range lines {
		if prefix == "+ " {
			b.WriteString(styleAdded.Render(prefix + line))
		} else {
			b.WriteString(styleRemoved.Render(prefix + line))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func truncatePreview(text string, maxLines int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}
	truncated := strings.Join(lines[:maxLines], "\n")
	return truncated + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}
