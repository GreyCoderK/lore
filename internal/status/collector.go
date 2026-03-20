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

	// Single-pass: express ratio + Angela score (reads each doc once)
	guide := angela.ParseStyleGuide(nil)
	var readErrors int
	for _, meta := range docs {
		content, err := storage.ReadDocContent(filepath.Join(docsDir, meta.Filename))
		if err != nil {
			readErrors++
			continue
		}
		// Express ratio
		if strings.Contains(content, "## Alternatives") || strings.Contains(content, "## Impact") {
			info.CompleteCount++
		} else {
			info.ExpressCount++
		}
		// Angela score
		suggestions := angela.AnalyzeDraft(content, meta, guide, docs, nil)
		suggestions = append(suggestions, angela.CheckCoherence(content, meta, docs)...)
		if len(suggestions) > 0 {
			info.AngelaDocsNeedReview++
			info.AngelaSuggestions += len(suggestions)
		}
	}
	info.ReadErrors = readErrors

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

	// Angela mode — check env > keychain > plaintext config
	info.AngelaMode = "draft"
	if envKey := os.Getenv("LORE_AI_API_KEY"); envKey != "" {
		info.AngelaMode = "polish"
		info.AIProvider = cfg.AI.Provider
	} else if cfg.AI.APIKey != "" && cfg.AI.APIKey != "@keychain" {
		info.AngelaMode = "polish"
		info.AIProvider = cfg.AI.Provider
	} else if cfg.AI.Provider != "" {
		store := credential.NewStore()
		if key, err := store.Get(cfg.AI.Provider); err == nil && len(key) > 0 {
			info.AngelaMode = "polish"
			info.AIProvider = cfg.AI.Provider
		}
	}
	if info.AIProvider == "" && info.AngelaMode == "polish" {
		info.AIProvider = "configured"
	}

	// Health check
	issues, err := storage.QuickHealthCheck(docsDir)
	if err != nil {
		return nil, fmt.Errorf("status: collect: health: %w", err)
	}
	info.HealthIssues = issues

	return info, nil
}
