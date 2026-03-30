// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveLoreBinary_OsExecutable(t *testing.T) {
	got, err := ResolveLoreBinary(PathOpts{
		Executable: func() (string, error) { return "/usr/local/bin/lore", nil },
	})
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/lore", got)
}

func TestResolveLoreBinary_LookPath(t *testing.T) {
	got, err := ResolveLoreBinary(PathOpts{
		Executable: func() (string, error) { return "", errors.New("no exec") },
		LookPath:   func(string) (string, error) { return "/usr/bin/lore", nil },
	})
	require.NoError(t, err)
	assert.Contains(t, got, "lore")
}

func TestResolveLoreBinary_KnownPaths(t *testing.T) {
	tmp := t.TempDir()
	lorePath := filepath.Join(tmp, "go/bin/lore")
	require.NoError(t, os.MkdirAll(filepath.Dir(lorePath), 0o755))
	require.NoError(t, os.WriteFile(lorePath, []byte("#!/bin/sh"), 0o755))

	got, err := ResolveLoreBinary(PathOpts{
		Executable: func() (string, error) { return "", errors.New("no exec") },
		LookPath:   func(string) (string, error) { return "", errors.New("not in path") },
		Home:       func() (string, error) { return tmp, nil },
	})
	require.NoError(t, err)
	assert.Equal(t, lorePath, got)
}

func TestResolveLoreBinary_NotFound(t *testing.T) {
	_, err := ResolveLoreBinary(PathOpts{
		Executable: func() (string, error) { return "", errors.New("no exec") },
		LookPath:   func(string) (string, error) { return "", errors.New("not in path") },
		Home:       func() (string, error) { return "/nonexistent", nil },
	})
	assert.ErrorIs(t, err, errLoreNotFound)
}

func TestResolveRepoRoot_GitCommand(t *testing.T) {
	got, err := ResolveRepoRoot(PathOpts{
		GitCommand: func(args ...string) (string, error) {
			return "/Users/dev/project", nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "/Users/dev/project", got)
}

func TestResolveRepoRoot_FallbackCwd(t *testing.T) {
	got, err := ResolveRepoRoot(PathOpts{
		GitCommand: func(args ...string) (string, error) {
			return "", errors.New("not a git repo")
		},
		Getwd: func() (string, error) { return "/tmp/fallback", nil },
	})
	require.NoError(t, err)
	assert.Equal(t, "/tmp/fallback", got)
}

func TestResolveLoreBinary_Symlink(t *testing.T) {
	tmp := t.TempDir()
	realPath := filepath.Join(tmp, "lore-real")
	linkPath := filepath.Join(tmp, "lore-link")

	require.NoError(t, os.WriteFile(realPath, []byte("#!/bin/sh"), 0o755))
	require.NoError(t, os.Symlink(realPath, linkPath))

	got, err := ResolveLoreBinary(PathOpts{
		Executable: func() (string, error) { return linkPath, nil },
	})
	require.NoError(t, err)
	// On macOS, EvalSymlinks resolves /var → /private/var.
	// Compare the base name to avoid platform-specific path differences.
	assert.Equal(t, filepath.Base(realPath), filepath.Base(got))
	assert.NotEqual(t, linkPath, got) // Must not return the symlink itself.
}
