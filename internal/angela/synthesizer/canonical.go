// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CanonicalJSON renders v as JSON in a byte-stable, deterministic form. The
// output preserves the order of keys provided through the keys argument
// (rather than alphabetical sorting) so that synthesizers can present
// required fields before optional ones, while still guaranteeing identical
// output across runs for identical inputs.
//
// Strict format guarantees:
//   - 2-space indent (no tabs).
//   - LF line endings (no CRLF).
//   - No trailing whitespace.
//   - Trailing newline at end of output.
//   - Sub-maps keyed by string are recursed in their own declared order
//     when wrapped via OrderedMap; otherwise standard json.Marshal applies.
//
// CanonicalJSON is the cornerstone of invariant I6 (idempotency): two runs
// on identical evidence MUST produce byte-identical output, and that output
// must remain stable across Go versions and platform line endings.
func CanonicalJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return "", fmt.Errorf("canonical json: %w", err)
	}
	out := buf.String()
	out = stripTrailingSpaces(out)
	return out, nil
}

// OrderedMap is a key-ordered string-keyed map used to render JSON objects
// in declared (not alphabetical) order. The standard library's json package
// sorts map keys alphabetically; OrderedMap implements json.Marshaler to
// emit keys in insertion order.
//
// Use it when the order conveys meaning to the reader (required fields
// before optional, identity fields before metadata).
type OrderedMap struct {
	keys   []string
	values map[string]any
}

// NewOrderedMap returns an empty OrderedMap with capacity for size keys.
func NewOrderedMap(size int) *OrderedMap {
	return &OrderedMap{
		keys:   make([]string, 0, size),
		values: make(map[string]any, size),
	}
}

// Set adds or replaces the value at key. Re-setting an existing key keeps
// its original position - it does NOT move it to the end. This matches
// what most synthesizers want: stable position regardless of update order.
func (m *OrderedMap) Set(key string, value any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

// Get returns the value at key and whether the key was set.
func (m *OrderedMap) Get(key string) (any, bool) {
	v, ok := m.values[key]
	return v, ok
}

// Keys returns a copy of the keys in declared order.
func (m *OrderedMap) Keys() []string {
	out := make([]string, len(m.keys))
	copy(out, m.keys)
	return out
}

// Len returns the number of entries.
func (m *OrderedMap) Len() int {
	return len(m.keys)
}

// MarshalJSON implements json.Marshaler. It emits keys in declared order
// using the same indentation conventions as CanonicalJSON's pretty-print.
// Note: nested OrderedMap values are recursively respected; nested standard
// maps use json's default alphabetical sort. Mix only when needed.
func (m *OrderedMap) MarshalJSON() ([]byte, error) {
	if m == nil || len(m.keys) == 0 {
		return []byte("{}"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, fmt.Errorf("orderedmap key %q: %w", k, err)
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		valJSON, err := json.Marshal(m.values[k])
		if err != nil {
			return nil, fmt.Errorf("orderedmap value for %q: %w", k, err)
		}
		buf.Write(valJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// stripTrailingSpaces removes trailing whitespace from each line. The
// json.Encoder pretty-print never emits trailing spaces in current Go
// versions, but defensively normalizing protects against future format
// drift that would silently break I6.
func stripTrailingSpaces(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "\n")
}

// SortedStrings returns a sorted copy of in. Convenience for callers that
// need stable list rendering for hashes or signature fields.
func SortedStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}
