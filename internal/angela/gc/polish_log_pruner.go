// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package gc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/fileutil"
)

// polishLogPruner applies retention and cap policies to polish.log.
//
// Two-pass algorithm on each run:
//   1. Drop lines whose ts is older than
//      cfg.Angela.Polish.Log.RetentionDays.
//   2. If the surviving content still exceeds
//      cfg.Angela.Polish.Log.MaxSizeMB, drop the oldest survivors
//      until the cap is respected.
//
// Under dry-run the surviving content is computed but the file is
// left untouched. Counts are reported normally so callers know what
// WOULD be removed.
type polishLogPruner struct{}

func init() {
	Register(&polishLogPruner{})
}

func (polishLogPruner) Name() string    { return "polish-log" }
func (polishLogPruner) Pattern() string { return "polish.log" }

func (p polishLogPruner) Prune(_ context.Context, workDir string, cfg *config.Config, dryRun bool) PruneReport {
	report := PruneReport{Feature: "polish-log", DryRun: dryRun}

	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	logPath := angela.PolishLogPath(stateDir)

	// Missing log = nothing to prune. Not an error.
	info, statErr := os.Stat(logPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return report
		}
		report.Err = fmt.Errorf("polish-log pruner: stat: %w", statErr)
		return report
	}

	retentionDays := cfg.Angela.Polish.Log.RetentionDays
	maxSizeMB := cfg.Angela.Polish.Log.MaxSizeMB

	// If neither policy is set, do nothing — the log grows forever
	// intentionally in that configuration.
	if retentionDays <= 0 && maxSizeMB <= 0 {
		report.Kept = -1 // sentinel: "policy off"
		return report
	}

	// Acquire the same advisory lock the writer uses so we serialize
	// with concurrent polish invocations writing new entries.
	lock, err := fileutil.NewFileLock(logPath)
	if err != nil {
		report.Err = fmt.Errorf("polish-log pruner: lock: %w", err)
		return report
	}
	defer lock.Unlock()

	// Re-stat under the lock so the size/mtime we use as "baseline" is
	// consistent with what we're about to read. Story 8-23 P0: a
	// writer that bypasses the advisory lock (test harness, shell
	// echo) could have appended between the pre-lock Stat and the
	// rewrite; we compare this baseline again just before the rewrite
	// and abort the prune if the file grew to avoid silent data loss
	// through the AtomicWriteStream rename-replace flow.
	info, statErr = os.Stat(logPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			return report
		}
		report.Err = fmt.Errorf("polish-log pruner: restat: %w", statErr)
		return report
	}
	baselineSize := info.Size()
	baselineMtime := info.ModTime()

	f, err := os.Open(logPath)
	if err != nil {
		report.Err = fmt.Errorf("polish-log pruner: open: %w", err)
		return report
	}
	type lineInfo struct {
		raw string
		ts  time.Time
		ok  bool // ts parsed cleanly
	}
	var all []lineInfo
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		li := lineInfo{raw: line}
		var probe struct {
			TS time.Time `json:"ts"`
		}
		if err := json.Unmarshal([]byte(line), &probe); err == nil {
			li.ts = probe.TS
			li.ok = true
		}
		all = append(all, li)
	}
	_ = f.Close()

	totalLines := len(all)

	// Pass 1: retention filter.
	var survivors []lineInfo
	if retentionDays > 0 {
		cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		for _, li := range all {
			// Keep unparseable lines — we prefer to over-retain than
			// to lose information we don't understand.
			if !li.ok || !li.ts.Before(cutoff) {
				survivors = append(survivors, li)
			}
		}
	} else {
		survivors = all
	}

	// Pass 2: size cap. Compute current size of survivors and drop
	// oldest until under the cap.
	if maxSizeMB > 0 {
		capBytes := int64(maxSizeMB) * 1024 * 1024
		var survivorSize int64
		for _, li := range survivors {
			survivorSize += int64(len(li.raw)) + 1 // +1 for newline
		}
		// Drop from the front (oldest first) — assumes JSONL is
		// append-ordered, which the writer guarantees.
		for survivorSize > capBytes && len(survivors) > 0 {
			survivorSize -= int64(len(survivors[0].raw)) + 1
			survivors = survivors[1:]
		}
	}

	removed := totalLines - len(survivors)
	report.Removed = removed
	report.Kept = len(survivors)

	// Compute bytes freed (approximate — counts newline terminators).
	if removed > 0 {
		report.Bytes = info.Size()
		var newSize int64
		for _, li := range survivors {
			newSize += int64(len(li.raw)) + 1
		}
		report.Bytes = info.Size() - newSize
		if report.Bytes < 0 {
			report.Bytes = 0
		}
	}

	if dryRun || removed == 0 {
		return report
	}

	// Re-stat once more under the still-held lock. If the file grew
	// or the mtime advanced between our baseline and now, a concurrent
	// writer has bypassed the advisory lock. Abort rather than
	// rename-replace: the extra lines we didn't read would disappear
	// silently from the rewritten file.
	preWriteInfo, preWriteErr := os.Stat(logPath)
	if preWriteErr != nil {
		report.Err = fmt.Errorf("polish-log pruner: pre-write stat: %w", preWriteErr)
		return report
	}
	if preWriteInfo.Size() != baselineSize || !preWriteInfo.ModTime().Equal(baselineMtime) {
		report.Err = fmt.Errorf("polish-log pruner: concurrent write detected (size %d→%d), aborting to preserve appended entries",
			baselineSize, preWriteInfo.Size())
		// Do not count removals we did not actually apply.
		report.Removed = 0
		report.Bytes = 0
		return report
	}

	// Rewrite the log atomically.
	var buf bytes.Buffer
	for _, li := range survivors {
		buf.WriteString(li.raw)
		buf.WriteByte('\n')
	}
	if err := fileutil.AtomicWriteStream(logPath, &buf, 0o600); err != nil {
		report.Err = fmt.Errorf("polish-log pruner: write: %w", err)
		return report
	}
	return report
}
