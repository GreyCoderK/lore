// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package git

import _ "embed"

//go:embed scripts/post-commit.sh
var postCommitScript string

// readHookScript returns the embedded hook script content.
func readHookScript() string {
	return postCommitScript
}
