// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
)

// requireLoreDir checks that .lore/ exists in the current directory.
// On failure it prints an actionable error to streams.Err and returns
// a wrapped domain.ErrNotInitialized.
func requireLoreDir(streams domain.IOStreams) error {
	if _, err := os.Stat(".lore"); err != nil {
		if os.IsNotExist(err) {
			ui.ActionableError(streams, "Lore not initialized.", "lore init")
			return fmt.Errorf("cmd: %w", domain.ErrNotInitialized)
		}
		return fmt.Errorf("cmd: access .lore/: %w", err)
	}
	return nil
}
