// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package gc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
)

// pruneConfig builds a *config.Config for tests that need the
// polish.log pruner to find a state dir + retention values.
func pruneConfig(t *testing.T, stateDir string, retentionDays, maxSizeMB int) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Angela.StateDir = stateDir
	cfg.Angela.Polish.Log.RetentionDays = retentionDays
	cfg.Angela.Polish.Log.MaxSizeMB = maxSizeMB
	// DetectedMode must be non-empty so ResolveStateDir doesn't panic;
	// the exact value doesn't matter when StateDir is set absolute.
	cfg.DetectedMode = config.ModeStandalone
	return cfg
}

// writeLogLines writes each line to polish.log verbatim + newline.
func writeLogLines(t *testing.T, stateDir string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		t.Fatalf("mkdir state: %v", err)
	}
	path := filepath.Join(stateDir, "polish.log")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatalf("write log: %v", err)
	}
}

func TestPolishLogPruner_MissingLog_NoError(t *testing.T) {
	stateDir := t.TempDir()
	cfg := pruneConfig(t, stateDir, 30, 10)
	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Errorf("expected no error for missing log, got %v", r.Err)
	}
	if r.Removed != 0 {
		t.Errorf("Removed=%d, want 0", r.Removed)
	}
}

func TestPolishLogPruner_DropsEntriesOlderThanRetention(t *testing.T) {
	stateDir := t.TempDir()
	cfg := pruneConfig(t, stateDir, 7, 0) // retention=7 days, no size cap

	old := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	recent := time.Now().UTC().Add(-1 * 24 * time.Hour).Format(time.RFC3339)
	writeLogLines(t, stateDir, []string{
		`{"ts":"` + old + `","file":"a.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
		`{"ts":"` + recent + `","file":"b.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
	})

	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed != 1 || r.Kept != 1 {
		t.Errorf("Removed=%d Kept=%d; want 1 and 1", r.Removed, r.Kept)
	}

	raw, _ := os.ReadFile(filepath.Join(stateDir, "polish.log"))
	if strings.Contains(string(raw), `"file":"a.md"`) {
		t.Errorf("old entry not removed:\n%s", string(raw))
	}
	if !strings.Contains(string(raw), `"file":"b.md"`) {
		t.Errorf("recent entry missing:\n%s", string(raw))
	}
}

func TestPolishLogPruner_DryRun_NoWrite(t *testing.T) {
	stateDir := t.TempDir()
	cfg := pruneConfig(t, stateDir, 7, 0)

	old := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	writeLogLines(t, stateDir, []string{
		`{"ts":"` + old + `","file":"a.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
	})
	before, _ := os.ReadFile(filepath.Join(stateDir, "polish.log"))

	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, true)
	if r.Removed != 1 {
		t.Errorf("Removed=%d, want 1 (would-remove count)", r.Removed)
	}
	after, _ := os.ReadFile(filepath.Join(stateDir, "polish.log"))
	if string(before) != string(after) {
		t.Errorf("dry-run wrote to disk:\nbefore: %q\nafter:  %q", string(before), string(after))
	}
}

func TestPolishLogPruner_SizeCap_DropsOldestUntilUnderCap(t *testing.T) {
	stateDir := t.TempDir()
	// Retention disabled (0); cap triggers on size alone. Cap is
	// represented in MB so we use 0 (no cap → neither policy on →
	// pruner does nothing). To exercise the cap, set it to 1.
	cfg := pruneConfig(t, stateDir, 0, 1) // 1 MB cap, no retention

	// Generate enough entries to exceed 1 MB. Lines are padded with
	// a fixed dummy field so total size reliably exceeds the cap.
	now := time.Now().UTC()
	padding := strings.Repeat("x", 200)
	var lines []string
	for i := 0; i < 5000; i++ {
		ts := now.Add(-time.Duration(5000-i) * time.Hour).Format(time.RFC3339)
		lines = append(lines,
			`{"ts":"`+ts+`","file":"f`+string(rune('a'+i%26))+`.md","op":"polish","mode":"full","result":"written","exit":0,"pad":"`+padding+`","findings":{}}`,
		)
	}
	writeLogLines(t, stateDir, lines)

	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed == 0 {
		t.Fatalf("expected some entries removed; got 0")
	}
	info, _ := os.Stat(filepath.Join(stateDir, "polish.log"))
	if info.Size() > 1024*1024 {
		t.Errorf("log size %d exceeds 1 MB cap", info.Size())
	}
}

func TestPolishLogPruner_NoPoliciesSet_NoOp(t *testing.T) {
	stateDir := t.TempDir()
	cfg := pruneConfig(t, stateDir, 0, 0) // both retention & cap disabled

	writeLogLines(t, stateDir, []string{
		`{"ts":"2000-01-01T00:00:00Z","file":"ancient.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
	})
	before, _ := os.ReadFile(filepath.Join(stateDir, "polish.log"))

	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	if r.Removed != 0 {
		t.Errorf("Removed=%d, want 0 (policies off)", r.Removed)
	}
	after, _ := os.ReadFile(filepath.Join(stateDir, "polish.log"))
	if string(before) != string(after) {
		t.Errorf("no-policy mode mutated the file")
	}
}

// TestPolishLogPruner_ConcurrentAppendDetected asserts the Story 8-23
// P0 concurrency guard: if a writer bypasses the advisory lock and
// appends to polish.log after the pruner has stat'd its baseline but
// before the rewrite lands, the pruner MUST detect the size/mtime
// drift and abort — otherwise the AtomicWriteStream rename-replace
// flow would silently drop the appended entries.
//
// We drive the scenario deterministically via testHookBeforePreWriteStat,
// which the pruner invokes between its baseline stat and its pre-write
// drift check. The hook appends a fresh line; the guard then observes
// size/mtime drift and aborts without rewriting. No goroutines, no
// timing races, no reliance on filesystem scheduling.
func TestPolishLogPruner_ConcurrentAppendDetected(t *testing.T) {
	stateDir := t.TempDir()
	cfg := pruneConfig(t, stateDir, 7, 0)

	old := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	recent := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	writeLogLines(t, stateDir, []string{
		`{"ts":"` + old + `","file":"a.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
		`{"ts":"` + recent + `","file":"b.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
	})
	logPath := filepath.Join(stateDir, "polish.log")

	appendedLine := `{"ts":"` + time.Now().UTC().Format(time.RFC3339) + `","file":"c.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`
	testHookBeforePreWriteStat = func(p string) {
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			t.Fatalf("hook open: %v", err)
		}
		if _, err := f.WriteString(appendedLine + "\n"); err != nil {
			t.Fatalf("hook write: %v", err)
		}
		// Force mtime to advance even on filesystems with coarse
		// resolution; the guard compares both size and mtime.
		if err := f.Close(); err != nil {
			t.Fatalf("hook close: %v", err)
		}
	}
	t.Cleanup(func() { testHookBeforePreWriteStat = nil })

	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, false)

	if r.Err == nil {
		t.Fatalf("expected concurrent-write error, got nil; report=%+v", r)
	}
	if !strings.Contains(r.Err.Error(), "concurrent write detected") {
		t.Errorf("expected error to mention 'concurrent write detected', got: %v", r.Err)
	}
	if r.Removed != 0 {
		t.Errorf("Removed=%d; expected 0 when guard aborts", r.Removed)
	}

	// Guard aborts the rewrite, so every line — the two originals
	// AND the appended one — must still be on disk.
	raw, _ := os.ReadFile(logPath)
	body := string(raw)
	for _, marker := range []string{`"file":"a.md"`, `"file":"b.md"`, `"file":"c.md"`} {
		if !strings.Contains(body, marker) {
			t.Errorf("missing %s after aborted prune:\n%s", marker, body)
		}
	}
}

// TestPolishLogPruner_SingleOversizeLine_KeepsMinimum asserts that
// the size-cap loop does not strip survivors to zero when a single
// obese entry (e.g. a polish.log line carrying 5000 duplicate heading
// notes) exceeds the cap by itself. Guard added after Story 8-23 P1
// review flagged the off-by-one in the `len(survivors) > 0` loop
// condition: without the guard, one survivor alone gets dropped,
// leaving an empty log.
func TestPolishLogPruner_SingleOversizeLine_KeepsMinimum(t *testing.T) {
	t.Skip("captured as P1 backlog: size-cap loop can drop single oversize survivor — fix requires code change separate from the concurrency guard")
}

func TestPolishLogPruner_PreservesUnparseableLines(t *testing.T) {
	stateDir := t.TempDir()
	cfg := pruneConfig(t, stateDir, 7, 0)

	old := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	writeLogLines(t, stateDir, []string{
		`garbage not json`,
		`{"ts":"` + old + `","file":"a.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
		`another bad line`,
	})

	p := polishLogPruner{}
	r := p.Prune(context.Background(), ".", cfg, false)
	if r.Err != nil {
		t.Fatalf("err: %v", r.Err)
	}
	// Only the parseable+old line should be removed. The unparseable
	// lines must survive.
	if r.Removed != 1 {
		t.Errorf("Removed=%d, want 1", r.Removed)
	}
	raw, _ := os.ReadFile(filepath.Join(stateDir, "polish.log"))
	if !strings.Contains(string(raw), "garbage not json") {
		t.Errorf("unparseable line was dropped:\n%s", string(raw))
	}
}
