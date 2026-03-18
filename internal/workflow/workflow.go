// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"

	"github.com/greycoderk/lore/internal/domain"
)

// Dispatch is the central router for the post-commit hook workflow.
// All contextual detection (non-TTY, doc-skip, merge, rebase, cherry-pick,
// amend) is delegated to Detect() inside HandleReactive (Story 2.5).
func Dispatch(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter) error {
	return HandleReactive(ctx, workDir, streams, gitAdapter)
}
