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
	"github.com/spf13/cobra"
)

// Use credential.KnownProviders as the single source of truth.

func newConfigCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config",
		Short:         "Manage Lore configuration",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	store := credential.NewStore()

	cmd.AddCommand(
		newSetKeyCmd(store, streams),
		newDeleteKeyCmd(store, streams),
		newListKeysCmd(store, streams),
	)

	return cmd
}

func newSetKeyCmd(store credential.CredentialStore, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "set-key <provider>",
		Short:         "Store an API key in the system keychain",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if !credential.IsKnownProvider(provider) {
				return fmt.Errorf("unknown provider %q, supported: %s", provider, strings.Join(credential.KnownProviders, ", "))
			}

			// Read key from stdin
			_, _ = fmt.Fprintf(streams.Err, "Enter API key for %s: ", provider)
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

			_, _ = fmt.Fprintf(streams.Err, "Stored   api key for %s in system keychain\n", provider)
			return nil
		},
	}
}

func newDeleteKeyCmd(store credential.CredentialStore, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "delete-key <provider>",
		Short:         "Remove an API key from the system keychain",
		Args:          cobra.ExactArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if !credential.IsKnownProvider(provider) {
				return fmt.Errorf("unknown provider %q, supported: %s", provider, strings.Join(credential.KnownProviders, ", "))
			}

			if err := store.Delete(provider); err != nil {
				return fmt.Errorf("config: delete-key: %w", err)
			}

			_, _ = fmt.Fprintf(streams.Err, "Deleted   api key for %s from system keychain\n", provider)
			return nil
		},
	}
}

func newListKeysCmd(store credential.CredentialStore, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "list-keys",
		Short:         "List stored API keys (masked)",
		SilenceUsage:  true,
		SilenceErrors: true,
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
					_, _ = fmt.Fprintf(streams.Out, "%s: ****\n", p)
				} else {
					_, _ = fmt.Fprintf(streams.Out, "%s: (not set)\n", p)
				}
			}
			return nil
		},
	}
}

