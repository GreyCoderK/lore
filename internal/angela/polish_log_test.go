// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestFormatResolution_AllChoices(t *testing.T) {
	cases := []struct {
		source string
		choice ArbitrateChoice
		want   string
	}{
		{"user", ChoiceFirst, "user:first"},
		{"user", ChoiceSecond, "user:second"},
		{"user", ChoiceBoth, "user:both"},
		{"user", ChoiceAbort, "user:abort"},
		{"rule", ChoiceFirst, "rule:first"},
		{"rule", ChoiceAbort, "rule:abort"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := FormatResolution(tc.source, tc.choice); got != tc.want {
				t.Errorf("got=%q, want=%q", got, tc.want)
			}
		})
	}
}

func TestPolishLogPath(t *testing.T) {
	got := PolishLogPath("/tmp/foo/.angela")
	want := filepath.Join("/tmp/foo/.angela", "polish.log")
	if got != want {
		t.Errorf("got=%q, want=%q", got, want)
	}
}

func TestAppendLogEntry_WritesOneJSONLinePerCall(t *testing.T) {
	stateDir := t.TempDir()

	e1 := LogEntry{
		File:   "docs/a.md",
		Mode:   LogModeFull,
		Result: LogResultWritten,
		Exit:   0,
	}
	if err := AppendLogEntry(stateDir, e1); err != nil {
		t.Fatalf("AppendLogEntry #1: %v", err)
	}
	e2 := LogEntry{
		File:   "docs/b.md",
		Mode:   LogModeIncremental,
		Result: LogResultAbortedArbitrate,
		Exit:   1,
	}
	if err := AppendLogEntry(stateDir, e2); err != nil {
		t.Fatalf("AppendLogEntry #2: %v", err)
	}

	raw, err := os.ReadFile(PolishLogPath(stateDir))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), string(raw))
	}

	var l1, l2 LogEntry
	if err := json.Unmarshal([]byte(lines[0]), &l1); err != nil {
		t.Fatalf("parse #1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &l2); err != nil {
		t.Fatalf("parse #2: %v", err)
	}

	if l1.File != "docs/a.md" || l1.Op != "polish" || l1.Mode != LogModeFull || l1.Exit != 0 {
		t.Errorf("line 1 mismatch: %+v", l1)
	}
	if l2.File != "docs/b.md" || l2.Mode != LogModeIncremental || l2.Exit != 1 {
		t.Errorf("line 2 mismatch: %+v", l2)
	}
}

func TestAppendLogEntry_DefaultsTimestampAndOp(t *testing.T) {
	stateDir := t.TempDir()
	before := time.Now().UTC().Add(-1 * time.Second)

	entry := LogEntry{
		File:   "docs/c.md",
		Mode:   LogModeDryRun,
		Result: LogResultDryRun,
		Exit:   0,
	}
	if err := AppendLogEntry(stateDir, entry); err != nil {
		t.Fatalf("AppendLogEntry: %v", err)
	}

	entries, err := ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	got := entries[0]
	if got.Op != "polish" {
		t.Errorf("default Op = %q, want 'polish'", got.Op)
	}
	after := time.Now().UTC().Add(1 * time.Second)
	if got.Timestamp.Before(before) || got.Timestamp.After(after) {
		t.Errorf("timestamp %v not in [%v, %v]", got.Timestamp, before, after)
	}
}

func TestAppendLogEntry_PreservesExplicitTimestamp(t *testing.T) {
	stateDir := t.TempDir()
	fixed := time.Date(2026, 4, 19, 14, 23, 5, 0, time.UTC)

	entry := LogEntry{
		Timestamp: fixed,
		File:      "docs/d.md",
		Mode:      LogModeFull,
		Result:    LogResultWritten,
	}
	if err := AppendLogEntry(stateDir, entry); err != nil {
		t.Fatalf("AppendLogEntry: %v", err)
	}
	entries, err := ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if !entries[0].Timestamp.Equal(fixed) {
		t.Errorf("timestamp = %v, want %v", entries[0].Timestamp, fixed)
	}
}

func TestAppendLogEntry_WithFindingsAndAIInfo(t *testing.T) {
	stateDir := t.TempDir()
	entry := LogEntry{
		File: "docs/decision.md",
		Mode: LogModeIncremental,
		AI: &LogAIInfo{
			Provider:         "anthropic",
			Model:            "claude-sonnet-4-6",
			PromptTokens:     1240,
			CompletionTokens: 890,
		},
		Findings: LogFindings{
			LeakedFM: &LogLeakedFM{Stripped: true, Bytes: 47},
			Duplicates: []LogDuplicate{
				{Heading: "## Why", Count: 2, Resolution: "user:first"},
				{Heading: "## Context", Count: 3, Resolution: "rule:both"},
			},
		},
		Result: LogResultWritten,
		Exit:   0,
	}
	if err := AppendLogEntry(stateDir, entry); err != nil {
		t.Fatalf("AppendLogEntry: %v", err)
	}
	entries, err := ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	got := entries[0]
	if got.AI == nil || got.AI.Provider != "anthropic" || got.AI.PromptTokens != 1240 {
		t.Errorf("AI info mismatch: %+v", got.AI)
	}
	if got.Findings.LeakedFM == nil || !got.Findings.LeakedFM.Stripped {
		t.Errorf("LeakedFM mismatch: %+v", got.Findings.LeakedFM)
	}
	if len(got.Findings.Duplicates) != 2 {
		t.Fatalf("Duplicates len=%d, want 2", len(got.Findings.Duplicates))
	}
	if got.Findings.Duplicates[0].Resolution != "user:first" {
		t.Errorf("duplicate[0].Resolution=%q", got.Findings.Duplicates[0].Resolution)
	}
	if got.Findings.Duplicates[1].Resolution != "rule:both" {
		t.Errorf("duplicate[1].Resolution=%q", got.Findings.Duplicates[1].Resolution)
	}
}

func TestReadLogEntries_NoFile_ReturnsEmpty(t *testing.T) {
	stateDir := t.TempDir()
	entries, err := ReadLogEntries(stateDir)
	if err != nil {
		t.Errorf("err=%v, want nil", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestReadLogEntries_SkipsMalformedLines(t *testing.T) {
	stateDir := t.TempDir()
	logPath := PolishLogPath(stateDir)
	raw := strings.Join([]string{
		`{"ts":"2026-04-19T14:23:05Z","file":"a.md","op":"polish","mode":"full","result":"written","exit":0,"findings":{}}`,
		`this is not json at all`,
		``, // blank
		`{"ts":"2026-04-19T14:24:05Z","file":"b.md","op":"polish","mode":"dry-run","result":"dryrun","exit":0,"findings":{}}`,
		`{malformed`,
	}, "\n") + "\n"
	if err := os.WriteFile(logPath, []byte(raw), 0o600); err != nil {
		t.Fatalf("seed log: %v", err)
	}

	entries, err := ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 valid entries, got %d", len(entries))
	}
	if entries[0].File != "a.md" || entries[1].File != "b.md" {
		t.Errorf("order or content mismatch: %+v", entries)
	}
}

func TestAppendLogEntry_ConcurrentWritesAllLandCleanly(t *testing.T) {
	// This exercises the FileLock contract: N goroutines each appending
	// one line must produce N well-formed JSON lines with no
	// interleaving or truncation.
	stateDir := t.TempDir()
	const N = 32
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			entry := LogEntry{
				File:   "docs/concurrent.md",
				Mode:   LogModeFull,
				Result: LogResultWritten,
				Exit:   i, // we use Exit as a tag to tell which goroutine wrote which
			}
			if err := AppendLogEntry(stateDir, entry); err != nil {
				t.Errorf("goroutine %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	entries, err := ReadLogEntries(stateDir)
	if err != nil {
		t.Fatalf("ReadLogEntries: %v", err)
	}
	if len(entries) != N {
		t.Errorf("expected %d entries, got %d (log may have been interleaved)", N, len(entries))
	}
	// Each goroutine used a unique Exit value in [0, N); all N values
	// must appear exactly once.
	seen := make(map[int]bool, N)
	for _, e := range entries {
		if seen[e.Exit] {
			t.Errorf("duplicate entry with Exit=%d", e.Exit)
		}
		seen[e.Exit] = true
	}
	for i := 0; i < N; i++ {
		if !seen[i] {
			t.Errorf("missing entry with Exit=%d", i)
		}
	}
}

func TestAppendLogEntry_TimestampIsUTC(t *testing.T) {
	stateDir := t.TempDir()
	// Provide a timestamp in a non-UTC zone — it should be converted.
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("time zone data unavailable: %v", err)
	}
	ny := time.Date(2026, 4, 19, 10, 0, 0, 0, loc)

	entry := LogEntry{
		Timestamp: ny,
		File:      "docs/tz.md",
		Mode:      LogModeFull,
		Result:    LogResultWritten,
	}
	if err := AppendLogEntry(stateDir, entry); err != nil {
		t.Fatalf("AppendLogEntry: %v", err)
	}
	entries, _ := ReadLogEntries(stateDir)
	if got := entries[0].Timestamp.Location(); got.String() != "UTC" {
		t.Errorf("timestamp location=%q, want UTC", got.String())
	}
	// The instant must be preserved regardless of zone.
	if !entries[0].Timestamp.Equal(ny) {
		t.Errorf("timestamp instant changed: got=%v want=%v", entries[0].Timestamp, ny)
	}
}
