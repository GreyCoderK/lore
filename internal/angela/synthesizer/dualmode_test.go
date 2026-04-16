// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"strings"
	"testing"
)

// verify the framework's dual-mode behavior (I2).
//
// The framework MUST work identically in all three modes detected by story
// 8-4 (lore-native, hybrid, standalone). Two design choices make this
// trivially true and worth pinning with a test:
//
//  1. Signatures live in frontmatter (decision Q3, 2026-04-15), not in
//     .lore/state/. There is no on-disk state the framework consults outside
//     the doc itself.
//  2. ParseDoc uses storage.UnmarshalPermissive, which accepts non-lore
//     frontmatter (mkdocs, hugo, docusaurus). It does NOT enforce
//     ValidDocType, missing date, etc.
//
// The tests below pin both choices.

const minimalStandaloneDoc = `---
type: blog-post
title: standalone
---

## Endpoints

- POST /api/foo with month
`

func TestDualMode_StandaloneDocParsesWithoutLoreFrontmatter(t *testing.T) {
	doc, err := ParseDoc("blog.md", []byte(minimalStandaloneDoc))
	if err != nil {
		t.Fatalf("permissive parser must accept non-lore frontmatter: %v", err)
	}
	if doc.Meta.Type != "blog-post" {
		t.Fatalf("frontmatter not parsed: %q", doc.Meta.Type)
	}
	if len(doc.Sections) == 0 {
		t.Fatal("sections must be parsed even without lore-validated frontmatter")
	}
}

func TestDualMode_SignaturesRoundTripWithoutLoreState(t *testing.T) {
	docBody := strings.ReplaceAll(minimalStandaloneDoc, "title: standalone\n",
		"title: standalone\nsynthesized:\n  api-postman:\n    hash: \"sha256:xxx\"\n    at: \"2026-04-15T00:00:00Z\"\n    version: \"1.0.0\"\n    evidence_count: 3\n")

	doc, err := ParseDoc("blog.md", []byte(docBody))
	if err != nil {
		t.Fatal(err)
	}
	sig, ok := doc.Signatures["api-postman"]
	if !ok {
		t.Fatal("signature not loaded from frontmatter in standalone-style doc")
	}
	if sig.Hash != "sha256:xxx" {
		t.Fatalf("hash drift: %q", sig.Hash)
	}
}

func TestDualMode_NoExternalStateRequired(t *testing.T) {
	// A clean ParseDoc on a doc with no Synthesized frontmatter must yield
	// an empty (not nil-broken) signature map. This ensures lookups like
	// doc.Signatures["x"] always work without nil checks.
	doc, err := ParseDoc("d.md", []byte(minimalStandaloneDoc))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Signatures == nil {
		t.Fatal("Signatures must always be initialized, even when frontmatter has no synthesized key")
	}
	// Lookup of unknown key returns zero value, not panic.
	if _, ok := doc.Signatures["never-registered"]; ok {
		t.Fatal("unknown synthesizer should not be in Signatures")
	}
}
