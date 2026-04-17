// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
)

// ═══════════════════════════════════════════════════════════════════════════
// Invariant I7 — No silent merge.
//
// Contract: once a synthesized block exists in a document, any edit to the
// source must flip the signature so the next polish/draft/review run
// DETECTS the change and requires user confirmation before overwriting.
// Silent auto-merge is strictly forbidden — it would let an upstream doc
// edit erase a downstream synthesizer output without the user knowing.
//
// Two enforcement layers (matrix requires ≥2):
//   1. Runtime signature check (TestI7_SignatureFlipsOnSourceEdit).
//   2. Static wiring check (TestI7_GatingHooksWireToIsFresh) — any new
//      command path that writes synthesized blocks MUST import IsFresh.
// ═══════════════════════════════════════════════════════════════════════════

// TestI7_SignatureFlipsOnSourceEdit is the runtime anchor for I7. It builds
// a signature over a small evidence set, mutates the source snippet, and
// asserts IsFresh returns false — proving the gate would refuse to
// auto-merge.
func TestI7_SignatureFlipsOnSourceEdit(t *testing.T) {
	evs := []synthesizer.Evidence{
		{Field: "month", File: "a.md", Line: 2, ColStart: 10, ColEnd: 15, Snippet: "month", Rule: "literal"},
		{Field: "amount", File: "a.md", Line: 2, ColStart: 20, ColEnd: 26, Snippet: "amount", Rule: "literal"},
	}
	sig := synthesizer.MakeSignature("v1", evs, []string{"endpoints"}, nil)

	// Same evidence → fresh.
	if !synthesizer.IsFresh(sig, evs, "v1") {
		t.Error("signature must be fresh when evidence unchanged")
	}

	// Mutate the first snippet (simulates a source doc edit).
	mutated := make([]synthesizer.Evidence, len(evs))
	copy(mutated, evs)
	mutated[0].Snippet = mutated[0].Snippet + "_EDITED"

	if synthesizer.IsFresh(sig, mutated, "v1") {
		t.Error("I7 violation: IsFresh returned true after source edit — silent merge would occur")
	}

	// Version bump alone also flips freshness.
	if synthesizer.IsFresh(sig, evs, "v2") {
		t.Error("I7 violation: version bump must invalidate freshness")
	}
}

// TestI7_GatingHooksWireToIsFresh is the static-check layer. It scans the
// source files that write synthesized blocks (draft/polish/review hooks)
// and verifies each one imports the signature gate. A new hook that forgot
// to route through IsFresh would leak past this check AND silently
// auto-merge — which is exactly the I7 violation we want to prevent.
//
// This is a "presence grep" rather than a proper static analysis because
// Go makes it hard to prove the gate is actually CALLED on the write path
// without running the code. The presence of the import + a call-site is
// strong signal that the maintainer wired the gate; combined with the
// runtime test above, it gives the matrix's 2-layer requirement.
func TestI7_GatingHooksWireToIsFresh(t *testing.T) {
	hookFiles := []string{
		"synthesizer_draft.go",
		"synthesizer_polish.go",
		"synthesizer_review.go",
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	missing := []string{}
	for _, name := range hookFiles {
		path := filepath.Join(wd, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		content := string(data)

		// The import line — every hook that writes synthesized blocks must
		// pull the signature package.
		if !strings.Contains(content, `"github.com/greycoderk/lore/internal/angela/synthesizer"`) {
			missing = append(missing,
				name+": missing import of synthesizer package")
			continue
		}
		// Must call IsFresh on the write path. A nil check is a red flag —
		// it'd bypass the gate when no prior signature existed.
		if !strings.Contains(content, "synthesizer.IsFresh(") {
			missing = append(missing,
				name+": does not call synthesizer.IsFresh — I7 gate not wired")
		}
	}

	if len(missing) > 0 {
		t.Errorf("I7 static wiring check failed:\n  %s", strings.Join(missing, "\n  "))
	}
}
