// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package gc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
)

// corruptQuarantinePruner cleans up state-file quarantine artifacts
// created by angela.QuarantineCorruptState when a JSON state file
// fails to parse. Those files sit alongside the live state files
// under the state dir as `<name>.corrupt-<stamp>` where stamp is
// formatted per angela.QuarantineTimestampLayout (single source of
// truth shared with the producer).
//
// Retention is driven by cfg.Angela.GC.CorruptQuarantine.RetentionDays
// (default 14). Files whose timestamp parses cleanly AND is older
// than the cutoff are removed; files with an unparseable stamp are
// always kept (defensive: we don't delete what we don't understand).
//
// Symlink safety: the sweep uses os.Lstat and skips any entry that is
// not a regular file. Without this, os.Remove on a symlink
// `attacker.corrupt-<old>` pointing to /etc/passwd would unlink the
// dangling link but the earlier os.Stat (which follows links) would
// have credited the target's size to Bytes freed — a misleading
// report. Regular-files-only also aligns with polish_backup.go's
// symlink-reject policy.
type corruptQuarantinePruner struct{}

func init() {
	Register(&corruptQuarantinePruner{})
}

func (corruptQuarantinePruner) Name() string    { return "corrupt-quarantine" }
func (corruptQuarantinePruner) Pattern() string { return "*.corrupt-*" }

func (corruptQuarantinePruner) Prune(_ context.Context, workDir string, cfg *config.Config, dryRun bool) PruneReport {
	report := PruneReport{Feature: "corrupt-quarantine", DryRun: dryRun}

	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	retentionDays := cfg.Angela.GC.CorruptQuarantine.RetentionDays
	if retentionDays <= 0 {
		// 0 = keep forever — no scan needed.
		return report
	}
	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)

	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return report
		}
		report.Err = fmt.Errorf("corrupt-quarantine pruner: read dir: %w", err)
		return report
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Expected shape: "<base>.corrupt-<YYYYMMDDTHHMMSS.sss>".
		idx := strings.Index(name, ".corrupt-")
		if idx < 0 {
			continue
		}
		stampPart := name[idx+len(".corrupt-"):]
		stamp, perr := time.Parse(angela.QuarantineTimestampLayout, stampPart)
		if perr != nil {
			// Unparseable — keep. Defensive: we don't delete files
			// whose age we can't establish.
			report.Kept++
			continue
		}
		if !stamp.Before(cutoff) {
			report.Kept++
			continue
		}
		// Eligible for removal — but refuse to follow symlinks or touch
		// special files. Use Lstat (does not follow links).
		full := filepath.Join(stateDir, name)
		info, ierr := os.Lstat(full)
		if ierr != nil {
			report.Kept++
			continue
		}
		if !info.Mode().IsRegular() {
			// Symlink, socket, device, pipe — never delete.
			report.Kept++
			continue
		}
		size := info.Size()
		if dryRun {
			report.Removed++
			report.Bytes += size
			continue
		}
		if rerr := os.Remove(full); rerr != nil {
			// Keep trying the rest — first error surfaces on Err but
			// does not stop the sweep.
			if report.Err == nil {
				report.Err = fmt.Errorf("corrupt-quarantine pruner: remove %s: %w", name, rerr)
			}
			report.Kept++
			continue
		}
		report.Removed++
		report.Bytes += size
	}
	return report
}
