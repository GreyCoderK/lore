// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/notify"
	"github.com/greycoderk/lore/internal/workflow/decision"
)

// Dispatch is the central router for the post-commit hook workflow.
// engine and store may be nil (backward compat — graceful degradation).
func Dispatch(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, engine *decision.Engine, store domain.LoreStore) error {
	return DispatchWithNotifyConfig(ctx, workDir, streams, gitAdapter, engine, store, nil)
}

// DispatchWithNotifyConfig is Dispatch with explicit notification configuration (ADR-023).
// notifyCfg may be nil (defaults to DefaultNotifyConfig).
func DispatchWithNotifyConfig(ctx context.Context, workDir string, streams domain.IOStreams, gitAdapter domain.GitAdapter, engine *decision.Engine, store domain.LoreStore, notifyCfg *notify.NotifyConfig) error {
	opts := DetectOpts{Store: store, NotifyConfig: notifyCfg}
	if engine != nil {
		opts.Engine = engine
	}
	return handleReactiveWithOpts(ctx, workDir, streams, gitAdapter, opts, store)
}
