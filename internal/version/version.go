// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package version

import (
	"fmt"
	"runtime/debug"
)

// These variables are set at build time via ldflags. When ldflags are
// missing (e.g., release pipeline drops them, or a contributor builds
// from source without the Makefile), init() enriches them from the
// Go binary's embedded VCS info (debug.ReadBuildInfo), so --version
// always gives meaningful output.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func init() {
	// Only fall back to embedded VCS info if ldflags weren't set.
	// If a release shipped with proper ldflags, those win.
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	// Module version (e.g., "v1.2.2") is populated when the binary
	// was installed via `go install path@vX.Y.Z` or built by goreleaser
	// from a tagged commit. Local `go build` reports "(devel)".
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value == "" {
				continue
			}
			if len(s.Value) >= 8 {
				Commit = s.Value[:8]
			} else {
				Commit = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				Date = s.Value
			}
		}
	}
}

// Info returns a formatted version string suitable for --version output.
func Info() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
}
