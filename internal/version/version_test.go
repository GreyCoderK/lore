// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package version

import (
	"strings"
	"testing"
)

func TestInfo_DefaultValues(t *testing.T) {
	t.Parallel()
	info := Info()
	if !strings.Contains(info, "dev") {
		t.Errorf("Info() = %q, want to contain 'dev'", info)
	}
	if !strings.Contains(info, "none") {
		t.Errorf("Info() = %q, want to contain 'none'", info)
	}
	if !strings.Contains(info, "unknown") {
		t.Errorf("Info() = %q, want to contain 'unknown'", info)
	}
}

func TestInfo_Format(t *testing.T) {
	t.Parallel()
	info := Info()
	if !strings.Contains(info, "commit:") {
		t.Errorf("Info() = %q, want to contain 'commit:'", info)
	}
	if !strings.Contains(info, "built:") {
		t.Errorf("Info() = %q, want to contain 'built:'", info)
	}
}

func TestInfo_CustomValues(t *testing.T) {
	// NOT parallel-safe: mutates package-level vars Version, Commit, Date.
	// Save and restore
	origVersion, origCommit, origDate := Version, Commit, Date
	defer func() { Version, Commit, Date = origVersion, origCommit, origDate }()

	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-03-23"

	info := Info()
	if !strings.Contains(info, "v1.2.3") {
		t.Errorf("Info() = %q, want to contain 'v1.2.3'", info)
	}
	if !strings.Contains(info, "abc1234") {
		t.Errorf("Info() = %q, want to contain 'abc1234'", info)
	}
	if !strings.Contains(info, "2026-03-23") {
		t.Errorf("Info() = %q, want to contain '2026-03-23'", info)
	}
}
