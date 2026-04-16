// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
)

func TestSynthesizerProposalsForDoc_EmitsForUnsynthesizedDoc(t *testing.T) {
	doc, err := synthesizer.ParseDoc("doc.md", []byte(reviewSampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	cfg := config.SynthesizersConfig{Enabled: []string{"stub-review"}}
	props, err := SynthesizerProposalsForDoc(doc, registry, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(props) != 1 {
		t.Fatalf("want 1 proposal, got %d", len(props))
	}
	if props[0].SynthesizerName != "stub-review" {
		t.Fatalf("unexpected synthesizer: %q", props[0].SynthesizerName)
	}
	if !strings.Contains(props[0].RenderedMarkdown, "Example") {
		t.Fatalf("rendered markdown missing title: %q", props[0].RenderedMarkdown)
	}
	if !strings.Contains(props[0].RenderedMarkdown, "```http+json") {
		t.Fatalf("rendered markdown missing fence: %q", props[0].RenderedMarkdown)
	}
}

func TestApplySynthesizerProposal_InsertsAfterHeading(t *testing.T) {
	doc, err := synthesizer.ParseDoc("doc.md", []byte(reviewSampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	block := synthesizer.Block{
		Title:              "Example",
		Language:           "json",
		Content:            "{}",
		InsertAfterHeading: "## Endpoints",
	}
	prop := SynthesizerProposal{
		Doc:              doc,
		SynthesizerName:  "stub",
		Block:            block,
		Signature:        synthesizer.Signature{Hash: "sha256:xxx", Version: "1.0.0"},
		RenderedMarkdown: RenderBlockMarkdown(doc, block),
	}

	newBody, newMeta, err := ApplySynthesizerProposal(prop)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(newBody, "### Example") {
		t.Fatalf("new body missing inserted heading: %s", newBody)
	}
	if newMeta.Synthesized["stub"]["hash"] != "sha256:xxx" {
		t.Fatalf("signature not propagated to meta: %+v", newMeta.Synthesized)
	}
}

func TestRenderBlockMarkdown_HeadingDepthBelowParent(t *testing.T) {
	doc, err := synthesizer.ParseDoc("doc.md", []byte(reviewSampleDoc))
	if err != nil {
		t.Fatal(err)
	}
	block := synthesizer.Block{
		Title:              "Sub",
		Language:           "json",
		Content:            "{}",
		InsertAfterHeading: "## Endpoints", // level 2 -> rendered should be ###
	}
	rendered := RenderBlockMarkdown(doc, block)
	if !strings.HasPrefix(rendered, "### ") {
		t.Fatalf("expected ### heading, got prefix: %q", rendered[:10])
	}
}

func TestRenderBlockMarkdown_NotesAsBullets(t *testing.T) {
	block := synthesizer.Block{
		Title:    "X",
		Language: "json",
		Content:  "{}",
		Notes:    []string{"first note", "second note"},
	}
	rendered := RenderBlockMarkdown(&synthesizer.Doc{}, block)
	if !strings.Contains(rendered, "- first note") || !strings.Contains(rendered, "- second note") {
		t.Fatalf("notes not rendered as bullets: %s", rendered)
	}
}

func TestSynthesizerProposalsForDoc_SkipsWhenSignatureFresh(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	hash := stubReviewEvidenceHash(t)
	doc := strings.ReplaceAll(reviewSampleDoc, "status: draft\n",
		"status: draft\nsynthesized:\n  stub-review:\n    hash: \""+hash+"\"\n    at: \"2026-04-15T10:00:00Z\"\n    version: \"\"\n    evidence_count: 1\n")

	parsed, err := synthesizer.ParseDoc("doc.md", []byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.SynthesizersConfig{Enabled: []string{"stub-review"}}
	props, err := SynthesizerProposalsForDoc(parsed, registry, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(props) != 0 {
		t.Fatalf("fresh signature must yield no proposals, got %d: %+v", len(props), props)
	}
}
