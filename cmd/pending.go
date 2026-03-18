package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/greycoderk/lore/internal/cli"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/spf13/cobra"
)

func newPendingCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:   "pending",
		Short: "Resume skipped documentation",
		Long:  "List, resolve, or skip pending documentation from interrupted or deferred commits.",
		Example: `  lore pending
  lore pending list
  lore pending resolve
  lore pending resolve 2
  lore pending skip abc1234`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagQuiet {
				return runPendingListQuiet(cmd, streams)
			}
			return runPendingList(cmd, streams)
		},
	}

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Machine-readable output on stdout")

	cmd.AddCommand(
		newPendingListCmd(cfg, streams),
		newPendingResolveCmd(cfg, streams),
		newPendingSkipCmd(cfg, streams),
	)

	return cmd
}

func newPendingListCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List pending documentation items",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagQuiet {
				return runPendingListQuiet(cmd, streams)
			}
			return runPendingList(cmd, streams)
		},
	}

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, "Machine-readable output on stdout")
	return cmd
}

func newPendingResolveCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "resolve [number]",
		Short:         "Resolve a pending item by answering remaining questions",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPendingResolve(cmd, streams, args)
		},
	}
}

func newPendingSkipCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "skip <hash>",
		Short:         "Skip a pending item without creating a document",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPendingSkip(cmd, streams, args)
		},
	}
}

func checkLoreDir(streams domain.IOStreams) error {
	if _, err := os.Stat(".lore"); err != nil {
		if os.IsNotExist(err) {
			_, _ = fmt.Fprintln(streams.Err, "Error: Lore not initialized.")
			_, _ = fmt.Fprintln(streams.Err, "  Run: lore init")
		} else {
			fmt.Fprintf(streams.Err, "Error: cannot access .lore/: %v\n", err)
		}
		return &cli.ExitCodeError{Code: cli.ExitError}
	}
	return nil
}

func runPendingList(cmd *cobra.Command, streams domain.IOStreams) error {
	if err := checkLoreDir(streams); err != nil {
		return err
	}

	pendingDir := filepath.Join(".lore", "pending")
	warnWriter := func(msg string) {
		_, _ = fmt.Fprintln(streams.Err, msg)
	}
	items, err := workflow.ListPending(cmd.Context(), pendingDir, warnWriter)
	if err != nil {
		return fmt.Errorf("cmd: pending list: %w", err)
	}

	if len(items) == 0 {
		fmt.Fprintln(streams.Err, "No pending documentation. All commits documented.")
		return nil
	}

	fprintPendingList(streams, items)

	fmt.Fprintln(streams.Err)
	fmt.Fprintln(streams.Err, "Resolve: lore pending resolve [number]")
	fmt.Fprintln(streams.Err, "Skip:    lore pending skip <hash>")

	return nil
}

func runPendingListQuiet(cmd *cobra.Command, streams domain.IOStreams) error {
	if err := checkLoreDir(streams); err != nil {
		return err
	}

	pendingDir := filepath.Join(".lore", "pending")
	items, err := workflow.ListPending(cmd.Context(), pendingDir, nil)
	if err != nil {
		return fmt.Errorf("cmd: pending list: %w", err)
	}

	for _, item := range items {
		fmt.Fprintf(streams.Out, "%s\t%s\t%s\n", item.CommitHash, item.CommitMessage, item.Progress)
	}

	return nil
}

func runPendingResolve(cmd *cobra.Command, streams domain.IOStreams, args []string) error {
	if err := checkLoreDir(streams); err != nil {
		return err
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cmd: pending resolve: getwd: %w", err)
	}

	pendingDir := filepath.Join(".lore", "pending")
	warnWriter := func(msg string) {
		fmt.Fprintln(streams.Err, msg)
	}
	items, err := workflow.ListPending(cmd.Context(), pendingDir, warnWriter)
	if err != nil {
		return fmt.Errorf("cmd: pending resolve: %w", err)
	}

	if len(items) == 0 {
		fmt.Fprintln(streams.Err, "No pending documentation. All commits documented.")
		return nil
	}

	var selected workflow.PendingItem

	if len(args) == 1 {
		// Number provided as argument
		num, parseErr := strconv.Atoi(args[0])
		if parseErr != nil || num < 1 || num > len(items) {
			fmt.Fprintf(streams.Err, "Error: invalid selection '%s'. Choose 1-%d.\n", args[0], len(items))
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		selected = items[num-1]
	} else if len(items) == 1 {
		// Single item -> resolve directly
		selected = items[0]
	} else if !workflow.IsInteractiveTTY(streams) {
		// Non-TTY: auto-resolve most recent (first in list, sorted desc)
		selected = items[0]
	} else {
		// Multiple items, interactive -> show numbered list and prompt
		fprintPendingList(streams, items)
		fmt.Fprintln(streams.Err)
		fmt.Fprintf(streams.Err, "Select item to resolve [1-%d]: ", len(items))

		scanner := bufio.NewScanner(streams.In)
		if !scanner.Scan() {
			return fmt.Errorf("cmd: pending resolve: no input")
		}
		input := strings.TrimSpace(scanner.Text())
		num, parseErr := strconv.Atoi(input)
		if parseErr != nil || num < 1 || num > len(items) {
			fmt.Fprintf(streams.Err, "Error: invalid selection '%s'. Choose 1-%d.\n", input, len(items))
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		selected = items[num-1]
	}

	adapter := git.NewAdapter(workDir)
	return workflow.ResolvePending(cmd.Context(), workDir, streams, selected, adapter, workflow.ResolveOpts{})
}

func runPendingSkip(cmd *cobra.Command, streams domain.IOStreams, args []string) error {
	if err := checkLoreDir(streams); err != nil {
		return err
	}

	pendingDir := filepath.Join(".lore", "pending")
	item, err := workflow.SkipPending(cmd.Context(), pendingDir, args[0])
	if err != nil {
		fmt.Fprintf(streams.Err, "Error: %v\n", err)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	hash := shortHash(item.CommitHash)
	ui.Verb(streams, "Skipped", fmt.Sprintf("%s — %s", hash, item.CommitMessage))
	return nil
}

// fprintPendingList renders the numbered pending list to stderr.
func fprintPendingList(streams domain.IOStreams, items []workflow.PendingItem) {
	fmt.Fprintln(streams.Err, "Pending documentation:")
	fmt.Fprintln(streams.Err)

	for i, item := range items {
		hash := shortHash(item.CommitHash)
		msg := truncate(item.CommitMessage, 40)
		fmt.Fprintf(streams.Err, "  %d  %s  %-40s  %s  %s\n",
			i+1, hash, msg, item.Progress, item.RelativeAge)
	}
}

// shortHash returns the first 7 characters of a hash, or the full string if shorter.
func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// truncate shortens s to maxLen runes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string([]rune(s)[:maxLen])
	}
	return string([]rune(s)[:maxLen-3]) + "..."
}
