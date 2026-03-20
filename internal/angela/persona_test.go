// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
)

// --- Registry tests (AC-1, AC-7) ---

func TestGetRegistry_Has6Profiles(t *testing.T) {
	reg := GetRegistry()
	if len(reg) != 6 {
		t.Fatalf("expected 6 personas in Registry, got %d", len(reg))
	}
}

func TestGetRegistry_ReturnsCopy(t *testing.T) {
	r1 := GetRegistry()
	r1[0].Name = "tampered"
	r2 := GetRegistry()
	if r2[0].Name == "tampered" {
		t.Fatal("GetRegistry must return an independent copy")
	}
}

func TestGetRegistry_ExpectedNames(t *testing.T) {
	expected := map[string]bool{
		"storyteller":      false,
		"tech-writer":      false,
		"qa-reviewer":      false,
		"architect":        false,
		"ux-designer":      false,
		"business-analyst": false,
	}
	for _, p := range GetRegistry() {
		if _, ok := expected[p.Name]; !ok {
			t.Errorf("unexpected persona name: %s", p.Name)
		}
		expected[p.Name] = true
	}
	for name, found := range expected {
		if !found {
			t.Errorf("missing persona: %s", name)
		}
	}
}

func TestGetRegistry_AllFieldsPopulated(t *testing.T) {
	for _, p := range GetRegistry() {
		if p.Name == "" {
			t.Error("persona has empty Name")
		}
		if p.DisplayName == "" {
			t.Errorf("persona %s has empty DisplayName", p.Name)
		}
		if p.Icon == "" {
			t.Errorf("persona %s has empty Icon", p.Name)
		}
		if p.Expertise == "" {
			t.Errorf("persona %s has empty Expertise", p.Name)
		}
		if len(p.Principles) == 0 {
			t.Errorf("persona %s has no Principles", p.Name)
		}
		if len(p.DraftChecks) == 0 {
			t.Errorf("persona %s has no DraftChecks", p.Name)
		}
		if p.PromptDirective == "" {
			t.Errorf("persona %s has empty PromptDirective", p.Name)
		}
		if len(p.DocTypes) == 0 && len(p.ContentSignals) == 0 {
			t.Errorf("persona %s has no DocTypes and no ContentSignals", p.Name)
		}
	}
}

func TestGetRegistry_NoBMADImport(t *testing.T) {
	if GetRegistry() == nil {
		t.Fatal("GetRegistry must not return nil")
	}
}

// --- containsWord tests (word boundary matching) ---

func TestContainsWord_ExactMatch(t *testing.T) {
	if !containsWord("the ui is broken", "ui") {
		t.Error("should match standalone 'ui'")
	}
}

func TestContainsWord_NoSubstringMatch(t *testing.T) {
	if containsWord("building a guide", "ui") {
		t.Error("should NOT match 'ui' inside 'building' or 'guide'")
	}
}

func TestContainsWord_PunctuationStripped(t *testing.T) {
	if !containsWord("check the (ui) now", "ui") {
		t.Error("should match 'ui' wrapped in parens")
	}
	if !containsWord("the fix.", "fix") {
		t.Error("should match 'fix' followed by period")
	}
}

func TestContainsWord_CaseInsensitive(t *testing.T) {
	if !containsWord("the UI is great", "ui") {
		t.Error("should match case-insensitively")
	}
}

func TestContainsWord_NoFalsePositive_Fix(t *testing.T) {
	if containsWord("prefix and suffix", "fix") {
		t.Error("should NOT match 'fix' inside 'prefix' or 'suffix'")
	}
}

func TestContainsWord_NoFalsePositive_Test(t *testing.T) {
	if containsWord("the latest contest results", "test") {
		t.Error("should NOT match 'test' inside 'latest' or 'contest'")
	}
}

func TestContainsWord_Hyphenated(t *testing.T) {
	if !containsWord("the trade-off was clear", "trade-off") {
		t.Error("should match hyphenated word 'trade-off'")
	}
}

func TestContainsWord_FrenchElision_Apostrophe(t *testing.T) {
	if !containsWord("l'utilisateur peut configurer", "utilisateur") {
		t.Error("should match 'utilisateur' after French elision l'")
	}
}

func TestContainsWord_FrenchElision_DArchitecture(t *testing.T) {
	if !containsWord("le choix d'architecture est important", "architecture") {
		t.Error("should match 'architecture' after French elision d'")
	}
}

func TestContainsWord_FrenchElision_TypographicApostrophe(t *testing.T) {
	if !containsWord("l\u2019interface est claire", "interface") {
		t.Error("should match 'interface' after typographic apostrophe \\u2019")
	}
}

func TestContainsWord_FrenchElision_NoFalsePositive(t *testing.T) {
	// "l'animal" should NOT match "ami" — only exact parts after split
	if containsWord("l'animal est la", "ami") {
		t.Error("should NOT match 'ami' inside 'animal' after elision split")
	}
}

// --- Deep copy tests ---

func TestGetRegistry_DeepCopy_SlicesIndependent(t *testing.T) {
	r1 := GetRegistry()
	if len(r1[0].DocTypes) == 0 {
		t.Fatal("expected non-empty DocTypes")
	}
	original := r1[0].DocTypes[0]
	r1[0].DocTypes[0] = "tampered"
	r2 := GetRegistry()
	if r2[0].DocTypes[0] != original {
		t.Fatal("GetRegistry must deep-copy DocTypes — mutation leaked to registry")
	}
}

func TestGetRegistry_DeepCopy_ContentSignalsIndependent(t *testing.T) {
	r1 := GetRegistry()
	if len(r1[0].ContentSignals) == 0 {
		t.Fatal("expected non-empty ContentSignals")
	}
	original := r1[0].ContentSignals[0]
	r1[0].ContentSignals[0] = "tampered"
	r2 := GetRegistry()
	if r2[0].ContentSignals[0] != original {
		t.Fatal("GetRegistry must deep-copy ContentSignals — mutation leaked to registry")
	}
}

// --- ResolvePersonas tests (AC-2, AC-5, AC-6) ---

func TestResolvePersonas_Decision_StorytellerAndArchitect(t *testing.T) {
	scored := ResolvePersonas("decision", "We chose this architecture because of the trade-off between speed and reliability.")
	if len(scored) == 0 {
		t.Fatal("expected at least 1 persona")
	}
	names := scoredNames(scored)
	if !sliceContains(names, "storyteller") {
		t.Error("expected storyteller for decision type")
	}
	if !sliceContains(names, "architect") {
		t.Error("expected architect for content signal 'architecture'")
	}
}

func TestResolvePersonas_Feature_WithUI_TechWriterAndUX(t *testing.T) {
	scored := ResolvePersonas("feature", "This feature adds a new interface for the utilisateur dashboard.")
	names := scoredNames(scored)
	if !sliceContains(names, "tech-writer") {
		t.Error("expected tech-writer for feature type")
	}
	if !sliceContains(names, "ux-designer") {
		t.Errorf("expected ux-designer for content signal 'interface'/'utilisateur', got %v", names)
	}
}

func TestResolvePersonas_Bugfix_QAReviewer(t *testing.T) {
	scored := ResolvePersonas("bugfix", "Fixed the validation bogue in the login form.")
	names := scoredNames(scored)
	if !sliceContains(names, "qa-reviewer") {
		t.Error("expected qa-reviewer for bugfix type")
	}
}

func TestResolvePersonas_Unknown_NoSignals_FallbackTechWriter(t *testing.T) {
	scored := ResolvePersonas("unknown", "Nothing special here.")
	if len(scored) != 1 {
		t.Fatalf("expected 1 fallback persona, got %d", len(scored))
	}
	if scored[0].Profile.Name != "tech-writer" {
		t.Errorf("expected fallback to tech-writer, got %s", scored[0].Profile.Name)
	}
}

func TestResolvePersonas_Max3(t *testing.T) {
	body := "décision architecture interface utilisateur bogue correctif exigence stakeholder business client test validation api endpoint schema"
	scored := ResolvePersonas("feature", body)
	if len(scored) > 3 {
		t.Errorf("expected max 3 personas, got %d", len(scored))
	}
}

func TestResolvePersonas_OrderedByScore(t *testing.T) {
	scored := ResolvePersonas("decision", "We made a décision about the architecture design and the compromis.")
	if len(scored) < 2 {
		t.Fatal("expected at least 2 personas")
	}
	if scored[0].Score < scored[1].Score {
		t.Errorf("personas should be ordered by score desc: %d < %d", scored[0].Score, scored[1].Score)
	}
}

func TestResolvePersonas_CaseInsensitiveDocType(t *testing.T) {
	scored := ResolvePersonas("Decision", "Something here.")
	names := scoredNames(scored)
	if !sliceContains(names, "storyteller") {
		t.Error("expected storyteller for 'Decision' (case-insensitive type match)")
	}
}

func TestResolvePersonas_EmptyInputs_FallbackTechWriter(t *testing.T) {
	scored := ResolvePersonas("", "")
	if len(scored) != 1 {
		t.Fatalf("expected 1 fallback, got %d", len(scored))
	}
	if scored[0].Profile.Name != "tech-writer" {
		t.Errorf("expected tech-writer fallback, got %s", scored[0].Profile.Name)
	}
}

// --- French signal tests ---

func TestResolvePersonas_FrenchSignals_Storyteller(t *testing.T) {
	scored := ResolvePersonas("note", "Voici la décision et le contexte de ce compromis.")
	names := scoredNames(scored)
	if !sliceContains(names, "storyteller") {
		t.Errorf("expected storyteller for French signals, got %v", names)
	}
}

func TestResolvePersonas_FrenchSignals_Architect(t *testing.T) {
	scored := ResolvePersonas("refactor", "La conception du système et ses composants principaux.")
	names := scoredNames(scored)
	if !sliceContains(names, "architect") {
		t.Errorf("expected architect for French signals 'conception'/'système'/'composant', got %v", names)
	}
}

func TestResolvePersonas_FrenchSignals_QA(t *testing.T) {
	scored := ResolvePersonas("bugfix", "Ce correctif résout le bogue de régression.")
	names := scoredNames(scored)
	if !sliceContains(names, "qa-reviewer") {
		t.Errorf("expected qa-reviewer for French signals 'correctif'/'bogue'/'régression', got %v", names)
	}
}

func TestResolvePersonas_FrenchSignals_UX(t *testing.T) {
	scored := ResolvePersonas("feature", "L'ergonomie et l'accessibilité pour l'utilisateur.")
	names := scoredNames(scored)
	if !sliceContains(names, "ux-designer") {
		t.Errorf("expected ux-designer for French signals, got %v", names)
	}
}

func TestResolvePersonas_FrenchSignals_Business(t *testing.T) {
	scored := ResolvePersonas("feature", "Les exigences du client et les parties-prenantes du métier.")
	names := scoredNames(scored)
	if !sliceContains(names, "business-analyst") {
		t.Errorf("expected business-analyst for French signals, got %v", names)
	}
}

// --- Score tests ---

func TestAverageScore_TwoPersonas(t *testing.T) {
	scored := []ScoredPersona{
		{Score: 14},
		{Score: 12},
	}
	avg := AverageScore(scored)
	if avg != 13.0 {
		t.Errorf("expected 13.0, got %.1f", avg)
	}
}

func TestAverageScore_Empty(t *testing.T) {
	avg := AverageScore(nil)
	if avg != 0 {
		t.Errorf("expected 0 for empty, got %.1f", avg)
	}
}

func TestProfiles_ExtractsCorrectly(t *testing.T) {
	reg := GetRegistry()
	scored := []ScoredPersona{
		{Profile: reg[0], Score: 10},
		{Profile: reg[3], Score: 8},
	}
	profiles := Profiles(scored)
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}
	if profiles[0].Name != reg[0].Name || profiles[1].Name != reg[3].Name {
		t.Error("profiles don't match input order")
	}
}

// --- BuildPersonaPrompt tests ---

func TestBuildPersonaPrompt_TwoPersonas(t *testing.T) {
	reg := GetRegistry()
	personas := []PersonaProfile{reg[0], reg[3]} // storyteller + architect
	prompt := BuildPersonaPrompt(personas)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "Affoue") {
		t.Error("prompt should contain Affoue")
	}
	if !strings.Contains(prompt, "Doumbia") {
		t.Error("prompt should contain Doumbia")
	}
	if !strings.Contains(prompt, "STORYTELLING LENS") {
		t.Error("prompt should contain storyteller directive")
	}
	if !strings.Contains(prompt, "ARCHITECTURE LENS") {
		t.Error("prompt should contain architect directive")
	}
}

func TestBuildPersonaPrompt_Empty(t *testing.T) {
	prompt := BuildPersonaPrompt(nil)
	if prompt != "" {
		t.Error("expected empty prompt for nil personas")
	}
}

// --- RunPersonaDraftChecks tests ---

func TestRunPersonaDraftChecks_Storyteller_WhyNarrative(t *testing.T) {
	body := "## Why\n- reason 1\n- reason 2\n- reason 3\n- reason 4\nShort."
	storyteller := GetRegistry()[0]
	suggestions := RunPersonaDraftChecks(body, []PersonaProfile{storyteller})
	var found bool
	for _, s := range suggestions {
		if strings.Contains(s.Message, "Affoue") && strings.Contains(s.Message, "list") {
			found = true
		}
	}
	if !found {
		t.Error("expected Affoue draft check to flag listy Why section")
	}
}

func TestRunPersonaDraftChecks_MessageDecoratedWithIcon(t *testing.T) {
	body := "## Why\n- a\n- b\n- c\n- d\nX."
	storyteller := GetRegistry()[0]
	suggestions := RunPersonaDraftChecks(body, []PersonaProfile{storyteller})
	for _, s := range suggestions {
		if !strings.HasPrefix(s.Message, "[📖 Affoue]") {
			t.Errorf("message should be decorated with icon+name prefix, got: %s", s.Message)
		}
	}
}

// --- extractAllSections tests ---

func TestExtractAllSections(t *testing.T) {
	body := "Intro text\n## What\nSome content\n## Why\nReason here\n## Impact\nBig impact"
	sections := extractAllSections(body)
	if _, ok := sections["## What"]; !ok {
		t.Error("missing ## What section")
	}
	if _, ok := sections["## Why"]; !ok {
		t.Error("missing ## Why section")
	}
	if _, ok := sections["## Impact"]; !ok {
		t.Error("missing ## Impact section")
	}
}

func TestExtractAllSections_EmptyBody(t *testing.T) {
	sections := extractAllSections("")
	if len(sections) != 0 {
		t.Errorf("expected empty map for empty body, got %d entries", len(sections))
	}
}

func TestExtractAllSections_NoHeaders(t *testing.T) {
	sections := extractAllSections("Just plain text without any headers.")
	if len(sections) != 0 {
		t.Errorf("expected empty map for body without headers, got %d entries", len(sections))
	}
}

// --- helpers ---

func scoredNames(scored []ScoredPersona) []string {
	names := make([]string, len(scored))
	for i, sp := range scored {
		names[i] = sp.Profile.Name
	}
	return names
}

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
