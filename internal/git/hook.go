package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	loreStartMarker = "# LORE-START"
	loreEndMarker   = "# LORE-END"
	shebang         = "#!/bin/sh"
)

// hookBlock returns the lore hook block content read from the embedded script.
func hookBlock() string {
	return strings.TrimSpace(readHookScript())
}

// hooksDir returns the git hooks directory, respecting core.hooksPath.
func hooksDir(workDir string) (string, bool, error) {
	cmd := exec.Command("git", "config", "core.hooksPath")
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err == nil {
		customPath := strings.TrimSpace(string(out))
		if customPath != "" {
			return customPath, true, nil
		}
	}
	// default: .git/hooks
	gitDir := filepath.Join(workDir, ".git", "hooks")
	return gitDir, false, nil
}

func installHook(workDir, hookType string) error {
	dir, custom, err := hooksDir(workDir)
	if err != nil {
		return fmt.Errorf("git: hooks dir: %w", err)
	}
	if custom {
		return fmt.Errorf("git: install hook: core.hooksPath is set — manual integration required")
	}

	hookPath := filepath.Join(dir, hookType)

	// Check if already installed
	if data, err := os.ReadFile(hookPath); err == nil {
		content := string(data)
		if strings.Contains(content, loreStartMarker) {
			return nil // idempotent
		}
		// Append to existing hook
		newContent := content + "\n" + hookBlock() + "\n"
		return atomicWriteHook(hookPath, []byte(newContent))
	}

	// Create new hook file
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("git: create hooks dir: %w", err)
	}
	content := shebang + "\n" + hookBlock() + "\n"
	return atomicWriteHook(hookPath, []byte(content))
}

func uninstallHook(workDir, hookType string) error {
	dir, _, err := hooksDir(workDir)
	if err != nil {
		return fmt.Errorf("git: hooks dir: %w", err)
	}

	hookPath := filepath.Join(dir, hookType)
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
	newContent = strings.TrimSpace(newContent)

	if newContent == "" || newContent == shebang {
		return os.Remove(hookPath)
	}

	return atomicWriteHook(hookPath, []byte(newContent+"\n"))
}

func hookExists(workDir, hookType string) (bool, error) {
	dir, _, err := hooksDir(workDir)
	if err != nil {
		return false, fmt.Errorf("git: hooks dir: %w", err)
	}

	hookPath := filepath.Join(dir, hookType)
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("git: read hook %s: %w", hookType, err)
	}

	return strings.Contains(string(data), loreStartMarker), nil
}

func atomicWriteHook(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0755); err != nil {
		return fmt.Errorf("git: write hook tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("git: rename hook: %w", err)
	}
	return nil
}
