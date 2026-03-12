package storage

import (
	"bytes"
	"fmt"
	"strings"

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

	// Find closing delimiter
	rest := content[4:] // skip opening "---\n"
	idx := strings.Index(rest, "---\n")
	if idx < 0 {
		return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal: missing closing front matter delimiter")
	}

	yamlPart := rest[:idx]
	body := rest[idx+4:]

	var meta domain.DocMeta
	if err := yaml.Unmarshal([]byte(yamlPart), &meta); err != nil {
		return domain.DocMeta{}, "", fmt.Errorf("storage: unmarshal front matter: %w", err)
	}

	return meta, body, nil
}
