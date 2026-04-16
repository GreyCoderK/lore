// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

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
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/spf13/cobra"
)

func newPendingCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:   "pending",
		Short: i18n.T().Cmd.PendingShort,
		Long:  i18n.T().Cmd.PendingLong,
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

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, i18n.T().Cmd.PendingFlagQuiet)

	cmd.AddCommand(
		newPendingListCmd(nil, streams),
		newPendingResolveCmd(nil, streams),
		newPendingSkipCmd(nil, streams),
	)

	return cmd
}

func newPendingListCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var flagQuiet bool

	cmd := &cobra.Command{
		Use:           "list",
		Short:         i18n.T().Cmd.PendingListShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flagQuiet {
				return runPendingListQuiet(cmd, streams)
			}
			return runPendingList(cmd, streams)
		},
	}

	cmd.Flags().BoolVar(&flagQuiet, "quiet", false, i18n.T().Cmd.PendingFlagQuiet)
	return cmd
}

func newPendingResolveCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	var (
		flagCommit       string
		flagType         string
		flagWhat         string
		flagWhy          string
		flagAlternatives string
		flagImpact       string
	)

	cmd := &cobra.Command{
		Use:           "resolve [number]",
		Short:         i18n.T().Cmd.PendingResolveShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPendingResolve(cmd, streams, args, workflow.ResolveOpts{
				Type:         flagType,
				What:         flagWhat,
				Why:          flagWhy,
				Alternatives: flagAlternatives,
				Impact:       flagImpact,
			}, flagCommit)
		},
	}

	t := i18n.T().Cmd
	cmd.Flags().StringVar(&flagCommit, "commit", "", t.PendingFlagCommit)
	cmd.Flags().StringVar(&flagType, "type", "", t.PendingFlagType)
	cmd.Flags().StringVar(&flagWhat, "what", "", t.PendingFlagWhat)
	cmd.Flags().StringVar(&flagWhy, "why", "", t.PendingFlagWhy)
	cmd.Flags().StringVar(&flagAlternatives, "alternatives", "", t.PendingFlagAlt)
	cmd.Flags().StringVar(&flagImpact, "impact", "", t.PendingFlagImpact)

	_ = cmd.RegisterFlagCompletionFunc("type", docTypeFlagCompletion)
	_ = cmd.RegisterFlagCompletionFunc("commit", gitRefFlagCompletion)

	return cmd
}

func newPendingSkipCmd(_ *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "skip <hash>",
		Short:         i18n.T().Cmd.PendingSkipShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPendingSkip(cmd, streams, args)
		},
	}
}


func runPendingList(cmd *cobra.Command, streams domain.IOStreams) error {
	if err := requireLoreDir(streams); err != nil {
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
		_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.PendingNoPending)
		return nil
	}

	fprintPendingList(streams, items)

	_, _ = fmt.Fprintln(streams.Err)
	_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.PendingResolveHint)
	_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.PendingSkipHint)

	return nil
}

func runPendingListQuiet(cmd *cobra.Command, streams domain.IOStreams) error {
	if err := requireLoreDir(streams); err != nil {
		return err
	}

	pendingDir := filepath.Join(".lore", "pending")
	items, err := workflow.ListPending(cmd.Context(), pendingDir, nil)
	if err != nil {
		return fmt.Errorf("cmd: pending list: %w", err)
	}

	for _, item := range items {
		_, _ = fmt.Fprintf(streams.Out, "%s\t%s\t%s\n", item.CommitHash, item.CommitMessage, item.Progress)
	}

	return nil
}

func runPendingResolve(cmd *cobra.Command, streams domain.IOStreams, args []string, resolveOpts workflow.ResolveOpts, commitFilter string) error {
	if err := requireLoreDir(streams); err != nil {
		return err
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("cmd: pending resolve: getwd: %w", err)
	}

	pendingDir := filepath.Join(".lore", "pending")
	warnWriter := func(msg string) {
		_, _ = fmt.Fprintln(streams.Err, msg)
	}
	items, err := workflow.ListPending(cmd.Context(), pendingDir, warnWriter)
	if err != nil {
		return fmt.Errorf("cmd: pending resolve: %w", err)
	}

	if len(items) == 0 {
		_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.PendingNoPendingRes)
		return nil
	}

	// --commit flag: find the pending item by commit hash prefix.
	if commitFilter != "" {
		for _, item := range items {
			if strings.HasPrefix(item.CommitHash, commitFilter) {
				adapter := git.NewAdapter(workDir)
				return workflow.ResolvePending(cmd.Context(), workDir, streams, item, adapter, resolveOpts)
			}
		}
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Notify.NoMatchingPending+"\n", commitFilter)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	var selected workflow.PendingItem

	if len(args) == 1 {
		// Number provided as argument
		num, parseErr := strconv.Atoi(args[0])
		if parseErr != nil || num < 1 || num > len(items) {
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.PendingInvalidSel+"\n", args[0], len(items))
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
		_, _ = fmt.Fprintln(streams.Err)
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.PendingSelectPrompt, len(items))

		scanner := bufio.NewScanner(streams.In)
		if !scanner.Scan() {
			return fmt.Errorf("cmd: pending resolve: no input")
		}
		input := strings.TrimSpace(scanner.Text())
		num, parseErr := strconv.Atoi(input)
		if parseErr != nil || num < 1 || num > len(items) {
			_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.PendingInvalidSelIn+"\n", input, len(items))
			return &cli.ExitCodeError{Code: cli.ExitError}
		}
		selected = items[num-1]
	}

	adapter := git.NewAdapter(workDir)
	return workflow.ResolvePending(cmd.Context(), workDir, streams, selected, adapter, resolveOpts)
}

func runPendingSkip(cmd *cobra.Command, streams domain.IOStreams, args []string) error {
	if err := requireLoreDir(streams); err != nil {
		return err
	}

	pendingDir := filepath.Join(".lore", "pending")
	item, err := workflow.SkipPending(cmd.Context(), pendingDir, args[0])
	if err != nil {
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.PendingSkipError+"\n", err)
		return &cli.ExitCodeError{Code: cli.ExitError}
	}

	hash := shortHash(item.CommitHash)
	ui.Verb(streams, "Skipped", fmt.Sprintf("%s — %s", hash, item.CommitMessage))
	return nil
}

// fprintPendingList renders the numbered pending list to stderr.
func fprintPendingList(streams domain.IOStreams, items []workflow.PendingItem) {
	_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.PendingListHeader)
	_, _ = fmt.Fprintln(streams.Err)

	for i, item := range items {
		hash := shortHash(item.CommitHash)
		msg := truncate(item.CommitMessage, 40)
		_, _ = fmt.Fprintf(streams.Err, "  %d  %s  %-40s  %s  %s\n",
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
