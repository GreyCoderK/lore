// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"gopkg.in/yaml.v3"
)

var hexHash = regexp.MustCompile(`^[0-9a-f]+$`)

var separator = []byte("---\n")

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

// Unmarshal parses front matter + body from a document.
func Unmarshal(data []byte) (domain.DocMeta, string, error) {
	const maxDocSize = 10 << 20 // 10 MB
	if len(data) > maxDocSize {
		return domain.DocMeta{}, "", fmt.Errorf("storage: document too large (%d bytes, max %d)", len(data), maxDocSize)
	}

	// Normalize CRLF to LF for Windows compatibility
	content := string(data)
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

	if err := ValidateMeta(meta); err != nil {
		return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal: %w", err)
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
