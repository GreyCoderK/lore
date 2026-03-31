// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package engagement

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadState_Missing(t *testing.T) {
	state := LoadState("/nonexistent/state.json")
	assert.False(t, state.StarPromptShown)
}

func TestLoadState_Corrupt(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json"), 0o644))

	state := LoadState(path)
	assert.False(t, state.StarPromptShown)
}

func TestSaveLoadState(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "state.json")

	state := EngagementState{StarPromptShown: true}
	require.NoError(t, SaveState(path, state))

	loaded := LoadState(path)
	assert.True(t, loaded.StarPromptShown)
}
