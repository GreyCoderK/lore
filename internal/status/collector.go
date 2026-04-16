// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package status

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// detectAIProvider determines if an AI provider is configured and returns its name.
// Returns "" if no provider is available. Checks: env var > plaintext config > keychain.
func detectAIProvider(cfg *config.Config) string {
	if os.Getenv("LORE_AI_API_KEY") != "" {
		if cfg.AI.Provider != "" {
			return cfg.AI.Provider
		}
		return "configured"
	}
	if cfg.AI.APIKey != "" && cfg.AI.APIKey != "@keychain" {
		if cfg.AI.Provider != "" {
			return cfg.AI.Provider
		}
		return "configured"
	}
	if cfg.AI.Provider != "" {
		store := credential.NewStore()
		if key, err := store.Get(cfg.AI.Provider); err == nil && len(key) > 0 {
			return cfg.AI.Provider
		}
	}
	return ""
}

// countUniqueDocsInFindings counts the number of unique document filenames
// referenced across all review findings.
func countUniqueDocsInFindings(findings []angela.ReviewFinding) int {
	seen := make(map[string]bool)
	for _, f := range findings {
		for _, d := range f.Documents {
			seen[d] = true
		}
	}
	return len(seen)
}

// StatusInfo holds all data needed to render the status dashboard.
type StatusInfo struct {
	ProjectName      string
	HookInstalled    bool
	DocCount         int
	PendingCount     int
	ExpressCount     int
	CompleteCount    int
	ReadErrors       int
	AngelaMode       string // "draft", "polish"
	AIProvider       string // "anthropic", "openai", ""
	HealthIssues     int
	AngelaSuggestions int // total suggestions across all docs
	AngelaDocsNeedReview int // docs with at least 1 suggestion
	Coverage         *CoverageResult // doc coverage (nil when git is unavailable)
}

// CollectStatus gathers all status information from the repository.
func CollectStatus(cfg *config.Config, git domain.GitAdapter, loreDir string) (*StatusInfo, error) {
	info := &StatusInfo{}

	// Project name from directory
	absDir, err := filepath.Abs(".")
	if err == nil {
		info.ProjectName = filepath.Base(absDir)
	}

	// Hook status
	hookInstalled, err := git.HookExists("post-commit")
	if err != nil {
		// Non-fatal: git might not be initialized
		info.HookInstalled = false
	} else {
		info.HookInstalled = hookInstalled
	}

	// Doc count + express ratio
	docsDir := filepath.Join(loreDir, "docs")
	store := &storage.CorpusStore{Dir: docsDir}
	docs, err := store.ListDocs(domain.DocFilter{})
	if err != nil {
		return nil, fmt.Errorf("status: collect: list docs: %w", err)
	}
	info.DocCount = len(docs)

	// Express ratio: read each doc's content once (O(n), no corpus cross-checks)
	// TODO(optimization): store is_complete flag in doc_index DB table
	// to avoid reading all document files on every status call.
	// For now, this is acceptable for corpora under 500 documents.
	var readErrors int
	for _, meta := range docs {
		content, err := storage.ReadDocContent(filepath.Join(docsDir, meta.Filename))
		if err != nil {
			readErrors++
			continue
		}
		if strings.Contains(content, "## Alternatives") || strings.Contains(content, "## Impact") {
			info.CompleteCount++
		} else {
			info.ExpressCount++
		}
	}
	info.ReadErrors = readErrors

	// Angela suggestions: use cached review results instead of recomputing
	// O(n²) AnalyzeDraft+CheckCoherence on every status call.
	// The cache is populated by `lore angela review`.
	reviewCache, _ := angela.LoadReviewCache(loreDir)
	if reviewCache != nil {
		info.AngelaSuggestions = len(reviewCache.Findings)
		// Each finding may reference multiple docs, but for the dashboard
		// we just show total findings as the review indicator.
		if len(reviewCache.Findings) > 0 {
			info.AngelaDocsNeedReview = countUniqueDocsInFindings(reviewCache.Findings)
		}
	}

	// Pending count
	pendingDir := filepath.Join(loreDir, "pending")
	entries, err := os.ReadDir(pendingDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				info.PendingCount++
			}
		}
	}

	// Documentation coverage (commit-level).
	if git.IsInsideWorkTree() {
		cov := CalculateCoverage(filepath.Join(loreDir, "docs"), git)
		info.Coverage = &cov
	}

	// Angela mode — check env > keychain > plaintext config
	info.AngelaMode = "draft"
	info.AIProvider = detectAIProvider(cfg)
	if info.AIProvider != "" {
		info.AngelaMode = "polish"
	}

	// Health check
	issues, err := storage.QuickHealthCheck(docsDir)
	if err != nil {
		return nil, fmt.Errorf("status: collect: health: %w", err)
	}
	info.HealthIssues = issues

	return info, nil
}
