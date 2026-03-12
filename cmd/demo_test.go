package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
)

func TestRunDemo_NotInitialized(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("\n"),
	}
	cfg := &config.Config{}

	cmd := newDemoCmd(cfg, streams)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for not initialized")
	}

	errOutput := errBuf.String()
	if !strings.Contains(errOutput, "Lore not initialized") {
		t.Errorf("expected 'Lore not initialized' in error output, got %q", errOutput)
	}
}

func TestRunDemo_HappyPath(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	// Create .lore/docs/ to simulate initialized state
	docsDir := filepath.Join(dir, ".lore", "docs")
	os.MkdirAll(docsDir, 0755)

	var errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &bytes.Buffer{},
		Err: &errBuf,
		In:  strings.NewReader("\n"), // Enter for consent
	}
	cfg := &config.Config{}

	cmd := newDemoCmd(cfg, streams)
	cmd.SetContext(t.Context())
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("demo: %v", err)
	}

	output := errBuf.String()

	// Verify consent prompt
	if !strings.Contains(output, "Press Enter to begin") {
		t.Error("expected consent prompt")
	}

	// Verify document was created
	if !strings.Contains(output, "Created") {
		t.Error("expected 'Created' verb in output")
	}

	// Verify simulated lore show
	if !strings.Contains(output, "Simulating: lore show auth") {
		t.Error("expected simulated lore show")
	}

	// Verify tagline at the end
	if !strings.Contains(output, "Your code knows what. Lore knows why.") {
		t.Error("expected tagline")
	}

	// Verify file exists in .lore/docs/
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		t.Fatalf("read docs dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("expected demo document in .lore/docs/")
	}

	// Verify front matter has status: demo
	if len(entries) > 0 {
		data, err := os.ReadFile(filepath.Join(docsDir, entries[0].Name()))
		if err != nil {
			t.Fatalf("read demo doc: %v", err)
		}
		if !strings.Contains(string(data), "status: demo") {
			t.Error("expected 'status: demo' in front matter")
		}
	}
}
