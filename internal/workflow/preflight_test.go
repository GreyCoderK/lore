// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/greycoderk/lore/internal/testutil"
)

func TestPreflightCheck_HappyPath(t *testing.T) {
	dir := testutil.SetupLoreDir(t)
	if err := PreflightCheck(dir); err != nil {
		t.Fatalf("PreflightCheck should pass on valid .lore dir: %v", err)
	}
}

func TestPreflightCheck_NoLoreDir(t *testing.T) {
	dir := t.TempDir()
	err := PreflightCheck(dir)
	if err == nil {
		t.Fatal("PreflightCheck should fail when .lore/ does not exist")
	}
	if !containsStr(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestPreflightCheck_LoreDirIsFile(t *testing.T) {
	dir := t.TempDir()
	// Create .lore as a file instead of directory
	if err := os.WriteFile(filepath.Join(dir, ".lore"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := PreflightCheck(dir)
	if err == nil {
		t.Fatal("PreflightCheck should fail when .lore is a file")
	}
	if !containsStr(err.Error(), "not a directory") {
		t.Errorf("error should mention 'not a directory', got: %v", err)
	}
}

func TestPreflightCheck_DocsNotWritable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping permission test on Windows (chmod doesn't prevent writes)")
	}
	if os.Getuid() == 0 {
		t.Skip("skipping permission test as root")
	}
	dir := testutil.SetupLoreDir(t)
	docsDir := filepath.Join(dir, ".lore", "docs")

	// Make docs dir read-only
	if err := os.Chmod(docsDir, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(docsDir, 0o755) })

	err := PreflightCheck(dir)
	if err == nil {
		t.Fatal("PreflightCheck should fail when docs/ is not writable")
	}
	if !containsStr(err.Error(), "preflight") {
		t.Errorf("error should mention 'preflight', got: %v", err)
	}
}

// Note: corrupt local templates are loaded lazily by the template engine
// (at Render time, not New time). PreflightCheck verifies that the engine
// initializes (embedded defaults parse OK), but cannot catch corrupt local
// templates without performing a render. This is acceptable — the error
// surfaces during generateAndWrite, and answers are saved as pending.

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && findStr(s, substr)
}

func findStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
