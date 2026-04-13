// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package angela — draft_interactive.go
//
// Interactive draft fix-it mode using Bubbletea. Walks the user through
// each draft finding with contextual actions: add stub, edit in $EDITOR,
// add to related, ignore, skip. Reuses Bubbletea infrastructure from
// tui_common.go.

package angela

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
)

// draftViewMode tracks which screen the draft TUI is displaying.
type draftViewMode int

const (
	draftModeFinding draftViewMode = iota
)

// DraftFinding wraps a Suggestion with file context for the interactive TUI.
type DraftFinding struct {
	Filename   string
	Suggestion Suggestion
	Hash       string // stable identity for ignore tracking
}

// draftEditorFinishedMsg signals that $EDITOR exited.
type draftEditorFinishedMsg struct{ err error }

// draftReanalyzedMsg carries re-analysis results after a file modification.
type draftReanalyzedMsg struct {
	filename    string
	suggestions []Suggestion
}

// DraftInteractiveModel is the Bubbletea model for the interactive draft TUI.
type DraftInteractiveModel struct {
	findings []DraftFinding
	cursor   int
	mode     draftViewMode
	width    int
	height   int
	quitting bool

	// Session ignore set
	ignored map[string]bool // finding hash → ignored

	// Stats for quit summary
	resolvedCount int
	ignoredCount  int
	skippedCount  int

	// Context for re-analysis
	docsDir    string
	meta       map[string]domain.DocMeta // filename → meta
	guide      *StyleGuide
	corpus     []domain.DocMeta
	personas   []PersonaProfile
	standalone bool

	// Quit summary text
	QuitSummary string
}

// NewDraftInteractiveModel creates the draft TUI model.
func NewDraftInteractiveModel(
	findings []DraftFinding,
	docsDir string,
	meta map[string]domain.DocMeta,
	guide *StyleGuide,
	corpus []domain.DocMeta,
	personas []PersonaProfile,
	standalone bool,
) DraftInteractiveModel {
	SortDraftFindings(findings)
	return DraftInteractiveModel{
		findings:   findings,
		ignored:    make(map[string]bool),
		docsDir:    docsDir,
		meta:       meta,
		guide:      guide,
		corpus:     corpus,
		personas:   personas,
		standalone: standalone,
		width:      80,
		height:     24,
	}
}

// Init implements tea.Model.
func (m DraftInteractiveModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m DraftInteractiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case draftEditorFinishedMsg:
		// Re-analyze the file that was just edited
		if m.cursor < len(m.findings) {
			f := m.findings[m.cursor]
			return m, m.reanalyzeCmd(f.Filename)
		}
		return m, nil

	case draftReanalyzedMsg:
		return m.handleReanalyzed(msg), nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m DraftInteractiveModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "q" {
		return m.doQuit()
	}
	if msg.String() == "ctrl+c" {
		return m.doQuit()
	}

	switch m.mode {
	case draftModeFinding:
		return m.handleFindingKey(msg)
	}
	return m, nil
}

func (m DraftInteractiveModel) handleFindingKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.movePrev()
	case "down", "j":
		m.moveNext()
	case "s":
		// Skip
		m.skippedCount++
		m.moveNext()
	case "a":
		// Add stub — only for structure findings
		if m.cursor < len(m.findings) && m.findings[m.cursor].Suggestion.Category == "structure" {
			return m.doAddStub()
		}
	case "r":
		// Add to related — only for coherence findings
		if m.cursor < len(m.findings) && m.findings[m.cursor].Suggestion.Category == "coherence" {
			return m.doAddToRelated()
		}
	case "e":
		// Edit in $EDITOR
		return m.doEdit()
	case "i":
		// Ignore single
		return m.doIgnore()
	case "I":
		// Batch ignore
		return m.doBatchIgnore()
	}
	return m, nil
}

// --- Navigation ---

// moveNext and movePrev use pointer receivers so they can mutate
// m.cursor directly. This is safe because every call site (handleFindingKey,
// doIgnore, doBatchIgnore) always returns `m` as the updated model value,
// so the pointer writes are visible to the Bubbletea runtime.

func (m *DraftInteractiveModel) moveNext() {
	for {
		if m.cursor >= len(m.findings)-1 {
			return
		}
		m.cursor++
		if !m.ignored[m.findings[m.cursor].Hash] {
			return
		}
	}
}

func (m *DraftInteractiveModel) movePrev() {
	for {
		if m.cursor <= 0 {
			return
		}
		m.cursor--
		if !m.ignored[m.findings[m.cursor].Hash] {
			return
		}
	}
}

// --- Actions ---

func (m DraftInteractiveModel) doQuit() (tea.Model, tea.Cmd) {
	m.quitting = true
	m.QuitSummary = fmt.Sprintf("%d findings resolved, %d ignored, %d skipped",
		m.resolvedCount, m.ignoredCount, m.skippedCount)
	return m, tea.Quit
}

func (m DraftInteractiveModel) doIgnore() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.findings) {
		return m, nil
	}
	h := m.findings[m.cursor].Hash
	m.ignored[h] = true
	m.ignoredCount++
	m.moveNext()
	return m, nil
}

func (m DraftInteractiveModel) doBatchIgnore() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.findings) {
		return m, nil
	}
	cat := m.findings[m.cursor].Suggestion.Category
	for i := range m.findings {
		if m.findings[i].Suggestion.Category == cat && !m.ignored[m.findings[i].Hash] {
			m.ignored[m.findings[i].Hash] = true
			m.ignoredCount++
		}
	}
	m.moveNext()
	return m, nil
}

func (m DraftInteractiveModel) doAddStub() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.findings) {
		return m, nil
	}
	f := m.findings[m.cursor]
	if !isSafePath(f.Filename) {
		return m, nil
	}
	section := detectMissingSection(f.Suggestion.Message)
	if section == "" {
		return m, nil
	}

	docPath := filepath.Join(m.docsDir, f.Filename)
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return m, nil
	}

	stub := fmt.Sprintf("\n## %s\n\n<!-- TODO: describe %s -->\n", section, strings.ToLower(section))
	updated := string(raw) + stub
	if err := fileutil.AtomicWrite(docPath, []byte(updated), 0o644); err != nil {
		return m, nil
	}

	m.resolvedCount++
	return m, m.reanalyzeCmd(f.Filename)
}

func (m DraftInteractiveModel) doAddToRelated() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.findings) {
		return m, nil
	}
	f := m.findings[m.cursor]
	if !isSafePath(f.Filename) {
		return m, nil
	}
	mentioned := extractMentionedFilename(f.Suggestion.Message)
	if mentioned == "" || !isSafePath(mentioned) {
		return m, nil
	}

	docPath := filepath.Join(m.docsDir, f.Filename)
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return m, nil
	}

	updated := addRelatedToFrontMatter(string(raw), mentioned)
	if updated == string(raw) {
		return m, nil // no change
	}

	if err := fileutil.AtomicWrite(docPath, []byte(updated), 0o644); err != nil {
		return m, nil
	}

	m.resolvedCount++
	return m, m.reanalyzeCmd(f.Filename)
}

func (m DraftInteractiveModel) doEdit() (tea.Model, tea.Cmd) {
	if m.cursor >= len(m.findings) {
		return m, nil
	}
	editorArgs := splitEditorCmd(os.Getenv("EDITOR"))
	if len(editorArgs) == 0 {
		return m, nil
	}
	f := m.findings[m.cursor]
	if !isSafePath(f.Filename) {
		return m, nil
	}
	docPath := filepath.Join(m.docsDir, f.Filename)
	args := append(append([]string(nil), editorArgs[1:]...), docPath)
	c := exec.Command(editorArgs[0], args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		return draftEditorFinishedMsg{err: err}
	})
}

// --- Re-analysis ---

func (m DraftInteractiveModel) reanalyzeCmd(filename string) tea.Cmd {
	return func() tea.Msg {
		docPath := filepath.Join(m.docsDir, filename)
		raw, err := os.ReadFile(docPath)
		if err != nil {
			return draftReanalyzedMsg{filename: filename, suggestions: nil}
		}
		content := string(raw)
		meta := m.meta[filename]
		suggestions := AnalyzeDraft(content, meta, m.guide, m.corpus, m.personas)
		suggestions = append(suggestions, CheckCoherence(content, meta, m.corpus)...)
		return draftReanalyzedMsg{filename: filename, suggestions: suggestions}
	}
}

func (m DraftInteractiveModel) handleReanalyzed(msg draftReanalyzedMsg) DraftInteractiveModel {
	// M3 fix: collect old hashes for this file so we can transfer ignore
	// state to new findings with the same category (message may change).
	oldIgnoredCategories := make(map[string]bool)
	for _, f := range m.findings {
		if f.Filename == msg.filename && m.ignored[f.Hash] {
			oldIgnoredCategories[f.Suggestion.Category] = true
		}
	}

	// Remove old findings for this file
	var kept []DraftFinding
	for _, f := range m.findings {
		if f.Filename != msg.filename {
			kept = append(kept, f)
		}
	}

	// Add new findings, inheriting ignore state by category
	for _, s := range msg.suggestions {
		df := DraftFinding{
			Filename:   msg.filename,
			Suggestion: s,
			Hash:       DraftFindingHash(msg.filename, s),
		}
		if oldIgnoredCategories[s.Category] {
			m.ignored[df.Hash] = true
		}
		kept = append(kept, df)
	}

	SortDraftFindings(kept)
	m.findings = kept

	// Adjust cursor
	if m.cursor >= len(m.findings) {
		m.cursor = len(m.findings) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	return m
}

// --- View ---

func (m DraftInteractiveModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Active (non-ignored) counts
	total, remaining := m.countActive()
	fileRemaining := m.countActiveForFile()

	// Progress header
	// If the cursor is on an ignored finding, skip the
	// per-finding header to avoid showing misleading context.
	if m.cursor < len(m.findings) && !m.ignored[m.findings[m.cursor].Hash] {
		b.WriteString(TUIStyleTitle.Render(fmt.Sprintf(
			"Finding %d/%d · %d remaining in %s · %d remaining total",
			m.cursor+1, total, fileRemaining, m.findings[m.cursor].Filename, remaining,
		)))
	} else if remaining == 0 {
		b.WriteString(TUIStyleTitle.Render("All visible findings processed!"))
	} else {
		b.WriteString(TUIStyleTitle.Render(fmt.Sprintf(
			"%d/%d findings · %d remaining total",
			total-remaining, total, remaining,
		)))
	}
	b.WriteString("\n\n")

	// Current finding
	if m.cursor < len(m.findings) {
		f := m.findings[m.cursor]
		// Skip ignored findings in display
		if m.ignored[f.Hash] {
			b.WriteString(TUIStyleDim.Render("(ignored)"))
		} else {
			sev := formatDraftSeverity(f.Suggestion.Severity)
			fmt.Fprintf(&b, "%s  %s\n", sev, TUIStyleDim.Render(f.Suggestion.Category))
			b.WriteString(fmt.Sprintf("File: %s\n\n", f.Filename))
			b.WriteString(f.Suggestion.Message)
			b.WriteString("\n\n")

			// Actions based on category
			b.WriteString(m.renderActions(f))
		}
	}

	b.WriteString("\n")
	b.WriteString(TUIStyleDim.Render("↑/↓ j/k: navigate    q: quit"))
	return b.String()
}

func (m DraftInteractiveModel) renderActions(f DraftFinding) string {
	var actions []string
	switch f.Suggestion.Category {
	case "structure":
		if detectMissingSection(f.Suggestion.Message) != "" {
			actions = append(actions, TUIStyleHelpKey.Render("[a]")+" Add stub")
		}
		actions = append(actions, TUIStyleHelpKey.Render("[e]")+" Edit")
	case "coherence":
		if extractMentionedFilename(f.Suggestion.Message) != "" {
			actions = append(actions, TUIStyleHelpKey.Render("[r]")+" Add to related")
		}
		actions = append(actions, TUIStyleHelpKey.Render("[e]")+" Edit")
	default:
		actions = append(actions, TUIStyleHelpKey.Render("[e]")+" Edit")
	}
	actions = append(actions,
		TUIStyleHelpKey.Render("[i]")+" Ignore",
		TUIStyleHelpKey.Render("[I]")+" Ignore all "+f.Suggestion.Category,
		TUIStyleHelpKey.Render("[s]")+" Skip",
	)
	return strings.Join(actions, "    ")
}

func (m DraftInteractiveModel) countActive() (total, remaining int) {
	for i, f := range m.findings {
		if !m.ignored[f.Hash] {
			total++
			if i >= m.cursor {
				remaining++
			}
		}
	}
	return
}

func (m DraftInteractiveModel) countActiveForFile() int {
	if m.cursor >= len(m.findings) {
		return 0
	}
	currentFile := m.findings[m.cursor].Filename
	count := 0
	for i := m.cursor; i < len(m.findings); i++ {
		if m.findings[i].Filename == currentFile && !m.ignored[m.findings[i].Hash] {
			count++
		}
	}
	return count
}

// --- Sorting ---

// categoryPriority defines the sort order for draft finding categories.
var categoryPriority = map[string]int{
	"structure":    0,
	"completeness": 1,
	"coherence":    2,
	"persona":      3,
	"style":        4,
}

// severityPriority defines sort order within a category.
var severityPriority = map[string]int{
	SeverityError:   0,
	SeverityWarning: 1,
	SeverityInfo:    2,
}

// SortDraftFindings sorts findings by category priority, then severity,
// then filename alphabetically.
func SortDraftFindings(findings []DraftFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		ci := categoryPriority[findings[i].Suggestion.Category]
		cj := categoryPriority[findings[j].Suggestion.Category]
		if ci != cj {
			return ci < cj
		}
		si := severityPriority[findings[i].Suggestion.Severity]
		sj := severityPriority[findings[j].Suggestion.Severity]
		if si != sj {
			return si < sj
		}
		return findings[i].Filename < findings[j].Filename
	})
}

// DraftFindingHash computes a stable identity for a draft finding.
func DraftFindingHash(filename string, s Suggestion) string {
	input := filename + "\x00" + s.Category + "\x00" + s.Severity + "\x00" + s.Message
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:8])
}

// --- Helpers ---

func formatDraftSeverity(sev string) string {
	switch sev {
	case SeverityError:
		return TUIStyleError.Render("ERROR")
	case SeverityWarning:
		return TUIStyleWarning.Render("WARNING")
	default:
		return TUIStyleInfo.Render("INFO")
	}
}

// detectMissingSection extracts the section name from a "missing ## X"
// style message. Returns "" if the message doesn't match.
func detectMissingSection(msg string) string {
	lower := strings.ToLower(msg)
	for _, section := range []string{"What", "Why", "Alternatives", "Impact"} {
		if strings.Contains(lower, strings.ToLower(section)) &&
			(strings.Contains(lower, "missing") || strings.Contains(lower, "manquant")) {
			return section
		}
	}
	return ""
}

// extractMentionedFilename tries to extract a filename from a coherence
// message like 'Document "faq.md" mentioned in body'. Returns "" if not found.
func extractMentionedFilename(msg string) string {
	// Look for quoted filename pattern
	idx := strings.Index(msg, "\"")
	if idx < 0 {
		// Try pattern: filename.md mentioned / filename.md appears
		for _, word := range strings.Fields(msg) {
			if strings.HasSuffix(word, ".md") || strings.HasSuffix(word, ".md,") {
				return strings.TrimRight(word, ",)")
			}
		}
		return ""
	}
	end := strings.Index(msg[idx+1:], "\"")
	if end < 0 {
		return ""
	}
	candidate := msg[idx+1 : idx+1+end]
	if strings.HasSuffix(candidate, ".md") {
		return candidate
	}
	return ""
}

// addRelatedToFrontMatter adds a filename to the `related:` field in YAML
// front matter. If the field doesn't exist, it's created. Returns the
// original string unchanged if no front matter is found or the file is
// already listed.
func addRelatedToFrontMatter(doc string, filename string) string {
	if !strings.HasPrefix(doc, "---\n") {
		return doc
	}
	endIdx := strings.Index(doc[4:], "\n---\n")
	if endIdx < 0 {
		return doc
	}
	frontMatter := doc[4 : 4+endIdx]
	body := doc[4+endIdx+5:]

	slug := strings.TrimSuffix(filename, ".md")

	// Check if already present
	if strings.Contains(frontMatter, filename) || strings.Contains(frontMatter, slug) {
		return doc
	}

	// Find existing related: field
	lines := strings.Split(frontMatter, "\n")
	relatedIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "related:") {
			relatedIdx = i
			break
		}
	}

	if relatedIdx >= 0 {
		// Append to existing related list
		// Find last related list entry
		insertAt := relatedIdx + 1
		for insertAt < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[insertAt]), "- ") {
			insertAt++
		}
		entry := fmt.Sprintf("  - %s", slug)
		lines = append(lines[:insertAt], append([]string{entry}, lines[insertAt:]...)...)
	} else {
		// Add new related field at end
		lines = append(lines, fmt.Sprintf("related:\n  - %s", slug))
	}

	return "---\n" + strings.Join(lines, "\n") + "\n---\n" + body
}
