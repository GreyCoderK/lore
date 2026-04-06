// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/greycoderk/lore/internal/domain"
	loretemplate "github.com/greycoderk/lore/internal/template"
)

// PreflightCheck validates that the documentation pipeline can succeed
// BEFORE asking the user any questions. Returns nil if everything is OK,
// or a descriptive error if the pipeline would fail post-questions.
//
// Why: the user should never spend 90 seconds answering questions only to
// hit a template or filesystem error at the end. This preserves the
// "90 seconds or nothing" contract.
func PreflightCheck(workDir string) error {
	loreDir := filepath.Join(workDir, domain.LoreDir)

	// 1. .lore/ exists and is a directory
	info, err := os.Stat(loreDir)
	if err != nil {
		return fmt.Errorf("preflight: %s not found — run 'lore init' first", domain.LoreDir)
	}
	if !info.IsDir() {
		return fmt.Errorf("preflight: %s is not a directory", domain.LoreDir)
	}

	// 2. .lore/docs/ is writable
	docsDir := filepath.Join(loreDir, domain.DocsDir)
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return fmt.Errorf("preflight: cannot create docs directory: %w", err)
	}
	testPath := filepath.Join(docsDir, ".preflight-test")
	if err := os.WriteFile(testPath, []byte("ok"), 0o644); err != nil {
		return fmt.Errorf("preflight: docs directory not writable: %w", err)
	}
	_ = os.Remove(testPath)

	// 3. Template engine initializes without error
	_, err = loretemplate.New(
		filepath.Join(loreDir, domain.TemplatesDir),
		loretemplate.GlobalDir(),
	)
	if err != nil {
		return fmt.Errorf("preflight: template engine: %w", err)
	}

	return nil
}
