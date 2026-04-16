// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/spf13/cobra"
)

// personaFlagCompletion powers tab-completion for --persona and
// --manual-personas. Returns every registered persona identifier
// (storyteller, tech-writer, api-designer, ...). Suppresses file
// completion since these flags take enum-like values.
func personaFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return personaNames(), cobra.ShellCompDirectiveNoFileComp
}

// synthesizerFlagCompletion powers tab-completion for --synthesizers.
// Pulls the list from the process-wide registry so the completion stays
// accurate even as new synthesizer packages register via init().
func synthesizerFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return synthesizer.DefaultRegistry.Names(), cobra.ShellCompDirectiveNoFileComp
}

// statusFlagCompletion powers tab-completion for --set-status. The
// returned list matches the conventions documented in domain.DocStatus
// constants but isn't constrained to them — operators can still pass a
// custom value if their workflow demands.
func statusFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{"draft", "reviewed", "published", "archived"}, cobra.ShellCompDirectiveNoFileComp
}

// docTypeFlagCompletion powers tab-completion for --type on commands that
// filter/select by document type (list, show, pending resolve, new).
// Sourced from domain.DocTypeNames so the list stays in sync with the
// single source of truth.
func docTypeFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return domain.DocTypeNames(), cobra.ShellCompDirectiveNoFileComp
}

// gitRefFlagCompletion powers tab-completion for flags that accept a git
// commit ref (--commit, --from, --to). Returns recent short SHAs plus
// tags; shells out to `git` once per TAB press. Silently returns nil
// outside a git work tree so completion falls through to the default.
func gitRefFlagCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	refs := collectGitRefs()
	if len(refs) == 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return refs, cobra.ShellCompDirectiveNoFileComp
}

// collectGitRefs merges recent commit short SHAs and local tags into a
// single deduped slice capped at 40 entries — enough for a quick pick
// without flooding the shell completion menu.
func collectGitRefs() []string {
	seen := make(map[string]struct{})
	var out []string

	// Recent commits (short SHAs, last 20).
	if data, err := exec.Command("git", "log", "--format=%h", "-n", "20").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			sha := strings.TrimSpace(line)
			if sha == "" {
				continue
			}
			if _, dup := seen[sha]; dup {
				continue
			}
			seen[sha] = struct{}{}
			out = append(out, sha)
		}
	}
	// Local tags.
	if data, err := exec.Command("git", "tag", "--sort=-creatordate").Output(); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			tag := strings.TrimSpace(line)
			if tag == "" {
				continue
			}
			if _, dup := seen[tag]; dup {
				continue
			}
			seen[tag] = struct{}{}
			out = append(out, tag)
			if len(out) >= 40 {
				break
			}
		}
	}
	return out
}

// docsFileCompletion lists markdown files in the lore docs directory so
// operators get tab-completion on the positional filename argument of
// draft/polish/consult — even when they run the command from outside
// `.lore/docs/`. Falls back to the cobra default (whole filesystem) when
// the docs dir cannot be resolved.
//
// Resolution order:
//  1. <cwd>/.lore/docs/ when a .lore/ directory exists (lore-native mode)
//  2. cwd itself when --path or $PWD is already inside a docs tree
//     (standalone mode)
func docsFileCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	dir := resolveDocsDirForCompletion(cmd)
	if dir == "" {
		return nil, cobra.ShellCompDirectiveDefault
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveDefault
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if toComplete != "" && !strings.HasPrefix(name, toComplete) {
			continue
		}
		out = append(out, name)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

// resolveDocsDirForCompletion is a lightweight mirror of resolveDocsDir
// used exclusively by the shell-completion path. It reads the optional
// --path flag (if the command has one) and falls back to .lore/docs/
// relative to the current working directory.
func resolveDocsDirForCompletion(cmd *cobra.Command) string {
	if cmd != nil {
		if f := cmd.Flag("path"); f != nil && f.Value.String() != "" {
			return f.Value.String()
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(cwd, domain.LoreDir, domain.DocsDir)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}
