package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/testutil"
)

func testConfig() *config.Config {
	return &config.Config{}
}

func TestHookCmd_HasSubcommands(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	hookCmd := newHookCmd(cfg, streams)

	cmds := hookCmd.Commands()
	names := make([]string, 0, len(cmds))
	for _, c := range cmds {
		names = append(names, c.Name())
	}

	joined := strings.Join(names, ",")
	if !strings.Contains(joined, "install") {
		t.Error("expected 'install' subcommand")
	}
	if !strings.Contains(joined, "uninstall") {
		t.Error("expected 'uninstall' subcommand")
	}
}

func TestIntegration_HookInstallCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initRealGitRepo(t)
	// chdir required: hook cmd uses os.Getwd() to find .git/
	testutil.Chdir(t, dir)

	streams, _, errBuf := testStreams()
	cfg := testConfig()

	cmd := newHookInstallCmd(cfg, streams)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook install: %v", err)
	}

	// Verify hook was actually installed
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("read hook: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# LORE-START") {
		t.Error("hook should contain LORE-START marker")
	}
	if !strings.Contains(content, "exec lore _hook-post-commit") {
		t.Error("hook should contain exec command")
	}

	// Verify output
	output := errBuf.String()
	if !strings.Contains(output, "Installed") {
		t.Errorf("output should contain 'Installed', got %q", output)
	}
}

func TestIntegration_HookInstallCmd_CoreHooksPath(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initRealGitRepo(t)
	runGit(t, dir, "config", "core.hooksPath", "/custom/hooks")
	// chdir required: hook cmd uses os.Getwd() to find .git/
	testutil.Chdir(t, dir)

	streams, _, errBuf := testStreams()
	cfg := testConfig()

	cmd := newHookInstallCmd(cfg, streams)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("hook install: %v", err)
	}

	// Verify no hook file was created in .git/hooks
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("no hook file should be created when core.hooksPath is set")
	}

	// Verify warning output
	output := errBuf.String()
	if !strings.Contains(output, "core.hooksPath") {
		t.Errorf("output should warn about core.hooksPath, got %q", output)
	}
}

func TestIntegration_HookUninstallCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir := initRealGitRepo(t)
	// chdir required: hook cmd uses os.Getwd() to find .git/
	testutil.Chdir(t, dir)

	streams, _, errBuf := testStreams()
	cfg := testConfig()

	// First install
	installCmd := newHookInstallCmd(cfg, streams)
	installCmd.SetArgs([]string{})
	if err := installCmd.Execute(); err != nil {
		t.Fatalf("hook install: %v", err)
	}

	// Then uninstall
	streams2, _, errBuf2 := testStreams()
	uninstallCmd := newHookUninstallCmd(cfg, streams2)
	uninstallCmd.SetArgs([]string{})
	if err := uninstallCmd.Execute(); err != nil {
		t.Fatalf("hook uninstall: %v", err)
	}

	// Verify hook file is gone (only had lore content)
	hookPath := filepath.Join(dir, ".git", "hooks", "post-commit")
	if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
		t.Error("hook file should be deleted after uninstall")
	}

	// Verify output
	_ = errBuf
	output := errBuf2.String()
	if !strings.Contains(output, "Removed") {
		t.Errorf("output should contain 'Removed', got %q", output)
	}
}
