package template

import (
	"testing"
	"time"
)

func TestFormatDate(t *testing.T) {
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

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Hello World", "hello-world"},
		{"JWT Auth Middleware", "jwt-auth-middleware"},
		{"  spaces  ", "spaces"},
		{"already-slugged", "already-slugged"},
	}
	for _, tt := range tests {
		got := slugify(tt.input)
		if got != tt.want {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
