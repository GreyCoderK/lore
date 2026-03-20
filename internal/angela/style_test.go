// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "testing"

func TestParseStyleGuide_WithRules(t *testing.T) {
	rules := map[string]interface{}{
		"require_why":          false,
		"require_alternatives": true,
		"max_body_length":      3000,
		"min_tags":             2,
	}
	guide := ParseStyleGuide(rules)
	if guide.RequireWhy != false {
		t.Error("RequireWhy should be false")
	}
	if guide.RequireAlternatives != true {
		t.Error("RequireAlternatives should be true")
	}
	if guide.MaxBodyLength != 3000 {
		t.Errorf("MaxBodyLength = %d, want 3000", guide.MaxBodyLength)
	}
	if guide.MinTags != 2 {
		t.Errorf("MinTags = %d, want 2", guide.MinTags)
	}
	if len(guide.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(guide.Warnings))
	}
}

func TestParseStyleGuide_Nil_Defaults(t *testing.T) {
	guide := ParseStyleGuide(nil)
	if guide.RequireWhy != true {
		t.Error("default RequireWhy should be true")
	}
	if guide.RequireAlternatives != false {
		t.Error("default RequireAlternatives should be false")
	}
	if guide.MaxBodyLength != 0 {
		t.Errorf("default MaxBodyLength = %d, want 0", guide.MaxBodyLength)
	}
	if guide.MinTags != 0 {
		t.Errorf("default MinTags = %d, want 0", guide.MinTags)
	}
}

func TestParseStyleGuide_UnknownKey_Warning(t *testing.T) {
	rules := map[string]interface{}{
		"require_why": true,
		"typo_key":    "oops",
	}
	guide := ParseStyleGuide(rules)
	if len(guide.Warnings) != 1 {
		t.Fatalf("expected 1 warning for unknown key, got %d", len(guide.Warnings))
	}
	if guide.Warnings[0].Category != "style" {
		t.Errorf("warning category = %q, want style", guide.Warnings[0].Category)
	}
}

func TestParseStyleGuide_CustomRules_Applied(t *testing.T) {
	rules := map[string]interface{}{
		"require_alternatives": true,
		"min_tags":             3,
	}
	guide := ParseStyleGuide(rules)
	if !guide.RequireAlternatives {
		t.Error("RequireAlternatives should be true")
	}
	if guide.MinTags != 3 {
		t.Errorf("MinTags = %d, want 3", guide.MinTags)
	}
	// Defaults preserved for unset keys
	if !guide.RequireWhy {
		t.Error("RequireWhy default should remain true")
	}
}
