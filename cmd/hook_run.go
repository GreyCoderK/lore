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
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/greycoderk/lore/internal/workflow/decision"
	"github.com/spf13/cobra"
)

func newHookPostCommitCmd(cfg *config.Config, streams domain.IOStreams, storePtr *domain.LoreStore) *cobra.Command {
	return &cobra.Command{
		Use:           "_hook-post-commit",
		Short:         i18n.T().Cmd.HookPostCommitShort,
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: hook-post-commit getwd: %w", err)
			}

			adapter := git.NewAdapter(workDir)

			// Build Decision Engine from config (nil-safe: works without store)
			engineCfg := engineConfigFromApp(cfg)

			// Use store from root.go PersistentPreRunE (may be nil — graceful degradation)
			var loreStore domain.LoreStore
			if storePtr != nil && *storePtr != nil {
				loreStore = *storePtr
			}
			engine := decision.NewEngine(loreStore, engineCfg)

			// Wire notification + amend config from .lorerc.
			notifyCfg := notifyConfigFromApp(cfg)
			amendPrompt := cfg.Hooks.AmendPrompt

			if err := workflow.DispatchFull(cmd.Context(), workDir, streams, adapter, engine, loreStore, workflow.DispatchConfig{
				NotifyConfig: notifyCfg,
				AmendPrompt:  &amendPrompt,
			}); err != nil {
				return fmt.Errorf("cmd: hook-post-commit: %w", err)
			}
			return nil
		},
	}
}
