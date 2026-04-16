// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"reflect"
	"testing"
)

type stubSynth struct {
	name    string
	applies bool
}

func (s stubSynth) Name() string       { return s.name }
func (s stubSynth) Applies(*Doc) bool  { return s.applies }
func (s stubSynth) Detect(*Doc) ([]Candidate, error) {
	return nil, nil
}
func (s stubSynth) Synthesize(Candidate, Config) (Block, []Evidence, []Warning, error) {
	return Block{}, nil, nil, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "api-postman"})

	if got := r.Get("api-postman"); got == nil {
		t.Fatal("Get returned nil for registered synthesizer")
	}
	if got := r.Get("nonexistent"); got != nil {
		t.Fatal("Get should return nil for missing name")
	}
}

func TestRegistry_RegisterEmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Register with empty name must panic")
		}
	}()
	NewRegistry().Register(stubSynth{name: ""})
}

func TestRegistry_RegisterDuplicatePanics(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "dup"})
	defer func() {
		if rec := recover(); rec == nil {
			t.Fatal("duplicate Register must panic")
		}
	}()
	r.Register(stubSynth{name: "dup"})
}

func TestRegistry_NamesSorted(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "sql-query"})
	r.Register(stubSynth{name: "api-postman"})
	r.Register(stubSynth{name: "env-vars"})

	got := r.Names()
	want := []string{"api-postman", "env-vars", "sql-query"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Names unsorted: got %v, want %v", got, want)
	}
}

func TestRegistry_EnabledRespectsOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "a"})
	r.Register(stubSynth{name: "b"})
	r.Register(stubSynth{name: "c"})

	got := r.Enabled(EnabledConfig{Enabled: []string{"c", "a"}})
	if len(got) != 2 {
		t.Fatalf("want 2 enabled, got %d", len(got))
	}
	if got[0].Name() != "c" || got[1].Name() != "a" {
		t.Fatalf("order not preserved: got [%s %s], want [c a]", got[0].Name(), got[1].Name())
	}
}

func TestRegistry_EnabledSkipsUnknown(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "real"})

	got := r.Enabled(EnabledConfig{Enabled: []string{"ghost", "real", "spectre"}})
	if len(got) != 1 || got[0].Name() != "real" {
		t.Fatalf("unknown names not skipped cleanly: %v", got)
	}
}

func TestRegistry_EnabledDisabledShortCircuits(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "real"})

	if got := r.Enabled(EnabledConfig{Enabled: []string{"real"}, Disabled: true}); len(got) != 0 {
		t.Fatalf("Disabled=true must return zero synthesizers, got %d", len(got))
	}
}

func TestRegistry_EnabledDeduplicates(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "s"})
	got := r.Enabled(EnabledConfig{Enabled: []string{"s", "s", "s"}})
	if len(got) != 1 {
		t.Fatalf("duplicate names not deduplicated: %v", got)
	}
}

func TestRegistry_ForDocFiltersByApplies(t *testing.T) {
	r := NewRegistry()
	r.Register(stubSynth{name: "yes", applies: true})
	r.Register(stubSynth{name: "no", applies: false})

	got := r.ForDoc(&Doc{}, EnabledConfig{Enabled: []string{"yes", "no"}})
	if len(got) != 1 || got[0].Name() != "yes" {
		t.Fatalf("ForDoc must drop non-applicable synthesizers, got %v", got)
	}
}
