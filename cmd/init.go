package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
	"github.com/museigen/lore/internal/git"
	"github.com/museigen/lore/internal/ui"
	"github.com/spf13/cobra"
)

const lorercContent = `# Lore configuration — shared with team (commit this file)
# See: https://github.com/museigen/lore/docs/configuration

ai:
  provider: ""       # anthropic, openai, ollama
  model: ""          # model name (e.g., claude-sonnet-4-20250514)

angela:
  mode: draft        # draft, review, full
  max_tokens: 2000

templates:
  dir: .lore/templates

hooks:
  post_commit: true

output:
  format: markdown
  dir: .lore/docs
`

const lorercLocalContent = `# Lore local configuration — personal overrides (DO NOT commit)
# This file is in .gitignore

ai:
  api_key: ""  # paste your API key here
`

type initDeps struct {
	git     domain.GitAdapter
	workDir string
}

func newInitCmd(cfg *config.Config, streams domain.IOStreams) *cobra.Command {
	var noDemo bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Set up Lore in this repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			workDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cmd: init getwd: %w", err)
			}

			deps := initDeps{
				git:     git.NewAdapter(workDir),
				workDir: workDir,
			}

			return runInit(ctx, deps, streams, noDemo)
		},
	}

	cmd.Flags().BoolVar(&noDemo, "no-demo", false, "Skip the demo prompt")

	return cmd
}

func runInit(_ context.Context, deps initDeps, streams domain.IOStreams, noDemo bool) error {
	// AC-2: Not a git repo
	if !deps.git.IsInsideWorkTree() {
		ui.ActionableError(streams, "Not a git repository.", "git init")
		return domain.ErrNotGitRepo
	}

	loreDir := filepath.Join(deps.workDir, ".lore")

	// AC-3: Already initialized
	if _, err := os.Stat(loreDir); err == nil {
		fmt.Fprintf(streams.Err, "%s\n", ui.Warning("Lore already initialized."))
		return nil
	}

	// AC-1: Happy path
	// 1. Create .lore/docs/
	docsDir := filepath.Join(loreDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("cmd: init create .lore/docs: %w", err)
	}
	ui.Verb(streams, "Created", ".lore/")

	// 2. Generate .lorerc
	lorercPath := filepath.Join(deps.workDir, ".lorerc")
	if err := atomicWrite(lorercPath, []byte(lorercContent)); err != nil {
		return fmt.Errorf("cmd: init write .lorerc: %w", err)
	}
	ui.Verb(streams, "Created", ".lorerc")

	// 3. Generate .lorerc.local
	lorercLocalPath := filepath.Join(deps.workDir, ".lorerc.local")
	if err := atomicWrite(lorercLocalPath, []byte(lorercLocalContent)); err != nil {
		return fmt.Errorf("cmd: init write .lorerc.local: %w", err)
	}
	ui.Verb(streams, "Created", ".lorerc.local")

	// 4. Update .gitignore
	gitignorePath := filepath.Join(deps.workDir, ".gitignore")
	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		return fmt.Errorf("cmd: init update .gitignore: %w", err)
	}
	if modified {
		ui.Verb(streams, "Modified", ".gitignore")
	}

	// 5. Install post-commit hook
	if err := deps.git.InstallHook("post-commit"); err != nil {
		fmt.Fprintf(streams.Err, "%s %s\n", ui.Warning("  Warning:"), err.Error())
	} else {
		ui.Verb(streams, "Installed", "post-commit hook")
	}

	fmt.Fprintf(streams.Err, "\nYour code knows what. Lore knows why.\n")

	// AC-4: Demo opt-in
	if !noDemo {
		promptDemo(streams)
	}

	return nil
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("init: write tmp %s: %w", tmp, err)
	}
	return os.Rename(tmp, path)
}

func ensureGitignore(path string, entry string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("init: read .gitignore: %w", err)
	}

	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			return false, nil // already present
		}
	}

	var newContent string
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		newContent = content + "\n" + entry + "\n"
	} else {
		newContent = content + entry + "\n"
	}

	if err := os.WriteFile(path, []byte(newContent), 0644); err != nil {
		return false, fmt.Errorf("init: write .gitignore: %w", err)
	}
	return true, nil
}

func promptDemo(streams domain.IOStreams) {
	if !ui.IsTerminal(streams) {
		return
	}

	fmt.Fprintf(streams.Err, "\nRun demo? (~45s) [o/N] ")

	reader := bufio.NewReader(streams.In)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "o" {
		fmt.Fprintf(streams.Err, "%s\n", ui.Dim("Demo not yet implemented."))
	}
}
