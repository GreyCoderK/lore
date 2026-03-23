// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const totalQuestions = 5

// PendingItem is the view-model for a single pending entry, used by
// ListPending and the cmd layer.
type PendingItem struct {
	Filename      string    // e.g. "abc1234.yaml"
	CommitHash    string    // short hash (from filename / record.Commit)
	CommitMessage string    // from record.Message
	CommitDate    time.Time // parsed from record.Date
	Answers       PendingAnswers
	Progress      string // "3/5"
	RelativeAge   string // "2 days ago"
}

// ListPending reads and parses all YAML files in pendingDir, returning
// them sorted by date descending (most recent first). Corrupt files are
// skipped with a warning written to warnWriter (if non-nil).
func ListPending(ctx context.Context, pendingDir string, warnWriter func(string)) ([]PendingItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("workflow: pending: %w", err)
	}

	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("workflow: pending: read dir: %w", err)
	}

	var items []PendingItem
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(pendingDir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if warnWriter != nil {
				warnWriter(fmt.Sprintf("Warning: could not read %s: %v", entry.Name(), readErr))
			}
			continue
		}

		var record PendingRecord
		if parseErr := yaml.Unmarshal(data, &record); parseErr != nil {
			if warnWriter != nil {
				warnWriter(fmt.Sprintf("Warning: could not parse %s: %v", entry.Name(), parseErr))
			}
			continue
		}

		commitDate, _ := time.Parse(time.RFC3339, record.Date)

		items = append(items, PendingItem{
			Filename:      entry.Name(),
			CommitHash:    record.Commit,
			CommitMessage: record.Message,
			CommitDate:    commitDate,
			Answers:       record.Answers,
			Progress:      computeProgress(record.Answers),
			RelativeAge:   RelativeAge(time.Since(commitDate)),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CommitDate.After(items[j].CommitDate)
	})

	return items, nil
}

// SkipPending deletes the pending file matching the given commit hash
// without creating a document. The hash must match exactly or be a unique
// prefix among all pending items.
func SkipPending(ctx context.Context, pendingDir string, commitHash string) (PendingItem, error) {
	items, err := ListPending(ctx, pendingDir, nil)
	if err != nil {
		return PendingItem{}, fmt.Errorf("workflow: skip pending: %w", err)
	}

	// Collect all matches: exact hash or filename prefix
	var matches []PendingItem
	for _, item := range items {
		if item.CommitHash == commitHash {
			// Exact match — use immediately
			matches = []PendingItem{item}
			break
		}
		if strings.HasPrefix(item.CommitHash, commitHash) || strings.HasPrefix(item.Filename, commitHash) {
			matches = append(matches, item)
		}
	}

	if len(matches) == 0 {
		return PendingItem{}, fmt.Errorf("workflow: skip pending: no pending item for hash %q", commitHash)
	}
	if len(matches) > 1 {
		return PendingItem{}, fmt.Errorf("workflow: skip pending: ambiguous hash %q matches %d items", commitHash, len(matches))
	}

	item := matches[0]
	if err := deletePendingFile(pendingDir, item.Filename); err != nil {
		return PendingItem{}, fmt.Errorf("workflow: skip pending: %w", err)
	}
	return item, nil
}

// deletePendingFile removes a single pending file by filename, with path
// traversal validation.
func deletePendingFile(pendingDir, filename string) error {
	path := filepath.Join(pendingDir, filename)
	resolved, resolveErr := filepath.EvalSymlinks(filepath.Dir(path))
	if resolveErr != nil {
		// Do not fall back to unresolved path — reject if symlink resolution fails.
		return fmt.Errorf("workflow: pending: cannot resolve path: %w", resolveErr)
	}
	expectedDir, expectedErr := filepath.EvalSymlinks(pendingDir)
	if expectedErr != nil {
		return fmt.Errorf("workflow: pending: cannot resolve pending dir: %w", expectedErr)
	}
	if resolved != expectedDir {
		return fmt.Errorf("workflow: pending: path traversal detected")
	}
	return os.Remove(path)
}

// computeProgress counts non-empty answer fields and returns "N/5".
func computeProgress(a PendingAnswers) string {
	count := 0
	if a.Type != "" {
		count++
	}
	if a.What != "" {
		count++
	}
	if a.Why != "" {
		count++
	}
	if a.Alternatives != "" {
		count++
	}
	if a.Impact != "" {
		count++
	}
	return fmt.Sprintf("%d/%d", count, totalQuestions)
}

// RelativeAge formats a duration into a human-readable relative age string.
func RelativeAge(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	minutes := int(d.Minutes())
	hours := int(d.Hours())
	days := hours / 24
	weeks := days / 7

	switch {
	case minutes < 5:
		return "just now"
	case minutes < 60:
		return fmt.Sprintf("%d minutes ago", minutes)
	case hours < 24:
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case days < 7:
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case weeks < 4:
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	default:
		months := days / 30
		if months <= 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}
}
