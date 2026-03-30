// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package generator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	loretemplate "github.com/greycoderk/lore/internal/template"
)

func TestGenerate_ProducesContent(t *testing.T) {
	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template: new: %v", err)
	}

	input := GenerateInput{
		DocType: "decision",
		What:    "Add JWT auth middleware",
		Why:     "Stateless authentication for microservices",
		CommitInfo: &domain.CommitInfo{
			Hash: "abc1234",
			Date: time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC),
		},
		GeneratedBy:  "lore-demo",
		Alternatives: "- Session-based auth with Redis\n- OAuth2 with external provider",
		Impact:       "- API routes now require Bearer token",
	}

	result, err := Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("generator: generate: %v", err)
	}

	if !strings.Contains(result.Body, "Add JWT auth middleware") {
		t.Error("expected 'what' in body")
	}
	if !strings.Contains(result.Body, "Stateless authentication") {
		t.Error("expected 'why' in body")
	}
}

func TestGenerate_CancelledContext(t *testing.T) {
	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template: new: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = Generate(ctx, engine, GenerateInput{DocType: "decision"}) //nolint:errcheck — error is the subject
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestGenerate_DateFromCommitInfo(t *testing.T) {
	fixed := time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)
	input := GenerateInput{
		CommitInfo: &domain.CommitInfo{Date: fixed},
	}
	if got := input.Date(); got != "2026-03-07" {
		t.Errorf("Date() = %q, want %q", got, "2026-03-07")
	}
}

func TestGenerate_DateFallsBackToToday(t *testing.T) {
	before := time.Now().Format("2006-01-02")
	input := GenerateInput{} // no CommitInfo
	got := input.Date()
	after := time.Now().Format("2006-01-02")
	// Accept either before or after to handle midnight crossing.
	if got != before && got != after {
		t.Errorf("Date() = %q, want %q or %q", got, before, after)
	}
}

func TestGenerate_CommitInfoNilNocrash(t *testing.T) {
	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template: new: %v", err)
	}

	input := GenerateInput{
		DocType:     "feature",
		What:        "add search",
		Why:         "usability",
		GeneratedBy: "hook",
		// CommitInfo intentionally nil
	}

	result, err := Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("Generate with nil CommitInfo: %v", err)
	}
	if !strings.Contains(result.Body, "add search") {
		t.Error("expected 'what' in body")
	}
}

func TestGenerate_AllInputFields(t *testing.T) {
	engine, err := loretemplate.New("", "")
	if err != nil {
		t.Fatalf("template: new: %v", err)
	}

	input := GenerateInput{
		DocType:      "feature",
		What:         "add pagination",
		Why:          "performance",
		Alternatives: "cursor-based",
		Impact:       "breaking change",
		CommitInfo: &domain.CommitInfo{
			Hash:   "deadbeef",
			Author: "Dev",
			Date:   time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		},
		GeneratedBy: "hook",
	}

	result, err := Generate(context.Background(), engine, input)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if result.Body == "" {
		t.Error("expected non-empty body")
	}
	if result.Meta.Type != "feature" {
		t.Errorf("Meta.Type = %q, want %q", result.Meta.Type, "feature")
	}
	if result.Meta.Commit != "deadbeef" {
		t.Errorf("Meta.Commit = %q, want %q", result.Meta.Commit, "deadbeef")
	}
	if result.Meta.Status != "draft" {
		t.Errorf("Meta.Status = %q, want %q", result.Meta.Status, "draft")
	}
}
