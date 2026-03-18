// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package template

import (
	"testing"
	"time"
)

func TestFormatDate_TimeInput(t *testing.T) {
	tests := []struct {
		input time.Time
		want  string
	}{
		{time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC), "2026-03-07"},
		{time.Date(2025, 12, 25, 15, 30, 0, 0, time.UTC), "2025-12-25"},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), "2026-01-01"},
	}
	for _, tt := range tests {
		got := formatDate(tt.input)
		if got != tt.want {
			t.Errorf("formatDate(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDate_Zero(t *testing.T) {
	got := formatDate(time.Time{})
	if got != "" {
		t.Errorf("formatDate(zero) = %q, want empty string", got)
	}
}

func TestFormatDate_StringInput(t *testing.T) {
	got := formatDate("2026-03-12")
	if got != "2026-03-12" {
		t.Errorf("formatDate(string) = %q, want %q", got, "2026-03-12")
	}

	got = formatDate("")
	if got != "" {
		t.Errorf("formatDate(empty string) = %q, want empty", got)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Hello World", "hello-world"},
		{"JWT Auth Middleware", "jwt-auth-middleware"},
		{"  spaces  ", "spaces"},
		{"already-slugged", "already-slugged"},
		{"", ""},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCommitLink(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"abc1234def5678", "[`abc1234`](../../commit/abc1234def5678)"},
		{"deadbeef", "[`deadbee`](../../commit/deadbeef)"},
		{"", ""},
	}
	for _, tt := range tests {
		got := commitLink(tt.input)
		if got != tt.want {
			t.Errorf("commitLink(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
