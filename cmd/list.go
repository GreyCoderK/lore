// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"sort"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

func newListCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var (
		flagType  string
		flagQuiet bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: i18n.T().Cmd.ListShort,
		Long:  i18n.T().Cmd.ListLong,
		Example: `  lore list
  lore list --type feature
  lore list --quiet | wc -l`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}

			store := &storage.CorpusStore{Dir: domain.DocsPath(".")}
			filter := domain.DocFilter{
				Type: flagType,
			}

			results, parseErr := store.ListDocs(filter)
			if parseErr != nil && len(results) == 0 {
				return fmt.Errorf("cmd: list: %w", parseErr)
			}
			if parseErr != nil && !flagQuiet {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.ListParseWarning+"\n", parseErr)
			}

			// Empty results
			if len(results) == 0 {
				if !flagQuiet {
					if flagType != "" {
						_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.ListNoDocsOfType+"\n", flagType)
					} else {
						_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ListNoDocsYet)
					}
				}
				return nil
			}

			// Sort by date descending
			sort.Slice(results, func(i, j int) bool {
				return results[i].Date > results[j].Date
			})

			// Format output — one line per doc, parseable
			for _, meta := range results {
				slug := storage.ExtractSlug(meta.Filename)
				tagCount := len(meta.Tags)
				tagWord := i18n.T().Cmd.ListTagPlural
				if tagCount == 1 {
					tagWord = i18n.T().Cmd.ListTagSingular
				}
				_, _ = fmt.Fprintf(streams.Out, "%-10s %-25s %s  %d %s\n",
					meta.Type, slug, meta.Date, tagCount, tagWord)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flagType, "type", "", "Filter by document type (decision, feature, bugfix, refactor, note)")
	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Suppress human messages on stderr")

	return cmd
}
