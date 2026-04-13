// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Mode describes how Angela is operating in the current working directory.
// This abstraction lets draft/review/polish apply mode-aware defaults
// (e.g. auto-JSON output on standalone CI) without scattering filesystem
// probes across the codebase.
type Mode int

const (
	// ModeLoreNative is selected when a .lore/ directory is present. The
	// full feature set is available: scope/related conventions, LKS
	// signals, commit-linked docs, personas tuned for lore types.
	ModeLoreNative Mode = iota

	// ModeHybrid is selected when .git/ exists but .lore/ does not.
	// Git-based signals (commit history, diff size) are still available,
	// but lore-specific conventions (related:, scope:) are skipped.
	ModeHybrid

	// ModeStandalone is selected when neither .lore/ nor .git/ is found.
	// This is the canonical CI mode — the user is running Angela on a
	// pure-markdown repository (mkdocs, docusaurus, hugo, ...) that was
	// never initialized with lore. Defaults lean toward machine-readable
	// output and skip lore-specific suggestions.
	ModeStandalone
)

// String returns the canonical name for a Mode (matches the accepted
// values of angela.mode_detection in .lorerc).
func (m Mode) String() string {
	switch m {
	case ModeLoreNative:
		return "lore-native"
	case ModeHybrid:
		return "hybrid"
	case ModeStandalone:
		return "standalone"
	default:
		return "unknown"
	}
}

// parseModeString converts a user-facing config value to a Mode. Returns
// ModeStandalone and ok=false for unknown values (standalone is the
// safest fallback: it disables lore-specific assumptions).
func parseModeString(s string) (Mode, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "lore-native", "lore_native", "lorenative":
		return ModeLoreNative, true
	case "hybrid":
		return ModeHybrid, true
	case "standalone":
		return ModeStandalone, true
	default:
		return ModeStandalone, false
	}
}

// DetectMode returns the operating mode for a working directory. The
// resolution rules are:
//
//  1. If cfg.Angela.ModeDetection is explicit (anything but "auto" or
//     empty), parse it and return.
//  2. Otherwise, probe the filesystem: .lore/ wins, then .git/, then
//     standalone as the final fallback.
//
// DetectMode never returns an error. Unreadable directories are treated
// as "not a lore project" — the same outcome as a missing directory.
func DetectMode(workDir string, cfg *Config) Mode {
	// Explicit override from config (.lorerc or --angela.mode_detection=...).
	trimmed := strings.TrimSpace(cfg.Angela.ModeDetection)
	if trimmed != "" && strings.ToLower(trimmed) != "auto" {
		if m, ok := parseModeString(trimmed); ok {
			return m
		}
		// Unknown value in config → fall through to auto-detect rather
		// than silently returning standalone. A user-visible warning is
		// emitted by validate.go's unknown-field detection elsewhere.
	}

	// Auto mode: filesystem probing. Order matters — .lore/ implies
	// .git/ in most projects, so check .lore/ first.
	if dirExists(filepath.Join(workDir, ".lore")) {
		return ModeLoreNative
	}
	if dirExists(filepath.Join(workDir, ".git")) {
		return ModeHybrid
	}
	return ModeStandalone
}

// ResolveStateDir returns the absolute path to the state directory that
// should be used for differential caches, backups, and ignore rules.
// Resolution rules:
//
//  1. cfg.Angela.StateDir wins when set (joined with workDir if relative).
//  2. lore-native mode → <workDir>/.lore/angela/
//  3. hybrid / standalone modes → <workDir>/.angela-state/
//
// The returned path is not guaranteed to exist — callers that need to
// write into it should os.MkdirAll first.
func ResolveStateDir(workDir string, cfg *Config, mode Mode) string {
	if explicit := strings.TrimSpace(cfg.Angela.StateDir); explicit != "" {
		if filepath.IsAbs(explicit) {
			return explicit
		}
		return filepath.Join(workDir, explicit)
	}
	if mode == ModeLoreNative {
		return filepath.Join(workDir, ".lore", "angela")
	}
	return filepath.Join(workDir, ".angela-state")
}

// ApplyModeOverrides mutates cfg in-place to apply mode-specific defaults
// that the user has not explicitly overridden. Called once at command
// startup, AFTER config load and mode detection.
//
// The only automatic override in MVP v1:
//
//   - Standalone mode + non-TTY stdout + output.format still at "human"
//     → promote to "json". This makes `lore angela draft ./docs` produce
//     machine-readable output in a CI container without the user having
//     to remember the flag.
//
// Explicit user choices (the user wrote format: json OR human in their
// .lorerc) are respected — the auto-promotion only fires when the field
// is still at its zero-config default.
//
// The caller signals whether stdout is a TTY via stdoutIsTTY. Tests pass
// a controlled value; the production caller uses isNonInteractive().
func ApplyModeOverrides(cfg *Config, mode Mode, stdoutIsTTY bool) {
	if mode == ModeStandalone && !stdoutIsTTY {
		if cfg.Angela.Draft.Output.Format == "human" {
			cfg.Angela.Draft.Output.Format = "json"
		}
	}
	// Future mode-specific overrides land here. Keep per-mode branches
	// tight — anything that should be user-controllable belongs in the
	// config defaults, not here.
}

// IsNonInteractive reports whether stdout is NOT a terminal. Used by
// root command wiring to decide whether ApplyModeOverrides should
// promote draft output to json. Exposed for testing and for other
// packages that need the same signal.
func IsNonInteractive() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		// If we cannot stat stdout something is very wrong; treat as
		// non-interactive so CI pipelines do not accidentally get
		// colored terminal output.
		return true
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// dirExists returns true when path exists and is a real directory.
// Symlinks are NOT followed (os.Lstat) so a symlink named e.g. `.lore`
// does not incorrectly trigger lore-native mode.
func dirExists(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
