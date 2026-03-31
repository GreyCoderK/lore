// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package status

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// CoverageResult holds documentation coverage metrics.
type CoverageResult struct {
	TotalCommits  int
	MergeCommits  int
	RebaseCommits int
	Eligible      int
	Documented    int
	DocSkipped    int
	Covered       int
	Gaps          int
	Coverage      int     // 0-100 percentage
	SkipRate      float64 // 0-1
}

// CalculateCoverage computes documentation coverage for the repository.
func CalculateCoverage(docsDir string, gitAdapter domain.GitAdapter) CoverageResult {
	var result CoverageResult

	commits := listRecentCommits(gitAdapter)
	result.TotalCommits = len(commits)

	for _, c := range commits {
		if c.IsMerge {
			result.MergeCommits++
		}
		if strings.Contains(c.Message, "[doc-skip]") {
			result.DocSkipped++
		}
	}

	result.Eligible = result.TotalCommits - result.MergeCommits - result.RebaseCommits
	if result.Eligible <= 0 {
		return result
	}

	// Count documented commits by scanning .lore/docs/.
	corpus := &storage.CorpusStore{Dir: docsDir}
	docs, _ := corpus.ListDocs(domain.DocFilter{})

	documentedHashes := make(map[string]bool)
	for _, doc := range docs {
		if doc.Status == "demo" {
			continue
		}
		if doc.Commit != "" {
			documentedHashes[doc.Commit] = true
		}
	}
	result.Documented = len(documentedHashes)

	result.Covered = result.Documented + result.DocSkipped
	result.Gaps = result.Eligible - result.Covered
	if result.Gaps < 0 {
		result.Gaps = 0
	}

	result.Coverage = int(float64(result.Covered) / float64(result.Eligible) * 100)
	if result.Coverage > 100 {
		result.Coverage = 100
	}

	if result.Eligible > 0 {
		result.SkipRate = float64(result.DocSkipped) / float64(result.Eligible)
	}

	return result
}

type commitInfo struct {
	Hash    string
	Message string
	IsMerge bool
}

func listRecentCommits(gitAdapter domain.GitAdapter) []commitInfo {
	commits, err := gitAdapter.LogAll()
	if err != nil {
		return nil
	}
	result := make([]commitInfo, 0, len(commits))
	for _, c := range commits {
		result = append(result, commitInfo{
			Hash:    c.Hash,
			Message: c.Message,
			IsMerge: c.IsMerge,
		})
	}
	return result
}

// BadgeColor returns the shields.io color hex for a coverage percentage.
func BadgeColor(coverage int) string {
	switch {
	case coverage >= 80:
		return "d4a017" // gold
	case coverage >= 50:
		return "5c2" // green
	default:
		return "555" // grey
	}
}

// FormatBadgeMarkdown generates a shields.io badge markdown snippet.
func FormatBadgeMarkdown(coverage int, label string) string {
	color := BadgeColor(coverage)
	var display string
	if coverage == 100 {
		display = "💯"
	} else {
		display = fmt.Sprintf("%d%%25", coverage)
	}
	return fmt.Sprintf(
		"[![lore: %s %s](https://img.shields.io/badge/lore-%s_%s-%s)](https://github.com/greycoderk/lore)",
		fmt.Sprintf("%d%%", coverage), label,
		display, label, color,
	)
}
