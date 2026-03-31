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
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/git"
	"github.com/greycoderk/lore/internal/i18n"
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
		Short:         i18n.T().Cmd.InitShort,
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
		ui.ActionableError(streams, i18n.T().Cmd.InitNotGitRepo, i18n.T().Cmd.InitNotGitRepoHint)
		return domain.ErrNotGitRepo
	}

	loreDir := filepath.Join(deps.workDir, ".lore")

	// AC-3: Already initialized
	if _, err := os.Stat(loreDir); err == nil {
		fmt.Fprintf(streams.Err, "%s\n", ui.Warning(i18n.T().Cmd.InitAlreadyInitialized))
		return nil
	}

	// AC-1: Happy path
	// 1. Create .lore/docs/
	docsDir := filepath.Join(loreDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("cmd: init create .lore/docs: %w", err)
	}
	ui.Verb(streams, "Created", i18n.T().Cmd.InitCreatedLore)

	// 2. Generate .lorerc
	lorercPath := filepath.Join(deps.workDir, ".lorerc")
	if err := storage.AtomicWrite(lorercPath, []byte(lorercContent)); err != nil {
		return fmt.Errorf("cmd: init write .lorerc: %w", err)
	}
	ui.Verb(streams, "Created", i18n.T().Cmd.InitCreatedLorerc)

	// 3. Generate .lorerc.local
	lorercLocalPath := filepath.Join(deps.workDir, ".lorerc.local")
	if err := fileutil.AtomicWrite(lorercLocalPath, []byte(lorercLocalContent), 0600); err != nil {
		return fmt.Errorf("cmd: init write .lorerc.local: %w", err)
	}
	ui.Verb(streams, "Created", i18n.T().Cmd.InitCreatedLorercLocal)

	// 4. Update .gitignore
	gitignorePath := filepath.Join(deps.workDir, ".gitignore")
	modified, err := ensureGitignore(gitignorePath, ".lorerc.local")
	if err != nil {
		return fmt.Errorf("cmd: init update .gitignore: %w", err)
	}
	if modified {
		ui.Verb(streams, "Modified", i18n.T().Cmd.InitModifiedGitignore)
	}

	// 5. Install post-commit hook
	result, err := deps.git.InstallHook("post-commit")
	if err != nil {
		fmt.Fprintf(streams.Err, "%s %s\n", ui.Warning("  "+i18n.T().Cmd.InitWarningPrefix), err.Error())
	} else if result.HooksPathWarn != "" {
		fmt.Fprintf(streams.Err, "%s %s\n", ui.Warning("  "+i18n.T().Cmd.InitWarningPrefix), fmt.Sprintf(i18n.T().Cmd.InitHooksPathWarn, result.HooksPathWarn))
	} else {
		ui.Verb(streams, "Installed", i18n.T().Cmd.InitInstalledHook)
	}

	// 6. Generate .lore/README.md discovery bridge (AC1 — Story 7f.3)
	if err := storage.GenerateReadmeBridge(loreDir); err != nil {
		fmt.Fprintf(streams.Err, "%s %s\n", ui.Warning("  "+i18n.T().Cmd.InitWarningPrefix), err.Error())
	}

	// Check if lore is in PATH and suggest adding it if not
	if _, lookErr := exec.LookPath("lore"); lookErr != nil {
		fmt.Fprintf(streams.Err, "\n%s %s\n", ui.Warning(i18n.T().Cmd.InitWarningPrefix), i18n.T().Cmd.InitNotInPathWarn)
		fmt.Fprintf(streams.Err, "  %s\n", i18n.T().Cmd.InitInstallHint)
		fmt.Fprintf(streams.Err, "  %s\n", i18n.T().Cmd.InitAddToPathHint)
		fmt.Fprintf(streams.Err, "    %s\n", i18n.T().Cmd.InitBashPathHint)
		fmt.Fprintf(streams.Err, "    %s\n", i18n.T().Cmd.InitZshPathHint)
		fmt.Fprintf(streams.Err, "    %s\n", i18n.T().Cmd.InitFishPathHint)
		fmt.Fprintf(streams.Err, "    %s\n", i18n.T().Cmd.InitPowerShellHint)
	}

	fmt.Fprintf(streams.Err, "\n%s\n", i18n.T().Cmd.InitTagline)

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

	fmt.Fprintf(streams.Err, "\n%s", i18n.T().Cmd.InitDemoPrompt)

	reader := bufio.NewReader(streams.In)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "y" {
		_, _ = fmt.Fprintln(streams.Err)
		if err := runDemo(ctx, cfg, streams); err != nil {
			fmt.Fprintf(streams.Err, "%s %v\n", ui.Warning(i18n.T().Cmd.InitDemoWarning), err)
		}
	}
}
