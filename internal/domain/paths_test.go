// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import (
	"path/filepath"
	"testing"
)

func TestDocsPath(t *testing.T) {
	tests := []struct {
		workDir string
		want    string
	}{
		{"/home/user/project", filepath.Join("/home/user/project", ".lore", "docs")},
		{".", filepath.Join(".", ".lore", "docs")},
		{"/tmp", filepath.Join("/tmp", ".lore", "docs")},
	}
	for _, tt := range tests {
		got := DocsPath(tt.workDir)
		if got != tt.want {
			t.Errorf("DocsPath(%q) = %q, want %q", tt.workDir, got, tt.want)
		}
	}
}

func TestConstants(t *testing.T) {
	if LoreDir != ".lore" {
		t.Errorf("LoreDir = %q, want %q", LoreDir, ".lore")
	}
	if DocsDir != "docs" {
		t.Errorf("DocsDir = %q, want %q", DocsDir, "docs")
	}
	if TemplatesDir != "templates" {
		t.Errorf("TemplatesDir = %q, want %q", TemplatesDir, "templates")
	}
}
