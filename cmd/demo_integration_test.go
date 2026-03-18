// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/testutil"
)

func testEmptyConfig() *config.Config {
	return &config.Config{}
}

func TestIntegration_InitThenDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	dir := initRealGitRepo(t)
	runGit(t, dir, "commit", "--allow-empty", "-m", "initial")

	// chdir required: init and demo commands use os.Getwd() to find .lore/
	testutil.Chdir(t, dir)

	// Run lore init
	streams, _, errBuf := testStreams("\n")
	cfg := testEmptyConfig()
	initCmd := newInitCmd(cfg, streams)
	if err := initCmd.Execute(); err != nil {
		t.Fatalf("init: %v\n%s", err, errBuf.String())
	}

	// Verify .lore/docs/ exists
	docsDir := filepath.Join(dir, ".lore", "docs")
	if _, err := os.Stat(docsDir); os.IsNotExist(err) {
		t.Fatal("expected .lore/docs/ after init")
	}

	// Run lore demo
	streams2, _, errBuf2 := testStreams("\n")
	demoCmd := newDemoCmd(cfg, streams2)
	demoCmd.SetContext(t.Context())
	if err := demoCmd.Execute(); err != nil {
		t.Fatalf("demo: %v\n%s", err, errBuf2.String())
	}

	// Verify document created
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}

	found := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "decision-") && strings.HasSuffix(e.Name(), ".md") {
			found = true
			data, err := os.ReadFile(filepath.Join(docsDir, e.Name()))
			if err != nil {
				t.Fatalf("read doc: %v", err)
			}
			content := string(data)
			// AC-3: Verify front matter
			if !strings.Contains(content, "status: demo") {
				t.Error("expected 'status: demo' in front matter")
			}
			if !strings.Contains(content, "generated_by: lore-demo") {
				t.Error("expected 'generated_by: lore-demo'")
			}
			break
		}
	}
	if !found {
		t.Error("expected decision-*.md in .lore/docs/")
	}
}
