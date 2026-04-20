// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

// newAngelaPolishRestoreCmd wires `lore angela polish restore <filename>`.
//
// List available polish backups for a document, then either
// restore the newest one (default), a specific timestamp (`--timestamp`), or
// just print the list without touching anything (`--list`).
//
// The command is deliberately tiny — the heavy lifting lives in
// internal/angela/polish_backup.go so other callers (a future TUI, the
// daemon, etc.) can reuse ListBackups/RestoreBackup without having to go
// through cobra.
func newAngelaPolishRestoreCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagTimestamp string
	var flagList bool

	cmd := &cobra.Command{
		Use:          "restore <filename>",
		Short:        i18n.T().Cmd.AngelaPolishRestoreShort,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			filename := args[0]

			if err := storage.ValidateFilename(filename); err != nil {
				return fmt.Errorf("angela: polish: restore: %w", err)
			}

			workDir, wderr := os.Getwd()
			if wderr != nil {
				return fmt.Errorf("angela: polish: restore: cwd: %w", wderr)
			}
			stateDir := config.ResolveStateDir(workDir, cfg, cfg.DetectedMode)
			if stateDir == "" {
				return fmt.Errorf("angela: polish: restore: cannot resolve state directory (no .lore/ found and no state_dir configured)")
			}

			docsDir := filepath.Join(domain.LoreDir, domain.DocsDir)
			docRelPath := filepath.Join(docsDir, filename)
			backupSubdir := cfg.Angela.Polish.Backup.Path
			if backupSubdir == "" {
				backupSubdir = "polish-backups"
			}

			entries, err := angela.ListBackups(stateDir, backupSubdir, docRelPath)
			if err != nil {
				return fmt.Errorf("angela: polish: restore: %w", err)
			}
			if len(entries) == 0 {
				return fmt.Errorf(i18n.T().Cmd.AngelaPolishRestoreNoBackup, filename)
			}

			if flagList {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishRestoreListHdr+"\n", filename)
				for _, e := range entries {
					_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishRestoreListRow+"\n",
						e.Stamp, e.Timestamp.Format("2006-01-02 15:04:05"))
				}
				return nil
			}

			target := entries[0] // newest by default (ListBackups sorts desc)
			if flagTimestamp != "" {
				match, ok := angela.FindBackupByStamp(entries, flagTimestamp)
				if !ok {
					return fmt.Errorf(i18n.T().Cmd.AngelaPolishRestoreUnknown, flagTimestamp)
				}
				target = match
			}

			destPath := filepath.Join(workDir, filepath.Clean(docRelPath))
			lock, lockErr := fileutil.NewFileLock(destPath)
			if lockErr != nil {
				return fmt.Errorf("angela: polish: restore: dest lock: %w", lockErr)
			}
			defer lock.Unlock()

			if err := angela.RestoreBackup(workDir, docRelPath, target.Path); err != nil {
				return fmt.Errorf("angela: polish: restore: %w", err)
			}
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.AngelaPolishRestoreOK+"\n", filename, target.Stamp)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagTimestamp, "timestamp", "", i18n.T().Cmd.AngelaPolishRestoreFlagTimestamp)
	cmd.Flags().BoolVar(&flagList, "list", false, i18n.T().Cmd.AngelaPolishRestoreFlagList)

	return cmd
}
