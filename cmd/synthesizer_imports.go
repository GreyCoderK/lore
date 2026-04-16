// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

// Side-effect imports that register concrete Example Synthesizers with the
// framework's DefaultRegistry at process startup. Each listed package MUST
// call synthesizer.DefaultRegistry.Register in its init(), exactly once.
//
// Add a new line here for each additional synthesizer.
// The order is not significant - Registry.Enabled preserves
// the order declared in cfg.Angela.Synthesizers.Enabled at run time.
import (
	_ "github.com/greycoderk/lore/internal/angela/synthesizer/impls/apipostman"
)
