// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestResolveDocTypeFlags_NoFlags(t *testing.T) {
	docType, err := resolveDocTypeFlags("", false, false, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docType != "" {
		t.Errorf("expected empty type, got %q", docType)
	}
}

func TestResolveDocTypeFlags_TypeFlag(t *testing.T) {
	docType, err := resolveDocTypeFlags("feature", false, false, false, false, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if docType != "feature" {
		t.Errorf("expected 'feature', got %q", docType)
	}
}

func TestResolveDocTypeFlags_SingleShorthand(t *testing.T) {
	tests := []struct {
		name    string
		feature bool
		decision bool
		bugfix  bool
		refactor bool
		note    bool
		want    string
	}{
		{"feature", true, false, false, false, false, domain.DocTypeFeature},
		{"decision", false, true, false, false, false, domain.DocTypeDecision},
		{"bugfix", false, false, true, false, false, domain.DocTypeBugfix},
		{"refactor", false, false, false, true, false, domain.DocTypeRefactor},
		{"note", false, false, false, false, true, domain.DocTypeNote},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docType, err := resolveDocTypeFlags("", tt.feature, tt.decision, tt.bugfix, tt.refactor, tt.note)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if docType != tt.want {
				t.Errorf("got %q, want %q", docType, tt.want)
			}
		})
	}
}

func TestResolveDocTypeFlags_MultipleShorthands(t *testing.T) {
	_, err := resolveDocTypeFlags("", true, true, false, false, false)
	if err == nil {
		t.Error("expected error for multiple shorthands")
	}
}

func TestResolveDocTypeFlags_TypeAndShorthand(t *testing.T) {
	_, err := resolveDocTypeFlags("feature", false, true, false, false, false)
	if err == nil {
		t.Error("expected error when --type and shorthand both set")
	}
}
