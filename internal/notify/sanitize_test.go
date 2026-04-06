// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeCommitHash(t *testing.T) {
	assert.Equal(t, "abc1234", sanitizeCommitHash("abc1234"))
	assert.Equal(t, "abc1234f", sanitizeCommitHash("abc1234; rm -rf /")) // keeps hex chars (f from -rf)
	assert.Equal(t, "ac", sanitizeCommitHash("$(malicious)"))            // keeps a, c (hex chars)
	assert.Equal(t, "deadbeef", sanitizeCommitHash("dead beef"))         // strips space
	assert.Equal(t, "", sanitizeCommitHash("!@#$%"))                     // all non-hex stripped
}

func TestSanitizeForShell(t *testing.T) {
	assert.Equal(t, "hello world", sanitizeForShell("hello world"))
	assert.Equal(t, "helloworld", sanitizeForShell("hello\x00world"))    // strips null, no separator
	assert.Equal(t, "helloworld", sanitizeForShell("hello\x01world"))    // strips control char
	assert.Equal(t, "hello\nworld", sanitizeForShell("hello\nworld"))    // keeps newline
	assert.Equal(t, "hello\tworld", sanitizeForShell("hello\tworld"))    // keeps tab
}

func TestEscapeForJSON(t *testing.T) {
	assert.Equal(t, `hello world`, escapeForJSON("hello world"))
	assert.Equal(t, `say \\\"hi\\\"`, escapeForJSON(`say \"hi\"`))
	assert.Equal(t, `line1\nline2`, escapeForJSON("line1\nline2"))
	assert.Equal(t, `tab\there`, escapeForJSON("tab\there"))
	assert.Equal(t, `cr\r`, escapeForJSON("cr\r"))
}

func TestEscapeAppleScript(t *testing.T) {
	assert.Equal(t, `hello`, escapeAppleScript("hello"))
	assert.Equal(t, `say \"hi\"`, escapeAppleScript(`say "hi"`))
	assert.Equal(t, `path\\to\\file`, escapeAppleScript(`path\to\file`))
	assert.Equal(t, "line1 line2", escapeAppleScript("line1\nline2"))   // newline → space
	assert.Equal(t, "line1  line2", escapeAppleScript("line1\r\nline2")) // CRLF → two spaces
}

func TestEscapePowerShell(t *testing.T) {
	assert.Equal(t, "hello", escapePowerShell("hello"))
	assert.Equal(t, "it''s", escapePowerShell("it's"))
	assert.Equal(t, "it''''s", escapePowerShell("it''s"))
}

func TestEscapeXML(t *testing.T) {
	assert.Equal(t, "hello", escapeXML("hello"))
	assert.Equal(t, "&amp;&lt;&gt;&quot;&apos;", escapeXML(`&<>"'`))
	assert.Equal(t, "fix: &lt;script&gt;alert&lt;/script&gt;", escapeXML("fix: <script>alert</script>"))
}

func TestBashQuote(t *testing.T) {
	assert.Equal(t, "'hello'", bashQuote("hello"))
	assert.Equal(t, "'/path with spaces'", bashQuote("/path with spaces"))
	assert.Equal(t, "'/it'\\''s here'", bashQuote("/it's here"))
}

func TestShellQuote(t *testing.T) {
	assert.Equal(t, "'/usr/local/bin/lore'", shellQuote("/usr/local/bin/lore"))
}

func TestCoalesce(t *testing.T) {
	assert.Equal(t, "first", coalesce("first", "second"))
	assert.Equal(t, "second", coalesce("", "second"))
	assert.Equal(t, "third", coalesce("", "", "third"))
	assert.Equal(t, "", coalesce("", "", ""))
	assert.Equal(t, "", coalesce())
}

func TestSanitizeForShell_Truncation(t *testing.T) {
	long := strings.Repeat("a", 600)
	result := sanitizeForShell(long)
	assert.Equal(t, maxSanitizedLen, len(result))
}

func TestEscapeForJSON_ControlChars(t *testing.T) {
	// \b and \f escapes
	assert.Equal(t, `\b`, escapeForJSON("\b"))
	assert.Equal(t, `\f`, escapeForJSON("\f"))
	// Other control chars (< 0x20) are stripped
	assert.Equal(t, "", escapeForJSON("\x01"))
	assert.Equal(t, "", escapeForJSON("\x1f"))
}
