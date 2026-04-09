// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"testing"
)

func setupVHSTestDirs(t *testing.T) (tapeDir, docsDir string) {
	t.Helper()
	base := t.TempDir()
	tapeDir = filepath.Join(base, "tapes")
	docsDir = filepath.Join(base, "docs")
	if err := os.MkdirAll(tapeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatal(err)
	}
	return tapeDir, docsDir
}

func TestAnalyzeVHSSignals_ParsesTapeCommands(t *testing.T) {
	tapeDir, docsDir := setupVHSTestDirs(t)

	tape := `# Demo tape
Output docs/assets/vhs/demo.gif
Set Shell "bash"
Type "lore angela review"
Enter
Sleep 3s
Type "lore list"
Enter
`
	_ = os.WriteFile(filepath.Join(tapeDir, "demo.tape"), []byte(tape), 0644)

	signals := AnalyzeVHSSignals(tapeDir, docsDir, nil)

	cmds, ok := signals.TapeCommands["demo.tape"]
	if !ok {
		t.Fatal("expected commands for demo.tape")
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d: %v", len(cmds), cmds)
	}
	if cmds[0] != "lore angela review" {
		t.Errorf("cmds[0] = %q, want %q", cmds[0], "lore angela review")
	}
	if cmds[1] != "lore list" {
		t.Errorf("cmds[1] = %q, want %q", cmds[1], "lore list")
	}
}

func TestAnalyzeVHSSignals_DetectsOrphanTape(t *testing.T) {
	tapeDir, docsDir := setupVHSTestDirs(t)

	tape := "Output docs/assets/vhs/orphan.gif\nType \"lore list\"\nEnter\n"
	_ = os.WriteFile(filepath.Join(tapeDir, "orphan.tape"), []byte(tape), 0644)

	// Doc that does NOT reference orphan.gif
	doc := "# Guide\n\nNo images here.\n"
	_ = os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte(doc), 0644)

	signals := AnalyzeVHSSignals(tapeDir, docsDir, nil)

	if len(signals.OrphanTapes) != 1 {
		t.Fatalf("expected 1 orphan tape, got %d", len(signals.OrphanTapes))
	}
	if signals.OrphanTapes[0] != "orphan.tape" {
		t.Errorf("OrphanTapes[0] = %q, want %q", signals.OrphanTapes[0], "orphan.tape")
	}
}

func TestAnalyzeVHSSignals_DetectsOrphanGIF(t *testing.T) {
	tapeDir, docsDir := setupVHSTestDirs(t)

	// Doc references a GIF that has no tape source
	doc := "# Guide\n\n![demo](../assets/vhs/missing.gif)\n"
	_ = os.WriteFile(filepath.Join(docsDir, "guide.md"), []byte(doc), 0644)

	signals := AnalyzeVHSSignals(tapeDir, docsDir, nil)

	if len(signals.OrphanGIFs) != 1 {
		t.Fatalf("expected 1 orphan GIF, got %d", len(signals.OrphanGIFs))
	}
	if signals.OrphanGIFs[0].GIFPath != "../assets/vhs/missing.gif" {
		t.Errorf("OrphanGIFs[0].GIFPath = %q", signals.OrphanGIFs[0].GIFPath)
	}
}

func TestAnalyzeVHSSignals_MatchedTapeAndDoc(t *testing.T) {
	tapeDir, docsDir := setupVHSTestDirs(t)

	tape := "Output docs/assets/vhs/review.gif\nType \"lore angela review\"\nEnter\n"
	_ = os.WriteFile(filepath.Join(tapeDir, "review.tape"), []byte(tape), 0644)

	doc := "# Review\n\n![review](../assets/vhs/review.gif)\n"
	_ = os.WriteFile(filepath.Join(docsDir, "review.md"), []byte(doc), 0644)

	signals := AnalyzeVHSSignals(tapeDir, docsDir, nil)

	if len(signals.OrphanTapes) != 0 {
		t.Errorf("expected 0 orphan tapes, got %d: %v", len(signals.OrphanTapes), signals.OrphanTapes)
	}
	if len(signals.OrphanGIFs) != 0 {
		t.Errorf("expected 0 orphan GIFs, got %d: %v", len(signals.OrphanGIFs), signals.OrphanGIFs)
	}
}

func TestAnalyzeVHSSignals_UnknownCommand(t *testing.T) {
	tapeDir, docsDir := setupVHSTestDirs(t)

	tape := "Output demo.gif\nType \"lore fakecmd --verbose\"\nEnter\n"
	_ = os.WriteFile(filepath.Join(tapeDir, "bad.tape"), []byte(tape), 0644)

	known := []string{"angela review", "angela draft", "list", "status"}
	signals := AnalyzeVHSSignals(tapeDir, docsDir, known)

	if len(signals.CommandMismatches) != 1 {
		t.Fatalf("expected 1 mismatch, got %d", len(signals.CommandMismatches))
	}
	if signals.CommandMismatches[0].Command != "lore fakecmd --verbose" {
		t.Errorf("mismatch command = %q", signals.CommandMismatches[0].Command)
	}
}

func TestAnalyzeVHSSignals_NoTapeDir(t *testing.T) {
	_, docsDir := setupVHSTestDirs(t)

	// Pass a nonexistent tape dir — should return empty signals, not error
	signals := AnalyzeVHSSignals("/tmp/nonexistent-vhs-dir", docsDir, nil)
	if len(signals.TapeCommands) != 0 {
		t.Error("expected empty signals for nonexistent tape dir")
	}
}

func TestExtractRootCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"lore angela review --quiet", "angela review"},
		{"lore list", "list"},
		{"lore angela draft --all --path ./docs", "angela draft"},
		{"lore", ""},
		{"echo hello", ""},
	}
	for _, tt := range tests {
		got := extractRootCommand(tt.cmd)
		if got != tt.want {
			t.Errorf("extractRootCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}
