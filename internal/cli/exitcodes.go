package cli

// Exit codes for the lore CLI, following Unix conventions.
// Used by cmd/root.go and commands that need non-zero exits.
const (
	ExitOK    = 0
	ExitError = 1
	ExitSkip  = 2 // no match found (e.g. lore show with zero results)
)
