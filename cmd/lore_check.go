// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
)

// requireLoreDir checks that .lore/ exists in the current directory.
// On failure it prints an actionable error to streams.Err and returns
// a wrapped domain.ErrNotInitialized.
func requireLoreDir(streams domain.IOStreams) error {
	if _, err := os.Stat(domain.LoreDir); err != nil {
		if os.IsNotExist(err) {
			ui.ActionableError(streams, i18n.T().Cmd.LoreCheckNotInit, i18n.T().Cmd.LoreCheckNotInitHint)
			return fmt.Errorf("cmd: %w", domain.ErrNotInitialized)
		}
		return fmt.Errorf("cmd: access %s/: %w", domain.LoreDir, err)
	}
	return nil
}
