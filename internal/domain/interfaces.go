// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import (
	"context"
	"time"
)

type GitAdapter interface {
	Diff(ref string) (string, error)
	Log(ref string) (*CommitInfo, error)

	CommitExists(ref string) (bool, error)
	IsMergeCommit(ref string) (bool, error)
	
	IsInsideWorkTree() bool

	HeadRef() (string, error)
	// HeadCommit returns the full commit info for HEAD in a single git
	// invocation, replacing separate HeadRef() + Log(ref) calls.
	HeadCommit() (*CommitInfo, error)
	IsRebaseInProgress() (bool, error)

	CommitMessageContains(ref, marker string) (bool, error)

	GitDir() (string, error)

	InstallHook(hookType string) (InstallResult, error)
	UninstallHook(hookType string) error
	HookExists(hookType string) (bool, error)

	CommitRange(from, to string) ([]string, error)
	LatestTag() (string, error)

	LogAll() ([]CommitInfo, error)
	CurrentBranch() (string, error)
}

type AIProvider interface {
	Complete(ctx context.Context, prompt string, opts ...Option) (string, error)
}

type CorpusReader interface {
	ReadDoc(id string) (string, error)
	ListDocs(filter DocFilter) ([]DocMeta, error)
}

// CommitStore handles commit recording and querying.
type CommitStore interface {
	RecordCommit(rec CommitRecord) error
	GetCommit(hash string) (*CommitRecord, error)
	CommitsByScope(scope string, days int) ([]CommitRecord, error)
	CommitsByBranch(branch string) ([]CommitRecord, error)
	CommitsSince(since time.Time) ([]CommitRecord, error)
	UndocumentedCommits() ([]CommitRecord, error)
	CommitCountByDecision() (map[string]int, error)
	ScopeStats(scope string, days int) (ScopeStatsResult, error)
}

// DocIndexStore handles document index operations.
type DocIndexStore interface {
	IndexDoc(entry DocIndexEntry) error
	RemoveDoc(filename string) error
	GetDoc(filename string) (*DocIndexEntry, error)
	DocsByScope(scope string) ([]DocIndexEntry, error)
	DocsByBranch(branch string) ([]DocIndexEntry, error)
	DocsByType(docType string) ([]DocIndexEntry, error)
	UnconsolidatedDocs(scope string) ([]DocIndexEntry, error)
	AllDocSummaries(limit int) ([]DocIndexEntry, error)
	DocsByCommitHash(hash string) ([]DocIndexEntry, error)
	SearchDocs(ctx context.Context, query string) ([]DocIndexEntry, error)
	DocCount() (int, error)
}

// SignatureStore handles code signature tracking (Phase 1 — stub).
type SignatureStore interface {
	StoreSignatures(commitHash string, sigs []CodeSignature) error
	FindBySignatureHash(sigHash string) ([]CodeSignature, error)
	SignaturesForCommit(commitHash string) ([]CodeSignature, error)
	EntityHistory(entityName, lang string) ([]CodeSignature, error)
}

// AIUsageStore handles AI usage recording (Phase 1 — stub).
type AIUsageStore interface {
	RecordAIUsage(usage AIUsageRecord) error
	AIStatsSince(since time.Time) (*AIStatsAggregate, error)
	AIStatsByDay(days int) ([]DailyAIStats, error)
}

// ReviewCacheStore handles review caching (Phase 1 — stub).
type ReviewCacheStore interface {
	CacheReview(report ReviewCacheEntry) error
	GetCachedReview(corpusHash string) (*ReviewCacheEntry, error)
	ReviewHistory(limit int) ([]ReviewCacheEntry, error)
}

// PatternStore handles commit pattern tracking (Phase 2 — stub).
type PatternStore interface {
	UpdatePattern(convType, scope string, decision string, diffLines, score int) error
	GetPattern(convType, scope string) (*CommitPattern, error)
	AllPatterns() ([]CommitPattern, error)
}

// StoreMaintenanceOps handles store lifecycle operations.
type StoreMaintenanceOps interface {
	Rebuild(ctx context.Context, docsDir string, git GitAdapter) error
	Vacuum() error
	Close() error
}

// LoreStore is the composed interface for the full store.
// Consumers that only need a subset should accept the focused interface
// (CommitStore, DocIndexStore, etc.) instead of LoreStore.
type LoreStore interface {
	CommitStore
	DocIndexStore
	SignatureStore
	AIUsageStore
	ReviewCacheStore
	PatternStore
	StoreMaintenanceOps
}