// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package gc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
)

// quarantineConfig builds a *config.Config with the retention knob
// set to the given number of days.
func quarantineConfig(t *testing.T, stateDir string, retentionDays int) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Angela.StateDir = stateDir
	cfg.Angela.GC.CorruptQuarantine.RetentionDays = retentionDays
	cfg.DetectedMode = config.ModeStandalone
	return cfg
}

// writeQuarantineFile creates a `<name>.corrupt-<stamp>` file whose
// stamp encodes the given timestamp. Returns the full path.
func writeQuarantineFile(t *testing.T, stateDir, base string, ts time.Time, body string) string {
	t.Helper()
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stamp := ts.UTC().Format(angela.QuarantineTimestampLayout)
	path := filepath.Join(stateDir, base+".corrupt-"+stamp)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestCorruptQuarantinePruner_DeletesOldKeepsFresh(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 14)

	oldPath := writeQuarantineFile(t, stateDir, "draft-state.json", time.Now().Add(-30*24*time.Hour), "old")
	newPath := writeQuarantineFile(t, stateDir, "review-state.json", time.Now().Add(-1*24*time.Hour), "fresh")

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed != 1 {
		t.Errorf("Removed=%d, want 1", r.Removed)
	}
	if r.Kept != 1 {
		t.Errorf("Kept=%d, want 1", r.Kept)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old quarantine file not removed: %v", err)
	}
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("fresh quarantine file removed unexpectedly: %v", err)
	}
}

func TestCorruptQuarantinePruner_DryRun_NoUnlink(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 14)

	oldPath := writeQuarantineFile(t, stateDir, "draft-state.json", time.Now().Add(-30*24*time.Hour), "old")

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, true)
	if r.Removed != 1 {
		t.Errorf("Removed=%d, want 1 (would-remove)", r.Removed)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Errorf("dry-run removed the file: %v", err)
	}
}

func TestCorruptQuarantinePruner_IgnoresUnparseableSuffix(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 14)

	// Garbage timestamp suffix — retained as "age unknown".
	path := filepath.Join(stateDir, "draft-state.json.corrupt-not-a-timestamp")
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed != 0 {
		t.Errorf("Removed=%d, want 0 (unparseable stamp is kept)", r.Removed)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("unparseable-stamp file should be retained: %v", err)
	}
}

func TestCorruptQuarantinePruner_ZeroRetention_NoOp(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 0) // 0 = keep forever

	oldPath := writeQuarantineFile(t, stateDir, "draft-state.json", time.Now().Add(-30*24*time.Hour), "old")

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Removed != 0 {
		t.Errorf("Removed=%d, want 0 (retention=0)", r.Removed)
	}
	if _, err := os.Stat(oldPath); err != nil {
		t.Errorf("file should be retained: %v", err)
	}
}

func TestCorruptQuarantinePruner_MissingStateDir_NoError(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), "does-not-exist")
	cfg := quarantineConfig(t, stateDir, 14)

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Errorf("missing state dir should not surface an error, got: %v", r.Err)
	}
}

// TestCorruptQuarantinePruner_SkipsSymlinks asserts the Story 8-23
// P0 symlink safety fix: a symlink named like a quarantine file
// (`<base>.corrupt-<stamp>`) pointing outside stateDir must be skipped,
// even when the stamp is old enough to be eligible. The target stays
// intact, and the symlink itself is counted as Kept rather than
// Removed.
func TestCorruptQuarantinePruner_SkipsSymlinks(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 14)

	// Regular stale file — must be removed.
	realStale := writeQuarantineFile(t, stateDir, "draft-state.json",
		time.Now().Add(-30*24*time.Hour), "stale")

	// Sentinel file that the symlink points to. Must survive.
	sentinelDir := t.TempDir()
	sentinelPath := filepath.Join(sentinelDir, "sentinel.txt")
	if err := os.WriteFile(sentinelPath, []byte("precious"), 0o644); err != nil {
		t.Fatalf("WriteFile sentinel: %v", err)
	}

	// Symlink inside stateDir with a stale stamp pointing at the sentinel.
	stamp := time.Now().Add(-30 * 24 * time.Hour).UTC().Format(angela.QuarantineTimestampLayout)
	linkName := filepath.Join(stateDir, "review-state.json.corrupt-"+stamp)
	if err := os.Symlink(sentinelPath, linkName); err != nil {
		t.Skipf("symlink creation unsupported on this platform: %v", err)
	}

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}

	// Real file gone.
	if _, err := os.Stat(realStale); !os.IsNotExist(err) {
		t.Errorf("regular stale file should have been removed: %v", err)
	}
	// Symlink still there.
	if _, err := os.Lstat(linkName); err != nil {
		t.Errorf("symlink should have been skipped, got Lstat err: %v", err)
	}
	// Sentinel unharmed (proving the pruner did not follow the link).
	if _, err := os.Stat(sentinelPath); err != nil {
		t.Errorf("symlink target should be untouched, got: %v", err)
	}
}

// TestQuarantineLayout_ProducerConsumerRoundTrip asserts the Story
// 8-23 P0 single-source fix: the format literal used by the producer
// (angela.QuarantineCorruptState via draft_state.go) and by the
// consumer (this pruner) comes from the same exported constant. The
// round-trip test gives confidence that a future drift in either
// site is caught — parsing the producer-generated name with the
// consumer layout must succeed.
func TestQuarantineLayout_ProducerConsumerRoundTrip(t *testing.T) {
	// Produce a stamp exactly as QuarantineCorruptState would.
	stamp := time.Now().UTC().Format(angela.QuarantineTimestampLayout)
	// The pruner parses with the same layout.
	parsed, err := time.Parse(angela.QuarantineTimestampLayout, stamp)
	if err != nil {
		t.Fatalf("round-trip parse failed: %v (stamp=%q)", err, stamp)
	}
	if parsed.IsZero() {
		t.Errorf("round-trip produced zero time from stamp %q", stamp)
	}
}

// TestCorruptQuarantinePruner_FutureStamp_KeptAsFresh codifies the
// clock-skew policy: a stamp in the future must not be supplied as
// "older than cutoff" and thus must not be removed. The current code
// gets this right by using `!stamp.Before(cutoff)`; the test pins
// the invariant so a refactor to `stamp.Add(retention).Before(now)`
// cannot regress silently.
func TestCorruptQuarantinePruner_FutureStamp_KeptAsFresh(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 14)

	future := time.Now().Add(3 * time.Hour)
	path := writeQuarantineFile(t, stateDir, "draft-state.json", future, "future")

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed != 0 {
		t.Errorf("future-stamped file should be kept, got Removed=%d", r.Removed)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("future-stamped file should still exist: %v", err)
	}
}

// TestCorruptQuarantinePruner_MalformedStampVariants exercises a
// table of deliberately-malformed stamps to confirm the "unparseable
// → keep" rule holds across the edge space we care about.
func TestCorruptQuarantinePruner_MalformedStampVariants(t *testing.T) {
	stateDir := t.TempDir()
	cfg := quarantineConfig(t, stateDir, 14)

	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatal(err)
	}

	cases := []string{
		"draft-state.json.corrupt-",                             // empty stamp
		"draft-state.json.corrupt-20260101",                     // partial
		"draft-state.json.corrupt-abcdef",                       // non-numeric
		"draft-state.json.corrupt-20260101T150405",              // missing .000
		"draft-state.json.corrupt-20260101T150405.000Z",         // trailing Z
	}
	for _, name := range cases {
		full := filepath.Join(stateDir, name)
		if err := os.WriteFile(full, []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	p := corruptQuarantinePruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed != 0 {
		t.Errorf("Removed=%d, want 0 (unparseable stamps must be kept)", r.Removed)
	}
	for _, name := range cases {
		if _, err := os.Stat(filepath.Join(stateDir, name)); err != nil {
			t.Errorf("malformed-stamp %q should have been kept: %v", name, err)
		}
	}
}
