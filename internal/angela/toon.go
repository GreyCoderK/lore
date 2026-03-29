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
	sb.WriteString("filename|type|date|tags|summary\n")
	for _, doc := range summaries {
		tags := strings.Join(doc.Tags, ",")
		fmt.Fprintf(&sb, "%s|%s|%s|%s|%s\n",
			escapeTOON(sanitizePromptContent(doc.Filename)),
			escapeTOON(sanitizePromptContent(doc.Type)),
			escapeTOON(sanitizePromptContent(doc.Date)),
			escapeTOON(sanitizePromptContent(tags)),
			escapeTOON(sanitizePromptContent(doc.Summary)),
		)
	}

	// Signals section
	if signals != nil && (len(signals.PotentialPairs) > 0 || len(signals.IsolatedDocs) > 0) {
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
	}

	return sb.String()
}
