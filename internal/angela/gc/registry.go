// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package gc holds the Pruner registry used by `lore doctor --prune`.
// A Pruner owns the lifecycle of one family of Lore-generated
// artifacts (polish backups, polish.log, *.corrupt-* quarantine
// files). It exposes a stable Name, a file Pattern for invariant
// coverage, and a Prune method that applies the family's retention
// policy to a given working directory.
//
// Story 8-23 / invariant I32: every growing artifact produced by
// Lore is either (a) an overwritten single-version file, or (b) has
// a Pruner registered in this package. A test walks the known
// patterns and verifies each maps to a registered Pruner — adding a
// new growing-artifact family without a Pruner fails the build.
package gc

import (
	"context"
	"sort"
	"sync"

	"github.com/greycoderk/lore/internal/config"
)

// PruneReport describes the outcome of one Pruner's run.
//
// Error semantics: a non-nil Err means the Pruner encountered an I/O
// error and the report may be partial. RunAll always returns every
// Pruner's report regardless of errors, so callers can surface
// partial progress + the error.
type PruneReport struct {
	Feature string // stable human identifier: "polish-backups", "polish-log", "corrupt-quarantine"
	Removed int    // number of items pruned (files, lines — unit is Pruner-specific)
	Kept    int    // number of items retained
	Bytes   int64  // bytes freed (or would have been freed in dry-run)
	DryRun  bool   // true when the Pruner was asked to simulate only
	Err     error
}

// Pruner is implemented by each family of artifact-managing code.
// Implementations live next to the writers they prune (e.g.
// polish_backup_pruner.go sits next to polish_backup.go). They
// register themselves with the package-level DefaultRegistry via
// init().
type Pruner interface {
	// Name returns the stable identifier for this Pruner. Used as
	// the PruneReport.Feature field and in invariant test output.
	Name() string

	// Pattern returns the file glob (relative to the state dir) that
	// this Pruner manages. Purely descriptive — used by the I32
	// invariant test to ensure every known growing artifact has
	// coverage.
	Pattern() string

	// Prune applies the retention policy. workDir is the repo root;
	// cfg carries the user's configured retention values. When
	// dryRun is true the Pruner must compute what it WOULD remove
	// without touching the filesystem. A non-nil PruneReport.Err
	// indicates partial failure — the report still carries the
	// counts that succeeded before the error surfaced.
	Prune(ctx context.Context, workDir string, cfg *config.Config, dryRun bool) PruneReport
}

var (
	mu       sync.RWMutex
	registry []Pruner
)

// Register adds a Pruner to the default registry. Safe for init()
// calls from sibling files in this package. Registering the same
// Pruner.Name() twice panics — this is a programmer error that
// should surface loudly at test time.
func Register(p Pruner) {
	mu.Lock()
	defer mu.Unlock()
	for _, existing := range registry {
		if existing.Name() == p.Name() {
			panic("gc: duplicate Pruner registration: " + p.Name())
		}
	}
	registry = append(registry, p)
}

// Registered returns a snapshot of the current Pruner set. Callers
// mutate the slice safely — it is a copy.
func Registered() []Pruner {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Pruner, len(registry))
	copy(out, registry)
	return out
}

// RegisteredPatterns returns the sorted list of Pattern values for
// every registered Pruner. Used by the I32 invariant test.
func RegisteredPatterns() []string {
	prs := Registered()
	out := make([]string, 0, len(prs))
	for _, p := range prs {
		out = append(out, p.Pattern())
	}
	sort.Strings(out)
	return out
}

// RunAll runs every registered Pruner in registration order and
// returns one PruneReport per Pruner. Errors from individual
// Pruners do not stop the sequence — the aggregate report carries
// each outcome so a caller (e.g. `doctor --prune`) can surface
// partial success.
func RunAll(ctx context.Context, workDir string, cfg *config.Config, dryRun bool) []PruneReport {
	prs := Registered()
	out := make([]PruneReport, 0, len(prs))
	for _, p := range prs {
		out = append(out, p.Prune(ctx, workDir, cfg, dryRun))
	}
	return out
}

// resetForTest clears the default registry. Test-only; the function
// is unexported and lives in this file so package-internal tests can
// call it without exposing a reset path to production callers.
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	registry = nil
}
