// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"strings"
	"testing"
)

func TestCanonicalJSON_StableAcrossRuns(t *testing.T) {
	in := map[string]int{"alpha": 1, "beta": 2, "gamma": 3}
	first, err := CanonicalJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		again, err := CanonicalJSON(in)
		if err != nil {
			t.Fatal(err)
		}
		if again != first {
			t.Fatalf("CanonicalJSON drift: run %d differs", i)
		}
	}
}

func TestCanonicalJSON_NoTrailingWhitespace(t *testing.T) {
	out, err := CanonicalJSON(map[string]string{"a": "1", "b": "2"})
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range strings.Split(out, "\n") {
		if line != strings.TrimRight(line, " \t") {
			t.Fatalf("line %d has trailing whitespace: %q", i, line)
		}
	}
}

func TestCanonicalJSON_TwoSpaceIndent(t *testing.T) {
	out, err := CanonicalJSON(map[string]any{"k": map[string]any{"nested": 1}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "\n  ") {
		t.Fatalf("expected 2-space indent, got: %s", out)
	}
	if strings.Contains(out, "\t") {
		t.Fatalf("tabs not allowed in canonical output: %s", out)
	}
}

func TestOrderedMap_PreservesInsertionOrder(t *testing.T) {
	m := NewOrderedMap(0)
	m.Set("z", "first")
	m.Set("a", "second")
	m.Set("m", "third")

	out, err := CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	// "z" must come before "a" must come before "m".
	zIdx := strings.Index(out, `"z"`)
	aIdx := strings.Index(out, `"a"`)
	mIdx := strings.Index(out, `"m"`)
	if zIdx >= aIdx || aIdx >= mIdx {
		t.Fatalf("OrderedMap did not preserve insertion order: %s", out)
	}
}

func TestOrderedMap_SetExistingKeyKeepsPosition(t *testing.T) {
	m := NewOrderedMap(0)
	m.Set("first", 1)
	m.Set("second", 2)
	m.Set("first", 99) // update — should NOT move to end

	keys := m.Keys()
	if len(keys) != 2 || keys[0] != "first" || keys[1] != "second" {
		t.Fatalf("update should preserve position, got %v", keys)
	}
	v, _ := m.Get("first")
	if v != 99 {
		t.Fatalf("update should change value, got %v", v)
	}
}

func TestOrderedMap_EmptyMarshalsAsEmptyObject(t *testing.T) {
	m := NewOrderedMap(0)
	out, err := CanonicalJSON(m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "{}" {
		t.Fatalf("empty OrderedMap should marshal to {}, got %q", out)
	}
}

func TestSortedStrings_ReturnsSortedCopy(t *testing.T) {
	in := []string{"c", "a", "b"}
	out := SortedStrings(in)
	if out[0] != "a" || out[1] != "b" || out[2] != "c" {
		t.Fatalf("not sorted: %v", out)
	}
	// Original must be unchanged.
	if in[0] != "c" {
		t.Fatalf("input was mutated: %v", in)
	}
}
