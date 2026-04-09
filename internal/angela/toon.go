// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"fmt"
	"strings"
)

// escapeTOON escapes a field value for TOON serialization.
// Order: (1) newlines → spaces, (2) all \ → \\, (3) | → \|
func escapeTOON(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "|", `\|`)
	return s
}

// SerializeTOON produces a pipe-separated corpus + signals block for review prompts.
// Headers are declared once per section. Each data row is pipe-separated.
func SerializeTOON(summaries []DocSummary, signals *CorpusSignals) string {
	var sb strings.Builder

	// Corpus section
	sb.WriteString("corpus:\n")
	sb.WriteString("filename|type|date|tags|branch|scope|summary\n")
	for _, doc := range summaries {
		tags := strings.Join(doc.Tags, ",")
		fmt.Fprintf(&sb, "%s|%s|%s|%s|%s|%s|%s\n",
			escapeTOON(sanitizePromptContent(doc.Filename)),
			escapeTOON(sanitizePromptContent(doc.Type)),
			escapeTOON(sanitizePromptContent(doc.Date)),
			escapeTOON(sanitizePromptContent(tags)),
			escapeTOON(sanitizePromptContent(doc.Branch)),
			escapeTOON(sanitizePromptContent(doc.Scope)),
			escapeTOON(sanitizePromptContent(doc.Summary)),
		)
	}

	// Signals section
	hasSignals := signals != nil && (len(signals.PotentialPairs) > 0 || len(signals.IsolatedDocs) > 0 || len(signals.UnconsolidatedScopes) > 0)
	if hasSignals {
		sb.WriteString("signals:\n")
		sb.WriteString("signal_type|docs|detail\n")
		for _, p := range signals.PotentialPairs {
			docs := fmt.Sprintf("%s,%s", escapeTOON(p.DocA), escapeTOON(p.DocB))
			detail := fmt.Sprintf("type:%s, tags:%s, %dd apart",
				escapeTOON(p.Type), escapeTOON(p.Tags), p.DaysDiff)
			fmt.Fprintf(&sb, "contradiction|%s|%s\n", docs, detail)
		}
		for _, doc := range signals.IsolatedDocs {
			fmt.Fprintf(&sb, "isolated|%s|no shared tags\n", escapeTOON(doc))
		}
		for _, sg := range signals.UnconsolidatedScopes {
			scopeDocs := strings.Join(signals.ScopeClusters[sg.Scope], ",")
			fmt.Fprintf(&sb, "unconsolidated|%s|scope:%s, %d docs, no summary\n",
				escapeTOON(scopeDocs), escapeTOON(sg.Scope), sg.DocCount)
		}
	}

	return sb.String()
}

// SerializeTOONWithVHS extends SerializeTOON to include VHS cross-reference signals.
// These signals help the AI reviewer detect documentation ↔ demo inconsistencies.
func SerializeTOONWithVHS(summaries []DocSummary, signals *CorpusSignals, vhs *VHSSignals) string {
	base := SerializeTOON(summaries, signals)

	if vhs == nil {
		return base
	}

	hasVHS := len(vhs.OrphanTapes) > 0 || len(vhs.OrphanGIFs) > 0 || len(vhs.CommandMismatches) > 0
	if !hasVHS {
		return base
	}

	var sb strings.Builder
	sb.WriteString(base)
	sb.WriteString("vhs_signals:\n")
	sb.WriteString("signal_type|source|detail\n")

	for _, tape := range vhs.OrphanTapes {
		fmt.Fprintf(&sb, "orphan_tape|%s|output GIF not referenced in docs\n", escapeTOON(tape))
	}
	for _, ref := range vhs.OrphanGIFs {
		fmt.Fprintf(&sb, "orphan_gif|%s|%s references non-existent tape output\n",
			escapeTOON(ref.DocFilename), escapeTOON(ref.GIFPath))
	}
	for _, mm := range vhs.CommandMismatches {
		fmt.Fprintf(&sb, "command_mismatch|%s|%s (%s)\n",
			escapeTOON(mm.TapeFile), escapeTOON(mm.Command), escapeTOON(mm.Reason))
	}

	return sb.String()
}
