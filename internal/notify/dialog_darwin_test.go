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
