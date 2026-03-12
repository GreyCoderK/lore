package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallHook_Fresh(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	if err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "#!/bin/sh") {
		t.Error("expected shebang in new hook file")
	}
	if !strings.Contains(content, loreStartMarker) {
		t.Error("expected LORE-START marker")
	}
	if !strings.Contains(content, loreEndMarker) {
		t.Error("expected LORE-END marker")
	}
	if !strings.Contains(content, "exec lore _hook-post-commit") {
		t.Error("expected hook command")
	}

	// Verify executable
	info, _ := os.Stat(hookPath)
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable")
	}
}

func TestInstallHook_ExistingHookPreserved(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hookDir, 0755)
	hookPath := filepath.Join(hookDir, "post-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	os.WriteFile(hookPath, []byte(existingContent), 0755)

	a := NewAdapter(dir)
	if err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	data, _ := os.ReadFile(hookPath)
	content := string(data)

	if !strings.Contains(content, "echo 'existing hook'") {
		t.Error("existing hook content should be preserved")
	}
	if !strings.Contains(content, loreStartMarker) {
		t.Error("LORE block should be appended")
	}
}

func TestInstallHook_Idempotent(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	if err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("first InstallHook: %v", err)
	}
	if err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("second InstallHook: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	data, _ := os.ReadFile(hookPath)
	content := string(data)

	// Should only have ONE lore block
	count := strings.Count(content, loreStartMarker)
	if count != 1 {
		t.Errorf("expected 1 LORE-START marker, got %d", count)
	}
}

func TestUninstallHook_Clean(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	a.InstallHook("post-commit")
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("hook file should be deleted when only lore content remains")
	}
}

func TestUninstallHook_PreservesExistingContent(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hookDir, 0755)
	hookPath := filepath.Join(hookDir, "post-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	os.WriteFile(hookPath, []byte(existingContent), 0755)

	a := NewAdapter(dir)
	a.InstallHook("post-commit")
	a.UninstallHook("post-commit")

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("hook file should still exist: %v", err)
	}

	content := string(data)
	if strings.Contains(content, loreStartMarker) {
		t.Error("LORE block should be removed")
	}
	if !strings.Contains(content, "echo 'existing hook'") {
		t.Error("existing content should be preserved")
	}
}

func TestUninstallHook_NoFile(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	// Should not error when no hook file exists
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook on missing file: %v", err)
	}
}

func TestHookExists_True(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	a.InstallHook("post-commit")

	exists, err := a.HookExists("post-commit")
	if err != nil {
		t.Fatalf("HookExists: %v", err)
	}
	if !exists {
		t.Error("expected HookExists = true after install")
	}
}

func TestHookExists_False(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	exists, err := a.HookExists("post-commit")
	if err != nil {
		t.Fatalf("HookExists: %v", err)
	}
	if exists {
		t.Error("expected HookExists = false before install")
	}
}

func TestInstallHook_CoreHooksPath(t *testing.T) {
	dir := initGitRepo(t)

	// Set core.hooksPath
	run(t, dir, "git", "config", "core.hooksPath", "/custom/hooks")

	a := NewAdapter(dir)
	err := a.InstallHook("post-commit")
	if err == nil {
		t.Error("expected error when core.hooksPath is set")
	}
	if !strings.Contains(err.Error(), "core.hooksPath") {
		t.Errorf("error should mention core.hooksPath, got: %v", err)
	}
}
