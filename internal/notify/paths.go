// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PathOpts holds injectable dependencies for path resolution (testability).
type PathOpts struct {
	// Executable returns the path of the current binary. Defaults to os.Executable.
	Executable func() (string, error)

	// LookPath searches for a binary in PATH. Defaults to exec.LookPath.
	LookPath func(file string) (string, error)

	// GitCommand runs a git command and returns its output. Defaults to exec.Command("git", ...).Output().
	GitCommand func(args ...string) (string, error)

	// Getwd returns the current working directory. Defaults to os.Getwd.
	Getwd func() (string, error)

	// Home returns the user's home directory. Defaults to os.UserHomeDir.
	Home func() (string, error)
}

func (o *PathOpts) defaults() {
	if o.Executable == nil {
		o.Executable = os.Executable
	}
	if o.LookPath == nil {
		o.LookPath = exec.LookPath
	}
	if o.GitCommand == nil {
		o.GitCommand = defaultGitCommand
	}
	if o.Getwd == nil {
		o.Getwd = os.Getwd
	}
	if o.Home == nil {
		o.Home = os.UserHomeDir
	}
}

// ResolveLoreBinary finds the absolute path to the lore binary.
// Priority: os.Executable() → PATH lookup → known install locations.
func ResolveLoreBinary(opts PathOpts) (string, error) {
	opts.defaults()

	// 1. The hook IS the lore binary — os.Executable returns its path.
	if selfPath, err := opts.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(selfPath); err == nil {
			return resolved, nil
		}
		return selfPath, nil
	}

	// 2. PATH lookup.
	if path, err := opts.LookPath("lore"); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs, nil
		}
		return path, nil
	}

	// 3. Known install locations.
	home, _ := opts.Home()
	if home != "" {
		knownPaths := []string{
			"/usr/local/bin/lore",
			filepath.Join(home, "go/bin/lore"),
			filepath.Join(home, ".local/bin/lore"),
		}
		for _, p := range knownPaths {
			if _, err := os.Stat(p); err == nil {
				return p, nil
			}
		}
	}

	return "", errLoreNotFound
}

// ResolveRepoRoot finds the absolute path to the git repository root.
// Uses git rev-parse --show-toplevel, falls back to cwd.
func ResolveRepoRoot(opts PathOpts) (string, error) {
	opts.defaults()

	out, err := opts.GitCommand("rev-parse", "--show-toplevel")
	if err == nil && out != "" {
		return out, nil
	}

	return opts.Getwd()
}

func defaultGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
