// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"strings"
	"unicode"
)

// maxSanitizedLen is the maximum length for sanitized strings to prevent
// oversized arguments that could cause process launch failures.
const maxSanitizedLen = 500

// sanitizeForShell removes or replaces characters that are dangerous in shell
// contexts. Used as a defense-in-depth layer on top of proper quoting.
// Strips null bytes and control characters (except common whitespace).
// Truncates to maxSanitizedLen to prevent oversized arguments.
func sanitizeForShell(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	count := 0
	for _, r := range s {
		if count >= maxSanitizedLen {
			break
		}
		if r == 0 {
			continue // strip null bytes
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue // strip control chars except newline/tab
		}
		b.WriteRune(r)
		count++
	}
	return b.String()
}

// sanitizeCommitHash strips everything except hex chars from a commit hash.
// Git hashes are always [0-9a-f], so anything else is suspicious.
func sanitizeCommitHash(hash string) string {
	var b strings.Builder
	b.Grow(len(hash))
	for _, r := range hash {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeForJSON escapes a string for embedding in a JSON string value.
// Handles backslash, double quotes, and control characters.
func escapeForJSON(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				// Control character — skip
				continue
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// escapeAppleScript escapes a string for embedding in AppleScript double-quoted strings.
// Escapes backslashes, double quotes, and replaces newlines with spaces
// (AppleScript does not allow raw newlines inside "..." strings).
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

// escapePowerShell escapes a string for embedding in PowerShell single-quoted strings.
// In PowerShell, single quotes are escaped by doubling them.
func escapePowerShell(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// escapeXML escapes a string for safe embedding in XML/XAML attributes.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// coalesce returns the first non-empty string.
func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// bashQuote wraps a string in single quotes for safe bash embedding.
// Single quotes inside the string are handled via the quote-break-quote pattern: 'foo'\''bar'
func bashQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// shellQuote wraps a path in single quotes for shell safety.
func shellQuote(s string) string {
	return bashQuote(s)
}
