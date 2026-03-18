package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

func newListCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var (
		flagType  string
		flagQuiet bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "See all documented decisions",
		Long:  "List all documents in the Lore corpus with type, title, date, and tag count.",
		Example: `  lore list
  lore list --type feature
  lore list --quiet | wc -l`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// AC-6: Check .lore/ exists
			if _, err := os.Stat(".lore"); err != nil {
				if os.IsNotExist(err) {
					_, _ = fmt.Fprintln(streams.Err, "Error: Lore not initialized.")
					_, _ = fmt.Fprintln(streams.Err, "  Run: lore init")
				} else {
					fmt.Fprintf(streams.Err, "Error: cannot access .lore/: %v\n", err)
				}
				return &cli.ExitCodeError{Code: cli.ExitError}
			}

			store := &storage.CorpusStore{Dir: ".lore/docs"}
			filter := domain.DocFilter{
				Type: flagType,
			}

			results, parseErr := store.ListDocs(filter)
			if parseErr != nil && len(results) == 0 {
				return fmt.Errorf("cmd: list: %w", parseErr)
			}
			if parseErr != nil && !flagQuiet {
				fmt.Fprintf(streams.Err, "Warning: some documents could not be parsed: %v\n", parseErr)
			}

			// AC-2: Empty results
			if len(results) == 0 {
				if !flagQuiet {
					if flagType != "" {
						fmt.Fprintf(streams.Err, "No documents of type '%s'.\n", flagType)
					} else {
						_, _ = fmt.Fprintln(streams.Err, "No documents yet. Run: lore new")
					}
				}
				return nil
			}

			// AC-5: Sort by date descending
			sort.Slice(results, func(i, j int) bool {
				return results[i].Date > results[j].Date
			})

			// AC-1: Format output — one line per doc, parseable
			for _, meta := range results {
				slug := storage.ExtractSlug(meta.Filename)
				tagCount := len(meta.Tags)
				tagWord := "tags"
				if tagCount == 1 {
					tagWord = "tag"
				}
				fmt.Fprintf(streams.Out, "%-10s %-25s %s  %d %s\n",
					meta.Type, slug, meta.Date, tagCount, tagWord)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&flagType, "type", "", "Filter by document type (decision, feature, bugfix, refactor, note)")
	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Suppress human messages on stderr")

	return cmd
}
