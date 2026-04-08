// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"testing"
)

func TestValidateType_AllValidTypes(t *testing.T) {
	validTypes := []string{
		"decision", "feature", "bugfix", "refactor", "release", "note", "summary",
	}
	for _, typ := range validTypes {
		normalized, ok := validateType(typ)
		if !ok {
			t.Errorf("validateType(%q) returned false, want true", typ)
		}
		if normalized != typ {
			t.Errorf("validateType(%q) normalized to %q, want %q", typ, normalized, typ)
		}
	}
}

func TestValidateType_CaseNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Feature", "feature"},
		{"BUGFIX", "bugfix"},
		{"Decision", "decision"},
		{"REFACTOR", "refactor"},
		{"Note", "note"},
		{"RELEASE", "release"},
		{"Summary", "summary"},
	}
	for _, tt := range tests {
		normalized, ok := validateType(tt.input)
		if !ok {
			t.Errorf("validateType(%q) returned false, want true (after lowercase)", tt.input)
		}
		if normalized != tt.want {
			t.Errorf("validateType(%q) = %q, want %q", tt.input, normalized, tt.want)
		}
	}
}

func TestValidateType_WhitespaceStripping(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{" feature ", "feature"},
		{"  bugfix\t", "bugfix"},
		{"\tnote\n", "note"},
	}
	for _, tt := range tests {
		normalized, ok := validateType(tt.input)
		if !ok {
			t.Errorf("validateType(%q) returned false, want true", tt.input)
		}
		if normalized != tt.want {
			t.Errorf("validateType(%q) = %q, want %q", tt.input, normalized, tt.want)
		}
	}
}

func TestValidateType_InvalidTypes(t *testing.T) {
	invalidTypes := []string{
		"", "invalid", "feat", "fix", "docs", "chore", "test", "perf",
		"bug", "feat-fix", "123", "feature!", "feature feature",
	}
	for _, typ := range invalidTypes {
		_, ok := validateType(typ)
		if ok {
			t.Errorf("validateType(%q) returned true, want false", typ)
		}
	}
}
