// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package gc

import (
	"context"
	"errors"
	"testing"

	"github.com/greycoderk/lore/internal/config"
)

// fakePruner is a minimal Pruner used by registry-level tests.
type fakePruner struct {
	name     string
	pattern  string
	report   PruneReport
	callFlag *bool
}

func (f *fakePruner) Name() string    { return f.name }
func (f *fakePruner) Pattern() string { return f.pattern }
func (f *fakePruner) Prune(_ context.Context, _ string, _ *config.Config, dryRun bool) PruneReport {
	if f.callFlag != nil {
		*f.callFlag = true
	}
	r := f.report
	r.DryRun = dryRun
	return r
}

func TestRegistry_RegisterAndRetrieve(t *testing.T) {
	// Each test manipulates the package-level registry; restore the
	// default-registered Pruners after we're done so other tests in
	// the package still see the production set.
	prev := Registered()
	resetForTest()
	t.Cleanup(func() {
		resetForTest()
		for _, p := range prev {
			Register(p)
		}
	})

	p := &fakePruner{name: "test-feature", pattern: "test-*.bak"}
	Register(p)

	got := Registered()
	if len(got) != 1 {
		t.Fatalf("Registered len=%d, want 1", len(got))
	}
	if got[0].Name() != "test-feature" {
		t.Errorf("Name=%q, want 'test-feature'", got[0].Name())
	}
	if pats := RegisteredPatterns(); len(pats) != 1 || pats[0] != "test-*.bak" {
		t.Errorf("RegisteredPatterns=%v, want ['test-*.bak']", pats)
	}
}

func TestRegistry_DuplicateNamePanics(t *testing.T) {
	prev := Registered()
	resetForTest()
	t.Cleanup(func() {
		resetForTest()
		for _, p := range prev {
			Register(p)
		}
	})

	Register(&fakePruner{name: "dup", pattern: "a"})
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("duplicate Register should panic")
		}
	}()
	Register(&fakePruner{name: "dup", pattern: "b"})
}

func TestRegistry_RunAll_CallsEveryPruner(t *testing.T) {
	prev := Registered()
	resetForTest()
	t.Cleanup(func() {
		resetForTest()
		for _, p := range prev {
			Register(p)
		}
	})

	var called1, called2 bool
	Register(&fakePruner{name: "one", pattern: "one", callFlag: &called1, report: PruneReport{Removed: 3}})
	Register(&fakePruner{name: "two", pattern: "two", callFlag: &called2, report: PruneReport{Removed: 5}})

	cfg := &config.Config{}
	reports := RunAll(context.Background(), ".", cfg, false)

	if !called1 || !called2 {
		t.Errorf("expected both pruners called, got called1=%v called2=%v", called1, called2)
	}
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports, got %d", len(reports))
	}
	total := reports[0].Removed + reports[1].Removed
	if total != 8 {
		t.Errorf("total Removed=%d, want 8", total)
	}
}

func TestRegistry_RunAll_DryRunPropagates(t *testing.T) {
	prev := Registered()
	resetForTest()
	t.Cleanup(func() {
		resetForTest()
		for _, p := range prev {
			Register(p)
		}
	})

	Register(&fakePruner{name: "dry", pattern: "dry"})
	reports := RunAll(context.Background(), ".", &config.Config{}, true)
	if len(reports) != 1 || !reports[0].DryRun {
		t.Errorf("DryRun flag not propagated; reports=%+v", reports)
	}
}

func TestRegistry_RunAll_PartialFailureReturnsAllReports(t *testing.T) {
	prev := Registered()
	resetForTest()
	t.Cleanup(func() {
		resetForTest()
		for _, p := range prev {
			Register(p)
		}
	})

	Register(&fakePruner{name: "ok", pattern: "ok", report: PruneReport{Removed: 1}})
	Register(&fakePruner{name: "bad", pattern: "bad", report: PruneReport{Err: errors.New("boom")}})
	Register(&fakePruner{name: "also-ok", pattern: "also-ok", report: PruneReport{Removed: 2}})

	reports := RunAll(context.Background(), ".", &config.Config{}, false)
	if len(reports) != 3 {
		t.Fatalf("expected 3 reports, got %d", len(reports))
	}
	if reports[1].Err == nil {
		t.Errorf("middle report should carry Err")
	}
	if reports[0].Removed != 1 || reports[2].Removed != 2 {
		t.Errorf("other reports should be intact: %+v", reports)
	}
}

// --- Story 8-23 / I32: every known growing artifact has a Pruner -----

// TestI32_AllGrowingArtifactsHavePruners codifies invariant I32.
// The list of known patterns is maintained here; adding a new
// growing-artifact family in the future without registering a Pruner
// fails this test. When a family is intentionally decommissioned,
// remove it from the list.
func TestI32_AllGrowingArtifactsHavePruners(t *testing.T) {
	known := []string{
		"polish-backups/*.bak",
		"polish.log",
		"*.corrupt-*",
	}
	registered := RegisteredPatterns()
	registeredSet := make(map[string]bool, len(registered))
	for _, p := range registered {
		registeredSet[p] = true
	}
	for _, pat := range known {
		if !registeredSet[pat] {
			t.Errorf("I32 violation: known growing artifact %q has no registered Pruner", pat)
		}
	}
}
