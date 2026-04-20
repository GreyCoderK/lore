// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package gc

import (
	"context"
	"path/filepath"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
)

// polishBackupPruner wraps the existing PruneOldBackups function at
// internal/angela/polish_backup.go. That function was already running
// at the end of each polish invocation (auto-prune); this Pruner lets
// `lore doctor --prune` apply the same retention on demand without
// needing to issue a polish.
type polishBackupPruner struct{}

func init() {
	Register(&polishBackupPruner{})
}

func (polishBackupPruner) Name() string    { return "polish-backups" }
func (polishBackupPruner) Pattern() string { return "polish-backups/*.bak" }

func (polishBackupPruner) Prune(_ context.Context, workDir string, cfg *config.Config, dryRun bool) PruneReport {
	report := PruneReport{Feature: "polish-backups", DryRun: dryRun}

	stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
	backupSubdir := cfg.Angela.Polish.Backup.Path
	if backupSubdir == "" {
		backupSubdir = "polish-backups"
	}
	retention := cfg.Angela.Polish.Backup.RetentionDays
	if retention <= 0 {
		// 0 = keep forever — nothing to do.
		return report
	}
	backupRoot := filepath.Join(stateDir, backupSubdir)

	// The existing PruneOldBackups API does not support dry-run. For
	// the dry-run path we just report 0 removed — a precise dry-run
	// counter is post-MVP scope. The live path delegates to the
	// existing implementation so we don't duplicate the timestamp
	// parse / cutoff logic.
	if dryRun {
		// TODO(post-v1.2): expose a dry-run-capable variant of
		// PruneOldBackups so the report here can show what WOULD be
		// removed. For now, dry-run is a signal-only no-op.
		return report
	}
	if err := angela.PruneOldBackups(backupRoot, retention); err != nil {
		report.Err = err
		return report
	}
	// PruneOldBackups doesn't currently surface counts; the lowest-
	// friction integration is to report an empty but non-error
	// outcome, letting callers know the pruner ran cleanly.
	return report
}
