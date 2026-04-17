// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

// SynthesizerProposal is one insertion the polish pipeline can offer the
// user. It carries the rendered Block plus the signature that should be
// written to the doc's frontmatter once the user accepts.
//
// Polish never modifies docs without proposing first: proposals are routed
// through the interactive diff in interactive mode, or the dry-run reporter
// in --synthesizer-dry-run mode (I7 — no silent merge).
type SynthesizerProposal struct {
	Doc              *synthesizer.Doc
	SynthesizerName  string
	Block            synthesizer.Block
	Evidence         []synthesizer.Evidence
	Warnings         []synthesizer.Warning
	Signature        synthesizer.Signature
	CandidateKey     string

	// RenderedMarkdown is the markdown snippet to insert into the doc body
	// (Title heading + fenced code block + notes bullets). Already sized
	// for the parent section's depth.
	RenderedMarkdown string
}

// SynthesizerProposalsForDoc gathers polish proposals for a single doc by
// iterating every applicable synthesizer and short-circuiting any whose
// existing signature is still fresh (I6).
//
// version is looked up from the per-synthesizer config; absence falls back
// to "" which forces the freshness comparison to use raw hash equality.
func SynthesizerProposalsForDoc(
	doc *synthesizer.Doc,
	registry *synthesizer.Registry,
	cfg config.SynthesizersConfig,
) ([]SynthesizerProposal, error) {
	if doc == nil || registry == nil {
		return nil, nil
	}

	enabled := registry.ForDoc(doc, synthesizer.EnabledConfig{Enabled: cfg.Enabled})
	if len(enabled) == 0 {
		return nil, nil
	}

	synthCfg := synthesizer.Config{
		WellKnownServerFields: cfg.WellKnownServerFields,
		PerSynthesizer:        cfg.PerSynthesizer,
	}

	var proposals []SynthesizerProposal
	for _, s := range enabled {
		candidates, err := s.Detect(doc)
		if err != nil {
			return nil, fmt.Errorf("synthesizer %s detect: %w", s.Name(), err)
		}
		for _, c := range candidates {
			block, evs, warns, err := s.Synthesize(c, synthCfg)
			if err != nil {
				return nil, fmt.Errorf("synthesizer %s synthesize(%s): %w", s.Name(), c.Key, err)
			}

			version := lookupSynthesizerVersion(cfg, s.Name())
			existing := doc.Signatures[s.Name()]
			if synthesizer.IsFresh(existing, evs, version) {
				continue
			}

			sig := synthesizer.MakeSignature(version, evs, sectionTitlesUsed(evs, doc), warns)

			proposals = append(proposals, SynthesizerProposal{
				Doc:              doc,
				SynthesizerName:  s.Name(),
				Block:            block,
				Evidence:         evs,
				Warnings:         warns,
				Signature:        sig,
				CandidateKey:     c.Key,
				RenderedMarkdown: RenderBlockMarkdown(doc, block),
			})
		}
	}
	return proposals, nil
}

// ApplySynthesizerProposal returns the modified doc body and frontmatter
// after applying p. The caller is responsible for persisting both via
// storage.Marshal + the write pipeline (atomic write, backup).
//
// The function does NOT mutate p.Doc — it works on copies so callers can
// preview the result safely (used by the dry-run path).
func ApplySynthesizerProposal(p SynthesizerProposal) (string, domain.DocMeta, error) {
	if p.Doc == nil {
		return "", domain.DocMeta{}, fmt.Errorf("synthesizer proposal: nil doc")
	}

	newBody, err := insertBlockAfterHeading(p.Doc, p.Block, p.RenderedMarkdown)
	if err != nil {
		return "", domain.DocMeta{}, err
	}

	signatures := make(map[string]synthesizer.Signature, len(p.Doc.Signatures)+1)
	for k, v := range p.Doc.Signatures {
		signatures[k] = v
	}
	signatures[p.SynthesizerName] = p.Signature

	newMeta := synthesizer.MetaFromDoc(p.Doc, signatures)
	return newBody, newMeta, nil
}

// RenderBlockMarkdown produces the markdown snippet to insert into the doc.
// The heading level matches the doc's parent section level + 1 so the
// generated subsection nests cleanly under the section identified by
// block.InsertAfterHeading.
//
// Block format:
//
//	#### <Block.Title>
//
//	```<Block.Language>
//	<Block.Content>
//	```
//
//	- <Block.Notes[0]>
//	- <Block.Notes[1]>
func RenderBlockMarkdown(doc *synthesizer.Doc, block synthesizer.Block) string {
	level := 3 // sensible default for ## parent
	for _, sec := range doc.Sections {
		if sec.Heading == block.InsertAfterHeading {
			level = sec.Level + 1
			break
		}
	}
	if level > 6 {
		level = 6
	}

	var b strings.Builder
	b.WriteString(strings.Repeat("#", level))
	b.WriteByte(' ')
	b.WriteString(block.Title)
	b.WriteString("\n\n")

	if block.Language != "" {
		b.WriteString("```")
		b.WriteString(block.Language)
		b.WriteByte('\n')
	} else {
		b.WriteString("```\n")
	}
	b.WriteString(block.Content)
	if !strings.HasSuffix(block.Content, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("```\n")

	for _, note := range block.Notes {
		b.WriteString("\n- ")
		b.WriteString(note)
	}
	if len(block.Notes) > 0 {
		b.WriteByte('\n')
	}

	return b.String()
}

// insertBlockAfterHeading places rendered immediately after the section
// matching block.InsertAfterHeading. It walks the parsed sections (which
// already know their EndLine), then splices rendered into the body lines
// at that boundary.
//
// When the heading is not found, returns an error - this path indicates a
// drift between Detect's heading reference and the doc's current state.
func insertBlockAfterHeading(doc *synthesizer.Doc, block synthesizer.Block, rendered string) (string, error) {
	if block.InsertAfterHeading == "" {
		// Append at end of doc.
		return strings.TrimRight(doc.Body, "\n") + "\n\n" + rendered, nil
	}

	var section *synthesizer.Section
	for i := range doc.Sections {
		if doc.Sections[i].Heading == block.InsertAfterHeading {
			section = &doc.Sections[i]
			break
		}
	}
	if section == nil {
		return "", fmt.Errorf("synthesizer: insertion heading %q not found in doc", block.InsertAfterHeading)
	}

	// Defensively normalize CRLF before splitting so a caller that built a
	// Doc outside ParseDoc (e.g., in tests or future integrations) still
	// gets byte-clean splits (code review finding #8).
	body := strings.ReplaceAll(doc.Body, "\r\n", "\n")
	rawLines := strings.Split(body, "\n")
	insertAfter := section.EndLine // 1-based, EndLine is inclusive last line
	if insertAfter > len(rawLines) {
		insertAfter = len(rawLines)
	}
	if insertAfter < 0 {
		insertAfter = 0
	}

	before := strings.Join(rawLines[:insertAfter], "\n")
	after := strings.Join(rawLines[insertAfter:], "\n")

	var sep string
	if before != "" && !strings.HasSuffix(before, "\n\n") {
		if strings.HasSuffix(before, "\n") {
			sep = "\n"
		} else {
			sep = "\n\n"
		}
	}
	tail := ""
	if after != "" {
		tail = "\n" + after
	}
	return before + sep + rendered + tail, nil
}

// sectionTitlesUsed gathers the unique source section titles touched by
// evs for the Signature.Sections audit field. Determined by mapping each
// Evidence.Line back to its enclosing section.
func sectionTitlesUsed(evs []synthesizer.Evidence, doc *synthesizer.Doc) []string {
	if len(evs) == 0 || doc == nil {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, ev := range evs {
		title := sectionTitleAtLine(doc, ev.Line)
		if title == "" {
			continue
		}
		if _, dup := seen[title]; dup {
			continue
		}
		seen[title] = struct{}{}
		out = append(out, title)
	}
	return out
}

func sectionTitleAtLine(doc *synthesizer.Doc, line int) string {
	var current string
	for _, sec := range doc.Sections {
		if sec.StartLine > line {
			break
		}
		if sec.StartLine <= line && (sec.EndLine == 0 || sec.EndLine >= line) {
			current = sec.Title
		}
	}
	return current
}

func lookupSynthesizerVersion(cfg config.SynthesizersConfig, name string) string {
	return synthesizerVersionFromBag(cfg.PerSynthesizer, name)
}
