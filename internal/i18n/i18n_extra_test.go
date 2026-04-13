// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

import (
	"sync"
	"testing"
)

// TestSnapshot_RestoresPreviousLanguage verifies that Snapshot captures the
// current catalog and the returned function restores it.
func TestSnapshot_RestoresPreviousLanguage(t *testing.T) {
	Init("fr")
	restore := Snapshot()

	// Switch to EN while snapshot was taken under FR.
	Init("en")
	if T().Cmd.RootShort != catalogEN.Cmd.RootShort {
		t.Fatal("expected EN after Init(en)")
	}

	// Restore should bring back FR.
	restore()
	if T().Cmd.RootShort != catalogFR.Cmd.RootShort {
		t.Errorf("Snapshot restore: got %q, want FR catalog", T().Cmd.RootShort)
	}

	// Leave EN for subsequent tests.
	Init("en")
}

// TestSnapshot_RestoresEN verifies Snapshot works when the active lang is EN.
func TestSnapshot_RestoresEN(t *testing.T) {
	Init("en")
	restore := Snapshot()

	Init("fr")
	restore()

	if T().Cmd.RootShort != catalogEN.Cmd.RootShort {
		t.Errorf("Snapshot restore to EN: got %q", T().Cmd.RootShort)
	}
}

// TestSnapshot_CalledTwice verifies that calling the restore function more than
// once does not panic and leaves the catalog in the expected state.
func TestSnapshot_CalledTwice(t *testing.T) {
	Init("en")
	restore := Snapshot()
	Init("fr")

	restore() // first call — should restore EN
	restore() // second call — must not panic, catalog stays EN

	if T().Cmd.RootShort != catalogEN.Cmd.RootShort {
		t.Errorf("after double restore: got %q, want EN", T().Cmd.RootShort)
	}
}

// TestInit_ConcurrentWritesAndReads exercises concurrent Init() + T() calls
// under the race detector to verify atomic.Value is used correctly.
func TestInit_ConcurrentWritesAndReads(t *testing.T) {
	var wg sync.WaitGroup
	langs := []string{"en", "fr", "xx", ""}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			Init(langs[i%len(langs)])
		}(i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			m := T()
			if m == nil {
				t.Error("T() returned nil during concurrent Init")
			}
		}()
	}
	wg.Wait()
	// Restore deterministic state.
	Init("en")
}

// TestT_NeverReturnsNil exercises the nil-guard branch inside T(). In practice
// atomic.Value is pre-initialized by init(), so we verify the post-condition
// holds after every Init path.
func TestT_NeverReturnsNil(t *testing.T) {
	for _, lang := range []string{"en", "fr", "de", "", "zh"} {
		Init(lang)
		if m := T(); m == nil {
			t.Errorf("T() returned nil after Init(%q)", lang)
		}
	}
	Init("en")
}

// TestIsSupported_TableDriven gives table-driven coverage for all code paths.
func TestIsSupported_TableDriven(t *testing.T) {
	cases := []struct {
		lang string
		want bool
	}{
		{"en", true},
		{"fr", true},
		{"EN", false}, // case-sensitive
		{"FR", false},
		{"de", false},
		{"", false},
		{"español", false},
	}
	for _, tc := range cases {
		got := IsSupported(tc.lang)
		if got != tc.want {
			t.Errorf("IsSupported(%q) = %v, want %v", tc.lang, got, tc.want)
		}
	}
}

// TestSupportedLanguages_ContentsAreENAndFR ensures the slice holds exactly
// EN and FR, in that order, so callers that range over it see a stable order.
func TestSupportedLanguages_ContentsAreENAndFR(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) != 2 {
		t.Fatalf("len(SupportedLanguages()) = %d, want 2", len(langs))
	}
	if langs[0] != EN {
		t.Errorf("langs[0] = %q, want %q", langs[0], EN)
	}
	if langs[1] != FR {
		t.Errorf("langs[1] = %q, want %q", langs[1], FR)
	}
}
