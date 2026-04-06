// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newHookCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: i18n.T().Cmd.HookShort,
	}

	cmd.AddCommand(
		newHookInstallCmd(nil, streams),
		newHookUninstallCmd(nil, streams),
	)

	return cmd
}

func newHookInstallCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "install",
		Short:         i18n.T().Cmd.HookInstallShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: hook install getwd: %w", err)
			}

			adapter := git.NewAdapter(workDir)
			result, err := adapter.InstallHook("post-commit")
			if err != nil {
				return fmt.Errorf("cmd: hook install: %w", err)
			}

			if result.HooksPathWarn != "" {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.HookInstallHooksPathW+"\n", result.HooksPathWarn)
				_, _ = fmt.Fprintf(streams.Err, "%s\n", i18n.T().Cmd.HookInstallCannotAuto)
				_, _ = fmt.Fprintf(streams.Err, "%s\n\n", i18n.T().Cmd.HookInstallManualHint)
				_, _ = fmt.Fprintf(streams.Err, "  # LORE-START\n  if (: < /dev/tty) 2>/dev/null; then\n    exec lore _hook-post-commit < /dev/tty\n  else\n    exec lore _hook-post-commit\n  fi\n  # LORE-END\n")
				return nil
			}

			ui.Verb(streams, "Installed", i18n.T().Cmd.HookInstallVerb)
			return nil
		},
	}
}

func newHookUninstallCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "uninstall",
		Short:         i18n.T().Cmd.HookUninstallShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: hook uninstall getwd: %w", err)
			}

			adapter := git.NewAdapter(workDir)
			if err := adapter.UninstallHook("post-commit"); err != nil {
				return fmt.Errorf("cmd: hook uninstall: %w", err)
			}

			ui.Verb(streams, "Removed", i18n.T().Cmd.HookUninstallVerb)
			return nil
		},
	}
}
