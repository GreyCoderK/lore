package git

import "embed"

//go:embed scripts/post-commit.sh
var hookScriptFS embed.FS

// readHookScript reads the embedded hook script content.
func readHookScript() string {
	data, err := hookScriptFS.ReadFile("scripts/post-commit.sh")
	if err != nil {
		panic("git: embedded hook script missing: " + err.Error())
	}
	return string(data)
}
