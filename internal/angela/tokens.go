// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import "math"

// ResolveMaxTokens returns the max_tokens cap for a given Angela mode.
// For polish mode, the cap is dynamic based on document word count.
// Unknown modes default to 2000.
func ResolveMaxTokens(mode string, docWordCount int) int {
	switch mode {
	case "polish":
		if docWordCount <= 0 {
			return 2000
		}
		raw := float64(docWordCount) * 1.3 * 1.5 // ≈ words × 1.95 (1.3 token/word ratio × 1.5 reformulation margin)
		capped := math.Min(raw, 4096)
		floored := math.Max(capped, 512)
		return int(floored)
	case "review":
		return 1500
	case "render": // Wire via domain.WithMaxTokens(ResolveMaxTokens("render", 0)) when implementing lore render
		return 512
	case "ask": // Wire via domain.WithMaxTokens(ResolveMaxTokens("ask", 0)) when implementing lore ask
		return 1024
	default:
		return 2000
	}
}
