// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — review_interactive.go
//
// Interactive review REPL using Bubbletea. Provides a TUI for navigating
// review findings one by one, viewing evidence, and performing actions
// (resolve, ignore, deep dive, open in editor).

package angela

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
)

// viewMode tracks which screen the TUI is displaying.
type viewMode int

const (
	modeBrowse viewMode = iota
	modeDetail
	modeIgnorePrompt
)

// deepDiveResultMsg carries the AI response back to the TUI event loop.
type deepDiveResultMsg struct {
	text string
	err  error
}

// editorFinishedMsg signals that the external editor process exited.
type editorFinishedMsg struct{ err error }

// ReviewInteractiveModel is the Bubbletea model for the interactive review TUI.
type ReviewInteractiveModel struct {
	findings []ReviewFinding
	cursor   int
	mode     viewMode
	width    int
	height   int
	quitting bool

	// Deep dive state
	deepDiveText       string
	deepDiveLoading    bool
	deepDiveTargetHash string // hash of the finding that triggered the current deep dive

	// Ignore prompt state
	ignoreInput string

	// State integration
	state     *ReviewState
	statePath string
	audience  string

	// Counters for quit summary
	resolvedCount int
	ignoredCount  int
	deepDivedCount int

	// AI provider for deep dive
	provider domain.AIProvider
	reader   domain.CorpusReader

	// Quit summary text (rendered after program exits)
	QuitSummary string
}

// NewReviewInteractiveModel creates the TUI model with findings and optional
// state integration. Pass nil state/statePath to disable resolve/ignore.
func NewReviewInteractiveModel(
	findings []ReviewFinding,
	state *ReviewState,
	statePath string,
	audience string,
	provider domain.AIProvider,
	reader domain.CorpusReader,
) ReviewInteractiveModel {
	// The model sorts findings in place for persona-aware reviews. Without a
	// defensive copy, the caller's slice (typically `report.Findings`) is
	// mutated, which would alter the order the downstream cache / JSON
	// emitter sees. Copy before any sort.
	if anyFindingHasPersonas(findings) {
		copied := make([]ReviewFinding, len(findings))
		copy(copied, findings)
		findings = copied
		sortFindingsByAgreementThenSeverity(findings)
	}
	return ReviewInteractiveModel{
		findings:  findings,
		state:     state,
		statePath: statePath,
		audience:  audience,
		provider:  provider,
		reader:    reader,
		width:     80,
		height:    24,
	}
}

// anyFindingHasPersonas reports whether at least one finding has a non-empty
// Personas slice, signaling a story-8-19 persona-aware review.
func anyFindingHasPersonas(findings []ReviewFinding) bool {
	for i := range findings {
		if len(findings[i].Personas) > 0 {
			return true
		}
	}
	return false
}

// personaPoolSize returns the total number of distinct personas active in
// this review, derived from the UNION of persona names across all findings.
// It is the denominator for AgreementCount in the detail view: a "2 of 3"
// reading signals real dissent, a "2 of 2" reading signals unanimous
// concurrence among the active pool.
func (m ReviewInteractiveModel) personaPoolSize() int {
	seen := make(map[string]struct{})
	for _, f := range m.findings {
		for _, name := range f.Personas {
			seen[name] = struct{}{}
		}
	}
	return len(seen)
}

// sortFindingsByAgreementThenSeverity sorts in-place with AgreementCount DESC
// as the primary key and severityRank ASC as the tie-breaker. Stable so
// same-key findings preserve their relative input order.
func sortFindingsByAgreementThenSeverity(findings []ReviewFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].AgreementCount != findings[j].AgreementCount {
			return findings[i].AgreementCount > findings[j].AgreementCount
		}
		return severityRank(findings[i].Severity) < severityRank(findings[j].Severity)
	})
}

// Init implements tea.Model.
func (m ReviewInteractiveModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ReviewInteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case deepDiveResultMsg:
		m.deepDiveLoading = false
		// If the user resolved/ignored the finding while the deep dive
		// was in flight, the cursor may have shifted. Verify the current
		// finding's hash still matches the one that triggered the
		// request; discard stale results silently.
		if m.cursor >= len(m.findings) || m.findings[m.cursor].Hash != m.deepDiveTargetHash {
			return m, nil
		}
		if msg.err != nil {
			m.deepDiveText = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.deepDiveText = msg.text
		}
		return m, nil

	case editorFinishedMsg:
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m ReviewInteractiveModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global quit
	if msg.String() == "q" && m.mode != modeIgnorePrompt {
		return m.doQuit()
	}
	// ctrl+c during ignore-prompt discards the partial input and
	// quits. This is intentional — ctrl+c is an unconditional exit signal
	// regardless of current mode; the user can press Esc to cancel the
	// prompt without quitting.
	if msg.String() == "ctrl+c" {
		return m.doQuit()
	}

	switch m.mode {
	case modeBrowse:
		return m.handleBrowseKey(msg)
	case modeDetail:
		return m.handleDetailKey(msg)
	case modeIgnorePrompt:
		return m.handleIgnoreKey(msg)
	}
	return m, nil
}

func (m ReviewInteractiveModel) handleBrowseKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.findings)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.findings) > 0 {
			m.mode = modeDetail
			m.deepDiveText = ""
			m.deepDiveLoading = false
		}
	}
	return m, nil
}

func (m ReviewInteractiveModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBrowse
		return m, nil
	case "r":
		return m.doResolve()
	case "i":
		m.mode = modeIgnorePrompt
		m.ignoreInput = ""
		return m, nil
	case "d":
		if !m.deepDiveLoading {
			return m.doDeepDive()
		}
	case "o":
		return m.doOpenEditor()
	}
	return m, nil
}

func (m ReviewInteractiveModel) handleIgnoreKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeDetail
		m.ignoreInput = ""
		return m, nil
	case "enter":
		if strings.TrimSpace(m.ignoreInput) != "" {
			return m.doIgnore()
		}
	case "backspace":
		// Truncate by rune, not byte, for multi-byte chars.
		runes := []rune(m.ignoreInput)
		if len(runes) > 0 {
			m.ignoreInput = string(runes[:len(runes)-1])
		}
	default:
		// Only accept printable characters
		if len(msg.String()) == 1 && msg.String()[0] >= 32 {
			m.ignoreInput += msg.String()
		} else if msg.String() == " " {
			m.ignoreInput += " "
		}
	}
	return m, nil
}

// --- Actions ---

func (m ReviewInteractiveModel) doQuit() (tea.Model, tea.Cmd) {
	m.quitting = true
	// Save state on quit
	if m.state != nil && m.statePath != "" {
		_ = SaveReviewState(m.statePath, m.state)
	}
	m.QuitSummary = fmt.Sprintf("%d findings resolved, %d ignored, %d deep-dived",
		m.resolvedCount, m.ignoredCount, m.deepDivedCount)
	return m, tea.Quit
}

func (m ReviewInteractiveModel) doResolve() (tea.Model, tea.Cmd) {
	if m.state == nil || m.cursor >= len(m.findings) {
		return m, nil
	}
	f := m.findings[m.cursor]
	if f.Hash == "" {
		return m, nil
	}
	if err := MarkResolved(m.state, f.Hash, "interactive", time.Now().UTC()); err != nil {
		m.deepDiveText = fmt.Sprintf("state error: %v", err)
		return m, nil
	}
	m.resolvedCount++
	m.findings = append(m.findings[:m.cursor], m.findings[m.cursor+1:]...)
	if m.cursor >= len(m.findings) && m.cursor > 0 {
		m.cursor--
	}
	if len(m.findings) == 0 {
		return m.doQuit()
	}
	m.mode = modeBrowse
	return m, nil
}

func (m ReviewInteractiveModel) doIgnore() (tea.Model, tea.Cmd) {
	if m.state == nil || m.cursor >= len(m.findings) {
		return m, nil
	}
	f := m.findings[m.cursor]
	if f.Hash == "" {
		return m, nil
	}
	if err := MarkIgnored(m.state, f.Hash, m.ignoreInput, time.Now().UTC()); err != nil {
		m.deepDiveText = fmt.Sprintf("state error: %v", err)
		m.mode = modeDetail
		m.ignoreInput = ""
		return m, nil
	}
	m.ignoredCount++
	m.findings = append(m.findings[:m.cursor], m.findings[m.cursor+1:]...)
	if m.cursor >= len(m.findings) && m.cursor > 0 {
		m.cursor--
	}
	m.ignoreInput = ""
	if len(m.findings) == 0 {
		return m.doQuit()
	}
	m.mode = modeBrowse
	return m, nil
}

func (m ReviewInteractiveModel) doDeepDive() (tea.Model, tea.Cmd) {
	if m.provider == nil || m.cursor >= len(m.findings) {
		return m, nil
	}
	m.deepDiveLoading = true
	m.deepDivedCount++
	f := m.findings[m.cursor]
	m.deepDiveTargetHash = f.Hash
	return m, func() tea.Msg {
		sys, usr := BuildDeepDivePrompt(f, m.reader)
		result, err := m.provider.Complete(
			context.Background(), usr,
			domain.WithSystem(sys),
			domain.WithMaxTokens(1500),
		)
		return deepDiveResultMsg{text: result, err: err}
	}
}

func (m ReviewInteractiveModel) doOpenEditor() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.findings) || len(m.findings[m.cursor].Documents) == 0 {
		return m, nil
	}
	editorArgs := splitEditorCmd(os.Getenv("EDITOR"))
	if len(editorArgs) == 0 {
		m.deepDiveText = "Set $EDITOR to open files (e.g. export EDITOR=vim)"
		return m, nil
	}
	filename := m.findings[m.cursor].Documents[0]
	// C1 fix: reject path traversal in AI-generated document names.
	if !isSafePath(filename) {
		m.deepDiveText = fmt.Sprintf("Refusing to open unsafe path: %s", filename)
		return m, nil
	}
	args := append(append([]string(nil), editorArgs[1:]...), filename)
	c := exec.Command(editorArgs[0], args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}

// --- View ---

// L1 fix: review-specific styles only. Shared styles come from tui_common.go.
var (
	styleSeverityContradiction = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	styleSeverityGap           = lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow
	styleSeverityObsolete      = lipgloss.NewStyle().Foreground(lipgloss.Color("13")) // magenta
	styleSeverityStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray

	styleStatusNew       = lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
	styleStatusRegressed = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red
	styleStatusPersist   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // gray
)

func (m ReviewInteractiveModel) View() string {
	if m.quitting {
		return ""
	}
	switch m.mode {
	case modeBrowse:
		return m.viewBrowse()
	case modeDetail:
		return m.viewDetail()
	case modeIgnorePrompt:
		return m.viewIgnorePrompt()
	}
	return ""
}

func (m ReviewInteractiveModel) viewBrowse() string {
	var b strings.Builder

	// Header with counts
	newC, persC, regC := 0, 0, 0
	for _, f := range m.findings {
		switch f.DiffStatus {
		case ReviewDiffNew:
			newC++
		case ReviewDiffPersisting:
			persC++
		case ReviewDiffRegressed:
			regC++
		}
	}
	header := fmt.Sprintf("Review — %d findings", len(m.findings))
	if newC+persC+regC > 0 {
		header += fmt.Sprintf(" (%d NEW, %d PERSISTING, %d REGRESSED)", newC, persC, regC)
	}
	b.WriteString(TUIStyleTitle.Render(header))
	b.WriteString("\n\n")

	// Finding list
	visible := m.height - 6 // header + footer
	if visible < 3 {
		visible = 3
	}
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}
	end := start + visible
	if end > len(m.findings) {
		end = len(m.findings)
	}
	// Bounds check: ensure cursor is visible when list is taller than window.
	if m.cursor < start || m.cursor >= end {
		start = m.cursor
		end = start + visible
		if end > len(m.findings) {
			end = len(m.findings)
		}
	}

	// When personas are active, reserve a column for the persona tag (e.g.
	// "[sec+dx] (2/2)") next to the title. Baseline reviews skip the column
	// entirely so rendering stays identical to a non-persona run.
	showPersonas := anyFindingHasPersonas(m.findings)
	for i := start; i < end; i++ {
		f := m.findings[i]
		prefix := "  "
		if i == m.cursor {
			prefix = TUIStyleCursor.Render("> ")
		}
		idx := fmt.Sprintf("[%d/%d]", i+1, len(m.findings))
		status := formatDiffStatusTag(f.DiffStatus)
		sev := formatSeverityTag(f.Severity)
		var line string
		if showPersonas {
			line = fmt.Sprintf("%s %-7s %-10s %-14s %-22s %s",
				prefix, idx, status, sev, formatPersonaTag(f), f.Title)
		} else {
			line = fmt.Sprintf("%s %-7s %-10s %-14s %s", prefix, idx, status, sev, f.Title)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(TUIStyleDim.Render("↑/↓ or j/k: navigate    Enter: detail    q: quit"))
	return b.String()
}

func (m ReviewInteractiveModel) viewDetail() string {
	if m.cursor >= len(m.findings) {
		return ""
	}
	f := m.findings[m.cursor]
	var b strings.Builder

	// Header
	idx := fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.findings))
	b.WriteString(TUIStyleTitle.Render(fmt.Sprintf("%s %s %s", idx, f.Severity, formatDiffStatusTag(f.DiffStatus))))
	b.WriteString("\n")
	b.WriteString(TUIStyleTitle.Render(f.Title))
	b.WriteString("\n\n")

	// Show each flagging persona's lens so the reader understands WHY this
	// finding was surfaced, not just WHO flagged it. Each line carries Icon
	// + DisplayName + Expertise when the persona is resolvable via the
	// registry; unknown names fall back to the raw string.
	//
	// The agreement line expresses the numerator/denominator as
	// `AgreementCount / PersonaPoolSize`, where the pool size is the TOTAL
	// active persona count for this review, derived from the union of persona
	// attributions across the whole findings list. A "2/3" finding signals
	// genuine dissent (one persona did not concur) — the core multi-persona
	// signal.
	if len(f.Personas) > 0 {
		ta := i18n.T().Angela
		if f.AgreementCount > 1 {
			pool := m.personaPoolSize()
			if pool < f.AgreementCount {
				pool = f.AgreementCount // safety: denominator cannot be smaller than numerator
			}
			fmt.Fprintf(&b, ta.UIReviewAgreementConcur+"\n", f.AgreementCount, pool)
		}
		// The TUI renders the personas as a vertical list below the header;
		// using Sprintf with empty %s yields the localized "Flagged by: "
		// label — the trailing space is trimmed to keep the header-only look.
		b.WriteString(strings.TrimRight(fmt.Sprintf(ta.UIReviewFlaggedBy, ""), " "))
		b.WriteString("\n")
		for _, name := range f.Personas {
			if p, ok := personaByName(name); ok {
				fmt.Fprintf(&b, "  %s %s — %s\n", p.Icon, p.DisplayName, p.Expertise)
			} else {
				fmt.Fprintf(&b, "  %s (unknown persona)\n", name)
			}
		}
		b.WriteString("\n")
	}

	// Description
	if f.Description != "" {
		b.WriteString("Description:\n")
		b.WriteString("  " + f.Description)
		b.WriteString("\n\n")
	}

	// Evidence
	if len(f.Evidence) > 0 {
		b.WriteString("Evidence:\n")
		for _, ev := range f.Evidence {
			fmt.Fprintf(&b, "  %s:\n", TUIStyleDim.Render(ev.File))
			fmt.Fprintf(&b, "    \"%s\"\n", ev.Quote)
		}
		b.WriteString("\n")
	}

	// Confidence
	if f.Confidence > 0 {
		fmt.Fprintf(&b, "Confidence: %.2f\n\n", f.Confidence)
	}

	// Deep dive result
	if m.deepDiveLoading {
		b.WriteString(TUIStyleSpinner.Render("⣾ thinking..."))
		b.WriteString("\n\n")
	} else if m.deepDiveText != "" {
		b.WriteString(TUIStyleDim.Render("── Deep Dive ──"))
		b.WriteString("\n")
		b.WriteString(m.deepDiveText)
		b.WriteString("\n\n")
	}

	// Actions
	actions := []string{
		TUIStyleHelpKey.Render("[r]") + " Resolve",
		TUIStyleHelpKey.Render("[i]") + " Ignore",
		TUIStyleHelpKey.Render("[d]") + " Deep dive",
		TUIStyleHelpKey.Render("[o]") + " Open files",
	}
	b.WriteString(strings.Join(actions, "    "))
	b.WriteString("\n")
	b.WriteString(TUIStyleDim.Render("[Esc] Back     [q] Quit"))
	return b.String()
}

func (m ReviewInteractiveModel) viewIgnorePrompt() string {
	var b strings.Builder
	b.WriteString(TUIStyleTitle.Render("Ignore reason:"))
	b.WriteString("\n\n")
	b.WriteString("> " + m.ignoreInput + "█")
	b.WriteString("\n\n")
	b.WriteString(TUIStyleDim.Render("[Enter] Confirm    [Esc] Cancel"))
	return b.String()
}

// --- Helpers ---

func formatDiffStatusTag(status string) string {
	switch status {
	case ReviewDiffNew:
		return styleStatusNew.Render("[NEW]")
	case ReviewDiffRegressed:
		return styleStatusRegressed.Render("[REGRESSED]")
	case ReviewDiffPersisting:
		return styleStatusPersist.Render("[PERSIST]")
	default:
		return ""
	}
}

// formatPersonaTag renders the compact persona column used in the browse list.
// Format:
//   - no personas       → "" (empty, column stays visually blank)
//   - 1 persona         → "[sec]"
//   - 2+ personas       → "[sec+dx] (2/N)" where N = AgreementCount
//
// Persona names are abbreviated to the first 3 runes to keep the line width
// reasonable on a typical 80-column terminal. Rune-based slicing is required
// because byte slicing would produce invalid UTF-8 on multi-byte characters
// (emoji, accented Latin, CJK).
func formatPersonaTag(f ReviewFinding) string {
	if len(f.Personas) == 0 {
		return ""
	}
	shorts := make([]string, 0, len(f.Personas))
	for _, n := range f.Personas {
		shorts = append(shorts, truncateRunes(n, 3))
	}
	tag := "[" + strings.Join(shorts, "+") + "]"
	if f.AgreementCount > 1 {
		tag += fmt.Sprintf(i18n.T().Angela.UIReviewAgreementLineFormat, f.AgreementCount, f.AgreementCount)
	}
	return tag
}

func formatSeverityTag(sev string) string {
	switch sev {
	case "contradiction":
		return styleSeverityContradiction.Render(sev)
	case "gap":
		return styleSeverityGap.Render(sev)
	case "obsolete":
		return styleSeverityObsolete.Render(sev)
	case "style":
		return styleSeverityStyle.Render(sev)
	default:
		return sev
	}
}

// BuildDeepDivePrompt constructs the system and user prompt for a targeted
// deep-dive analysis of a single finding. If reader is non-nil, the
// full content of referenced documents is included so the AI can verify.
func BuildDeepDivePrompt(finding ReviewFinding, reader domain.CorpusReader) (string, string) {
	sys := `You are Angela, a senior technical editor. A review finding has been flagged and the user wants a deep dive.

Analyze this finding in detail:
1. Is the finding real or a false positive?
2. What is the exact contradiction/gap/issue?
3. How should it be fixed?

Be specific, cite exact passages from the documents. Keep your answer under 500 words.`

	var usr strings.Builder
	fmt.Fprintf(&usr, "Finding: %s (%s)\n", finding.Title, finding.Severity)
	fmt.Fprintf(&usr, "Description: %s\n", finding.Description)
	fmt.Fprintf(&usr, "Documents: %s\n\n", strings.Join(finding.Documents, ", "))

	if reader != nil {
		for _, doc := range finding.Documents {
			content, err := reader.ReadDoc(doc)
			if err != nil {
				continue
			}
			fmt.Fprintf(&usr, "=== %s ===\n%s\n\n", doc, content)
		}
	}

	return sys, usr.String()
}

// IsInteractiveAvailable checks if stdout is a TTY.
// Delegates to the shared IsTTYAvailable in tui_common.go.
func IsInteractiveAvailable() bool {
	return IsTTYAvailable()
}
