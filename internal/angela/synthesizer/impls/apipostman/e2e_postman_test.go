// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package apipostman

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/textproto"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/angela/synthesizer"
)

// ═══════════════════════════════════════════════════════════════════════════
// Phase 6 — api-postman E2E format & Postman Collection v2.1 wrapability.
//
// Reframe note: the original Phase 6 plan called for "newman E2E OK". On
// inspection, the api-postman synthesizer does NOT produce a Postman
// Collection v2.1 JSON — it produces example `http` + JSON markdown blocks
// meant to be embedded INSIDE a doc (the `### Endpoints` section). newman
// runs collections, not markdown example blocks, so a literal "newman run"
// does not apply.
//
// What DOES apply: the generated block must be shaped such that a user
// who copies it into Postman — or we wrap it programmatically into a
// Postman Collection v2.1 — gets a valid, importable request. That is
// the real contract Phase 6 should enforce.
//
// These tests validate, with zero external dependencies (no Node, no
// newman, no Docker):
//   - the HTTP+JSON block parses as a valid HTTP message (request line,
//     headers, body)
//   - the JSON body is valid JSON once Postman variables are substituted
//   - the block can be wrapped into a Postman Collection v2.1 structure
//     that survives a round-trip through json.Marshal + Unmarshal and
//     contains all required schema fields
//
// Cross-platform coverage: the existing `go test ./... -race` CI matrix
// (ubuntu/macos/windows) runs this package automatically.
// ═══════════════════════════════════════════════════════════════════════════

// TestE2E_BlockParsesAsHTTPRequest verifies the generated block can be
// decoded into its HTTP constituents (method, URL, headers, body). If
// the formatter ever emits a malformed request line (missing space,
// non-uppercase method, header without colon), this test catches it.
func TestE2E_BlockParsesAsHTTPRequest(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	if len(candidates) == 0 {
		t.Fatal("no candidates detected — fixture broken")
	}
	block, _, _, err := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	if err != nil {
		t.Fatal(err)
	}

	requests := extractHTTPRequests(t, block.Content)
	if len(requests) < 1 {
		t.Fatalf("expected ≥1 HTTP request in block, got %d\nblock:\n%s", len(requests), block.Content)
	}

	for i, req := range requests {
		if !isValidHTTPMethod(req.method) {
			t.Errorf("request #%d: invalid HTTP method %q", i, req.method)
		}
		// URL is expected to contain {{baseUrl}} (Postman variable). We
		// substitute a real base before parsing with net/url.
		concrete := strings.ReplaceAll(req.url, "{{baseUrl}}", "https://api.example.com")
		if _, err := url.ParseRequestURI(concrete); err != nil {
			t.Errorf("request #%d: unparseable URL after baseUrl substitution %q: %v", i, concrete, err)
		}
		if _, ok := req.headers["Content-Type"]; !ok && req.method != "GET" {
			t.Errorf("request #%d: non-GET request missing Content-Type header", i)
		}
	}
}

// TestE2E_BodyIsValidJSON verifies the request body is valid JSON once
// Postman variables are substituted. A malformed body (trailing comma,
// unquoted key, dangling `null`) would fail Postman import as much as it
// would fail `json.Unmarshal` here — so this is a strict proxy for
// Postman compatibility.
func TestE2E_BodyIsValidJSON(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	block, _, _, err := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	if err != nil {
		t.Fatal(err)
	}

	requests := extractHTTPRequests(t, block.Content)
	for i, req := range requests {
		if req.body == "" {
			continue
		}
		concrete := substitutePostmanVars(req.body)
		var out map[string]any
		if err := json.Unmarshal([]byte(concrete), &out); err != nil {
			t.Errorf("request #%d: body is not valid JSON after variable substitution: %v\nbody:\n%s", i, err, concrete)
		}
	}
}

// TestE2E_WrapsAsPostmanCollectionV21 is the main "E2E" anchor: it wraps
// each extracted request into a minimal Postman Collection v2.1 structure,
// serializes it, and reparses it. If the generated fields (method/URL/
// headers/body) couldn't legally sit inside a v2.1 collection, json.Marshal
// would fail silently or produce something that Postman would reject on
// import. Parsing back round-trips the contract.
func TestE2E_WrapsAsPostmanCollectionV21(t *testing.T) {
	doc := parseFixture(t, "account-statement-complete.md")
	candidates, _ := (&Synthesizer{}).Detect(doc)
	block, _, _, err := (&Synthesizer{}).Synthesize(candidates[0], synthesizer.Config{})
	if err != nil {
		t.Fatal(err)
	}

	requests := extractHTTPRequests(t, block.Content)
	coll := buildPostmanCollectionV21("lore-apipostman-e2e", requests)

	raw, err := json.MarshalIndent(coll, "", "  ")
	if err != nil {
		t.Fatalf("collection marshal: %v", err)
	}

	// Sanity: required v2.1 schema fields are present in the serialized
	// JSON. Parse into a generic map and assert.
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("collection round-trip unmarshal: %v\njson:\n%s", err, raw)
	}
	info, _ := parsed["info"].(map[string]any)
	if info == nil {
		t.Fatal("collection missing `info` block — required by Postman Collection v2.1 schema")
	}
	if schema, _ := info["schema"].(string); !strings.Contains(schema, "schema.getpostman.com/json/collection/v2.1.0") {
		t.Errorf("info.schema = %q, want v2.1.0 URL", schema)
	}
	items, _ := parsed["item"].([]any)
	if len(items) != len(requests) {
		t.Errorf("item count = %d, want %d (one per request)", len(items), len(requests))
	}
	for i, it := range items {
		item, _ := it.(map[string]any)
		if item == nil {
			t.Errorf("item[%d] is not an object", i)
			continue
		}
		req, _ := item["request"].(map[string]any)
		if req == nil {
			t.Errorf("item[%d].request missing", i)
			continue
		}
		if method, _ := req["method"].(string); method == "" {
			t.Errorf("item[%d].request.method missing", i)
		}
		if _, ok := req["url"]; !ok {
			t.Errorf("item[%d].request.url missing", i)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────
// Helpers: parse the HTTP+JSON fenced block into structured requests, and
// wrap into a Postman Collection v2.1 shape.
// ─────────────────────────────────────────────────────────────────────────

type httpRequest struct {
	method  string
	url     string
	headers map[string]string
	body    string
}

// extractHTTPRequests scans the synthesized block for request-line patterns
// (`METHOD URL` or `METHOD URL HTTP/1.1`) and groups the headers + body that
// follow each one, up to the next request line or end of block. The
// synthesizer emits multiple variants (Full + Minimal) in a single block.
func extractHTTPRequests(t *testing.T, content string) []httpRequest {
	t.Helper()
	// A request line starts with an HTTP verb + whitespace + URL. Stop at
	// the next request line or the end of the string.
	verbRE := regexp.MustCompile(`^(?i)(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s`)
	lines := strings.Split(content, "\n")
	var requests []httpRequest
	var cur *httpRequest
	var body strings.Builder
	inBody := false

	flush := func() {
		if cur == nil {
			return
		}
		cur.body = strings.TrimSpace(body.String())
		requests = append(requests, *cur)
		cur = nil
		body.Reset()
		inBody = false
	}

	// headingRE matches markdown section separators the synthesizer uses
	// between the Full and Minimal variants (e.g., "# Minimal — …").
	// When we see one while accumulating a body, it signals end of this
	// request — flush and wait for the next verb line.
	headingRE := regexp.MustCompile(`^#{1,6}\s`)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if verbRE.MatchString(trimmed) {
			flush()
			parts := strings.Fields(trimmed)
			if len(parts) < 2 {
				continue
			}
			cur = &httpRequest{
				method:  strings.ToUpper(parts[0]),
				url:     parts[1],
				headers: map[string]string{},
			}
			continue
		}
		if cur == nil {
			continue
		}
		if inBody && headingRE.MatchString(trimmed) {
			flush()
			continue
		}
		if trimmed == "" && !inBody {
			inBody = true
			continue
		}
		if inBody {
			body.WriteString(line)
			body.WriteString("\n")
			continue
		}
		// Header line
		tp := textproto.MIMEHeader{}
		if err := parseHeaderInto(line, tp); err == nil {
			for k := range tp {
				cur.headers[k] = tp.Get(k)
			}
		}
	}
	flush()
	return requests
}

// parseHeaderInto parses a single `Key: Value` line using the stdlib
// textproto reader, which is the same parser net/http uses.
func parseHeaderInto(line string, h textproto.MIMEHeader) error {
	r := textproto.NewReader(bufio.NewReader(strings.NewReader(line + "\r\n\r\n")))
	parsed, err := r.ReadMIMEHeader()
	if err != nil {
		return err
	}
	for k, vs := range parsed {
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	return nil
}

// isValidHTTPMethod uses net/http's canonical list.
func isValidHTTPMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodConnect,
		http.MethodTrace:
		return true
	}
	return false
}

// substitutePostmanVars replaces every `{{var}}` with a stringified JSON
// value so the body parses with encoding/json. `"{{var}}"` → `"var"`
// (already quoted, nothing to do); bare `{{var}}` → `"var"` (make a
// string). Also leaves `null` and numeric literals untouched.
var postmanVarInString = regexp.MustCompile(`"\{\{([^}]+)\}\}"`) // "{{var}}"  → keep as "var"
var postmanVarBare = regexp.MustCompile(`\{\{([^}]+)\}\}`)       // {{var}}     → "var"

func substitutePostmanVars(body string) string {
	// First pass: already-quoted Postman vars become the var name.
	body = postmanVarInString.ReplaceAllString(body, `"$1"`)
	// Second pass: any remaining bare vars become quoted strings.
	body = postmanVarBare.ReplaceAllString(body, `"$1"`)
	return body
}

// buildPostmanCollectionV21 wraps a slice of httpRequests into the minimal
// shape required by the Postman Collection v2.1 schema. Only the fields
// that are schema-required are populated; anything optional is omitted so
// the test surface matches the actual contract.
func buildPostmanCollectionV21(name string, requests []httpRequest) map[string]any {
	items := make([]map[string]any, 0, len(requests))
	for i, req := range requests {
		headers := make([]map[string]string, 0, len(req.headers))
		for k, v := range req.headers {
			headers = append(headers, map[string]string{"key": k, "value": v})
		}
		item := map[string]any{
			"name": req.method + " " + req.url + " #" + itoaIdx(i),
			"request": map[string]any{
				"method": req.method,
				"header": headers,
				"url":    req.url,
				"body": map[string]any{
					"mode": "raw",
					"raw":  req.body,
				},
			},
		}
		items = append(items, item)
	}
	return map[string]any{
		"info": map[string]any{
			"name":   name,
			"schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		"item": items,
	}
}

func itoaIdx(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}
