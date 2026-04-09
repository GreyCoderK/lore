// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "testing"

func TestResolveMaxTokens_Polish_DynamicCap(t *testing.T) {
	tests := []struct {
		name         string
		docWordCount int
		want         int
	}{
		{"zero words fallback", 0, 2000},
		{"10 words floor 512", 10, 512},       // 10*1.3*1.8=23.4 → floor 512
		{"100 words", 100, 512},               // 100*1.3*1.8=234 → floor 512
		{"300 words", 300, 702},               // 300*1.3*1.8=702
		{"1000 words", 1000, 2340},            // 1000*1.3*1.8=2340
		{"2000 words", 2000, 4680},            // 2000*1.3*1.8=4680
		{"3000 words", 3000, 7020},            // 3000*1.3*1.8=7020
		{"4000 words cap 8192", 4000, 8192},   // 4000*1.3*1.8=9360 → cap 8192
		{"5000 words cap 8192", 5000, 8192},   // way over cap
		{"negative words fallback", -1, 2000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveMaxTokens("polish", tt.docWordCount)
			if got != tt.want {
				t.Errorf("ResolveMaxTokens(polish, %d) = %d, want %d", tt.docWordCount, got, tt.want)
			}
		})
	}
}

func TestResolveMaxTokens_FixedModes(t *testing.T) {
	tests := []struct {
		mode string
		want int
	}{
		{"review", 1500},
		{"render", 512},
		{"ask", 1024},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := ResolveMaxTokens(tt.mode, 0)
			if got != tt.want {
				t.Errorf("ResolveMaxTokens(%q, 0) = %d, want %d", tt.mode, got, tt.want)
			}
		})
	}
}

func TestResolveMaxTokens_UnknownMode_Default2000(t *testing.T) {
	got := ResolveMaxTokens("unknown-mode", 500)
	if got != 2000 {
		t.Errorf("ResolveMaxTokens(unknown-mode, 500) = %d, want 2000", got)
	}
}

func TestResolveMaxTokens_EmptyMode_Default2000(t *testing.T) {
	got := ResolveMaxTokens("", 500)
	if got != 2000 {
		t.Errorf("ResolveMaxTokens(\"\", 500) = %d, want 2000", got)
	}
}
