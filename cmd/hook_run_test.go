package cmd

import (
	"testing"
)

func TestHookPostCommitCmd_CallsWorkflowDispatch(t *testing.T) {
	// AC-6: The command is no longer a stub — it wires to workflow.Dispatch.
	// Structural check: RunE is non-nil and command metadata is correct.
	// End-to-end behavior is tested in internal/workflow/reactive_test.go.
	streams, _, _ := testStreams()
	cfg := testConfig()

	cmd := newHookPostCommitCmd(cfg, streams)
	if cmd.Use != "_hook-post-commit" {
		t.Errorf("Use = %q, want %q", cmd.Use, "_hook-post-commit")
	}
	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestHookPostCommitCmd_IsHidden(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	cmd := newHookPostCommitCmd(cfg, streams)
	if !cmd.Hidden {
		t.Error("_hook-post-commit command should be hidden")
	}
}

func TestHookPostCommitCmd_Registered(t *testing.T) {
	streams, _, _ := testStreams()
	cfg := testConfig()

	rootCmd := newRootCmd(cfg, streams)

	// Verify _hook-post-commit is registered
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "_hook-post-commit" {
			found = true
			if !c.Hidden {
				t.Error("_hook-post-commit should be hidden")
			}
			break
		}
	}
	if !found {
		t.Error("_hook-post-commit command should be registered in root")
	}
}
