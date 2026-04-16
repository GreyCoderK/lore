// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestComputeHash_DeterministicAcrossCallOrder(t *testing.T) {
	a := Evidence{Field: "month", File: "doc.md", Line: 10, ColStart: 0, ColEnd: 5, Snippet: "month", Rule: "literal"}
	b := Evidence{Field: "year", File: "doc.md", Line: 11, ColStart: 0, ColEnd: 4, Snippet: "year", Rule: "literal"}

	h1 := ComputeHash([]Evidence{a, b})
	h2 := ComputeHash([]Evidence{b, a})

	if h1 != h2 {
		t.Fatalf("hash must be order-insensitive: %s vs %s", h1, h2)
	}
}

func TestComputeHash_EmptyEvidenceReturnsEmpty(t *testing.T) {
	if got := ComputeHash(nil); got != "" {
		t.Fatalf("empty evidence must produce empty hash, got %q", got)
	}
	if got := ComputeHash([]Evidence{}); got != "" {
		t.Fatalf("empty slice must produce empty hash, got %q", got)
	}
}

func TestComputeHash_ChangesWhenSnippetChanges(t *testing.T) {
	base := Evidence{Field: "f", File: "d.md", Line: 1, ColStart: 0, ColEnd: 5, Snippet: "alpha", Rule: "literal"}
	shifted := base
	shifted.Snippet = "beta"

	if ComputeHash([]Evidence{base}) == ComputeHash([]Evidence{shifted}) {
		t.Fatal("hash must shift when snippet text changes")
	}
}

func TestComputeHash_ChangesWhenSpanChanges(t *testing.T) {
	base := Evidence{Field: "f", File: "d.md", Line: 1, ColStart: 0, ColEnd: 5, Snippet: "alpha", Rule: "literal"}
	shifted := base
	shifted.Line = 2

	if ComputeHash([]Evidence{base}) == ComputeHash([]Evidence{shifted}) {
		t.Fatal("hash must shift when source line changes")
	}
}

func TestComputeHash_StartsWithPrefix(t *testing.T) {
	ev := Evidence{Field: "f", File: "d.md", Line: 1, ColStart: 0, ColEnd: 5, Snippet: "alpha", Rule: "literal"}
	got := ComputeHash([]Evidence{ev})
	if !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("hash must be prefixed with sha256:, got %q", got)
	}
}

func TestEscapeSnippet_HandlesPipesAndNewlines(t *testing.T) {
	got := escapeSnippet("a|b\nc")
	if got != `a\|b\nc` {
		t.Fatalf("escape: got %q", got)
	}
}

func TestMakeSignature_FieldsPopulated(t *testing.T) {
	evs := []Evidence{
		{Field: "month", File: "d.md", Line: 1, Snippet: "month", Rule: "literal"},
	}
	warnings := []Warning{{Code: "fuzzy-heading-match"}}
	sig := MakeSignature("1.0.0", evs, []string{"endpoints", "filters"}, warnings)

	if sig.Hash == "" {
		t.Fatal("Hash empty")
	}
	if sig.Version != "1.0.0" {
		t.Fatalf("Version: %q", sig.Version)
	}
	if sig.EvidenceCount != 1 {
		t.Fatalf("EvidenceCount: %d", sig.EvidenceCount)
	}
	if len(sig.Sections) != 2 {
		t.Fatalf("Sections: %v", sig.Sections)
	}
	if len(sig.Warnings) != 1 || sig.Warnings[0] != "fuzzy-heading-match" {
		t.Fatalf("Warnings: %v", sig.Warnings)
	}
	if sig.At == "" {
		t.Fatal("At not populated")
	}
}

func TestSignaturesRoundTrip_MetaAndBack(t *testing.T) {
	original := map[string]Signature{
		"api-postman": {
			Hash:          "sha256:abc",
			At:            "2026-04-15T10:00:00Z",
			Version:       "1.0.0",
			Sections:      []string{"endpoints", "filters"},
			EvidenceCount: 14,
			Warnings:      []string{"missing-security-section"},
		},
	}
	var meta domain.DocMeta
	SignaturesToMeta(&meta, original)

	if meta.Synthesized == nil {
		t.Fatal("Synthesized was not written to meta")
	}

	// Round-trip through SignaturesFromMeta.
	roundTrip := SignaturesFromMeta(meta)
	got, ok := roundTrip["api-postman"]
	if !ok {
		t.Fatal("api-postman key lost in round trip")
	}
	want := original["api-postman"]
	if got.Hash != want.Hash || got.At != want.At || got.Version != want.Version {
		t.Fatalf("scalar fields drifted: got %+v, want %+v", got, want)
	}
	if got.EvidenceCount != want.EvidenceCount {
		t.Fatalf("evidence count drift: got %d, want %d", got.EvidenceCount, want.EvidenceCount)
	}
	if len(got.Sections) != len(want.Sections) || len(got.Warnings) != len(want.Warnings) {
		t.Fatalf("slice lengths drifted: got %+v, want %+v", got, want)
	}
}

func TestSignaturesToMeta_EmptyClearsField(t *testing.T) {
	meta := domain.DocMeta{Synthesized: map[string]map[string]any{"existing": {"hash": "x"}}}
	SignaturesToMeta(&meta, nil)
	if meta.Synthesized != nil {
		t.Fatalf("empty input must clear the field, got %v", meta.Synthesized)
	}
}

func TestIsFresh_TrueOnMatchingHashAndVersion(t *testing.T) {
	evs := []Evidence{{Field: "f", File: "d.md", Line: 1, Snippet: "x", Rule: "literal"}}
	sig := MakeSignature("1.0.0", evs, nil, nil)

	if !IsFresh(sig, evs, "1.0.0") {
		t.Fatal("IsFresh must return true on identical evidence + matching version")
	}
}

func TestIsFresh_FalseOnEmptyHash(t *testing.T) {
	if IsFresh(Signature{}, []Evidence{{Field: "f", Snippet: "x"}}, "1.0.0") {
		t.Fatal("empty signature must report stale")
	}
}

func TestIsFresh_FalseOnVersionBump(t *testing.T) {
	evs := []Evidence{{Field: "f", File: "d.md", Line: 1, Snippet: "x", Rule: "literal"}}
	sig := MakeSignature("1.0.0", evs, nil, nil)
	if IsFresh(sig, evs, "2.0.0") {
		t.Fatal("version bump must invalidate cache")
	}
}

func TestIsFresh_FalseOnEvidenceShift(t *testing.T) {
	original := []Evidence{{Field: "f", File: "d.md", Line: 1, Snippet: "x", Rule: "literal"}}
	sig := MakeSignature("1.0.0", original, nil, nil)

	shifted := []Evidence{{Field: "f", File: "d.md", Line: 2, Snippet: "x", Rule: "literal"}}
	if IsFresh(sig, shifted, "1.0.0") {
		t.Fatal("source line shift must invalidate cache")
	}
}
