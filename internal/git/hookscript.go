package git

import _ "embed"

//go:embed scripts/post-commit.sh
var postCommitScript string

// readHookScript returns the embedded hook script content.
func readHookScript() string {
	return postCommitScript
}
