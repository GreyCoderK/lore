// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/fileutil"
)

const (
	loreStartMarker = "# LORE-START"
	loreEndMarker   = "# LORE-END"
	shebang         = "#!/bin/sh"
)

// hookBlock returns the lore hook block content read from the embedded script.
// CRLF is normalized to LF so the block is byte-deterministic regardless of
// how the script file was checked out (Windows runners with autocrlf=true
// would otherwise embed \r\n, which breaks idempotent reinstalls).
func hookBlock() string {
	s := readHookScript()
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.TrimRight(s, "\n")
}

// checkCoreHooksPath returns the configured core.hooksPath, or empty string if not set.
func checkCoreHooksPath(workDir string) (string, error) {
	cmd := exec.Command("git", "config", "--get", "core.hooksPath")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		// Exit code 1 means the key is not set — not an error
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

// installHook installs the lore hook block into hooksDir/hookType.
// workDir is the repository root (needed for checkCoreHooksPath).
// hooksDir is the resolved git hooks directory (from GitDir()/hooks — H3 fix).
func installHook(workDir, hooksDir, hookType string) (domain.InstallResult, error) {
	hooksPath, err := checkCoreHooksPath(workDir)
	if err != nil {
		return domain.InstallResult{}, fmt.Errorf("git: check hooks path: %w", err)
	}
	if hooksPath != "" {
		return domain.InstallResult{Installed: false, HooksPathWarn: hooksPath}, nil
	}

	hookPath := filepath.Join(hooksDir, hookType)

	// Check if already installed (idempotent: replace content between markers)
	if data, err := os.ReadFile(hookPath); err == nil {
		// Normalize CRLF to LF on read so replaceMarkerBlock's index arithmetic
		// (which advances past trailing "\n") stays correct even if the file
		// was previously written or edited with CRLF line endings.
		content := strings.ReplaceAll(string(data), "\r\n", "\n")
		if strings.Contains(content, loreStartMarker) || strings.Contains(content, loreEndMarker) {
			// Replace existing block between markers
			newContent, err := replaceMarkerBlock(content, hookBlock())
			if err != nil {
				return domain.InstallResult{}, fmt.Errorf("git: install hook %s: %w", hookType, err)
			}
			if err := atomicWriteHook(hookPath, []byte(newContent)); err != nil {
				return domain.InstallResult{}, fmt.Errorf("git: install hook %s: %w", hookType, err)
			}
			return domain.InstallResult{Installed: true}, nil
		}
		// Append to existing hook
		newContent := content + "\n" + hookBlock() + "\n"
		if err := atomicWriteHook(hookPath, []byte(newContent)); err != nil {
			return domain.InstallResult{}, fmt.Errorf("git: install hook %s: %w", hookType, err)
		}
		return domain.InstallResult{Installed: true}, nil
	}

	// Create new hook file
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return domain.InstallResult{}, fmt.Errorf("git: install hook %s: %w", hookType, err)
	}
	content := shebang + "\n" + hookBlock() + "\n"
	if err := atomicWriteHook(hookPath, []byte(content)); err != nil {
		return domain.InstallResult{}, fmt.Errorf("git: install hook %s: %w", hookType, err)
	}
	return domain.InstallResult{Installed: true}, nil
}

// replaceMarkerBlock replaces the content between LORE-START and LORE-END markers
// with the new block content.
// Returns the original content and nil error when both markers are absent (nothing to replace).
// Returns an error when markers are malformed (one present but not the other, or end before start).
func replaceMarkerBlock(content, newBlock string) (string, error) {
	hasStart := strings.Contains(content, loreStartMarker)
	hasEnd := strings.Contains(content, loreEndMarker)

	if !hasStart && !hasEnd {
		return content, nil
	}
	if hasStart && !hasEnd {
		return "", fmt.Errorf("git: hook corrupted: found LORE-START but missing LORE-END")
	}
	if !hasStart && hasEnd {
		return "", fmt.Errorf("git: hook corrupted: found LORE-END but missing LORE-START")
	}

	startIdx := strings.Index(content, loreStartMarker)
	endIdx := strings.Index(content, loreEndMarker)
	if endIdx < startIdx {
		return "", fmt.Errorf("git: hook corrupted: LORE-END appears before LORE-START")
	}

	endIdx += len(loreEndMarker)
	// Include trailing newline if present
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	return content[:startIdx] + newBlock + "\n" + content[endIdx:], nil
}

// uninstallHook removes the lore marker block from hooksDir/hookType.
// hooksDir is the resolved git hooks directory (from GitDir()/hooks — H3 fix).
func uninstallHook(hooksDir, hookType string) error {
	hookPath := filepath.Join(hooksDir, hookType)

	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("git: read hook %s: %w", hookType, err)
	}

	content := string(data)
	hasStart := strings.Contains(content, loreStartMarker)
	hasEnd := strings.Contains(content, loreEndMarker)
	if !hasStart && !hasEnd {
		return nil // nothing to remove
	}
	if hasStart && !hasEnd {
		return fmt.Errorf("git: hook corrupted: found LORE-START but missing LORE-END")
	}
	if !hasStart && hasEnd {
		return fmt.Errorf("git: hook corrupted: found LORE-END but missing LORE-START")
	}

	// Remove the LORE block
	startIdx := strings.Index(content, loreStartMarker)
	endIdx := strings.Index(content, loreEndMarker)
	if endIdx < startIdx {
		return fmt.Errorf("git: hook corrupted: LORE-END appears before LORE-START")
	}
	endIdx += len(loreEndMarker)
	// Remove trailing newline after end marker
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	// Remove leading newline before start marker
	if startIdx > 0 && content[startIdx-1] == '\n' {
		startIdx--
	}

	newContent := content[:startIdx] + content[endIdx:]
	// M3 fix: TrimRight preserves leading indentation; TrimSpace would strip it.
	newContent = strings.TrimRight(newContent, "\n\r")

	if newContent == "" || newContent == shebang {
		return os.Remove(hookPath)
	}

	return atomicWriteHook(hookPath, []byte(newContent+"\n"))
}

// hookExists reports whether the lore marker block is present in hooksDir/hookType.
// hooksDir is the resolved git hooks directory (from GitDir()/hooks — H3 fix).
func hookExists(hooksDir, hookType string) (bool, error) {
	hookPath := filepath.Join(hooksDir, hookType)

	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("git: read hook %s: %w", hookType, err)
	}

	hasStart := strings.Contains(string(data), loreStartMarker)
	hasEnd := strings.Contains(string(data), loreEndMarker)
	if hasStart != hasEnd {
		return false, fmt.Errorf("git: hook %s: corrupted (mismatched lore markers)", hookType)
	}
	return hasStart && hasEnd, nil
}

func atomicWriteHook(path string, data []byte) error {
	return fileutil.AtomicWrite(path, data, 0755)
}
