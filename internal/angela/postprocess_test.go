// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"strings"
	"testing"
)

func TestRestoreHeadingNumbers_SingleHeading(t *testing.T) {
	original := "## 4. Title\nSome content"
	polished := "## Title\nSome content"
	got := restoreHeadingNumbers(original, polished)
	if !strings.Contains(got, "## 4. Title") {
		t.Errorf("expected restored heading, got %q", got)
	}
}

func TestRestoreHeadingNumbers_NoNumber(t *testing.T) {
	original := "## Title\nSome content"
	polished := "## Title\nSome content"
	got := restoreHeadingNumbers(original, polished)
	if got != polished {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestRestoreHeadingNumbers_MultipleHeadings(t *testing.T) {
	original := "## 1. First\n## 2. Second\n## 3. Third"
	polished := "## First\n## Second\n## Third"
	got := restoreHeadingNumbers(original, polished)
	lines := strings.Split(got, "\n")
	expected := []string{"## 1. First", "## 2. Second", "## 3. Third"}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Errorf("line %d: expected %q, got %q", i, exp, lines[i])
		}
	}
}

func TestRestoreHeadingNumbers_H3Headings(t *testing.T) {
	original := "### 2.1 Details\nContent"
	polished := "### Details\nContent"
	got := restoreHeadingNumbers(original, polished)
	if !strings.Contains(got, "### 2.1 Details") {
		t.Errorf("expected restored ### heading, got %q", got)
	}
}

func TestStripNumber(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"4. Title", "Title"},
		{"4.2 Title", "Title"},
		{"Title", "Title"},
	}
	for _, tt := range tests {
		got := stripNumber(tt.input)
		if got != tt.want {
			t.Errorf("stripNumber(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseHeading(t *testing.T) {
	tests := []struct {
		input      string
		wantPrefix string
		wantText   string
	}{
		{"## Title", "## ", "Title"},
		{"### Title", "### ", "Title"},
		{"Not a heading", "", ""},
	}
	for _, tt := range tests {
		prefix, text := parseHeading(tt.input)
		if prefix != tt.wantPrefix || text != tt.wantText {
			t.Errorf("parseHeading(%q) = (%q, %q), want (%q, %q)",
				tt.input, prefix, text, tt.wantPrefix, tt.wantText)
		}
	}
}

func TestNormalizeCodeFenceLanguages_BareSQL(t *testing.T) {
	input := "```\nSELECT * FROM users;\n```"
	got := normalizeCodeFenceLanguages(input)
	if !strings.Contains(got, "```sql") {
		t.Errorf("expected sql fence, got %q", got)
	}
}

func TestNormalizeCodeFenceLanguages_BareGo(t *testing.T) {
	input := "```\nfunc main() {\n```"
	got := normalizeCodeFenceLanguages(input)
	if !strings.Contains(got, "```go") {
		t.Errorf("expected go fence, got %q", got)
	}
}

func TestNormalizeCodeFenceLanguages_AlreadyTagged(t *testing.T) {
	input := "```python\nprint('hello')\n```"
	got := normalizeCodeFenceLanguages(input)
	if got != input {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestNormalizeCodeFenceLanguages_EmptyFence(t *testing.T) {
	input := "```\n```"
	got := normalizeCodeFenceLanguages(input)
	if got != input {
		t.Errorf("expected unchanged for empty fence, got %q", got)
	}
}

func TestNormalizeMermaidIndent_NoIndent(t *testing.T) {
	input := "```mermaid\ngraph TD\nA-->B\n```"
	got := normalizeMermaidIndent(input)
	lines := strings.Split(got, "\n")
	// graph TD should be indented to 4 spaces
	if !strings.HasPrefix(lines[1], "    ") {
		t.Errorf("expected graph TD indented, got %q", lines[1])
	}
	// A-->B should be indented to 8 spaces
	if !strings.HasPrefix(lines[2], "        ") {
		t.Errorf("expected content indented to 8 spaces, got %q", lines[2])
	}
}

func TestNormalizeMermaidIndent_AlreadyIndented(t *testing.T) {
	input := "```mermaid\n    graph TD\n    A-->B\n```"
	got := normalizeMermaidIndent(input)
	lines := strings.Split(got, "\n")
	// graph TD is a diagram type, gets 4-space indent regardless
	if !strings.HasPrefix(lines[1], "    ") {
		t.Errorf("expected graph TD indented, got %q", lines[1])
	}
	// Already indented content should be kept as-is
	if lines[2] != "    A-->B" {
		t.Errorf("expected already-indented content unchanged, got %q", lines[2])
	}
}

func TestNormalizeMermaidIndent_NonMermaid(t *testing.T) {
	input := "Some text\nMore text"
	got := normalizeMermaidIndent(input)
	if got != input {
		t.Errorf("expected non-mermaid unchanged, got %q", got)
	}
}

func TestPostProcess_Integration(t *testing.T) {
	original := "## 1. Overview\nSome text\n```\nSELECT * FROM t;\n```\n```mermaid\ngraph TD\nA-->B\n```"
	polished := "## Overview\nImproved text\n```\nSELECT * FROM t;\n```\n```mermaid\ngraph TD\nA-->B\n```"

	got := PostProcess(original, polished)

	// Heading number restored
	if !strings.Contains(got, "## 1. Overview") {
		t.Error("expected heading number restored")
	}
	// SQL fence tagged
	if !strings.Contains(got, "```sql") {
		t.Error("expected sql code fence")
	}
	// Mermaid indented
	lines := strings.Split(got, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "A-->B" && !strings.HasPrefix(line, "    ") {
			t.Error("expected mermaid content indented")
		}
	}
}
