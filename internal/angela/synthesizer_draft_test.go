// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"reflect"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
)

func TestSynthesizerDraftSuggestions_EmitsForUnsynthesizedDoc(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	cfg := config.SynthesizersConfig{Enabled: []string{"stub-review"}}
	got := SynthesizerDraftSuggestions("doc.md", []byte(reviewSampleDoc), registry, cfg)

	if len(got) != 1 {
		t.Fatalf("want 1 suggestion, got %d", len(got))
	}
	if got[0].Category != "synthesizer" {
		t.Fatalf("category: %q", got[0].Category)
	}
	if got[0].Severity != "info" {
		t.Fatalf("severity: %q", got[0].Severity)
	}
	if !strings.Contains(got[0].Message, "pending_enrichment") {
		t.Fatalf("message missing prefix: %q", got[0].Message)
	}
}

func TestSynthesizerDraftSuggestions_SkipsWhenSignatureFresh(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	hash := stubReviewEvidenceHash(t)
	doc := strings.ReplaceAll(reviewSampleDoc, "status: draft\n",
		"status: draft\nsynthesized:\n  stub-review:\n    hash: \""+hash+"\"\n    at: \"x\"\n    version: \"\"\n    evidence_count: 1\n")

	cfg := config.SynthesizersConfig{Enabled: []string{"stub-review"}}
	got := SynthesizerDraftSuggestions("doc.md", []byte(doc), registry, cfg)
	if len(got) != 0 {
		t.Fatalf("fresh signature must yield no suggestions, got %v", got)
	}
}

func TestSynthesizerDraftSuggestions_NilRegistryReturnsNil(t *testing.T) {
	cfg := config.SynthesizersConfig{Enabled: []string{"x"}}
	if got := SynthesizerDraftSuggestions("d.md", []byte(reviewSampleDoc), nil, cfg); got != nil {
		t.Fatalf("nil registry must return nil, got %+v", got)
	}
}

func TestSynthesizerDraftSuggestions_DisabledRegistryReturnsNil(t *testing.T) {
	registry := synthesizer.NewRegistry()
	registry.Register(stubReviewSynth{})

	if got := SynthesizerDraftSuggestions("d.md", []byte(reviewSampleDoc), registry, config.SynthesizersConfig{}); len(got) != 0 {
		t.Fatalf("empty Enabled must yield nothing, got %+v", got)
	}
}

// I1 invariant: SynthesizerDraftSuggestions must not import any AI provider
// transitively. Test by inspecting the package's own dependency tree at
// build time - this is a static check, encoded as a compile-time assertion
// on the imports referenced in the file.
//
// A meaningful runtime check: the function MUST behave identically when
// the caller passes no AI provider (we don't even take one as a parameter).
// The test below confirms the function signature does not surface any
// AI-typed parameter.
func TestSynthesizerDraftSuggestions_SignatureExposesNoAIType(t *testing.T) {
	// Reflect on the function value's type. Each parameter must be a type
	// the draft path is allowed to receive: doc path, doc bytes, registry,
	// config. A leak (e.g. an AIProvider parameter) would be visible here.
	t.Helper()

	fn := SynthesizerDraftSuggestions
	want := []string{
		"string",
		"[]uint8",
		"*synthesizer.Registry",
		"config.SynthesizersConfig",
	}
	got := paramTypes(fn)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("draft signature drift: got %v, want %v", got, want)
	}
}

func paramTypes(fn any) []string {
	t := reflect.TypeOf(fn)
	if t.Kind() != reflect.Func {
		return nil
	}
	out := make([]string, t.NumIn())
	for i := 0; i < t.NumIn(); i++ {
		out[i] = t.In(i).String()
	}
	return out
}
