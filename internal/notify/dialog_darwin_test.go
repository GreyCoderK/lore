// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build darwin

package notify

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAppleScript_ContainsAbsolutePaths(t *testing.T) {
	data := DialogData{
		CommitHash:  "abc1234",
		CommitMsg:   "fix(auth): token refresh",
		DiffStat:    "+12 -3 auth/token.go",
		LorePath:    "/usr/local/bin/lore",
		RepoRoot:    "/Users/dev/project",
		PrefillType: "bugfix",
		PrefillWhat: "Fixed token refresh",
		PrefillWhy:  "Tokens expired",
	}

	script := buildAppleScript(data)

	assert.Contains(t, script, "/usr/local/bin/lore")
	assert.Contains(t, script, "/Users/dev/project")
	assert.Contains(t, script, "abc1234")
	assert.Contains(t, script, "Fixed token refresh")
	assert.Contains(t, script, "Tokens expired")
	assert.Contains(t, script, `"bugfix"`)
}

func TestNotifyOSDialog_Darwin_Detached(t *testing.T) {
	var captured struct {
		name string
		args []string
	}

	err := NotifyOSDialog(DialogData{
		CommitHash: "abc",
		LorePath:   "/bin/lore",
		RepoRoot:   "/tmp",
	}, DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			captured.name = name
			captured.args = args
			return nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "osascript", captured.name)
	require.Len(t, captured.args, 2)
	assert.Equal(t, "-e", captured.args[0])
	assert.Contains(t, captured.args[1], "choose from list") // script content
}

func TestBuildAppleScript_BranchAndScope(t *testing.T) {
	data := DialogData{
		CommitHash:  "abc1234",
		CommitMsg:   "feat(auth): add login",
		LorePath:    "/usr/local/bin/lore",
		RepoRoot:    "/tmp/project",
		PrefillType: "feature",
		Branch:      "feature/auth",
		Scope:       "auth",
	}

	script := buildAppleScript(data)

	assert.Contains(t, script, "Branch: feature/auth")
	assert.Contains(t, script, "Scope: auth")
}

func TestBuildAppleScript_NoBranchOrScope(t *testing.T) {
	data := DialogData{
		CommitHash:  "abc1234",
		CommitMsg:   "update readme",
		LorePath:    "/usr/local/bin/lore",
		RepoRoot:    "/tmp/project",
		PrefillType: "note",
	}

	script := buildAppleScript(data)

	assert.NotContains(t, script, "Branch:")
	assert.NotContains(t, script, "Scope:")
}

func TestBuildAppleScript_BranchOnly(t *testing.T) {
	data := DialogData{
		CommitHash:  "abc1234",
		CommitMsg:   "fix typo",
		LorePath:    "/usr/local/bin/lore",
		RepoRoot:    "/tmp/project",
		PrefillType: "bugfix",
		Branch:      "hotfix/typo",
	}

	script := buildAppleScript(data)

	assert.Contains(t, script, "Branch: hotfix/typo")
	assert.NotContains(t, script, "Scope:")
}

func TestDialogOpts_Defaults(t *testing.T) {
	opts := DialogOpts{}
	opts.defaults()

	require.NotNil(t, opts.StartCommand)
	require.NotNil(t, opts.LookPath)
}

func TestDialogOpts_Defaults_PreservesExisting(t *testing.T) {
	called := false
	opts := DialogOpts{
		StartCommand: func(name string, args []string, env []string) error {
			called = true
			return nil
		},
	}
	opts.defaults()

	// StartCommand should be preserved
	_ = opts.StartCommand("", nil, nil)
	assert.True(t, called, "original StartCommand should be preserved")
	// LookPath should be filled
	require.NotNil(t, opts.LookPath)
}

func TestBranchScopeContext(t *testing.T) {
	tests := []struct {
		name   string
		data   DialogData
		want   string
		absent string
	}{
		{"both", DialogData{Branch: "main", Scope: "auth"}, "Branch: main", ""},
		{"both_scope", DialogData{Branch: "main", Scope: "auth"}, "Scope: auth", ""},
		{"branch_only", DialogData{Branch: "develop"}, "Branch: develop", "Scope:"},
		{"scope_only", DialogData{Scope: "db"}, "Scope: db", "Branch:"},
		{"empty", DialogData{}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := branchScopeContext(tt.data)
			if tt.want == "" {
				assert.Empty(t, got)
			} else {
				assert.Contains(t, got, tt.want)
			}
			if tt.absent != "" {
				assert.NotContains(t, got, tt.absent)
			}
		})
	}
}
