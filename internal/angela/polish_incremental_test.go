// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// --- fake provider for incremental tests ---

type fakePolishProvider struct {
	calls   int
	lastIn  string
	fixedOut string
}

func (f *fakePolishProvider) Complete(_ context.Context, userContent string, _ ...domain.Option) (string, error) {
	f.calls++
	f.lastIn = userContent
	return f.fixedOut, nil
}

// helper to build a simple two-section doc
func twoSectionDoc() string {
	return "---\ntitle: Test\n---\n\n## Why\n\nBecause reasons.\n\n## How\n\nStep one.\nStep two.\nStep three.\n"
}

func baseIncrOpts(p domain.AIProvider) IncrementalOpts {
	return IncrementalOpts{
		Provider:       p,
		Meta:           domain.DocMeta{Filename: "test.md", Type: "adr"},
		MinChangeLines: 0,
	}
}

// --- AC-10 tests ---

// TestPolishIncremental_FirstRunIsFullPolish — AC-9: on first run
// (no stored hashes) the orchestrator falls back to full polish.
func TestPolishIncremental_FirstRunIsFullPolish(t *testing.T) {
	doc := twoSectionDoc()
	prov := &fakePolishProvider{fixedOut: doc}
	res, err := PolishIncremental(context.Background(), doc, nil, baseIncrOpts(prov))
	if err != nil {
		t.Fatal(err)
	}
	if res.WasIncremental {
		t.Error("first run should fall back to full polish (WasIncremental=false)")
	}
	if prov.calls != 1 {
		t.Errorf("expected 1 AI call, got %d", prov.calls)
	}
}

// TestPolishIncremental_UnchangedDocSkipsAI — AC-3: zero API calls
// when no section changed.
func TestPolishIncremental_UnchangedDocSkipsAI(t *testing.T) {
	doc := twoSectionDoc()
	sections := SplitSections(doc)
	stored := HashSections(sections)

	prov := &fakePolishProvider{fixedOut: "should not be called"}
	res, err := PolishIncremental(context.Background(), doc, stored, baseIncrOpts(prov))
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Error("expected Skipped=true when doc is unchanged")
	}
	if prov.calls != 0 {
		t.Errorf("expected 0 AI calls, got %d", prov.calls)
	}
	if res.Polished != doc {
		t.Error("polished output should equal original when skipped")
	}
}

// TestPolishIncremental_OnlyChangedSectionSent — AC-5: only the
// changed section appears in the AI prompt.
func TestPolishIncremental_OnlyChangedSectionSent(t *testing.T) {
	doc := twoSectionDoc()
	sections := SplitSections(doc)
	stored := HashSections(sections)

	// Mutate "## How" section.
	mutated := strings.Replace(doc, "Step one.", "Step one MODIFIED.", 1)

	// The AI echoes back the mutated section (no real polish).
	prov := &fakePolishProvider{fixedOut: "## How\n\nStep one POLISHED.\nStep two.\nStep three.\n"}
	res, err := PolishIncremental(context.Background(), mutated, stored, baseIncrOpts(prov))
	if err != nil {
		t.Fatal(err)
	}
	if !res.WasIncremental {
		t.Error("expected WasIncremental=true")
	}
	if res.ChangedCount != 1 {
		t.Errorf("expected 1 changed section, got %d", res.ChangedCount)
	}
	// The prompt should contain "## How" but NOT "Because reasons" (unchanged).
	if !strings.Contains(prov.lastIn, "## How") {
		t.Error("prompt should contain the changed heading")
	}
	if strings.Contains(prov.lastIn, "Because reasons") {
		t.Error("prompt should NOT contain unchanged section body")
	}
}

// TestPolishIncremental_ReassemblyPreservesOrder — AC-6: unchanged
// sections stay in their original positions after reassembly.
func TestPolishIncremental_ReassemblyPreservesOrder(t *testing.T) {
	doc := twoSectionDoc()
	sections := SplitSections(doc)
	stored := HashSections(sections)

	mutated := strings.Replace(doc, "Step one.", "Step one MODIFIED.", 1)
	prov := &fakePolishProvider{fixedOut: "## How\n\nStep one POLISHED.\nStep two.\nStep three.\n"}

	res, err := PolishIncremental(context.Background(), mutated, stored, baseIncrOpts(prov))
	if err != nil {
		t.Fatal(err)
	}
	// "## Why" should appear BEFORE "## How" in the output.
	whyIdx := strings.Index(res.Polished, "## Why")
	howIdx := strings.Index(res.Polished, "## How")
	if whyIdx < 0 || howIdx < 0 {
		t.Fatalf("missing sections in output: why=%d, how=%d", whyIdx, howIdx)
	}
	if whyIdx >= howIdx {
		t.Error("## Why should appear before ## How in reassembled output")
	}
	// "Because reasons" (unchanged) should be verbatim.
	if !strings.Contains(res.Polished, "Because reasons") {
		t.Error("unchanged section body should be preserved verbatim")
	}
}

// TestPolishIncremental_FallbackOnParseFailure — AC-8: when AI
// returns garbage the orchestrator falls back to full polish.
func TestPolishIncremental_FallbackOnParseFailure(t *testing.T) {
	doc := twoSectionDoc()
	sections := SplitSections(doc)
	stored := HashSections(sections)

	mutated := strings.Replace(doc, "Step one.", "Step one MODIFIED.", 1)
	// AI returns no headings → parse failure → fallback.
	prov := &fakePolishProvider{fixedOut: "This is just plain text with no headings."}
	res, err := PolishIncremental(context.Background(), mutated, stored, baseIncrOpts(prov))
	if err != nil {
		t.Fatal(err)
	}
	if res.WasIncremental {
		t.Error("expected fallback (WasIncremental=false) when AI returns no parseable sections")
	}
	// Should have made 2 calls: one incremental (failed parse) + one full fallback.
	if prov.calls != 2 {
		t.Errorf("expected 2 AI calls (incr + fallback), got %d", prov.calls)
	}
}

// TestPolishIncremental_MinChangeLinesRespected — AC-4: sections
// with fewer non-blank lines than the threshold are skipped.
func TestPolishIncremental_MinChangeLinesRespected(t *testing.T) {
	doc := twoSectionDoc()
	sections := SplitSections(doc)
	stored := HashSections(sections)

	// Change one word in "## Why" (1 non-blank line section).
	mutated := strings.Replace(doc, "Because reasons.", "Because other reasons.", 1)

	prov := &fakePolishProvider{fixedOut: "should not be called"}
	opts := baseIncrOpts(prov)
	opts.MinChangeLines = 5 // "## Why" body has only 1 non-blank line → skip

	res, err := PolishIncremental(context.Background(), mutated, stored, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Skipped {
		t.Error("expected Skipped=true when changed section is below MinChangeLines threshold")
	}
	if prov.calls != 0 {
		t.Errorf("expected 0 AI calls, got %d", prov.calls)
	}
}

// TestPolishIncremental_SingleSectionDocFallsBack — AC-8: docs with
// fewer than 2 sections fall back to full polish.
func TestPolishIncremental_SingleSectionDocFallsBack(t *testing.T) {
	doc := "---\ntitle: Short\n---\n\nJust a paragraph, no headings.\n"
	prov := &fakePolishProvider{fixedOut: doc}
	stored := map[string]string{"": ContentHash([]byte("old"))}
	res, err := PolishIncremental(context.Background(), doc, stored, baseIncrOpts(prov))
	if err != nil {
		t.Fatal(err)
	}
	if res.WasIncremental {
		t.Error("expected fallback for single-section doc")
	}
}

// TestPolishState_RoundTrip verifies save + load of polish state.
func TestPolishState_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "polish-state.json")

	state := &PolishState{
		Entries: map[string]PolishStateEntry{
			"auth.md": {
				SectionHashes: map[string]string{
					"## Why": "sha256:abc",
					"## How": "sha256:def",
				},
			},
		},
	}
	if err := SavePolishState(path, state); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPolishState(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != PolishStateVersion {
		t.Errorf("version = %d", loaded.Version)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("entries = %d", len(loaded.Entries))
	}
	entry := loaded.Entries["auth.md"]
	if entry.SectionHashes["## Why"] != "sha256:abc" {
		t.Error("section hash mismatch")
	}
}

// TestPolishState_MissingFileReturnsEmpty mirrors LoadDraftState contract.
func TestPolishState_MissingFileReturnsEmpty(t *testing.T) {
	state, err := LoadPolishState(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(state.Entries))
	}
}

// TestHashSections_DeterministicAndDistinct verifies that different
// section bodies produce different hashes and that the same body
// always produces the same hash.
func TestHashSections_DeterministicAndDistinct(t *testing.T) {
	sections := []Section{
		{Heading: "", Body: "preamble"},
		{Heading: "## A", Body: "content A"},
		{Heading: "## B", Body: "content B"},
	}
	h1 := HashSections(sections)
	h2 := HashSections(sections)
	if h1["## A"] != h2["## A"] {
		t.Error("same content should produce same hash")
	}
	if h1["## A"] == h1["## B"] {
		t.Error("different content should produce different hash")
	}
	if h1[""] == h1["## A"] {
		t.Error("preamble hash should differ from section hash")
	}
}

// TestDetectChangedSections_NewSection is detected as changed.
func TestDetectChangedSections_NewSection(t *testing.T) {
	sections := []Section{
		{Heading: "", Body: "pre"},
		{Heading: "## A", Body: "a"},
		{Heading: "## B", Body: "b"},
	}
	// stored only knows "## A"
	stored := map[string]string{
		"":     ContentHash([]byte("pre")),
		"## A": ContentHash([]byte("a")),
	}
	changed, allUnchanged := DetectChangedSections(sections, stored, 0)
	if allUnchanged {
		t.Error("should not be allUnchanged — ## B is new")
	}
	if len(changed) != 1 || changed[0] != 2 {
		t.Errorf("expected [2], got %v", changed)
	}
}

// TestPolishState_CorruptFileReturnsError ensures corrupt JSON is
// detected and ErrStateCorrupt is returned.
func TestPolishState_CorruptFileReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{{{"), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := LoadPolishState(path)
	if err == nil {
		t.Fatal("expected error on corrupt file")
	}
	// Should still return a usable empty state.
	if len(state.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(state.Entries))
	}
}
