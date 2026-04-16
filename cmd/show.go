// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newShowCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var (
		flagType     string
		flagAfter    string
		flagAll      bool
		flagQuiet    bool
		flagFeature  bool
		flagDecision bool
		flagBugfix   bool
		flagRefactor bool
		flagNote     bool
	)

	cmd := &cobra.Command{
		Use:   i18n.T().Cmd.ShowUse,
		Short: i18n.T().Cmd.ShowShort,
		Long:  i18n.T().Cmd.ShowLong,
		Example: `  lore show auth
  lore show --feature auth --after 2026-02
  lore show --all`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		ValidArgsFunction: docsFileCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check .lore/ exists
			if err := requireLoreDir(streams); err != nil {
				return err
			}

			// Resolve type from shorthand flags
			docType, err := resolveDocTypeFlags(flagType, flagFeature, flagDecision, flagBugfix, flagRefactor, flagNote)
			if err != nil {
				_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ShowTypeExclusive)
				return &cli.ExitCodeError{Code: cli.ExitError}
			}

			keyword := ""
			if len(args) > 0 {
				keyword = args[0]
			}

			// Validation: no keyword and no --all → usage hint
			if keyword == "" && !flagAll {
				_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ShowUsageHint)
				_, _ = fmt.Fprintln(streams.Err, "  "+i18n.T().Cmd.ShowTryHint)
				return &cli.ExitCodeError{Code: cli.ExitUserError}
			}

			// Deprecation: --all is redundant with `lore list`
			if flagAll && !flagQuiet {
				_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ShowAllDeprecated)
			}

			docsDir := domain.DocsPath(".")
			filter := domain.DocFilter{
				Type:  docType,
				After: flagAfter,
			}

			results, err := storage.SearchDocs(docsDir, keyword, filter)
			if err != nil {
				return fmt.Errorf("cmd: show: %w", err)
			}

			return displayResults(streams, results, keyword, flagQuiet)
		},
	}

	cmd.Flags().StringVar(&flagType, "type", "", "Filter by document type (decision, feature, bugfix, refactor, note)")
	cmd.Flags().StringVar(&flagAfter, "after", "", "Show documents after date (YYYY-MM or YYYY-MM-DD)")
	cmd.Flags().BoolVar(&flagAll, "all", false, "Show all documents")
	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Suppress human messages on stderr")
	cmd.Flags().BoolVar(&flagFeature, "feature", false, "Shorthand for --type feature")
	cmd.Flags().BoolVar(&flagDecision, "decision", false, "Shorthand for --type decision")
	cmd.Flags().BoolVar(&flagBugfix, "bugfix", false, "Shorthand for --type bugfix")
	cmd.Flags().BoolVar(&flagRefactor, "refactor", false, "Shorthand for --type refactor")
	cmd.Flags().BoolVar(&flagNote, "note", false, "Shorthand for --type note")

	_ = cmd.RegisterFlagCompletionFunc("type", docTypeFlagCompletion)

	return cmd
}

func displayResults(streams domain.IOStreams, results []storage.SearchResult, keyword string, quiet bool) error {
	count := len(results)

	switch count {
	case 0:
		// Zero results
		if !quiet {
			if keyword != "" {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.ShowNoMatchKeyword+"\n", keyword)
				_, _ = fmt.Fprintln(streams.Err, "  "+i18n.T().Cmd.ShowTryAll)
			} else {
				_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ShowNoDocsFound)
			}
		}
		return &cli.ExitCodeError{Code: cli.ExitSkip}

	case 1:
		// Single result — display directly on stdout
		content, err := storage.ReadDocContent(results[0].Path)
		if err != nil {
			return fmt.Errorf("cmd: show: %w", err)
		}
		_, _ = fmt.Fprint(streams.Out, content)

	default:
		// Multiple results
		items := make([]ui.ListItem, len(results))
		for i, r := range results {
			items[i] = ui.ListItem{
				Type:  r.Meta.Type,
				Title: r.Title,
				Date:  r.Meta.Date,
			}
		}

		isTTY := ui.IsTerminal(streams)

		if !quiet {
			// Truncation message handled by ui.List
			idx, err := ui.List(streams, items, i18n.T().Cmd.ShowSelectPrompt)
			if err != nil {
				return fmt.Errorf("cmd: show: %w", err)
			}

			// TTY: user selected a document
			if isTTY && idx >= 0 {
				content, err := storage.ReadDocContent(results[idx].Path)
				if err != nil {
					return fmt.Errorf("cmd: show: %w", err)
				}
				_, _ = fmt.Fprint(streams.Out, content)
			}
			// Non-TTY: ui.List already printed parseable list to stdout
		} else {
			// --quiet + multiple: stdout parseable list only
			for _, r := range results {
				_, _ = fmt.Fprintf(streams.Out, "%s\t%s\t%s\n", r.Meta.Type, r.Title, r.Meta.Date)
			}
		}
	}

	return nil
}
