// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "testing"

func TestDetectLanguage_VHS(t *testing.T) {
	tests := []struct {
		line string
		want string
	}{
		{"Output docs/assets/vhs/demo.gif", "vhs"},
		{"Type \"lore angela review\"", "vhs"},
		{"Set Shell \"bash\"", "vhs"},
		{"Set FontSize 14", "vhs"},
		{"Sleep 3s", "vhs"},
		{"Hide", "vhs"},
		{"Show", "vhs"},
		{"Enter", "vhs"},
		{"Require lore", "vhs"},
		{"Screenshot demo.png", "vhs"},
		{"Ctrl+C", "vhs"},
		{"Source setup.tape", "vhs"},
	}
	for _, tt := range tests {
		got := DetectLanguage(tt.line)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.line, got, tt.want)
		}
	}
}

func TestDetectLanguageMultiLine_VHS(t *testing.T) {
	lines := []string{
		"Output docs/assets/vhs/angela-review.gif",
		"Set Shell \"bash\"",
		"Set FontSize 14",
		"Type \"lore angela review\"",
		"Enter",
		"Sleep 25s",
	}
	got := DetectLanguageMultiLine(lines)
	if got != "vhs" {
		t.Errorf("DetectLanguageMultiLine(VHS tape) = %q, want %q", got, "vhs")
	}
}

func TestDetectLanguage_VHS_NotFalsePositive(t *testing.T) {
	// These should NOT match VHS
	tests := []struct {
		line string
		not  string
	}{
		{"func main() {", "vhs"},
		{"import os", "vhs"},
		{"SELECT * FROM users", "vhs"},
	}
	for _, tt := range tests {
		got := DetectLanguage(tt.line)
		if got == tt.not {
			t.Errorf("DetectLanguage(%q) = %q, should not match %q", tt.line, got, tt.not)
		}
	}
}
