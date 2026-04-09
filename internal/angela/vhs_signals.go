// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// VHSSignals holds analysis results from cross-referencing VHS tape files
// with documentation and CLI commands.
type VHSSignals struct {
	// TapeCommands maps tape filename → list of CLI commands found in Type/Exec lines.
	TapeCommands map[string][]string

	// TapeOutputs maps tape filename → output GIF/PNG path from Output directive.
	TapeOutputs map[string]string

	// OrphanTapes are tape files whose output GIF is not referenced in any doc.
	OrphanTapes []string

	// OrphanGIFs are GIF references in docs that have no corresponding .tape source.
	OrphanGIFs []GIFRef

	// CommandMismatches are CLI commands in tapes that don't appear in any doc.
	CommandMismatches []TapeMismatch
}

// GIFRef represents a GIF referenced in a documentation file.
type GIFRef struct {
	DocFilename string
	GIFPath     string
}

// TapeMismatch represents a command in a tape file that may be outdated or undocumented.
type TapeMismatch struct {
	TapeFile string
	Command  string
	Reason   string // "undocumented_command", "unknown_subcommand"
}

// tapeCommandRe matches Type "lore ..." or Type 'lore ...' lines in VHS tapes.
var tapeCommandRe = regexp.MustCompile(`^Type\s+["'](.+?)["']\s*$`)

// gifRefRe matches markdown image references to GIF/PNG files.
var gifRefRe = regexp.MustCompile(`!\[.*?\]\((.*?\.(?:gif|png))\)`)

// AnalyzeVHSSignals cross-references VHS tape files with documentation.
// tapeDir: directory containing .tape files (e.g., assets/vhs/)
// docsDir: directory containing .md documentation files
// knownCommands: set of known CLI commands (e.g., from cobra)
func AnalyzeVHSSignals(tapeDir, docsDir string, knownCommands []string) *VHSSignals {
	signals := &VHSSignals{
		TapeCommands: make(map[string][]string),
		TapeOutputs:  make(map[string]string),
	}

	// Parse all tape files
	tapes, err := os.ReadDir(tapeDir)
	if err != nil {
		return signals // no tape dir = no signals
	}

	for _, entry := range tapes {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tape") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tapeDir, entry.Name()))
		if err != nil {
			continue
		}
		parseTape(signals, entry.Name(), string(data))
	}

	// Collect all GIF references from docs
	docGIFs := collectDocGIFRefs(docsDir)

	// Cross-reference: find orphan tapes (output not referenced in any doc)
	referencedOutputs := make(map[string]bool)
	for _, ref := range docGIFs {
		referencedOutputs[normalizeGIFPath(ref.GIFPath)] = true
	}
	for tape, output := range signals.TapeOutputs {
		if output != "" && !referencedOutputs[normalizeGIFPath(output)] {
			signals.OrphanTapes = append(signals.OrphanTapes, tape)
		}
	}

	// Cross-reference: find orphan GIFs (referenced in docs but no tape source)
	tapeOutputSet := make(map[string]bool)
	for _, output := range signals.TapeOutputs {
		if output != "" {
			tapeOutputSet[normalizeGIFPath(output)] = true
		}
	}
	for _, ref := range docGIFs {
		if !tapeOutputSet[normalizeGIFPath(ref.GIFPath)] {
			signals.OrphanGIFs = append(signals.OrphanGIFs, ref)
		}
	}

	// Cross-reference: find commands in tapes that are not in known commands
	if len(knownCommands) > 0 {
		cmdSet := make(map[string]bool, len(knownCommands))
		for _, c := range knownCommands {
			cmdSet[c] = true
		}
		for tape, commands := range signals.TapeCommands {
			for _, cmd := range commands {
				root := extractRootCommand(cmd)
				if root != "" && !cmdSet[root] {
					signals.CommandMismatches = append(signals.CommandMismatches, TapeMismatch{
						TapeFile: tape,
						Command:  cmd,
						Reason:   "unknown_subcommand",
					})
				}
			}
		}
	}

	return signals
}

// parseTape extracts commands and output paths from a VHS tape file.
func parseTape(signals *VHSSignals, filename, content string) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Extract Output directive
		if strings.HasPrefix(trimmed, "Output ") {
			output := strings.TrimSpace(trimmed[7:])
			signals.TapeOutputs[filename] = output
			continue
		}

		// Extract Type "command" directives
		matches := tapeCommandRe.FindStringSubmatch(trimmed)
		if len(matches) >= 2 {
			cmd := matches[1]
			// Only track lore commands (the CLI we're analyzing)
			if strings.HasPrefix(cmd, "lore ") {
				signals.TapeCommands[filename] = append(signals.TapeCommands[filename], cmd)
			}
		}
	}
}

// collectDocGIFRefs scans markdown files for image references to GIF/PNG files.
func collectDocGIFRefs(docsDir string) []GIFRef {
	var refs []GIFRef

	resolvedDocsDir, err := filepath.EvalSymlinks(docsDir)
	if err != nil {
		return refs
	}

	// Walk recursively to handle nested doc directories
	_ = filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		// Reject symlinks that escape the docs directory
		resolved, evalErr := filepath.EvalSymlinks(path)
		if evalErr != nil || !strings.HasPrefix(resolved, resolvedDocsDir) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		relPath, _ := filepath.Rel(docsDir, path)
		matches := gifRefRe.FindAllStringSubmatch(string(data), -1)
		for _, m := range matches {
			if len(m) >= 2 {
				refs = append(refs, GIFRef{
					DocFilename: relPath,
					GIFPath:     m[1],
				})
			}
		}
		return nil
	})

	return refs
}

// normalizeGIFPath normalizes a GIF path for comparison by extracting the filename.
func normalizeGIFPath(p string) string {
	return filepath.Base(p)
}

// extractRootCommand extracts the subcommand chain from a lore CLI invocation.
// "lore angela review --quiet" → "angela review"
func extractRootCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) < 2 || parts[0] != "lore" {
		return ""
	}

	// Collect subcommands (stop at flags)
	var subs []string
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "-") {
			break
		}
		subs = append(subs, p)
	}
	return strings.Join(subs, " ")
}
