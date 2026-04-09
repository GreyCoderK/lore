// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
)

func TestSerializeTOON_BasicCorpus(t *testing.T) {
	docs := []DocSummary{
		{Filename: "decision-auth.md", Type: "decision", Date: "2026-03-15", Tags: []string{"auth", "api"}, Summary: "JWT chosen for stateless"},
		{Filename: "feature-api.md", Type: "feature", Date: "2026-03-16", Tags: []string{"api"}, Summary: "REST endpoints"},
		{Filename: "bugfix-login.md", Type: "bugfix", Date: "2026-03-17", Summary: "Fixed timeout"},
	}

	result := SerializeTOON(docs, nil)

	if !strings.HasPrefix(result, "corpus:\n") {
		t.Error("should start with corpus: section")
	}
	if !strings.Contains(result, "filename|type|date|tags|branch|scope|summary\n") {
		t.Error("should contain header row")
	}
	if !strings.Contains(result, "decision-auth.md|decision|2026-03-15|auth,api|||JWT chosen for stateless\n") {
		t.Error("should contain first doc row")
	}
	if !strings.Contains(result, "bugfix-login.md|bugfix|2026-03-17||||Fixed timeout\n") {
		t.Error("should contain doc with empty tags")
	}
	// No signals section when nil
	if strings.Contains(result, "signals:") {
		t.Error("should not contain signals section when nil")
	}
}

func TestSerializeTOON_WithSignals(t *testing.T) {
	docs := []DocSummary{
		{Filename: "a.md", Type: "decision", Date: "2026-01-01", Tags: []string{"auth"}, Summary: "old"},
		{Filename: "b.md", Type: "decision", Date: "2026-03-01", Tags: []string{"auth"}, Summary: "new"},
	}
	signals := &CorpusSignals{
		PotentialPairs: []DocPair{
			{DocA: "a.md", DocB: "b.md", Type: "decision", Tags: "auth", DaysDiff: 60},
		},
		IsolatedDocs: []string{"lonely.md"},
	}

	result := SerializeTOON(docs, signals)

	if !strings.Contains(result, "signals:\n") {
		t.Error("should contain signals section")
	}
	if !strings.Contains(result, "signal_type|docs|detail\n") {
		t.Error("should contain signals header row")
	}
	if !strings.Contains(result, "contradiction|a.md,b.md|type:decision, tags:auth, 60d apart\n") {
		t.Error("should contain contradiction pair row")
	}
	if !strings.Contains(result, "isolated|lonely.md|no shared tags\n") {
		t.Error("should contain isolated doc row")
	}
}

func TestSerializeTOON_Escaping_PipeInContent(t *testing.T) {
	docs := []DocSummary{
		{Filename: "test.md", Type: "decision", Date: "2026-01-01", Tags: []string{"a|b"}, Summary: "value|with|pipes"},
	}
	result := SerializeTOON(docs, nil)

	if !strings.Contains(result, `a\|b`) {
		t.Errorf("pipe in tags should be escaped, got: %s", result)
	}
	if !strings.Contains(result, `value\|with\|pipes`) {
		t.Errorf("pipes in summary should be escaped, got: %s", result)
	}
}

func TestSerializeTOON_Escaping_Backslash(t *testing.T) {
	docs := []DocSummary{
		{Filename: "test.md", Type: "decision", Date: "2026-01-01", Summary: `path\to\file`},
	}
	result := SerializeTOON(docs, nil)

	if !strings.Contains(result, `path\\to\\file`) {
		t.Errorf("backslashes should be escaped, got: %s", result)
	}
}

func TestSerializeTOON_Escaping_TrailingBackslash(t *testing.T) {
	docs := []DocSummary{
		{Filename: "test.md", Type: "decision", Date: "2026-01-01", Summary: `ends with\`},
	}
	result := SerializeTOON(docs, nil)

	if !strings.Contains(result, `ends with\\`) {
		t.Errorf("trailing backslash should be escaped, got: %s", result)
	}
}

func TestSerializeTOON_Escaping_Newlines(t *testing.T) {
	docs := []DocSummary{
		{Filename: "test.md", Type: "decision", Date: "2026-01-01", Summary: "line1\nline2\nline3"},
	}
	result := SerializeTOON(docs, nil)

	if strings.Contains(result, "line1\nline2") {
		t.Error("newlines in summary should be replaced with spaces")
	}
	if !strings.Contains(result, "line1 line2 line3") {
		t.Errorf("newlines should become spaces, got: %s", result)
	}
}

func TestSerializeTOON_EmptyCorpus(t *testing.T) {
	result := SerializeTOON(nil, nil)

	if !strings.HasPrefix(result, "corpus:\n") {
		t.Error("empty corpus should still have corpus header")
	}
	if !strings.Contains(result, "filename|type|date|tags|branch|scope|summary\n") {
		t.Error("empty corpus should still have column headers")
	}
	// Should be just the header, no data rows
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 2 {
		t.Errorf("empty corpus should have 2 lines (section + header), got %d", len(lines))
	}
}

func TestSerializeTOON_EmptySignals(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01"}}
	signals := &CorpusSignals{} // no pairs, no isolated

	result := SerializeTOON(docs, signals)

	if strings.Contains(result, "signals:") {
		t.Error("should not contain signals section when signals are empty")
	}
}

func TestEscapeTOON_EscapeOrder(t *testing.T) {
	// Verify order: newlines first, then backslash, then pipe
	// Input with all three: "a\n|b\" should become "a \\|b\\"
	// Step 1: \n → space: "a |b\"
	// Step 2: \ → \\: "a |b\\"
	// Step 3: | → \|: "a \|b\\"
	result := escapeTOON("a\n|b\\")
	if result != `a \|b\\` {
		t.Errorf("escapeTOON order wrong, got %q, want %q", result, `a \|b\\`)
	}
}

// --- SerializeTOONWithVHS tests ---

func TestSerializeTOONWithVHS_NilVHS(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Summary: "test"}}
	got := SerializeTOONWithVHS(docs, nil, nil)
	base := SerializeTOON(docs, nil)
	if got != base {
		t.Error("nil VHS should return same as SerializeTOON")
	}
}

func TestSerializeTOONWithVHS_EmptyVHS(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Summary: "test"}}
	vhs := &VHSSignals{}
	got := SerializeTOONWithVHS(docs, nil, vhs)
	if strings.Contains(got, "vhs_signals:") {
		t.Error("empty VHS signals should not produce vhs_signals section")
	}
}

func TestSerializeTOONWithVHS_WithOrphanTapes(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Summary: "test"}}
	vhs := &VHSSignals{
		OrphanTapes: []string{"demo.tape"},
	}
	got := SerializeTOONWithVHS(docs, nil, vhs)
	if !strings.Contains(got, "vhs_signals:\n") {
		t.Error("should contain vhs_signals section")
	}
	if !strings.Contains(got, "signal_type|source|detail\n") {
		t.Error("should contain VHS header row")
	}
	if !strings.Contains(got, "orphan_tape|demo.tape|") {
		t.Error("should contain orphan tape row")
	}
}

func TestSerializeTOONWithVHS_WithOrphanGIFs(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Summary: "test"}}
	vhs := &VHSSignals{
		OrphanGIFs: []GIFRef{
			{DocFilename: "readme.md", GIFPath: "demo.gif"},
		},
	}
	got := SerializeTOONWithVHS(docs, nil, vhs)
	if !strings.Contains(got, "orphan_gif|readme.md|") {
		t.Error("should contain orphan GIF row")
	}
	if !strings.Contains(got, "demo.gif") {
		t.Error("should reference the GIF path")
	}
}

func TestSerializeTOONWithVHS_WithCommandMismatches(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Summary: "test"}}
	vhs := &VHSSignals{
		CommandMismatches: []TapeMismatch{
			{TapeFile: "demo.tape", Command: "lore review", Reason: "undocumented_command"},
		},
	}
	got := SerializeTOONWithVHS(docs, nil, vhs)
	if !strings.Contains(got, "command_mismatch|demo.tape|") {
		t.Error("should contain command mismatch row")
	}
	if !strings.Contains(got, "lore review") {
		t.Error("should contain the mismatched command")
	}
	if !strings.Contains(got, "undocumented_command") {
		t.Error("should contain the reason")
	}
}

func TestSerializeTOONWithVHS_AllSignalTypes(t *testing.T) {
	docs := []DocSummary{{Filename: "a.md", Type: "decision", Date: "2026-01-01", Summary: "test"}}
	vhs := &VHSSignals{
		OrphanTapes:       []string{"tape1.tape"},
		OrphanGIFs:        []GIFRef{{DocFilename: "doc.md", GIFPath: "old.gif"}},
		CommandMismatches: []TapeMismatch{{TapeFile: "tape2.tape", Command: "lore init", Reason: "unknown_subcommand"}},
	}
	got := SerializeTOONWithVHS(docs, nil, vhs)
	// Count signal rows (3 data rows expected)
	if !strings.Contains(got, "orphan_tape|") {
		t.Error("should contain orphan_tape signal")
	}
	if !strings.Contains(got, "orphan_gif|") {
		t.Error("should contain orphan_gif signal")
	}
	if !strings.Contains(got, "command_mismatch|") {
		t.Error("should contain command_mismatch signal")
	}
}
