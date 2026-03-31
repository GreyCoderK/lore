// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
)

// EngagementState holds lightweight flags persisted in .lore/state.json.
// The documented commit count comes from the store/filesystem, not here.
type EngagementState struct {
	StarPromptShown bool `json:"star_prompt_shown"`
}

// StatePath returns the path to .lore/state.json for the given work directory.
func StatePath(workDir string) string {
	return filepath.Join(workDir, domain.LoreDir, "state.json")
}

// LoadState reads .lore/state.json. Returns zero-value state on any error.
func LoadState(path string) EngagementState {
	data, err := os.ReadFile(path)
	if err != nil {
		return EngagementState{}
	}
	var state EngagementState
	if err := json.Unmarshal(data, &state); err != nil {
		return EngagementState{}
	}
	return state
}

// SaveState persists .lore/state.json atomically.
func SaveState(path string, state EngagementState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return fileutil.AtomicWrite(path, append(data, '\n'), 0o644)
}
