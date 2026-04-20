// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/workflow"
	"github.com/greycoderk/lore/internal/workflow/decision"
	"github.com/spf13/cobra"
)

func newDecisionCmd(cfg *config.Config, streams domain.IOStreams, storePtr *domain.LoreStore) *cobra.Command {
	var explain string
	var calibration bool

	cmd := &cobra.Command{
		Use:           "decision",
		Short:         i18n.T().Cmd.DecisionShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if calibration {
				return runCalibration(streams, storePtr)
			}

			ref := explain
			if ref == "" {
				ref = "HEAD"
			}

			adapter := git.NewAdapter(".")
			commitInfo, err := adapter.Log(ref)
			if err != nil {
				return fmt.Errorf("cmd: decision: log %s: %w", ref, err)
			}

			diffContent, diffErr := adapter.Diff(ref)
			if diffErr != nil {
				_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DecisionDiffWarn+"\n", diffErr)
			}
			filesChanged := workflow.ExtractFilesFromDiff(diffContent)
			linesAdded, linesDeleted := workflow.CountDiffLines(diffContent)

			// TODO(post-mvp): extract decision orchestration to internal/service/decision.go
			engineCfg := engineConfigFromApp(cfg)

			var store domain.LoreStore
			if storePtr != nil && *storePtr != nil {
				store = *storePtr
			}
			engine := decision.NewEngine(store, engineCfg)
			ctx := decision.SignalContext{
				ConvType:     commitInfo.Type,
				Scope:        commitInfo.Scope,
				Subject:      commitInfo.Subject,
				Message:      commitInfo.Message,
				DiffContent:  diffContent,
				FilesChanged: filesChanged,
				LinesAdded:   linesAdded,
				LinesDeleted: linesDeleted,
			}

			result := engine.Evaluate(ctx)

			// Display
			_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.DecisionCommitLabel+"\n", commitInfo.Hash[:min(12, len(commitInfo.Hash))])
			_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.DecisionSubjectLabel+"\n", commitInfo.Subject)
			_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.DecisionScoreLabel+"\n", result.Score)
			_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.DecisionActionLabel+"\n", result.Action)
			_, _ = fmt.Fprintf(streams.Out, i18n.T().Cmd.DecisionConfidenceLabel+"\n\n", result.Confidence*100)

			_, _ = fmt.Fprintf(streams.Out, "%s\n", i18n.T().Cmd.DecisionSignalsHeader)
			for _, s := range result.Signals {
				_, _ = fmt.Fprintf(streams.Out, "  %-15s %+3d  %s\n", s.Name, s.Score, s.Reason)
			}

			if result.PrefilledWhy != "" {
				_, _ = fmt.Fprintf(streams.Out, "\n%s\n", i18n.T().Cmd.DecisionPrefillHeader)
				_, _ = fmt.Fprintf(streams.Out, "  "+i18n.T().Cmd.DecisionPrefillWhat+"\n", result.PrefilledWhat)
				_, _ = fmt.Fprintf(streams.Out, "  "+i18n.T().Cmd.DecisionPrefillWhy+"\n", result.PrefilledWhy, result.PrefilledWhyConfidence*100)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&explain, "explain", "", i18n.T().Cmd.DecisionFlagExplain)
	cmd.Flags().BoolVar(&calibration, "calibration", false, i18n.T().Cmd.DecisionFlagCalibration)

	_ = cmd.RegisterFlagCompletionFunc("explain", gitRefFlagCompletion)

	return cmd
}

func runCalibration(streams domain.IOStreams, storePtr *domain.LoreStore) error {
	if err := requireLoreDir(streams); err != nil {
		return err
	}
	if storePtr == nil || *storePtr == nil {
		return fmt.Errorf("%s", i18n.T().Cmd.DecisionStoreUnavail)
	}
	report, err := decision.ComputeCalibration(*storePtr)
	if err != nil {
		return fmt.Errorf("cmd: decision: calibration: %w", err)
	}
	_, _ = fmt.Fprintln(streams.Out, decision.FormatCalibration(report))
	return nil
}
