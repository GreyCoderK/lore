// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// PolishOptions holds optional parameters for PolishDocument.
type PolishOptions struct {
	Audience string                  // target audience for rewrite mode
	Progress angela.MultiPassProgress // callback for multi-pass progress (nil = silent)

	// Incremental enables section-level re-polish.
	// When true the orchestrator compares section hashes with the
	// stored polish state and only sends changed sections to the AI.
	Incremental bool

	// PolishStatePath is the absolute path to polish-state.json.
	// Required when Incremental is true.
	PolishStatePath string
}

// PolishDocument orchestrates the full document polish workflow:
// read document, resolve personas, call AI polish, compute diff.
func PolishDocument(ctx context.Context, provider domain.AIProvider, cfg *config.Config, docsDir string, filename string, opts ...PolishOptions) (*PolishResult, error) {
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

	// Resolve personas — boost for audience if specified
	var audience string
	if len(opts) > 0 {
		audience = opts[0].Audience
	}
	scored := angela.ResolvePersonasForAudience(meta.Type, originalContent, audience)
	personas := angela.Profiles(scored)

	// Incremental polish path — re-polish only changed
	// sections. Falls back to full polish when conditions aren't met
	// (first run, <2 sections, AI parse failure).
	incremental := len(opts) > 0 && opts[0].Incremental && opts[0].PolishStatePath != ""

	// Decide: single-pass or multi-pass based on document size
	docWordCount := len(strings.Fields(originalContent))
	var polished string

	if incremental {
		polishState, loadErr := angela.LoadPolishState(opts[0].PolishStatePath)
		if loadErr != nil {
			// Non-fatal: proceed with fresh state (full polish).
			polishState = &angela.PolishState{Entries: map[string]angela.PolishStateEntry{}}
		}
		storedEntry := polishState.Entries[filename]
		incOpts := angela.IncrementalOpts{
			Provider:       provider,
			Meta:           meta,
			StyleGuide:     styleGuideStr,
			CorpusSummary:  corpusSummary,
			Personas:       personas,
			Audience:       audience,
			ConfigMaxToks:  cfg.Angela.MaxTokens,
			MinChangeLines: cfg.Angela.Polish.Incremental.MinChangeLines,
		}
		res, incErr := angela.PolishIncremental(ctx, originalContent, storedEntry.SectionHashes, incOpts)
		if incErr != nil {
			return nil, incErr
		}
		polished = res.Polished
		// Update and save state.
		polishState.Entries[filename] = angela.PolishStateEntry{
			LastPolished:   time.Now().UTC(),
			SectionHashes: res.NewHashes,
		}
		if saveErr := angela.SavePolishState(opts[0].PolishStatePath, polishState); saveErr != nil {
			// Non-fatal: warn but don't fail the polish.
			_, _ = fmt.Fprintf(os.Stderr, "warning: polish state save: %v\n", saveErr)
		}
	} else if angela.ShouldMultiPass(docWordCount) {
		// Multi-pass: section by section
		var progress angela.MultiPassProgress
		if len(opts) > 0 && opts[0].Progress != nil {
			progress = opts[0].Progress
		}
		polished, err = angela.PolishMultiPass(ctx, provider, originalContent, meta, styleGuideStr, personas, progress, audience)
		if err != nil {
			return nil, err
		}
	} else {
		// Single-pass: whole document
		polishOpts := []angela.PolishOpts{{Audience: audience, ConfigMaxToks: cfg.Angela.MaxTokens}}
		polished, err = angela.Polish(ctx, provider, originalContent, meta, styleGuideStr, corpusSummary, personas, polishOpts...)
		if err != nil {
			return nil, err
		}
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
