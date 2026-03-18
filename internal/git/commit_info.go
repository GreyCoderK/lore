// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import (
	"regexp"
	"strings"
)

var conventionalCommitRe = regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?:\s*(.+)$`)

// ParseConventionalCommit extracts type, scope, and subject from a commit message.
// If the message doesn't match Conventional Commits format, type and scope are empty
// and subject is the full message.
func ParseConventionalCommit(message string) (ccType, scope, subject string) {
	// Only parse the first line
	firstLine := message
	if idx := strings.IndexByte(message, '\n'); idx >= 0 {
		firstLine = message[:idx]
	}

	m := conventionalCommitRe.FindStringSubmatch(firstLine)
	if m == nil {
		return "", "", message
	}
	return m[1], m[2], m[3]
}
