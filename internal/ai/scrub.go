// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import "regexp"

var sensitivePattern = regexp.MustCompile(`(?i)(authorization|api[_-]?key|secret|token|bearer)\s*[:=]\s*\S+`)

// scrubSensitive removes potential API keys or tokens from error text.
func scrubSensitive(s string) string {
	return sensitivePattern.ReplaceAllString(s, "$1=[REDACTED]")
}
