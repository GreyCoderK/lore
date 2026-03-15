package storage

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/museigen/lore/internal/domain"
	"gopkg.in/yaml.v3"
)

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
	// Normalize CRLF to LF for Windows compatibility
	content := strings.ReplaceAll(string(data), "\r\n", "\n")

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
		idx := strings.Index(rest, "\n---\n")
		if idx < 0 {
			return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal: missing closing front matter delimiter")
		}
		yamlPart = rest[:idx]
		body = rest[idx+5:] // skip "\n---\n"
	}

	var meta domain.DocMeta
	if err := yaml.Unmarshal([]byte(yamlPart), &meta); err != nil {
		return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal front matter: %w", err)
	}

	return meta, body, nil
}

// validDocTypes is the whitelist of accepted document type values.
var validDocTypes = map[string]bool{
	domain.DocTypeDecision: true,
	domain.DocTypeFeature:  true,
	domain.DocTypeBugfix:   true,
	domain.DocTypeRefactor: true,
	domain.DocTypeRelease:  true,
	domain.DocTypeNote:     true,
}

// ValidateMeta checks that required fields (type, date, status) are present,
// that type is a recognized DocType constant, and that date is in YYYY-MM-DD format.
func ValidateMeta(meta domain.DocMeta) error {
	if meta.Type == "" {
		return fmt.Errorf("storage: validate meta: type required")
	}
	if !validDocTypes[meta.Type] {
		return fmt.Errorf("storage: validate meta: unknown type %q (must be one of: decision, feature, bugfix, refactor, release, note)", meta.Type)
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
	return nil
}
