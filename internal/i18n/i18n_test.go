// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

import (
	"reflect"
	"sync"
	"testing"
)

func TestT_BeforeInit_ReturnsEN(t *testing.T) {
	m := T()
	if m == nil {
		t.Fatal("T() returned nil before Init()")
	}
	if m.Engagement.Milestone3 == "" {
		t.Error("T() before Init() should return EN catalog with non-empty engagement strings")
	}
}

func TestInit_EN(t *testing.T) {
	Init("en")
	m := T()
	if m == nil {
		t.Fatal("T() returned nil after Init(en)")
	}
	if m.Engagement.Milestone3 != catalogEN.Engagement.Milestone3 {
		t.Errorf("Milestone3 = %q, want %q", m.Engagement.Milestone3, catalogEN.Engagement.Milestone3)
	}
}

func TestInit_UnknownLang_FallbackEN(t *testing.T) {
	Init("xx")
	m := T()
	if m == nil {
		t.Fatal("T() returned nil after Init(xx)")
	}
	// Should fallback to EN
	if m.Engagement.Milestone3 != catalogEN.Engagement.Milestone3 {
		t.Error("Init(xx) should fallback to EN catalog")
	}
}

func TestInit_EmptyString_FallbackEN(t *testing.T) {
	Init("")
	m := T()
	if m == nil {
		t.Fatal("T() returned nil after Init(empty)")
	}
	if m.Engagement.Milestone3 != catalogEN.Engagement.Milestone3 {
		t.Error("Init('') should fallback to EN catalog")
	}
}

func TestT_ConcurrentAccess_NoRace(t *testing.T) {
	// This test is meaningful with `go test -race`
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := T()
			if m == nil {
				t.Error("T() returned nil in goroutine")
				return
			}
			_ = m.Engagement.Milestone3
		}()
	}
	wg.Wait()
}

func TestInit_FR(t *testing.T) {
	Init("fr")
	m := T()
	if m == nil {
		t.Fatal("T() returned nil after Init(fr)")
	}
	// Verify FR tagline uses the "L'or" brand
	if m.Engagement.Milestone55 != "55 décisions. Vous forgez l'or de votre équipe." {
		t.Errorf("FR Milestone55 = %q, expected French with L'or brand", m.Engagement.Milestone55)
	}
	if m.Cmd.RootShort != "Votre code sait quoi. Lore sait pourquoi." {
		t.Errorf("FR RootShort = %q", m.Cmd.RootShort)
	}
	// Confirm [o/N] instead of [y/N]
	if m.Workflow.SuggestSkipPrompt != "Documenter ce commit ? [o/N] " {
		t.Errorf("FR SuggestSkipPrompt = %q, expected [o/N]", m.Workflow.SuggestSkipPrompt)
	}
	// Reset to EN for other tests
	Init("en")
}

// TestCatalogFR_AllFieldsPopulated uses reflection to verify every string field
// in the FR catalog is non-empty. This prevents silent regressions when new
// strings are added to the Messages struct but not translated.
func TestCatalogFR_AllFieldsPopulated(t *testing.T) {
	checkAllStringsPopulated(t, reflect.ValueOf(*catalogFR), "catalogFR")
}

// TestCatalogEN_AllFieldsPopulated is the EN counterpart — ensures no empty strings in EN either.
func TestCatalogEN_AllFieldsPopulated(t *testing.T) {
	checkAllStringsPopulated(t, reflect.ValueOf(*catalogEN), "catalogEN")
}

func checkAllStringsPopulated(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Type().Field(i)
			checkAllStringsPopulated(t, v.Field(i), path+"."+field.Name)
		}
	case reflect.String:
		if v.String() == "" {
			t.Errorf("%s is empty", path)
		}
	}
}

func TestSupportedLanguages(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) != 2 {
		t.Errorf("SupportedLanguages() = %v, want [en, fr]", langs)
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported("en") {
		t.Error("IsSupported(en) should be true")
	}
	if !IsSupported("fr") {
		t.Error("IsSupported(fr) should be true")
	}
	if IsSupported("xx") {
		t.Error("IsSupported(xx) should be false")
	}
}
