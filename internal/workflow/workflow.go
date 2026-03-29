// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// Dispatch is the central router for the post-commit hook workflow.
// engine and store may be nil (backward compat — graceful degradation).
func Dispatch(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, engine *decision.Engine, store domain.LoreStore) error {
	return HandleReactiveWithEngine(ctx, workDir, streams, gitAdapter, engine, store)
}
