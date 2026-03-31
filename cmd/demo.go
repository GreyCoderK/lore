// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/generator"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	loretemplate "github.com/greycoderk/lore/internal/template"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

var demoAnswers = map[string]string{
	"type":         "decision",
	"what":         "Add JWT auth middleware for API routes",
	"why":          "Stateless authentication for microservices — JWT tokens avoid session storage",
	"alternatives": "- Session-based auth with Redis\n- OAuth2 with external provider",
	"impact":       "- API routes now require Bearer token\n- Auth middleware added to router chain",
}

const demoCommitHash = "abc1234"
const demoCommitMessage = "feat(auth): add JWT middleware"

func newDemoCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "demo",
		Short:         i18n.T().Cmd.DemoShort,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemo(cmd.Context(), cfg, streams)
		},
	}
}

func runDemo(ctx context.Context, _ *config.Config, streams domain.IOStreams) error {
	// AC-4: Check if Lore is initialized
	if err := requireLoreDir(streams); err != nil {
		return err
	}

	// AC-1: Temporal consent
	if err := ui.Confirm(streams, i18n.T().Cmd.DemoConsentPrompt); err != nil {
		return fmt.Errorf("cmd: demo consent: %w", err)
	}

	// Step 2: Logo
	ui.PrintLogo(streams)

	// Step 3: Simulated commit
	_, _ = fmt.Fprintf(streams.Err, "%s\n\n", ui.Dim(fmt.Sprintf(i18n.T().Cmd.DemoSimCommit, demoCommitMessage)))
	demoPause(ctx)

	// Step 4: Question flow (simulated — user watches, doesn't type)
	ui.Progress(streams, 1, 3, "Type")
	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DemoTypePrompt+"\n", demoAnswers["type"])
	_, _ = fmt.Fprintf(streams.Err, "%s %s\n\n", ui.Success("✓"), fmt.Sprintf(i18n.T().Cmd.DemoTypeConfirm, demoAnswers["type"]))
	demoPause(ctx)

	ui.Progress(streams, 2, 3, "What")
	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DemoWhatPrompt+"\n", demoAnswers["what"])
	_, _ = fmt.Fprintf(streams.Err, "%s %s\n\n", ui.Success("✓"), fmt.Sprintf(i18n.T().Cmd.DemoWhatConfirm, demoAnswers["what"]))
	demoPause(ctx)

	ui.Progress(streams, 3, 3, "Why")
	_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DemoWhyPrompt+"\n\n", demoAnswers["why"])
	demoPause(ctx)

	// Step 5: Generate document through real pipeline
	engine, err := loretemplate.New(
		filepath.Join(".lore", "templates"),
		loretemplate.GlobalDir(),
	)
	if err != nil {
		return fmt.Errorf("cmd: demo template: %w", err)
	}

	now := time.Now()
	input := generator.GenerateInput{
		DocType:      demoAnswers["type"],
		What:         demoAnswers["what"],
		Why:          demoAnswers["why"],
		Alternatives: demoAnswers["alternatives"],
		Impact:       demoAnswers["impact"],
		CommitInfo: &domain.CommitInfo{
			Hash: demoCommitHash,
			Date: now,
		},
		GeneratedBy: "lore-demo",
	}

	genResult, err := generator.Generate(ctx, engine, input)
	if err != nil {
		return fmt.Errorf("cmd: demo generate: %w", err)
	}

	// L11 fix: use the meta produced by Generate() as the base — eliminates the
	// redundant date construction (was: now.Format("2006-01-02") alongside
	// input.CommitInfo.Date = now). Override Status and Tags for demo-specific fields.
	demoMeta := genResult.Meta
	demoMeta.Status = "demo"
	demoMeta.Tags = []string{"authentication", "jwt", "middleware"}

	docsDir := filepath.Join(domain.LoreDir, domain.DocsDir)
	result, err := storage.WriteDoc(docsDir, demoMeta, demoAnswers["what"], genResult.Body)
	if err != nil {
		return fmt.Errorf("cmd: demo write: %w", err)
	}

	// Regenerate index after write (best-effort).
	if indexErr := storage.RegenerateIndex(docsDir); indexErr != nil {
		_, _ = fmt.Fprintf(streams.Err, i18n.T().Cmd.DemoIndexWarning+"\n", indexErr)
	}

	ui.Verb(streams, "Created", result.Filename)
	_, _ = fmt.Fprintln(streams.Err)
	demoPause(ctx)

	// Step 6: Simulated lore show
	_, _ = fmt.Fprintf(streams.Err, "%s\n\n", ui.Dim(i18n.T().Cmd.DemoSimShow))

	_, _ = fmt.Fprintf(streams.Err, "  %s\n\n", ui.Bold(result.Filename))
	_, _ = fmt.Fprintf(streams.Err, "  %s\n", fmt.Sprintf(i18n.T().Cmd.DemoShowType, demoAnswers["type"]))
	_, _ = fmt.Fprintf(streams.Err, "  %s\n", fmt.Sprintf(i18n.T().Cmd.DemoShowWhat, demoAnswers["what"]))
	_, _ = fmt.Fprintf(streams.Err, "  %s\n", fmt.Sprintf(i18n.T().Cmd.DemoShowWhy, demoAnswers["why"]))
	_, _ = fmt.Fprintf(streams.Err, "  %s\n", fmt.Sprintf(i18n.T().Cmd.DemoShowDate, now.Format("2006-01-02")))
	_, _ = fmt.Fprintf(streams.Err, "  %s\n", fmt.Sprintf(i18n.T().Cmd.DemoShowCommit, demoCommitHash))
	_, _ = fmt.Fprintf(streams.Err, "  %s\n\n", i18n.T().Cmd.DemoShowStatus)

	// Step 7: Tagline AFTER proof of value
	_, _ = fmt.Fprintf(streams.Err, "%s\n\n", ui.Bold(i18n.T().Cmd.DemoTagline))

	return nil
}

func demoPause(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(800 * time.Millisecond):
	}
}
