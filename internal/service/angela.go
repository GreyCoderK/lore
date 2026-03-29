// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/storage"
)

// PolishResult holds the output of a polish orchestration.
type PolishResult struct {
	Original    string
	Polished    string
	Diff        []angela.DiffHunk
	Meta        domain.DocMeta
	Filename    string
}

// PolishDocument orchestrates the full document polish workflow:
// read document, resolve personas, call AI polish, compute diff.
func PolishDocument(ctx context.Context, provider domain.AIProvider, cfg *config.Config, docsDir string, filename string) (*PolishResult, error) {
	docPath := filepath.Join(docsDir, filename)

	// Read document
	raw, err := os.ReadFile(docPath)
	if err != nil {
		return nil, fmt.Errorf("angela: polish: read: %w", err)
	}
	originalContent := string(raw)

	meta, _, err := storage.Unmarshal(raw)
	if err != nil {
		return nil, fmt.Errorf("angela: polish: parse: %w", err)
	}
	meta.Filename = filename

	// Style guide
	var styleGuideStr string
	if cfg.Angela.StyleGuide != nil {
		guide := angela.ParseStyleGuide(cfg.Angela.StyleGuide)
		styleGuideStr = angela.FormatStyleGuideRules(guide)
	}

	// Corpus summary
	corpusStore := &storage.CorpusStore{Dir: docsDir}
	corpus, _ := corpusStore.ListDocs(domain.DocFilter{})
	corpusSummary := angela.BuildCorpusSummary(corpus)

	// Resolve personas for this document
	scored := angela.ResolvePersonas(meta.Type, originalContent)
	personas := angela.Profiles(scored)

	// Exactly 1 API call
	polished, err := angela.Polish(ctx, provider, originalContent, meta, styleGuideStr, corpusSummary, personas)
	if err != nil {
		return nil, err
	}

	// Compute diff
	hunks := angela.ComputeDiff(originalContent, polished)

	return &PolishResult{
		Original: originalContent,
		Polished: polished,
		Diff:     hunks,
		Meta:     meta,
		Filename: filename,
	}, nil
}

// ReviewCorpus orchestrates the full corpus review workflow:
// prepare summaries, call AI review.
func ReviewCorpus(ctx context.Context, provider domain.AIProvider, reader domain.CorpusReader, cfg *config.Config, styleGuide map[string]interface{}) (*angela.ReviewReport, int, error) {
	// Prepare doc summaries
	summaries, totalCount, err := angela.PrepareDocSummaries(reader)
	if err != nil {
		return nil, 0, err
	}

	// Style guide
	var styleGuideStr string
	if styleGuide != nil {
		guide := angela.ParseStyleGuide(styleGuide)
		styleGuideStr = angela.FormatStyleGuideRules(guide)
	}

	// Exactly 1 API call
	report, err := angela.Review(ctx, provider, summaries, styleGuideStr)
	if err != nil {
		return nil, totalCount, err
	}

	return report, totalCount, nil
}
