// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "math"

// ResolveMaxTokens returns the max_tokens cap for a given Angela mode.
// For polish mode, the cap is dynamic based on document word count.
// If configMaxTokens > 0, it overrides the computed/default value.
// Unknown modes default to 2000.
func ResolveMaxTokens(mode string, docWordCount int, configMaxTokens ...int) int {
	// If user set angela.max_tokens in .lorerc, respect it
	if len(configMaxTokens) > 0 && configMaxTokens[0] > 0 {
		return configMaxTokens[0]
	}

	switch mode {
	case "polish":
		if docWordCount <= 0 {
			return 2000
		}
		// Document must be returned in full + enrichments (diagrams, tables).
		// Estimate: 1.3 tokens/word × 1.8 margin for added content.
		raw := float64(docWordCount) * 1.3 * 1.8
		capped := math.Min(raw, 8192)
		floored := math.Max(capped, 512)
		return int(floored)
	case "review":
		return 1500
	case "render":
		return 512
	case "ask":
		return 1024
	default:
		return 2000
	}
}
