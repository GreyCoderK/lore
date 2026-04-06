// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import (
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Add JWT auth", "add-jwt-auth"},
		{"  spaces  ", "spaces"},
		{"UPPER CASE", "upper-case"},
		{"special!@#$chars", "special-chars"},
		{"already-slug", "already-slug"},
		{"", ""},
		{"---dashes---", "dashes"},
		{"multiple   spaces", "multiple-spaces"},
		{"MixedCase123", "mixedcase123"},
	}
	for _, tt := range tests {
		got := Slugify(tt.in)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidDocType(t *testing.T) {
	valid := []string{"decision", "feature", "bugfix", "refactor", "release", "note", "summary"}
	for _, v := range valid {
		if !ValidDocType(v) {
			t.Errorf("ValidDocType(%q) = false, want true", v)
		}
	}

	invalid := []string{"", "invalid", "DECISION", "Feature", "story", "plan"}
	for _, v := range invalid {
		if ValidDocType(v) {
			t.Errorf("ValidDocType(%q) = true, want false", v)
		}
	}
}

func TestDocTypeNames(t *testing.T) {
	names := DocTypeNames()
	if len(names) < 7 {
		t.Fatalf("DocTypeNames() returned %d types, want at least 7", len(names))
	}
	// Verify sorted
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("DocTypeNames() not sorted: %q before %q", names[i-1], names[i])
		}
	}
	// Verify contains known types
	found := make(map[string]bool)
	for _, n := range names {
		found[n] = true
	}
	for _, expected := range []string{"decision", "feature", "bugfix", "summary"} {
		if !found[expected] {
			t.Errorf("DocTypeNames() missing %q", expected)
		}
	}
}

func TestWithModel(t *testing.T) {
	opts := &CallOptions{}
	WithModel("claude-3")(opts)
	if opts.Model != "claude-3" {
		t.Errorf("Model = %q, want %q", opts.Model, "claude-3")
	}
}

func TestWithMaxTokens(t *testing.T) {
	opts := &CallOptions{}
	WithMaxTokens(4096)(opts)
	if opts.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d", opts.MaxTokens, 4096)
	}
}

func TestWithTemperature(t *testing.T) {
	opts := &CallOptions{}
	WithTemperature(0.7)(opts)
	if opts.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want %f", opts.Temperature, 0.7)
	}
}

func TestWithTimeout(t *testing.T) {
	opts := &CallOptions{}
	WithTimeout(30 * time.Second)(opts)
	if opts.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want %v", opts.Timeout, 30*time.Second)
	}
}

func TestWithSystem(t *testing.T) {
	opts := &CallOptions{}
	WithSystem("You are Angela")(opts)
	if opts.System != "You are Angela" {
		t.Errorf("System = %q, want %q", opts.System, "You are Angela")
	}
}

func TestOptionChaining(t *testing.T) {
	opts := &CallOptions{}
	for _, opt := range []Option{
		WithModel("gpt-4"),
		WithMaxTokens(2000),
		WithTemperature(0.5),
		WithSystem("test"),
	} {
		opt(opts)
	}
	if opts.Model != "gpt-4" || opts.MaxTokens != 2000 || opts.Temperature != 0.5 || opts.System != "test" {
		t.Errorf("chained options not applied correctly: %+v", opts)
	}
}
