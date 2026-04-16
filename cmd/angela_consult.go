// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/storage"
	"github.com/spf13/cobra"
)

// newAngelaConsultCmd exposes a single persona's lens as an ad-hoc
// consultation. Useful to ask "what does Ouattara think of this doc right
// now" without re-running draft/polish/review.
//
// Offline (no AI). Runs the persona's DraftChecks on the current file
// content and reports suggestions. Never writes.
//
// Example usage:
//
//	lore angela consult api-designer feature-auth.md
//	lore angela consult storyteller adr-0042.md
//	lore angela consult                           # lists available personas
func newAngelaConsultCmd(cfg *config.Config, streams domain.IOStreams, flagPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consult <persona> [filename]",
		Short: "Consult a specific persona on an existing document (offline)",
		Long: `Invoke a single persona's draft-check lens on a document as it is right now.

Useful after a polish or manual edit when you want a specific expert's
opinion on the current content without re-running the full draft pipeline.
Pure offline — no AI call, no write.

Run without arguments to list available personas.`,
		Args:          cobra.MaximumNArgs(2),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return listPersonas(streams)
			}
			personaName := strings.TrimSpace(args[0])
			profile, ok := resolvePersonaByName(personaName)
			if !ok {
				_ = listPersonas(streams)
				return fmt.Errorf("angela: consult: unknown persona %q", personaName)
			}
			if len(args) < 2 {
				return fmt.Errorf("angela: consult: filename required — usage: lore angela consult <persona> <filename>")
			}

			filename := args[1]
			docsDir, standalone := resolveDocsDir(flagPath)
			if !standalone {
				if err := requireLoreDir(streams); err != nil {
					return err
				}
			}

			docPath := filename
			if !filepath.IsAbs(docPath) {
				if _, err := os.Stat(filename); err == nil {
					docPath = filename
				} else {
					docPath = filepath.Join(docsDir, filename)
				}
			}
			if _, err := os.Stat(docPath); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return fmt.Errorf("angela: consult: file not found: %s", docPath)
				}
				return fmt.Errorf("angela: consult: %w", err)
			}

			raw, err := os.ReadFile(docPath)
			if err != nil {
				return fmt.Errorf("angela: consult: read %s: %w", docPath, err)
			}
			_, body, err := storage.UnmarshalPermissive(raw)
			if err != nil {
				body = string(raw) // fall back to treating whole file as body
			}

			suggestions := angela.RunPersonaDraftChecks(body, []angela.PersonaProfile{profile})

			_, _ = fmt.Fprintf(streams.Err, "→ %s\n  "+i18n.T().Cmd.ConsultHeader+"\n\n",
				docPath, profile.Icon, profile.DisplayName, profile.Expertise)

			if len(suggestions) == 0 {
				_, _ = fmt.Fprintf(streams.Err, "  "+i18n.T().Cmd.ConsultNoSuggestion+"\n", profile.DisplayName)
				return nil
			}

			for _, s := range suggestions {
				_, _ = fmt.Fprintf(streams.Err, "  %-8s %-14s %s\n", s.Severity, s.Category, s.Message)
			}
			_, _ = fmt.Fprintf(streams.Err, "\n%d suggestion(s).\n", len(suggestions))
			_ = cfg // cfg reserved for future consult-time options
			return nil
		},
		ValidArgsFunction: consultValidArgs,
	}
	return cmd
}

// listPersonas prints the available personas to streams.Err, one per line,
// with a short description. Used when `consult` is called without args and
// for discoverability when the operator passes an unknown name.
func listPersonas(streams domain.IOStreams) error {
	reg := angela.GetRegistry()
	sort.Slice(reg, func(i, j int) bool { return reg[i].Name < reg[j].Name })
	_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ConsultListTitle)
	_, _ = fmt.Fprintln(streams.Err)
	for _, p := range reg {
		_, _ = fmt.Fprintf(streams.Err, "  %s %-20s %s\n", p.Icon, p.Name, p.DisplayName)
		_, _ = fmt.Fprintf(streams.Err, "  %-23s %s\n\n", "", p.Expertise)
	}
	_, _ = fmt.Fprintln(streams.Err, i18n.T().Cmd.ConsultListUsage)
	return nil
}

// resolvePersonaByName looks up a persona by its stable identifier
// (e.g. "api-designer"). Case-insensitive for operator convenience.
func resolvePersonaByName(name string) (angela.PersonaProfile, bool) {
	target := strings.ToLower(strings.TrimSpace(name))
	for _, p := range angela.GetRegistry() {
		if strings.EqualFold(p.Name, target) {
			return p, true
		}
	}
	return angela.PersonaProfile{}, false
}

// consultValidArgs powers shell completion: first arg → persona names,
// second arg → markdown files in the lore docs directory (falls back to
// regular filesystem completion when not inside a lore-native project).
func consultValidArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return personaNames(), cobra.ShellCompDirectiveNoFileComp
	}
	if len(args) == 1 {
		return docsFileCompletion(cmd, args, toComplete)
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// personaNames returns the list of persona identifiers for completion.
func personaNames() []string {
	reg := angela.GetRegistry()
	out := make([]string, 0, len(reg))
	for _, p := range reg {
		out = append(out, p.Name)
	}
	sort.Strings(out)
	return out
}
