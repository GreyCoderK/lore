// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
)

// IsInteractiveTTY reports whether the session is running in an interactive TTY.
// Detection order (deterministic, not timer-based):
//  1. TERM=dumb               → false (Emacs shell-mode, IDEs)
//  2. LORE_LINE_MODE=1        → false (forced plain output)
//  3. stdin or stderr not TTY → false
//  4. Otherwise               → true
//
// Story 2.5 reuses this helper in detection.go without duplication.
func IsInteractiveTTY(streams domain.IOStreams) bool {
	return ui.IsTerminal(streams)
}

// NewRenderer returns the appropriate Renderer for the given streams.
func NewRenderer(streams domain.IOStreams) Renderer {
	if IsInteractiveTTY(streams) {
		return NewProgressRenderer(streams)
	}
	return NewLineRenderer(streams)
}
