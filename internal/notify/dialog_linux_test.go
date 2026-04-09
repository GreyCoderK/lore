// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build linux

package notify

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// detectLinuxDialogTool
// ---------------------------------------------------------------------------

func TestDetectLinuxDialogTool_NoneFound(t *testing.T) {
	lookPath := func(file string) (string, error) {
		return "", fmt.Errorf("not found: %s", file)
	}
	got := detectLinuxDialogTool(lookPath)
	assert.Equal(t, "", got, "should return empty when no tool is found")
}

func TestDetectLinuxDialogTool_ZenityFound(t *testing.T) {
	lookPath := func(file string) (string, error) {
		if file == "zenity" {
			return "/usr/bin/zenity", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}
	got := detectLinuxDialogTool(lookPath)
	assert.Equal(t, "zenity", got)
}

func TestDetectLinuxDialogTool_KDialogFound(t *testing.T) {
	lookPath := func(file string) (string, error) {
		if file == "kdialog" {
			return "/usr/bin/kdialog", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}
	got := detectLinuxDialogTool(lookPath)
	assert.Equal(t, "kdialog", got)
}

func TestDetectLinuxDialogTool_YadFound(t *testing.T) {
	lookPath := func(file string) (string, error) {
		if file == "yad" {
			return "/usr/bin/yad", nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}
	got := detectLinuxDialogTool(lookPath)
	assert.Equal(t, "yad", got)
}

func TestDetectLinuxDialogTool_PrefersZenityOverKdialog(t *testing.T) {
	lookPath := func(file string) (string, error) {
		switch file {
		case "zenity", "kdialog":
			return "/usr/bin/" + file, nil
		}
		return "", fmt.Errorf("not found: %s", file)
	}
	got := detectLinuxDialogTool(lookPath)
	assert.Equal(t, "zenity", got, "zenity should be preferred when both are available")
}

// ---------------------------------------------------------------------------
// buildLinuxScript — tool selection
// ---------------------------------------------------------------------------

func TestBuildLinuxScript_Zenity(t *testing.T) {
	data := DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: login",
		LorePath:   "/usr/local/bin/lore",
		RepoRoot:   "/home/dev/project",
	}
	script := buildLinuxScript(data, "zenity")
	assert.Contains(t, script, "zenity")
	assert.NotContains(t, script, "kdialog")
}

func TestBuildLinuxScript_KDialog(t *testing.T) {
	data := DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: login",
		LorePath:   "/usr/local/bin/lore",
		RepoRoot:   "/home/dev/project",
	}
	script := buildLinuxScript(data, "kdialog")
	assert.Contains(t, script, "kdialog")
}

func TestBuildLinuxScript_YadFallsBackToZenity(t *testing.T) {
	data := DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: login",
		LorePath:   "/usr/local/bin/lore",
		RepoRoot:   "/home/dev/project",
	}
	script := buildLinuxScript(data, "yad")
	// yad uses the zenity-compatible script
	assert.Contains(t, script, "zenity")
}

// ---------------------------------------------------------------------------
// buildZenityScript
// ---------------------------------------------------------------------------

func TestBuildZenityScript_ContainsExpectedContent(t *testing.T) {
	data := DialogData{
		CommitHash:  "abc1234",
		CommitMsg:   "feat(auth): add OAuth",
		DiffStat:    "+42 -7 auth/oauth.go",
		LorePath:    "/usr/local/bin/lore",
		RepoRoot:    "/home/dev/project",
		PrefillWhat: "Added OAuth support",
		PrefillWhy:  "Users need SSO",
	}

	script := buildZenityScript(data)

	assert.Contains(t, script, "zenity --list")
	assert.Contains(t, script, "zenity --entry")
	assert.Contains(t, script, "/usr/local/bin/lore")
	assert.Contains(t, script, "/home/dev/project")
	assert.Contains(t, script, "abc1234")
	assert.Contains(t, script, "Added OAuth support")
	assert.Contains(t, script, "Users need SSO")
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "pending resolve")
}

func TestBuildZenityScript_CustomLabels(t *testing.T) {
	data := DialogData{
		CommitHash:     "abc1234",
		LorePath:       "/bin/lore",
		RepoRoot:       "/tmp",
		LabelTitle:     "Custom Title",
		LabelTitleWhat: "Custom What Title",
		LabelTitleWhy:  "Custom Why Title",
		LabelType:      "Kind:",
		LabelWhat:      "What changed?",
		LabelWhy:       "Why changed?",
	}

	script := buildZenityScript(data)

	assert.Contains(t, script, "Custom Title")
	assert.Contains(t, script, "Custom What Title")
	assert.Contains(t, script, "Custom Why Title")
	assert.Contains(t, script, "Kind:")
	assert.Contains(t, script, "What changed?")
	assert.Contains(t, script, "Why changed?")
}

func TestBuildZenityScript_BranchAndScope(t *testing.T) {
	data := DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "feat(auth): add login",
		LorePath:   "/bin/lore",
		RepoRoot:   "/tmp/project",
		Branch:     "feature/auth",
		Scope:      "auth",
	}

	script := buildZenityScript(data)

	assert.Contains(t, script, "Branch: feature/auth")
	assert.Contains(t, script, "Scope: auth")
}

// ---------------------------------------------------------------------------
// buildKDialogScript
// ---------------------------------------------------------------------------

func TestBuildKDialogScript_ContainsExpectedContent(t *testing.T) {
	data := DialogData{
		CommitHash:  "def5678",
		CommitMsg:   "refactor: simplify DB layer",
		DiffStat:    "+10 -30 db/query.go",
		LorePath:    "/usr/bin/lore",
		RepoRoot:    "/home/dev/myapp",
		PrefillWhat: "Simplified queries",
		PrefillWhy:  "Reduce complexity",
	}

	script := buildKDialogScript(data)

	assert.Contains(t, script, "kdialog --combobox")
	assert.Contains(t, script, "kdialog --inputbox")
	assert.Contains(t, script, "/usr/bin/lore")
	assert.Contains(t, script, "/home/dev/myapp")
	assert.Contains(t, script, "def5678")
	assert.Contains(t, script, "Simplified queries")
	assert.Contains(t, script, "Reduce complexity")
	assert.Contains(t, script, "#!/bin/bash")
	assert.Contains(t, script, "pending resolve")
}

func TestBuildKDialogScript_CustomLabels(t *testing.T) {
	data := DialogData{
		CommitHash:     "abc1234",
		LorePath:       "/bin/lore",
		RepoRoot:       "/tmp",
		LabelTitle:     "Mon Titre",
		LabelTitleWhat: "Mon Titre — Quoi",
		LabelTitleWhy:  "Mon Titre — Pourquoi",
		LabelWhat:      "Quoi?",
		LabelWhy:       "Pourquoi?",
	}

	script := buildKDialogScript(data)

	assert.Contains(t, script, "Mon Titre")
	assert.Contains(t, script, "Mon Titre — Quoi")
	assert.Contains(t, script, "Mon Titre — Pourquoi")
	assert.Contains(t, script, "Quoi?")
	assert.Contains(t, script, "Pourquoi?")
}

// ---------------------------------------------------------------------------
// Icon / --window-icon tests
// ---------------------------------------------------------------------------

func TestBuildZenityScript_ContainsWindowIcon(t *testing.T) {
	data := DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: icon test",
		LorePath:   "/bin/lore",
		RepoRoot:   "/tmp",
	}
	script := buildZenityScript(data)

	assert.Contains(t, script, "--window-icon=", "zenity script should contain --window-icon flag")
	assert.Contains(t, script, "lore-logo-", "zenity icon path should reference the embedded logo")
}

func TestBuildKDialogScript_ContainsIcon(t *testing.T) {
	data := DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: icon test",
		LorePath:   "/bin/lore",
		RepoRoot:   "/tmp",
	}
	script := buildKDialogScript(data)

	assert.Contains(t, script, "--icon ", "kdialog script should contain --icon flag")
	assert.Contains(t, script, "lore-logo-", "kdialog icon path should reference the embedded logo")
}

// ---------------------------------------------------------------------------
// NotifyOSDialog — no tools available
// ---------------------------------------------------------------------------

func TestNotifyOSDialog_NoToolsAvailable(t *testing.T) {
	err := NotifyOSDialog(DialogData{
		CommitHash: "abc",
		LorePath:   "/bin/lore",
		RepoRoot:   "/tmp",
	}, DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			t.Fatal("StartCommand should not be called when no dialog tool is found")
			return nil
		},
		LookPath: func(file string) (string, error) {
			return "", fmt.Errorf("not found: %s", file)
		},
	})

	assert.ErrorIs(t, err, errUnsupportedOS)
}

// ---------------------------------------------------------------------------
// NotifyOSDialog — with mocked LookPath + StartCommand
// ---------------------------------------------------------------------------

func TestNotifyOSDialog_WithZenity(t *testing.T) {
	var captured struct {
		name string
		args []string
	}

	err := NotifyOSDialog(DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: auth bug",
		LorePath:   "/usr/local/bin/lore",
		RepoRoot:   "/home/dev/project",
	}, DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			captured.name = name
			captured.args = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			if file == "zenity" {
				return "/usr/bin/zenity", nil
			}
			return "", fmt.Errorf("not found: %s", file)
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "bash", captured.name)
	require.Len(t, captured.args, 2)
	assert.Equal(t, "-c", captured.args[0])
	assert.Contains(t, captured.args[1], "zenity")
}

func TestNotifyOSDialog_WithKDialog(t *testing.T) {
	var captured struct {
		name string
		args []string
	}

	err := NotifyOSDialog(DialogData{
		CommitHash: "abc1234",
		CommitMsg:  "fix: auth bug",
		LorePath:   "/usr/local/bin/lore",
		RepoRoot:   "/home/dev/project",
	}, DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			captured.name = name
			captured.args = args
			return nil
		},
		LookPath: func(file string) (string, error) {
			if file == "kdialog" {
				return "/usr/bin/kdialog", nil
			}
			return "", fmt.Errorf("not found: %s", file)
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "bash", captured.name)
	require.Len(t, captured.args, 2)
	assert.Equal(t, "-c", captured.args[0])
	assert.Contains(t, captured.args[1], "kdialog")
}

func TestNotifyOSDialog_StartCommandError(t *testing.T) {
	err := NotifyOSDialog(DialogData{
		CommitHash: "abc",
		LorePath:   "/bin/lore",
		RepoRoot:   "/tmp",
	}, DialogOpts{
		StartCommand: func(name string, args, env []string) error {
			return fmt.Errorf("failed to start")
		},
		LookPath: func(file string) (string, error) {
			if file == "zenity" {
				return "/usr/bin/zenity", nil
			}
			return "", fmt.Errorf("not found: %s", file)
		},
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start")
}
