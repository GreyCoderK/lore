// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/store"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/version"
	"github.com/spf13/cobra"
)

func newRootCmd(cfg *config.Config, streams domain.IOStreams, storePtr *domain.LoreStore) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "lore",
		Short:   i18n.T().Cmd.RootShort,
		Long:    i18n.T().Cmd.RootShort,
		Version: version.Info(),
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			// Initialize i18n with env var BEFORE config loading.
			// This ensures init/doctor (which skip config) still respect LORE_LANGUAGE.
			lang := "en"
			if envLang := os.Getenv("LORE_LANGUAGE"); envLang != "" {
				lang = envLang
			}
			if !i18n.IsSupported(lang) {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.UnsupportedLangWarn+"\n", lang)
				lang = "en"
			}
			i18n.Init(lang)

			// Skip config loading for commands that must work without a valid config
			name := c.Name()
			if name == "init" || name == "doctor" {
				return nil
			}

			loaded, err := config.LoadFromDirWithFlags(".", c)
			if err != nil {
				return fmt.Errorf(i18n.T().Cmd.RootConfigErrHint, err)
			}
			if loaded == nil {
				return fmt.Errorf("cmd: config loaded as nil without error")
			}
			*cfg = *loaded

			// Re-init i18n with config language (overrides env var if set).
			// Cascade: flag --language > env LORE_LANGUAGE > .lorerc.local > .lorerc > default "en"
			if cfg.Language != "" && cfg.Language != lang {
				if !i18n.IsSupported(cfg.Language) {
					_, _ = fmt.Fprintf(streams.Err, "Warning: unsupported language '%s', falling back to English\n", cfg.Language)
				} else {
					i18n.Init(cfg.Language)
				}
			}

			// --no-color flag overrides terminal detection
			noColor, _ := c.Flags().GetBool("no-color")
			if noColor {
				ui.SetColorEnabled(false)
			}

			// Open store (graceful degradation if unavailable)
			storePath := filepath.Join(domain.LoreDir, "store.db")
			s, sErr := store.Open(storePath)
			if sErr != nil {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.StoreUnavailWarn+"\n", sErr)
			} else {
				*storePtr = s
			}

			return nil
		},
		PersistentPostRunE: func(c *cobra.Command, args []string) error {
			if *storePtr != nil {
				return (*storePtr).Close()
			}
			return nil
		},
	}

	config.RegisterFlags(cmd)

	cmd.AddCommand(
		newInitCmd(cfg, streams),
		newHookCmd(cfg, streams),
		newHookPostCommitCmd(cfg, streams, storePtr),
		newNewCmd(cfg, streams),
		newShowCmd(cfg, streams),
		newListCmd(cfg, streams),
		newStatusCmd(cfg, streams),
		newPendingCmd(cfg, streams),
		newAngelaCmd(cfg, streams),
		newConfigCmd(cfg, streams),
		newDoctorCmd(cfg, streams),
		newReleaseCmd(cfg, streams),
		newDeleteCmd(cfg, streams),
		newDemoCmd(cfg, streams),
		newDecisionCmd(cfg, streams, storePtr),
		newCompletionCmd(),
	)

	return cmd
}

func Execute() {
	streams := domain.IOStreams{
		Out: os.Stdout,
		Err: os.Stderr,
		In:  os.Stdin,
	}

	ui.SetColorEnabled(ui.ColorEnabled(streams))

	// Resolve language BEFORE Cobra command construction so that
	// i18n.T().Cmd.* strings (Short, Long, Use) are in the correct language.
	// Cascade: env LORE_LANGUAGE > .lorerc lightweight read > default "en".
	earlyLang := os.Getenv("LORE_LANGUAGE")
	if earlyLang == "" {
		earlyLang = config.ReadLanguageOnly(".")
	}
	if earlyLang != "" && i18n.IsSupported(earlyLang) {
		i18n.Init(earlyLang)
	}

	cfg := &config.Config{}
	var loreStore domain.LoreStore
	cmd := newRootCmd(cfg, streams, &loreStore)

	if err := cmd.Execute(); err != nil {
		if code := cli.ExitCodeFrom(err); code >= 0 {
			os.Exit(code)
		}
		os.Exit(cli.ExitError)
	}
}
