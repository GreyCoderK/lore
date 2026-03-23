// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/greycoderk/lore/internal/ui"
	"github.com/spf13/cobra"
)

const lorercContent = `# Lore configuration — shared with team (commit this file)
# See: https://github.com/greycoderk/lore/docs/configuration

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
		Use:           "init",
		Short:         "Set up Lore in this repository",
		SilenceUsage:  true,
		SilenceErrors: true,
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

			return runInit(ctx, cfg, deps, streams, noDemo)
		},
	}

	cmd.Flags().BoolVar(&noDemo, "no-demo", false, "Skip the demo prompt")

	return cmd
}

func runInit(ctx context.Context, cfg *config.Config, deps initDeps, streams domain.IOStreams, noDemo bool) error {
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
	if err := storage.AtomicWrite(lorercPath, []byte(lorercContent)); err != nil {
		return fmt.Errorf("cmd: init write .lorerc: %w", err)
	}
	ui.Verb(streams, "Created", ".lorerc")

	// 3. Generate .lorerc.local
	lorercLocalPath := filepath.Join(deps.workDir, ".lorerc.local")
	if err := storage.AtomicWrite(lorercLocalPath, []byte(lorercLocalContent)); err != nil {
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
	result, err := deps.git.InstallHook("post-commit")
	if err != nil {
		fmt.Fprintf(streams.Err, "%s %s\n", ui.Warning("  Warning:"), err.Error())
	} else if result.HooksPathWarn != "" {
		fmt.Fprintf(streams.Err, "%s core.hooksPath is set to %q — manual integration required\n", ui.Warning("  Warning:"), result.HooksPathWarn)
	} else {
		ui.Verb(streams, "Installed", "post-commit hook")
	}

	// Check if lore is in PATH and suggest adding it if not
	if _, lookErr := exec.LookPath("lore"); lookErr != nil {
		fmt.Fprintf(streams.Err, "\n%s lore is not in your PATH — the post-commit hook won't work until it is.\n", ui.Warning("Warning:"))
		fmt.Fprintf(streams.Err, "  Install: go install github.com/greycoderk/lore@latest\n")
		fmt.Fprintf(streams.Err, "  Or add it to your PATH:\n")
		fmt.Fprintf(streams.Err, "    bash:       echo 'export PATH=\"$PATH:/path/to/lore\"' >> ~/.bashrc\n")
		fmt.Fprintf(streams.Err, "    zsh:        echo 'export PATH=\"$PATH:/path/to/lore\"' >> ~/.zshrc\n")
		fmt.Fprintf(streams.Err, "    fish:       fish_add_path /path/to/lore\n")
		fmt.Fprintf(streams.Err, "    PowerShell: [Environment]::SetEnvironmentVariable('PATH', $env:PATH + ';C:\\path\\to\\lore', 'User')\n")
	}

	fmt.Fprintf(streams.Err, "\nYour code knows what. Lore knows why.\n")

	// AC-4: Demo opt-in
	if !noDemo {
		promptDemo(ctx, cfg, streams)
	}

	return nil
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

	if err := storage.AtomicWrite(path, []byte(newContent)); err != nil {
		return false, fmt.Errorf("init: write .gitignore: %w", err)
	}
	return true, nil
}

func promptDemo(ctx context.Context, cfg *config.Config, streams domain.IOStreams) {
	if !ui.IsTerminal(streams) {
		return
	}

	fmt.Fprintf(streams.Err, "\nRun demo? (~45s) [y/N] ")

	reader := bufio.NewReader(streams.In)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" {
		_, _ = fmt.Fprintln(streams.Err)
		if err := runDemo(ctx, cfg, streams); err != nil {
			fmt.Fprintf(streams.Err, "%s %v\n", ui.Warning("Demo:"), err)
		}
	}
}
