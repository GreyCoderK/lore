package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/generator"
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
		Short:         "See Lore in action with a guided walkthrough",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDemo(cmd.Context(), cfg, streams)
		},
	}
}

func runDemo(ctx context.Context, cfg *config.Config, streams domain.IOStreams) error {
	// AC-4: Check if Lore is initialized
	loreDir := filepath.Join(".", ".lore")
	if _, err := os.Stat(loreDir); os.IsNotExist(err) {
		ui.ActionableError(streams, "Lore not initialized.", "lore init")
		return fmt.Errorf("cmd: demo: %w", domain.ErrNotInitialized)
	} else if err != nil {
		return fmt.Errorf("cmd: demo: %w", err)
	}

	// AC-1: Temporal consent
	if err := ui.Confirm(streams, "Demo interactive ~45s. Press Enter to begin.\n"); err != nil {
		return fmt.Errorf("cmd: demo consent: %w", err)
	}

	// Step 2: Logo
	ui.PrintLogo(streams)

	// Step 3: Simulated commit
	fmt.Fprintf(streams.Err, "%s\n\n", ui.Dim("Simulating: git commit -m '"+demoCommitMessage+"'"))
	demoPause(ctx)

	// Step 4: Question flow (simulated — user watches, doesn't type)
	ui.Progress(streams, 1, 3, "Type")
	fmt.Fprintf(streams.Err, "? Type [feature]: %s\n", demoAnswers["type"])
	fmt.Fprintf(streams.Err, "%s Type: %s\n\n", ui.Success("✓"), demoAnswers["type"])
	demoPause(ctx)

	ui.Progress(streams, 2, 3, "What")
	fmt.Fprintf(streams.Err, "? What [add JWT auth middleware]: %s\n", demoAnswers["what"])
	fmt.Fprintf(streams.Err, "%s What: %s\n\n", ui.Success("✓"), demoAnswers["what"])
	demoPause(ctx)

	ui.Progress(streams, 3, 3, "Why")
	fmt.Fprintf(streams.Err, "? Why was this approach chosen?\n> %s\n\n", demoAnswers["why"])
	demoPause(ctx)

	// Step 5: Generate document through real pipeline
	engine, err := loretemplate.New(
		filepath.Join(loreDir, "templates"),
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

	docsDir := filepath.Join(loreDir, "docs")
	result, err := storage.WriteDoc(docsDir, demoMeta, demoAnswers["what"], genResult.Body)
	if err != nil {
		return fmt.Errorf("cmd: demo write: %w", err)
	}

	if result.IndexErr != nil {
		fmt.Fprintf(streams.Err, "Warning: index update failed: %v\n", result.IndexErr)
	}

	ui.Verb(streams, "Created", result.Filename)
	_, _ = fmt.Fprintln(streams.Err)
	demoPause(ctx)

	// Step 6: Simulated lore show
	fmt.Fprintf(streams.Err, "%s\n\n", ui.Dim("Simulating: lore show auth"))

	fmt.Fprintf(streams.Err, "  %s\n\n", ui.Bold(result.Filename))
	fmt.Fprintf(streams.Err, "  Type:    %s\n", demoAnswers["type"])
	fmt.Fprintf(streams.Err, "  What:    %s\n", demoAnswers["what"])
	fmt.Fprintf(streams.Err, "  Why:     %s\n", demoAnswers["why"])
	fmt.Fprintf(streams.Err, "  Date:    %s\n", now.Format("2006-01-02"))
	fmt.Fprintf(streams.Err, "  Commit:  %s\n", demoCommitHash)
	fmt.Fprintf(streams.Err, "  Status:  demo\n\n")

	// Step 7: Tagline AFTER proof of value
	fmt.Fprintf(streams.Err, "%s\n\n", ui.Bold("Your code knows what. Lore knows why."))

	return nil
}

func demoPause(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(800 * time.Millisecond):
	}
}
