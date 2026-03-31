// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallHook_Fresh(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	result, err := a.InstallHook("post-commit")
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !result.Installed {
		t.Error("expected Installed = true")
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

	// Verify executable (skip on Windows where chmod has no effect).
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(hookPath)
		if info.Mode()&0111 == 0 {
			t.Error("hook file should be executable")
		}
	}
}

func TestInstallHook_ExistingHookPreserved(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	if err := os.WriteFile(hookPath, []byte(existingContent), 0755); err != nil {
		t.Fatalf("write existing hook: %v", err)
	}

	a := NewAdapter(dir)
	if _, err := a.InstallHook("post-commit"); err != nil {
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

	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("first InstallHook: %v", err)
	}
	if _, err := a.InstallHook("post-commit"); err != nil {
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

	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
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
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	existingContent := "#!/bin/sh\necho 'existing hook'\n"
	if err := os.WriteFile(hookPath, []byte(existingContent), 0755); err != nil {
		t.Fatalf("write existing hook: %v", err)
	}

	a := NewAdapter(dir)
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

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

func TestUninstallHook_NoMarkers(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"), 0755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	a := NewAdapter(dir)
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook no markers: %v", err)
	}

	// File should still exist unchanged
	data, _ := os.ReadFile(hookPath)
	if !strings.Contains(string(data), "echo hello") {
		t.Error("file should be unchanged")
	}
}

func TestUninstallHook_EmptyAfterRemoval(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	// Install then uninstall — file created by install has only shebang + lore block
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	// File should be removed since only shebang remains
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("hook file should be deleted when only shebang remains")
	}
}

func TestHookExists_True(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

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

func TestInstallHook_MultipleTypes(t *testing.T) {
	dir := initGitRepo(t)
	a := NewAdapter(dir)

	// Install post-commit
	_, err := a.InstallHook("post-commit")
	if err != nil {
		t.Fatalf("InstallHook post-commit: %v", err)
	}

	// Install pre-push
	_, err = a.InstallHook("pre-push")
	if err != nil {
		t.Fatalf("InstallHook pre-push: %v", err)
	}

	// Both should exist
	exists1, _ := a.HookExists("post-commit")
	exists2, _ := a.HookExists("pre-push")
	if !exists1 {
		t.Error("post-commit hook should exist")
	}
	if !exists2 {
		t.Error("pre-push hook should exist")
	}

	// Uninstall one, other should remain
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook post-commit: %v", err)
	}
	exists1, _ = a.HookExists("post-commit")
	exists2, _ = a.HookExists("pre-push")
	if exists1 {
		t.Error("post-commit should be uninstalled")
	}
	if !exists2 {
		t.Error("pre-push should still exist")
	}
}

// --- Integration tests guarded by testing.Short() ---

func TestIntegration_FullHookLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)
	a := NewAdapter(dir)

	// Create .lore/ to simulate initialized repo
	if err := os.MkdirAll(filepath.Join(dir, ".lore"), 0755); err != nil {
		t.Fatalf("create .lore: %v", err)
	}

	// Step 1: Install hook
	result, err := a.InstallHook("post-commit")
	if err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	if !result.Installed {
		t.Error("expected Installed = true")
	}

	// Step 2: Verify hook file contains exact marker block
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "#!/bin/sh") {
		t.Error("missing shebang")
	}
	if !strings.Contains(content, "# LORE-START") || !strings.Contains(content, "# LORE-END") {
		t.Errorf("hook content missing LORE marker block, got:\n%s", content)
	}
	if !strings.Contains(content, "exec lore _hook-post-commit") {
		t.Errorf("hook content missing exec command, got:\n%s", content)
	}

	// Step 3: Verify executable
	info, _ := os.Stat(hookPath)
	if info.Mode()&0111 == 0 {
		t.Error("hook should be executable")
	}

	// Step 4: Uninstall hook
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	// Step 5: Verify markers removed and file deleted (only shebang was left)
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("hook file should be deleted after uninstall (only shebang remained)")
	}

	// Step 6: Verify HookExists returns false
	exists, err := a.HookExists("post-commit")
	if err != nil {
		t.Fatalf("HookExists: %v", err)
	}
	if exists {
		t.Error("expected HookExists = false after uninstall")
	}
}

func TestIntegration_HuskyHookPreserved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initGitRepo(t)

	// Write a pre-existing Husky-style hook
	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	huskyContent := "#!/bin/sh\n. \"$(dirname \"$0\")/_/husky.sh\"\nnpx lint-staged\n"
	if err := os.WriteFile(hookPath, []byte(huskyContent), 0755); err != nil {
		t.Fatalf("write husky hook: %v", err)
	}

	a := NewAdapter(dir)

	// Install Lore hook alongside Husky
	if _, err := a.InstallHook("post-commit"); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}

	data, _ := os.ReadFile(hookPath)
	content := string(data)

	// Verify Husky content is intact
	if !strings.Contains(content, "husky.sh") {
		t.Error("Husky content should be preserved")
	}
	if !strings.Contains(content, "npx lint-staged") {
		t.Error("Husky lint-staged should be preserved")
	}
	// Verify Lore block appended
	if !strings.Contains(content, loreStartMarker) {
		t.Error("LORE block should be appended")
	}

	// Uninstall Lore — Husky should remain
	if err := a.UninstallHook("post-commit"); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("hook file should still exist: %v", err)
	}
	content = string(data)

	if strings.Contains(content, loreStartMarker) {
		t.Error("LORE block should be removed")
	}
	if !strings.Contains(content, "husky.sh") {
		t.Error("Husky content should still be preserved after uninstall")
	}
}

func TestInstallHook_CoreHooksPath(t *testing.T) {
	dir := initGitRepo(t)

	// Set core.hooksPath
	run(t, dir, "git", "config", "core.hooksPath", "/custom/hooks")

	a := NewAdapter(dir)
	result, err := a.InstallHook("post-commit")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Installed {
		t.Error("expected Installed = false when core.hooksPath is set")
	}
	if result.HooksPathWarn != "/custom/hooks" {
		t.Errorf("expected HooksPathWarn = /custom/hooks, got %q", result.HooksPathWarn)
	}

	// Verify no hook file was created
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("no hook file should be created when core.hooksPath is set")
	}
}

// --- replaceMarkerBlock unit tests ---

func TestReplaceMarkerBlock_BothMarkersPresent(t *testing.T) {
	content := "before\n# LORE-START\nold stuff\n# LORE-END\nafter\n"
	got, err := replaceMarkerBlock(content, "# LORE-START\nnew stuff\n# LORE-END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "new stuff") {
		t.Error("expected new block content")
	}
	if strings.Contains(got, "old stuff") {
		t.Error("old block content should be replaced")
	}
}

func TestReplaceMarkerBlock_BothMissing(t *testing.T) {
	content := "#!/bin/sh\necho hello\n"
	got, err := replaceMarkerBlock(content, "# LORE-START\nstuff\n# LORE-END")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != content {
		t.Error("expected original content returned unchanged when both markers missing")
	}
}

func TestReplaceMarkerBlock_StartOnly(t *testing.T) {
	content := "#!/bin/sh\n# LORE-START\nexec lore _hook-post-commit\n"
	_, err := replaceMarkerBlock(content, "new block")
	if err == nil {
		t.Fatal("expected error when LORE-START present but LORE-END missing")
	}
	if !strings.Contains(err.Error(), "found LORE-START but missing LORE-END") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReplaceMarkerBlock_EndOnly(t *testing.T) {
	content := "#!/bin/sh\nexec lore _hook-post-commit\n# LORE-END\n"
	_, err := replaceMarkerBlock(content, "new block")
	if err == nil {
		t.Fatal("expected error when LORE-END present but LORE-START missing")
	}
	if !strings.Contains(err.Error(), "found LORE-END but missing LORE-START") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestReplaceMarkerBlock_EndBeforeStart(t *testing.T) {
	content := "#!/bin/sh\n# LORE-END\nstuff\n# LORE-START\n"
	_, err := replaceMarkerBlock(content, "new block")
	if err == nil {
		t.Fatal("expected error when LORE-END appears before LORE-START")
	}
	if !strings.Contains(err.Error(), "LORE-END appears before LORE-START") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestInstallHook_CorruptedStartOnly(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	corruptedContent := "#!/bin/sh\n# LORE-START\nexec lore _hook-post-commit\n"
	if err := os.WriteFile(hookPath, []byte(corruptedContent), 0755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	a := NewAdapter(dir)
	_, err := a.InstallHook("post-commit")
	if err == nil {
		t.Fatal("expected error for corrupted hook with only LORE-START")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error should mention corruption: %v", err)
	}
}

func TestInstallHook_CorruptedEndOnly(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	corruptedContent := "#!/bin/sh\nexec lore _hook-post-commit\n# LORE-END\n"
	if err := os.WriteFile(hookPath, []byte(corruptedContent), 0755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	a := NewAdapter(dir)
	_, err := a.InstallHook("post-commit")
	if err == nil {
		t.Fatal("expected error for corrupted hook with only LORE-END")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error should mention corruption: %v", err)
	}
}

func TestHookExists_CorruptedSingleMarker(t *testing.T) {
	dir := initGitRepo(t)
	hookDir := filepath.Join(dir, ".git", "hooks")
	os.MkdirAll(hookDir, 0o755)
	hookPath := filepath.Join(hookDir, "post-commit")
	// Write hook with only end marker (corrupted)
	os.WriteFile(hookPath, []byte("#!/bin/sh\n# LORE-END\n"), 0o755)

	a := NewAdapter(dir)
	_, err := a.HookExists("post-commit")
	if err == nil {
		t.Error("expected error for corrupted hook with mismatched markers")
	}
}

func TestUninstallHook_CorruptedStartOnly(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	corruptedContent := "#!/bin/sh\n# LORE-START\nexec lore _hook-post-commit\n"
	if err := os.WriteFile(hookPath, []byte(corruptedContent), 0755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	a := NewAdapter(dir)
	err := a.UninstallHook("post-commit")
	if err == nil {
		t.Fatal("expected error for corrupted hook with only LORE-START")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error should mention corruption: %v", err)
	}
}

func TestUninstallHook_CorruptedEndOnly(t *testing.T) {
	dir := initGitRepo(t)

	hookDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatalf("create hooks dir: %v", err)
	}
	hookPath := filepath.Join(hookDir, "post-commit")
	corruptedContent := "#!/bin/sh\nexec lore _hook-post-commit\n# LORE-END\n"
	if err := os.WriteFile(hookPath, []byte(corruptedContent), 0755); err != nil {
		t.Fatalf("write hook: %v", err)
	}

	a := NewAdapter(dir)
	err := a.UninstallHook("post-commit")
	if err == nil {
		t.Fatal("expected error for corrupted hook with only LORE-END")
	}
	if !strings.Contains(err.Error(), "corrupted") {
		t.Errorf("error should mention corruption: %v", err)
	}
}
