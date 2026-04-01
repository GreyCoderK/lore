// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"os"
	"strings"
)

// InstallMethod represents how lore was installed on the system.
type InstallMethod int

const (
	// InstallBinary means lore was installed as a standalone binary (self-update OK).
	InstallBinary InstallMethod = iota
	// InstallHomebrew means lore was installed via Homebrew.
	InstallHomebrew
	// InstallGoInstall means lore was installed via go install.
	InstallGoInstall
)

// DetectInstallMethod inspects the resolved binary path to determine how lore
// was installed. Returns the method and, for non-binary methods, the command
// the user should run instead.
func DetectInstallMethod(execPath string) (InstallMethod, string) {
	lower := strings.ToLower(execPath)

	// Homebrew: path typically contains /homebrew/, /linuxbrew/, or /Cellar/
	if strings.Contains(lower, "/homebrew/") ||
		strings.Contains(lower, "/linuxbrew/") ||
		strings.Contains(lower, "/cellar/") {
		return InstallHomebrew, "brew upgrade lore"
	}

	// Go install: binary lives in $GOPATH/bin or ~/go/bin
	if gopath := os.Getenv("GOPATH"); gopath != "" && strings.HasPrefix(execPath, gopath+"/bin") {
		return InstallGoInstall, "go install github.com/greycoderk/lore@latest"
	}
	if strings.Contains(execPath, "/go/bin/") {
		return InstallGoInstall, "go install github.com/greycoderk/lore@latest"
	}

	return InstallBinary, ""
}
