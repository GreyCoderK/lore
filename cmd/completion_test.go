// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionCmd_ValidArgs(t *testing.T) {
	cmd := newCompletionCmd()
	if cmd == nil {
		t.Fatal("newCompletionCmd returned nil")
	}
	if len(cmd.ValidArgs) != 4 {
		t.Errorf("expected 4 valid args, got %d", len(cmd.ValidArgs))
	}
	expected := map[string]bool{"bash": true, "zsh": true, "fish": true, "powershell": true}
	for _, arg := range cmd.ValidArgs {
		if !expected[arg] {
			t.Errorf("unexpected valid arg: %s", arg)
		}
	}
}

func TestCompletionCmd_AllShells(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			root := &cobra.Command{Use: "lore"}
			root.AddCommand(newCompletionCmd())
			root.SetArgs([]string{"completion", shell})
			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s: %v", shell, err)
			}
		})
	}
}
