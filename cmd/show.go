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
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

func newShowCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
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
		Use:   "show [keyword]",
		Short: "Find a past decision",
		Long:  "Search and display past decisions from the Lore corpus.",
		Example: `  lore show auth
  lore show --feature auth --after 2026-02
  lore show --all`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-10: Check .lore/ exists
			if _, err := os.Stat(".lore"); err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintln(streams.Err, "Error: Lore not initialized.")
					fmt.Fprintln(streams.Err, "  Run: lore init")
				} else {
					fmt.Fprintf(streams.Err, "Error: cannot access .lore/: %v\n", err)
				}
				return &cli.ExitCodeError{Code: cli.ExitError}
			}

			// Resolve type from shorthand flags
			docType := flagType
			shorthands := 0
			if flagFeature {
				docType = domain.DocTypeFeature
				shorthands++
			}
			if flagDecision {
				docType = domain.DocTypeDecision
				shorthands++
			}
			if flagBugfix {
				docType = domain.DocTypeBugfix
				shorthands++
			}
			if flagRefactor {
				docType = domain.DocTypeRefactor
				shorthands++
			}
			if flagNote {
				docType = domain.DocTypeNote
				shorthands++
			}
			if shorthands > 1 || (flagType != "" && shorthands > 0) {
				fmt.Fprintln(streams.Err, "Error: --type and type shorthand flags (--feature, --decision, etc.) are mutually exclusive.")
				return &cli.ExitCodeError{Code: cli.ExitError}
			}

			keyword := ""
			if len(args) > 0 {
				keyword = args[0]
			}

			// Validation: no keyword and no --all → usage hint
			if keyword == "" && !flagAll {
				fmt.Fprintln(streams.Err, "Usage: lore show [keyword] or lore show --all")
				fmt.Fprintln(streams.Err, "  Try: lore show auth")
				return &cli.ExitCodeError{Code: cli.ExitError}
			}

			docsDir := filepath.Join(".lore", "docs")
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

	return cmd
}

func displayResults(streams domain.IOStreams, results []storage.SearchResult, keyword string, quiet bool) error {
	count := len(results)

	switch count {
	case 0:
		// AC-5: Zero results
		if !quiet {
			if keyword != "" {
				fmt.Fprintf(streams.Err, "No documents matching '%s'.\n", keyword)
				fmt.Fprintln(streams.Err, "  Try: lore show --all")
			} else {
				fmt.Fprintln(streams.Err, "No documents found.")
			}
		}
		return &cli.ExitCodeError{Code: cli.ExitSkip}

	case 1:
		// AC-2: Single result — display directly on stdout
		content, err := storage.ReadDocContent(results[0].Path)
		if err != nil {
			return fmt.Errorf("cmd: show: %w", err)
		}
		_, _ = fmt.Fprint(streams.Out, content)

	default:
		// AC-3 / AC-4: Multiple results
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
			// AC-4: Truncation message handled by ui.List
			idx, err := ui.List(streams, items, "Select a document:")
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
				fmt.Fprintf(streams.Out, "%s\t%s\t%s\n", r.Meta.Type, r.Title, r.Meta.Date)
			}
		}
	}

	return nil
}
