// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"context"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

const reviewSampleDoc = `---
type: feature
date: "2026-04-15"
status: draft
---

# F

## Endpoints

- POST /api/foo with month
`

// fakeReader is a CorpusReader that serves docs from an in-memory map.
type fakeReader struct {
	docs map[string]string
}

func (f *fakeReader) ReadDoc(id string) (string, error) {
	if body, ok := f.docs[id]; ok {
		return body, nil
	}
	return "", nil
}

func (f *fakeReader) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error) {
	return nil, nil
}

// stubReviewSynth emits one Candidate plus one literal Evidence located at
// the "month" token in the doc.
type stubReviewSynth struct{}

func (stubReviewSynth) Name() string      { return "stub-review" }
func (stubReviewSynth) Applies(*synthesizer.Doc) bool { return true }

func (stubReviewSynth) Detect(*synthesizer.Doc) ([]synthesizer.Candidate, error) {
	return []synthesizer.Candidate{{Key: "POST /api/foo"}}, nil
}

func (stubReviewSynth) Synthesize(_ synthesizer.Candidate, _ synthesizer.Config) (synthesizer.Block, []synthesizer.Evidence, []synthesizer.Warning, error) {
	return synthesizer.Block{
			Title:    "Example",
			Language: "http+json",
			Content:  `{"month":"{{month}}"}`,
		},
		[]synthesizer.Evidence{
			{
				Field: "month", File: "doc.md", Line: 9,
				ColStart: strings.Index("- POST /api/foo with month", "month"),
				ColEnd:   strings.Index("- POST /api/foo with month", "month") + 5,
				Snippet:  "month",
				Rule:     "literal",
			},
		},
		nil,
		nil
}

func TestRunSynthesizerReview_EmitsFindingForUnsynthesizedDoc(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	reader := &fakeReader{docs: map[string]string{"doc.md": reviewSampleDoc}}
	docs := []domain.DocMeta{{Type: "feature", Filename: "doc.md"}}
	cfg := config.SynthesizersConfig{Enabled: []string{"stub-review"}}

	findings, err := RunSynthesizerReview(context.Background(), reader, docs, registry, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Fatalf("want 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Severity != "info" {
		t.Fatalf("default severity must be info, got %q", f.Severity)
	}
	if !strings.Contains(f.Title, "stub-review") {
		t.Fatalf("title should reference synthesizer name: %q", f.Title)
	}
	if len(f.Evidence) != 1 || f.Evidence[0].Quote != "month" {
		t.Fatalf("evidence not propagated: %+v", f.Evidence)
	}
}

func TestRunSynthesizerReview_NoFindingWhenSignatureFresh(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	// Compute the hash the stub will produce, then write it into the doc's
	// frontmatter so the freshness check short-circuits.
	// version must match cfg lookup (empty here since cfg.PerSynthesizer
	// is unset) — code review finding #10 strict-version gate.
	doc := strings.ReplaceAll(reviewSampleDoc, "status: draft\n", "status: draft\nsynthesized:\n  stub-review:\n    hash: \"{{HASH}}\"\n    at: \"2026-04-15T10:00:00Z\"\n    version: \"\"\n    evidence_count: 1\n")
	want := stubReviewEvidenceHash(t)
	doc = strings.ReplaceAll(doc, "{{HASH}}", want)

	reader := &fakeReader{docs: map[string]string{"doc.md": doc}}
	docs := []domain.DocMeta{{Type: "feature", Filename: "doc.md"}}
	cfg := config.SynthesizersConfig{Enabled: []string{"stub-review"}}

	findings, err := RunSynthesizerReview(context.Background(), reader, docs, registry, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("fresh signature must skip finding, got %d findings: %+v", len(findings), findings)
	}
}

func TestRunSynthesizerReview_DisabledRegistryReturnsNothing(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	reader := &fakeReader{docs: map[string]string{"doc.md": reviewSampleDoc}}
	docs := []domain.DocMeta{{Type: "feature", Filename: "doc.md"}}
	cfg := config.SynthesizersConfig{Enabled: nil} // empty -> no synthesizers active

	findings, err := RunSynthesizerReview(context.Background(), reader, docs, registry, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Fatalf("disabled registry must yield zero findings, got %d", len(findings))
	}
}

func TestResolveSynthesizerSeverity_RespectsOverride(t *testing.T) {
	cfg := config.SynthesizersConfig{
		PerSynthesizer: map[string]map[string]any{
			"review": {"severity": "warning"},
		},
	}
	if got := resolveSynthesizerSeverity(cfg); got != "warning" {
		t.Fatalf("override ignored: got %q, want %q", got, "warning")
	}
}

func TestResolveSynthesizerSeverity_DefaultsToInfo(t *testing.T) {
	if got := resolveSynthesizerSeverity(config.SynthesizersConfig{}); got != "info" {
		t.Fatalf("default must be info, got %q", got)
	}
}

// stubReviewEvidenceHash recomputes the hash the stubReviewSynth will
// produce so the freshness test can pre-populate frontmatter with it.
func stubReviewEvidenceHash(t *testing.T) string {
	t.Helper()
	idx := strings.Index("- POST /api/foo with month", "month")
	ev := synthesizer.Evidence{
		Field: "month", File: "doc.md", Line: 9,
		ColStart: idx, ColEnd: idx + 5,
		Snippet: "month", Rule: "literal",
	}
	return synthesizer.ComputeHash([]synthesizer.Evidence{ev})
}
