// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
	"github.com/greycoderk/lore/internal/storage"
)

// runStatusSet updates the frontmatter Status field of docPath in place.
// Triggered when the user passes --set-status to `lore angela polish`
// WITHOUT --synthesize.
//
// Fixes the long-standing "status doesn't evolve" issue: before this
// path, nothing in lore_cli ever mutated DocMeta.Status after doc
// creation. Re-polishing repeatedly is always safe — each run overwrites
// the field; no lifecycle enforcement gates the transition.
func runStatusSet(streams domain.IOStreams, docPath, newStatus string) error {
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return fmt.Errorf("angela: --set-status: read %s: %w", docPath, err)
	}
	meta, body, err := storage.UnmarshalPermissive(raw)
	if err != nil {
		return fmt.Errorf("angela: --set-status: parse %s: %w", docPath, err)
	}
	old := meta.Status
	if old == newStatus {
		_, _ = fmt.Fprintf(streams.Err, "→ %s\n  status déjà %q — aucun changement.\n", docPath, newStatus)
		return nil
	}
	meta.Status = newStatus
	out, err := storage.Marshal(meta, body)
	if err != nil {
		return fmt.Errorf("angela: --set-status: marshal %s: %w", docPath, err)
	}
	if err := fileutil.AtomicWrite(docPath, out, 0o644); err != nil {
		return fmt.Errorf("angela: --set-status: write %s: %w", docPath, err)
	}
	_, _ = fmt.Fprintf(streams.Err, "→ %s\n  status %q → %q\n", docPath, old, newStatus)
	return nil
}
