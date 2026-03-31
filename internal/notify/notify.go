// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/greycoderk/lore/internal/domain"
)

// NotificationMode controls notification behavior.
type NotificationMode = string

const (
	ModeAuto     NotificationMode = "auto"
	ModeTerminal NotificationMode = "terminal"
	ModeDialog   NotificationMode = "dialog"
	ModeNotify   NotificationMode = "notify"
	ModeSilent   NotificationMode = "silent"
)

// NotifyConfig holds user configuration for notification behavior.
type NotifyConfig struct {
	Mode         NotificationMode // auto, terminal, dialog, notify, silent
	DisabledEnvs []string         // environments to skip notification
}

// DefaultNotifyConfig returns sensible defaults.
func DefaultNotifyConfig() NotifyConfig {
	return NotifyConfig{
		Mode: ModeAuto,
	}
}

// NotifyOpts holds all dependencies for the main notification orchestrator.
type NotifyOpts struct {
	EnvOpts    EnvOpts
	PathOpts   PathOpts
	VSCodeOpts VSCodeOpts
	DialogOpts DialogOpts
	Config     NotifyConfig

	// I18nLabels populates i18n labels on DialogData.
	// When nil, English fallback labels are used inside the dialog builders.
	I18nLabels func(data *DialogData)
}

// NotifyNonTTY is the main entry point called by the hook when a non-TTY
// environment is detected. It orchestrates the full fallback chain:
//
//	VS Code terminal → OS dialog → OS notification → silent
//
// The pending record is already saved before this function is called.
// This function is best-effort: errors are swallowed, the hook always returns.
func NotifyNonTTY(hash string, env EnvironmentSource, commitMsg, diffStat string,
	prefillType, prefillWhat, prefillWhy string, opts NotifyOpts) {

	cfg := opts.Config
	if cfg.Mode == "" {
		cfg.Mode = ModeAuto
	}

	// Mode: silent — do nothing.
	if cfg.Mode == ModeSilent {
		return
	}

	// Check disabled environments.
	for _, disabled := range cfg.DisabledEnvs {
		if env == disabled {
			return
		}
	}

	// Resolve paths once — reused by all strategies.
	lorePath, loreErr := ResolveLoreBinary(opts.PathOpts)
	repoRoot, rootErr := ResolveRepoRoot(opts.PathOpts)

	if loreErr != nil || rootErr != nil {
		// Can't resolve paths → try simple notification as last resort.
		_ = NotifyOSSimple(commitMsg, opts.DialogOpts)
		return
	}

	// Acquire lock to prevent concurrent dialogs/terminals.
	lockPath := filepath.Join(repoRoot, domain.LoreDir, "notification.lock")
	if !acquireLock(lockPath) {
		return // Another notification is in progress.
	}
	defer releaseLock(lockPath)

	// Build dialog data (shared by dialog and simple).
	// I18n labels are populated from the active catalog.
	data := DialogData{
		CommitHash:  hash,
		CommitMsg:   commitMsg,
		DiffStat:    diffStat,
		LorePath:    lorePath,
		RepoRoot:    repoRoot,
		PrefillType: prefillType,
		PrefillWhat: prefillWhat,
		PrefillWhy:  prefillWhy,
	}
	if opts.I18nLabels != nil {
		opts.I18nLabels(&data)
	}

	switch cfg.Mode {
	case ModeTerminal:
		notifyTerminalOnly(hash, env, lorePath, repoRoot, opts)
		return
	case ModeDialog:
		notifyDialogOnly(data, commitMsg, opts)
		return
	case ModeNotify:
		_ = NotifyOSSimple(commitMsg, opts.DialogOpts)
		return
	}

	// Mode: auto — full fallback chain.

	// 1. Remote environments → skip terminal IDE.
	if IsRemoteEnvironment(opts.EnvOpts) {
		notifyDialogOnly(data, commitMsg, opts)
		return
	}

	// 2. VS Code family → try terminal IDE (best-effort).
	if IsVSCodeFamily(env) {
		ipcSocket := DetectIPCSocket(env, opts.EnvOpts)
		if ipcSocket != "" {
			if err := NotifyVSCodeTerminal(hash, env, ipcSocket, lorePath, repoRoot, opts.VSCodeOpts); err == nil {
				return
			}
		}
		// Fallback to dialog OS.
	}

	// 3. Dialog OS → fallback to simple notification.
	notifyDialogOnly(data, commitMsg, opts)
}

func notifyTerminalOnly(hash string, env EnvironmentSource, lorePath, repoRoot string, opts NotifyOpts) {
	if IsVSCodeFamily(env) {
		ipcSocket := DetectIPCSocket(env, opts.EnvOpts)
		if ipcSocket != "" {
			if err := NotifyVSCodeTerminal(hash, env, ipcSocket, lorePath, repoRoot, opts.VSCodeOpts); err == nil {
				return
			}
		}
	}
	// Terminal mode requested but not available → silent.
}

func notifyDialogOnly(data DialogData, commitMsg string, opts NotifyOpts) {
	if err := NotifyOSDialog(data, opts.DialogOpts); err == nil {
		return
	}
	// Dialog failed → try simple notification.
	_ = NotifyOSSimple(commitMsg, opts.DialogOpts)
}

// acquireLock creates an exclusive lock file using atomic link.
// Returns true if lock acquired, false if already locked.
// Note: there is a small TOCTOU window between os.Remove (stale lock cleanup)
// and os.Link (re-acquire). This is acceptable for best-effort notification —
// the worst case is a missed notification, not data corruption.
func acquireLock(path string) bool {
	dir := filepath.Dir(path)
	_ = os.MkdirAll(dir, 0o755)

	// Write PID to a temp file, then hard-link atomically.
	tmp := path + ".tmp." + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(tmp, []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		return false
	}
	defer os.Remove(tmp)

	if err := os.Link(tmp, path); err != nil {
		// Lock exists — check if the process is still alive.
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return false
		}
		pid, parseErr := strconv.Atoi(string(data))
		if parseErr != nil {
			// Corrupt lock → remove and retry.
			os.Remove(path)
			return os.Link(tmp, path) == nil
		}
		// Check if PID is still running.
		if !isProcessAlive(pid) {
			// Process dead → stale lock.
			os.Remove(path)
			return os.Link(tmp, path) == nil
		}
		return false // Process alive → lock held.
	}
	return true
}

// releaseLock removes the lock file.
func releaseLock(path string) {
	os.Remove(path)
}
