// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package version

import "fmt"

// These variables are set at build time via ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// Info returns a formatted version string suitable for --version output.
func Info() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, Date)
}
