package domain

import "context"

type GitAdapter interface {
	Diff(ref string) (string, error)
	Log(ref string) (*CommitInfo, error)

	CommitExists(ref string) (bool, error)
	IsMergeCommit(ref string) (bool, error)
	
	IsInsideWorkTree() bool

	HeadRef() (string, error)
	IsRebaseInProgress() (bool, error)

	CommitMessageContains(ref, marker string) (bool, error)

	InstallHook(hookType string) error
	UninstallHook(hookType string) error
	HookExists(hookType string) (bool, error)
}

type AIProvider interface {
	Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
}

type CorpusReader interface {
	ReadDoc(id string) (string, error)
	ListDocs(filter DocFilter) ([]DocMeta, error)
}