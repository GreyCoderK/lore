// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"gopkg.in/yaml.v3"
)

var hexHash = regexp.MustCompile(`^[0-9a-f]+$`)

var separator = []byte("---\n")

// utf8BOM is the 3-byte UTF-8 byte-order mark some editors (legacy
// Notepad, older VSCode with "files.encoding": "utf8bom") prepend
// invisibly to saved files. A BOM breaks `bytes.HasPrefix(data,
// "---\n")` delimiter detection even when the frontmatter is
// otherwise valid. Stripping the BOM before delimiter checks makes
// both ExtractFrontmatter and unmarshalInternal BOM-agnostic, which
// prevents doctor from mis-classifying a BOM-prefixed file as
// "frontmatter missing" and double-prepending a synthesized FM.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Typed errors returned by ExtractFrontmatter. Callers use errors.Is to
// distinguish between a truly-absent frontmatter block and one that is
// present but unparseable.
var (
	ErrFrontmatterMissing   = errors.New("storage: frontmatter missing")
	ErrFrontmatterMalformed = errors.New("storage: frontmatter malformed")
)

// ExtractFrontmatter captures the frontmatter bytes verbatim and returns
// them alongside the body bytes. Unlike Unmarshal, it does not parse the
// YAML into a DocMeta struct — it returns the raw byte regions.
//
// This is the foundation of invariant I24 (polish FM byte-identity): a
// caller that subsequently writes fmBytes to disk gets back the exact
// bytes it received, without YAML re-serialization (which would lose
// key ordering, comment lines, and quote styles).
//
// CRLF normalization matches Unmarshal: if the input contains "\r\n",
// it is converted to "\n" before delimiter detection, and the returned
// byte slices reflect the normalized form. The codebase is LF-first
// (enforced via .gitattributes) so the normalized form is the
// canonical representation.
//
// Postcondition on success: fmBytes + bodyBytes == normalized_input.
//
// Error semantics:
//   - ErrFrontmatterMissing    — no "---\n" opening delimiter.
//   - ErrFrontmatterMalformed  — opening delimiter present but the
//     YAML block is unclosed, empty, or unparseable. The underlying
//     yaml.Unmarshal error is wrapped via fmt.Errorf("%w: %v", ...) so
//     callers can surface the parse detail while still matching on the
//     sentinel via errors.Is.
func ExtractFrontmatter(data []byte) (fmBytes []byte, bodyBytes []byte, err error) {
	const maxDocSize = 10 << 20 // 10 MB — same cap as Unmarshal
	if len(data) > maxDocSize {
		return nil, nil, fmt.Errorf("storage: document too large (%d bytes, max %d)", len(data), maxDocSize)
	}

	content := string(data)
	// Strip UTF-8 BOM before any delimiter detection so files saved
	// with BOM ("\xEF\xBB\xBF---\n...") still classify correctly.
	if strings.HasPrefix(content, string(utf8BOM)) {
		content = content[len(utf8BOM):]
	}
	if strings.Contains(content, "\r\n") {
		content = strings.ReplaceAll(content, "\r\n", "\n")
	}
	normalized := []byte(content)

	if !bytes.HasPrefix(normalized, separator) {
		return nil, nil, ErrFrontmatterMissing
	}

	rest := normalized[len(separator):] // skip opening "---\n"

	// Empty YAML section: "---\n---\n..." — matches existing Unmarshal
	// behavior of rejecting empty frontmatter.
	if bytes.HasPrefix(rest, separator) {
		return nil, nil, ErrFrontmatterMalformed
	}

	// Look for the closing delimiter on its own line: "\n---\n".
	// This mirrors Unmarshal's detection logic.
	idx := bytes.Index(rest, []byte("\n---\n"))
	if idx < 0 {
		return nil, nil, ErrFrontmatterMalformed
	}

	// Compute the byte slices on the normalized buffer.
	//
	//   normalized[:len(separator)]                    — opening "---\n"
	//   normalized[len(separator):len(separator)+idx]  — YAML body
	//   normalized[len(separator)+idx:fmEnd]           — "\n---\n" closer
	//
	// where fmEnd = len(separator) + idx + len("\n---\n") = 4 + idx + 5
	fmEnd := len(separator) + idx + len("\n---\n")

	yamlPart := normalized[len(separator) : len(separator)+idx]
	if len(bytes.TrimSpace(yamlPart)) == 0 {
		return nil, nil, ErrFrontmatterMalformed
	}

	// Parse-check the YAML. We discard the parsed value — verifying
	// parseability is the only concern here. Callers that need the
	// typed DocMeta use Unmarshal.
	var probe map[string]interface{}
	if yerr := yaml.Unmarshal(yamlPart, &probe); yerr != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrFrontmatterMalformed, yerr)
	}

	fmBytes = normalized[:fmEnd]
	bodyBytes = normalized[fmEnd:]
	return fmBytes, bodyBytes, nil
}

// Marshal produces "---\n{yaml}\n---\n{body}" from DocMeta and body.
func Marshal(meta domain.DocMeta, body string) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("storage: marshal front matter: %w", err)
	}

	var buf bytes.Buffer
	buf.Write(separator)
	buf.Write(yamlBytes)
	buf.Write(separator)
	buf.WriteString(body)

	return buf.Bytes(), nil
}

// UnmarshalPermissive parses front matter + body from a document WITHOUT
// running ValidateMeta. Used by PlainCorpusStore (standalone mode) so
// external docs sites can have partial front matter (e.g. just `type` and
// `date`, or arbitrary types like "blog-post") without being rejected and
// silently downgraded to a synthetic "note" meta.
//
// Strict validation (ValidateMeta) is only appropriate for lore-managed
// corpora where the commit-capture workflow is guaranteed to fill every
// required field.
func UnmarshalPermissive(data []byte) (domain.DocMeta, string, error) {
	return unmarshalInternal(data, false)
}

// Unmarshal parses front matter + body from a document.
func Unmarshal(data []byte) (domain.DocMeta, string, error) {
	return unmarshalInternal(data, true)
}

// unmarshalInternal is the shared implementation. When validate is true,
// ValidateMeta enforces the full lore contract (type+date+status required);
// when false, any parseable YAML is accepted.
func unmarshalInternal(data []byte, validate bool) (domain.DocMeta, string, error) {
	const maxDocSize = 10 << 20 // 10 MB
	if len(data) > maxDocSize {
		return domain.DocMeta{}, "", fmt.Errorf("storage: document too large (%d bytes, max %d)", len(data), maxDocSize)
	}

	// Normalize CRLF to LF for Windows compatibility
	content := string(data)
	// Strip BOM before any delimiter check (see ExtractFrontmatter).
	if strings.HasPrefix(content, string(utf8BOM)) {
		content = content[len(utf8BOM):]
	}
	if strings.Contains(content, "\r\n") {
		content = strings.ReplaceAll(content, "\r\n", "\n")
	}

	if !strings.HasPrefix(content, "---\n") {
		return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal: missing front matter delimiter")
	}

	// Find closing delimiter — it must appear on its own line.
	// Search for "\n---\n" to avoid matching "---\n" embedded inside a YAML value.
	// Special case: empty front matter where rest begins immediately with "---\n".
	rest := content[4:] // skip opening "---\n"
	var yamlPart, body string
	if strings.HasPrefix(rest, "---\n") {
		// Empty YAML section
		body = rest[4:]
	} else {
		// NOTE: We match the FIRST "\n---\n" after the opening delimiter.
		// This means a Markdown horizontal rule (---) on its own line inside
		// the body will be mistakenly treated as the closing front matter
		// delimiter, truncating everything after it. This is standard behavior
		// consistent with Jekyll, Hugo, and other static site generators.
		// Authors should use *** or ___ for horizontal rules in documents
		// that have YAML front matter.
		idx := strings.Index(rest, "\n---\n")
		if idx < 0 {
			return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal: missing closing front matter delimiter")
		}
		yamlPart = rest[:idx]
		body = rest[idx+5:] // skip "\n---\n"
	}

	if strings.TrimSpace(yamlPart) == "" {
		return domain.DocMeta{}, "", fmt.Errorf("storage: empty front matter")
	}

	var meta domain.DocMeta
	if err := yaml.Unmarshal([]byte(yamlPart), &meta); err != nil {
		return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal front matter: %w", err)
	}

	if validate {
		if err := ValidateMeta(meta); err != nil {
			return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal: %w", err)
		}
	}

	return meta, body, nil
}

// ValidateMeta checks that required fields (type, date, status) are present,
// that type is a recognized DocType constant, and that date is in YYYY-MM-DD format.
func ValidateMeta(meta domain.DocMeta) error {
	if meta.Type == "" {
		return fmt.Errorf("storage: validate meta: type required")
	}
	if !domain.ValidDocType(meta.Type) {
		return fmt.Errorf("storage: validate meta: unknown type %q (must be one of: %s)", meta.Type, strings.Join(domain.DocTypeNames(), ", "))
	}
	if meta.Date == "" {
		return fmt.Errorf("storage: validate meta: date required")
	}
	if _, err := time.Parse("2006-01-02", meta.Date); err != nil {
		return fmt.Errorf("storage: validate meta: date must be YYYY-MM-DD, got %q", meta.Date)
	}
	if meta.Status == "" {
		return fmt.Errorf("storage: validate meta: status required")
	}
	if meta.Commit != "" {
		if !hexHash.MatchString(meta.Commit) {
			return fmt.Errorf("storage: validate meta: commit must be a hex hash, got %q", meta.Commit)
		}
	}
	return nil
}
