// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/spf13/cobra"
)

func newNoteCmd(_ *config.Config, _ domain.IOStreams) *cobra.Command {
	return &cobra.Command{
		Use:           "note",
		Short:         i18n.T().Cmd.NoteShort,
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("cmd: note not implemented")
		},
	}
}
