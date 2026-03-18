package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newHookCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Manage the post-commit hook",
	}

	cmd.AddCommand(
		newHookInstallCmd(cfg, streams),
		newHookUninstallCmd(cfg, streams),
	)

	return cmd
}

func newHookInstallCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "install",
		Short:         "Install the Lore post-commit hook",
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
				fmt.Fprintf(streams.Err, "Warning: core.hooksPath is set to %q.\n", result.HooksPathWarn)
				fmt.Fprintf(streams.Err, "Lore cannot install hooks automatically.\n")
				fmt.Fprintf(streams.Err, "Add the following to your hook manually:\n\n")
				fmt.Fprintf(streams.Err, "  # LORE-START\n  exec lore _hook-post-commit\n  # LORE-END\n")
				return nil
			}

			ui.Verb(streams, "Installed", "post-commit hook")
			return nil
		},
	}
}

func newHookUninstallCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "uninstall",
		Short:         "Remove the Lore post-commit hook",
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

			ui.Verb(streams, "Removed", "post-commit hook")
			return nil
		},
	}
}
