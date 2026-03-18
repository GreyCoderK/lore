package ui

import "testing"

func TestSuccessWithColor(t *testing.T) {
	SetColorEnabled(true)
	got := Success("ok")
	expected := "\033[32mok\033[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestSuccessWithoutColor(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()
	got := Success("ok")
	if got != "ok" {
		t.Errorf("expected plain 'ok', got %q", got)
	}
}

func TestWarningWithColor(t *testing.T) {
	SetColorEnabled(true)
	got := Warning("warn")
	expected := "\033[33mwarn\033[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestErrorWithColor(t *testing.T) {
	SetColorEnabled(true)
	got := Error("fail")
	expected := "\033[31mfail\033[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestDimWithColor(t *testing.T) {
	SetColorEnabled(true)
	got := Dim("grey")
	expected := "\033[2mgrey\033[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestBoldWithColor(t *testing.T) {
	SetColorEnabled(true)
	got := Bold("strong")
	expected := "\033[1mstrong\033[0m"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestAllColorsDisabled(t *testing.T) {
	restore := SaveAndDisableColor()
	defer restore()

	tests := []struct {
		name string
		fn   func(string) string
	}{
		{"Warning", Warning},
		{"Error", Error},
		{"Dim", Dim},
		{"Bold", Bold},
	}
	for _, tt := range tests {
		got := tt.fn("text")
		if got != "text" {
			t.Errorf("%s: expected plain 'text', got %q", tt.name, got)
		}
	}
}

// TestNoColorEnvVar verifies that ResetColorFromEnv() respects the NO_COLOR env var.
// L9 fix: init() runs once per process so t.Setenv alone cannot retrigger it;
// ResetColorFromEnv() must be called explicitly after the env is changed.
func TestNoColorEnvVar(t *testing.T) {
	prev := isColorEnabled()
	defer SetColorEnabled(prev) // restore previous state after test

	t.Setenv("NO_COLOR", "")
	ResetColorFromEnv()

	got := Success("ok")
	if got != "ok" {
		t.Errorf("NO_COLOR set: expected plain 'ok', got %q", got)
	}
}
