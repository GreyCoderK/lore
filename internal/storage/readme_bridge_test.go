// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/greycoderk/lore/internal/i18n"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateReadmeBridge_CreatesFile(t *testing.T) {
	i18n.Init("en")
	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	require.NoError(t, os.MkdirAll(loreDir, 0o755))

	err := GenerateReadmeBridge(loreDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(loreDir, "README.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "About .lore/")
	assert.Contains(t, content, "Lore")
	assert.Contains(t, content, "github.com/greycoderk/lore")
}

func TestGenerateReadmeBridge_FR(t *testing.T) {
	i18n.Init("fr")
	defer i18n.Init("en")

	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	require.NoError(t, os.MkdirAll(loreDir, 0o755))

	err := GenerateReadmeBridge(loreDir)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(loreDir, "README.md"))
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "À propos de .lore/")
	assert.Contains(t, content, "l'or")
}

func TestGenerateReadmeBridge_NoOverwrite(t *testing.T) {
	i18n.Init("en")
	dir := t.TempDir()
	loreDir := filepath.Join(dir, ".lore")
	require.NoError(t, os.MkdirAll(loreDir, 0o755))

	// Write a custom README first.
	custom := filepath.Join(loreDir, "README.md")
	require.NoError(t, os.WriteFile(custom, []byte("custom content"), 0o644))

	err := GenerateReadmeBridge(loreDir)
	require.NoError(t, err)

	data, err := os.ReadFile(custom)
	require.NoError(t, err)
	assert.Equal(t, "custom content", string(data), "existing file must not be overwritten")
}
