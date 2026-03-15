package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/museigen/lore/internal/domain"
)

const (
	loreStartMarker = "# LORE-START"
	loreEndMarker   = "# LORE-END"
	shebang         = "#!/bin/sh"
)

// hookBlock returns the lore hook block content read from the embedded script.
func hookBlock() string {
	return strings.TrimRight(readHookScript(), "\n")
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
		content := string(data)
		if strings.Contains(content, loreStartMarker) {
			// Replace existing block between markers
			newContent := replaceMarkerBlock(content, hookBlock())
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
func replaceMarkerBlock(content, newBlock string) string {
	startIdx := strings.Index(content, loreStartMarker)
	endIdx := strings.Index(content, loreEndMarker)
	if startIdx == -1 || endIdx == -1 {
		return content
	}
	endIdx += len(loreEndMarker)
	// Include trailing newline if present
	if endIdx < len(content) && content[endIdx] == '\n' {
		endIdx++
	}
	return content[:startIdx] + newBlock + "\n" + content[endIdx:]
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
	if !strings.Contains(content, loreStartMarker) {
		return nil // nothing to remove
	}

	// Remove the LORE block
	startIdx := strings.Index(content, loreStartMarker)
	endIdx := strings.Index(content, loreEndMarker)
	if startIdx == -1 || endIdx == -1 {
		return nil
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

	return strings.Contains(string(data), loreStartMarker), nil
}

// CONSOLIDATE: Story 2-2 — unifier avec storage.atomicWrite si possible
func atomicWriteHook(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return fmt.Errorf("git: write hook tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // cleanup on failure
		return fmt.Errorf("git: rename hook: %w", err)
	}
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("git: chmod hook %s: %w", filepath.Base(path), err)
	}
	return nil
}
