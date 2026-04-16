// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/spf13/cobra"
)

// Use credential.KnownProviders as the single source of truth.

func newConfigCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config",
		Short:         i18n.T().Cmd.ConfigShort,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	store := credential.NewStore()

	cmd.AddCommand(
		newSetKeyCmd(store, streams),
		newDeleteKeyCmd(store, streams),
		newListKeysCmd(store, streams),
	)

	return cmd
}

// credentialProviderCompletion returns the list of AI providers accepted
// by `lore config set-key` / `delete-key`. Sourced from
// credential.KnownProviders so the list stays in sync with the backend.
func credentialProviderCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return credential.KnownProviders, cobra.ShellCompDirectiveNoFileComp
}

func newSetKeyCmd(store credential.CredentialStore, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:               "set-key <provider>",
		Short:             i18n.T().Cmd.SetKeyShort,
		Args:              cobra.ExactArgs(1),
		SilenceUsage:      true,
		SilenceErrors:     false,
		ValidArgsFunction: credentialProviderCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if !credential.IsKnownProvider(provider) {
				return fmt.Errorf(i18n.T().Cmd.SetKeyUnknownProv, provider, strings.Join(credential.KnownProviders, ", "))
			}

			// Read key from stdin
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.SetKeyPrompt, provider)
			scanner := bufio.NewScanner(streams.In)
			if !scanner.Scan() {
				return fmt.Errorf("config: set-key: no input received")
			}
			key := strings.TrimSpace(scanner.Text())
			if key == "" {
				return fmt.Errorf("config: set-key: empty key")
			}

			if err := store.Set(provider, []byte(key)); err != nil {
				return fmt.Errorf("config: set-key: %w", err)
			}

			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.SetKeyStored+"\n", provider)
			return nil
		},
	}
}

func newDeleteKeyCmd(store credential.CredentialStore, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:               "delete-key <provider>",
		Short:             i18n.T().Cmd.DeleteKeyShort,
		Args:              cobra.ExactArgs(1),
		SilenceUsage:      true,
		SilenceErrors:     false,
		ValidArgsFunction: credentialProviderCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if !credential.IsKnownProvider(provider) {
				return fmt.Errorf(i18n.T().Cmd.DeleteKeyUnknownProv, provider, strings.Join(credential.KnownProviders, ", "))
			}

			if err := store.Delete(provider); err != nil {
				return fmt.Errorf("config: delete-key: %w", err)
			}

			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DeleteKeyDeleted+"\n", provider)
			return nil
		},
	}
}

func newListKeysCmd(store credential.CredentialStore, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "list-keys",
		Short:         i18n.T().Cmd.ListKeysShort,
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			stored, err := store.List()
			if err != nil {
				_, _ = fmt.Fprintf(streams.Err, "Warning: %v\n", err)
				stored = nil
			}

			storedSet := make(map[string]bool)
			for _, p := range stored {
				storedSet[p] = true
			}

			for _, p := range credential.KnownProviders {
				if storedSet[p] {
					_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.ListKeysStored+"\n", p)
				} else {
					_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.ListKeysNotSet+"\n", p)
				}
			}
			return nil
		},
	}
}

