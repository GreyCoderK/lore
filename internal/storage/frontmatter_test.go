// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
	"errors"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	meta := domain.DocMeta{
		Type:        "decision",
		Date:        "2026-03-07",
		Commit:      "abc1234",
		Status:      "demo",
		Tags:        []string{"auth", "jwt"},
		GeneratedBy: "lore-demo",
	}
	body := "# Test Document\n\nSome content here.\n"

	data, err := Marshal(meta, body)
	if err != nil {
		t.Fatalf("storage: marshal: %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Error("expected front matter to start with ---")
	}

	gotMeta, gotBody, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("storage: unmarshal: %v", err)
	}

	if gotMeta.Type != "decision" {
		t.Errorf("Type: got %q, want 'decision'", gotMeta.Type)
	}
	if gotMeta.Date != "2026-03-07" {
		t.Errorf("Date: got %q, want '2026-03-07'", gotMeta.Date)
	}
	if gotMeta.Status != "demo" {
		t.Errorf("Status: got %q, want 'demo'", gotMeta.Status)
	}
	if gotMeta.Commit != "abc1234" {
		t.Errorf("Commit: got %q, want 'abc1234'", gotMeta.Commit)
	}
	if len(gotMeta.Tags) != 2 {
		t.Errorf("Tags: got %d, want 2", len(gotMeta.Tags))
	}
	if gotMeta.GeneratedBy != "lore-demo" {
		t.Errorf("GeneratedBy: got %q, want 'lore-demo'", gotMeta.GeneratedBy)
	}
	if gotBody != body {
		t.Errorf("Body: got %q, want %q", gotBody, body)
	}
}

func TestMarshalUnmarshal_AllFields(t *testing.T) {
	meta := domain.DocMeta{
		Type:        "feature",
		Date:        "2026-03-10",
		Commit:      "def5678",
		Status:      "published",
		Tags:        []string{"api", "performance"},
		Related:     []string{"decision-auth-2026-03-07"},
		GeneratedBy: "hook",
		AngelaMode:  "draft",
	}
	body := "# Feature\n"

	data, err := Marshal(meta, body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	gotMeta, gotBody, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if gotMeta.Type != "feature" {
		t.Errorf("Type: got %q", gotMeta.Type)
	}
	if gotMeta.Date != "2026-03-10" {
		t.Errorf("Date: got %q", gotMeta.Date)
	}
	if gotMeta.Commit != "def5678" {
		t.Errorf("Commit: got %q", gotMeta.Commit)
	}
	if gotMeta.Status != "published" {
		t.Errorf("Status: got %q", gotMeta.Status)
	}
	if len(gotMeta.Tags) != 2 || gotMeta.Tags[0] != "api" {
		t.Errorf("Tags: got %v", gotMeta.Tags)
	}
	if len(gotMeta.Related) != 1 || gotMeta.Related[0] != "decision-auth-2026-03-07" {
		t.Errorf("Related: got %v", gotMeta.Related)
	}
	if gotMeta.GeneratedBy != "hook" {
		t.Errorf("GeneratedBy: got %q", gotMeta.GeneratedBy)
	}
	if gotMeta.AngelaMode != "draft" {
		t.Errorf("AngelaMode: got %q", gotMeta.AngelaMode)
	}
	if gotBody != body {
		t.Errorf("Body: got %q", gotBody)
	}
}

func TestMarshal_FieldOrder(t *testing.T) {
	meta := domain.DocMeta{
		Type:        "decision",
		Date:        "2026-03-07",
		Commit:      "abc1234",
		Status:      "published",
		Tags:        []string{"auth"},
		Related:     []string{"feature-x"},
		GeneratedBy: "hook",
		AngelaMode:  "draft",
	}

	data, err := Marshal(meta, "body")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	content := string(data)
	typeIdx := strings.Index(content, "type:")
	dateIdx := strings.Index(content, "date:")
	commitIdx := strings.Index(content, "commit:")
	statusIdx := strings.Index(content, "status:")
	tagsIdx := strings.Index(content, "tags:")
	relatedIdx := strings.Index(content, "related:")
	genByIdx := strings.Index(content, "generated_by:")
	angelaIdx := strings.Index(content, "angela_mode:")

	if typeIdx > dateIdx {
		t.Error("type should come before date")
	}
	if dateIdx > commitIdx {
		t.Error("date should come before commit")
	}
	if commitIdx > statusIdx {
		t.Error("commit should come before status")
	}
	if statusIdx > tagsIdx {
		t.Error("status should come before tags")
	}
	if tagsIdx > relatedIdx {
		t.Error("tags should come before related")
	}
	if relatedIdx > genByIdx {
		t.Error("related should come before generated_by")
	}
	if genByIdx > angelaIdx {
		t.Error("generated_by should come before angela_mode")
	}
}

func TestMarshal_OmitsEmptyOptionalFields(t *testing.T) {
	meta := domain.DocMeta{
		Type:   "note",
		Date:   "2026-03-07",
		Status: "published",
	}

	data, err := Marshal(meta, "body")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "commit:") {
		t.Error("empty commit should be omitted")
	}
	if strings.Contains(content, "tags:") {
		t.Error("empty tags should be omitted")
	}
	if strings.Contains(content, "related:") {
		t.Error("empty related should be omitted")
	}
	if strings.Contains(content, "generated_by:") {
		t.Error("empty generated_by should be omitted")
	}
	if strings.Contains(content, "angela_mode:") {
		t.Error("empty angela_mode should be omitted")
	}
}

func TestValidateMeta_Valid(t *testing.T) {
	meta := domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "published",
	}
	if err := ValidateMeta(meta); err != nil {
		t.Errorf("expected valid meta, got: %v", err)
	}
}

func TestValidateMeta_MissingType(t *testing.T) {
	meta := domain.DocMeta{
		Date:   "2026-03-07",
		Status: "published",
	}
	err := ValidateMeta(meta)
	if err == nil {
		t.Error("expected error for missing type")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Errorf("error should mention 'type', got: %v", err)
	}
}

func TestValidateMeta_MissingDate(t *testing.T) {
	meta := domain.DocMeta{
		Type:   "decision",
		Status: "published",
	}
	err := ValidateMeta(meta)
	if err == nil {
		t.Error("expected error for missing date")
	}
	if !strings.Contains(err.Error(), "date") {
		t.Errorf("error should mention 'date', got: %v", err)
	}
}

func TestValidateMeta_MissingStatus(t *testing.T) {
	meta := domain.DocMeta{
		Type: "decision",
		Date: "2026-03-07",
	}
	err := ValidateMeta(meta)
	if err == nil {
		t.Error("expected error for missing status")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("error should mention 'status', got: %v", err)
	}
}

func TestUnmarshal_MissingDelimiter(t *testing.T) {
	_, _, err := Unmarshal([]byte("no front matter here"))
	if err == nil {
		t.Error("expected error for missing delimiter")
	}
}

func TestUnmarshal_MissingClosingDelimiter(t *testing.T) {
	_, _, err := Unmarshal([]byte("---\ntype: test\n"))
	if err == nil {
		t.Error("expected error for missing closing delimiter")
	}
}

func TestUnmarshal_CRLF(t *testing.T) {
	input := "---\r\ntype: decision\r\nstatus: demo\r\ndate: \"2026-03-07\"\r\n---\r\n# Body\r\n"
	meta, body, err := Unmarshal([]byte(input))
	if err != nil {
		t.Fatalf("storage: unmarshal CRLF: %v", err)
	}
	if meta.Type != "decision" {
		t.Errorf("Type: got %q, want 'decision'", meta.Type)
	}
	if !strings.Contains(body, "Body") {
		t.Error("expected body content")
	}
}

func TestMarshal_DateFormat(t *testing.T) {
	meta := domain.DocMeta{
		Type:   "decision",
		Date:   "2026-03-07",
		Status: "demo",
	}
	data, err := Marshal(meta, "body")
	if err != nil {
		t.Fatalf("storage: marshal: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "2026-03-07") {
		t.Errorf("expected date in YYYY-MM-DD format, got:\n%s", content)
	}
	if strings.Contains(content, "T00:00:00") {
		t.Errorf("expected no RFC3339 timestamp, got:\n%s", content)
	}
}

func TestUnmarshal_MalformedYAML(t *testing.T) {
	input := "---\n: : : broken yaml\n---\nbody\n"
	_, _, err := Unmarshal([]byte(input))
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

// TestUnmarshal_HorizontalRuleInBody documents the known behavior of the
// front matter parser with respect to "---" horizontal rules in the Markdown
// body. The parser treats the FIRST "\n---\n" after the opening delimiter as
// the closing delimiter. This matches Jekyll, Hugo, and other static site
// generators. In normal usage (valid front matter followed by body), a "---"
// horizontal rule in the body is the SECOND "\n---\n" and is therefore
// preserved. However, if someone omits the closing delimiter and relies on a
// body "---" to accidentally close the front matter, the body will be
// truncated at that point.
func TestUnmarshal_HorizontalRuleInBody(t *testing.T) {
	t.Run("horizontal rule after valid front matter is preserved", func(t *testing.T) {
		// Standard document: front matter is properly closed, then body
		// contains a --- horizontal rule. The body is NOT truncated because
		// the real closing delimiter is found first.
		input := "---\ntype: decision\ndate: \"2026-03-07\"\nstatus: published\n---\n# Heading\n\nFirst section.\n\n---\n\nSecond section.\n"

		meta, body, err := Unmarshal([]byte(input))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if meta.Type != "decision" {
			t.Errorf("Type: got %q, want 'decision'", meta.Type)
		}

		// Both sections and the horizontal rule are present in the body.
		expectedBody := "# Heading\n\nFirst section.\n\n---\n\nSecond section.\n"
		if body != expectedBody {
			t.Errorf("Body: got %q, want %q", body, expectedBody)
		}
	})

	t.Run("missing closing delimiter with body horizontal rule causes truncation", func(t *testing.T) {
		// Malformed document: there is no real closing delimiter. The ---
		// horizontal rule in the body is mistaken for the closing delimiter.
		// Everything between the opening --- and the body --- is parsed as
		// YAML (which will likely fail or produce wrong results), and
		// everything after the body --- becomes the body.
		input := "---\ntype: decision\ndate: \"2026-03-07\"\nstatus: published\n# Heading\n\nFirst section.\n\n---\n\nSecond section.\n"

		_, body, err := Unmarshal([]byte(input))
		if err != nil {
			// YAML parsing may fail because the body text is not valid YAML.
			// That is expected — the point is the parser matched the wrong ---.
			t.Logf("YAML parse error (expected for malformed doc): %v", err)
			return
		}

		// If YAML parsing somehow succeeds, the body starts after the
		// horizontal rule, so "First section" is lost.
		if strings.Contains(body, "First section") {
			t.Error("expected First section to be consumed as YAML, not appear in body")
		}
		if !strings.Contains(body, "Second section") {
			t.Error("expected Second section to be in the body")
		}
	})
}

func TestUnmarshal_EmptyFrontmatter(t *testing.T) {
	data := []byte("---\n---\nBody content here\n")
	_, _, err := Unmarshal(data)
	if err == nil {
		t.Error("expected error for empty frontmatter")
	}
}

func TestUnmarshal_InvalidType(t *testing.T) {
	input := "---\ntype: banana\ndate: \"2026-03-07\"\nstatus: published\n---\n# Body\n"
	_, _, err := Unmarshal([]byte(input))
	if err == nil {
		t.Fatal("expected error for invalid type 'banana'")
	}
	if !strings.Contains(err.Error(), "unknown type") {
		t.Errorf("error should mention unknown type, got: %v", err)
	}
	if !strings.Contains(err.Error(), "banana") {
		t.Errorf("error should mention the invalid type value, got: %v", err)
	}
}

func TestValidateMeta_InvalidDateFormat(t *testing.T) {
	tests := []struct {
		name string
		date string
	}{
		{"slash format", "2026/03/07"},
		{"not a date", "not-a-date"},
		{"missing day", "2026-03"},
		{"US format", "03-07-2026"},
		{"impossible date", "2026-13-45"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := domain.DocMeta{Type: "note", Date: tt.date, Status: "published"}
			err := ValidateMeta(meta)
			if err == nil {
				t.Errorf("expected error for date %q", tt.date)
			}
			if !strings.Contains(err.Error(), "YYYY-MM-DD") {
				t.Errorf("error should mention format, got: %v", err)
			}
		})
	}
}

// --- ExtractFrontmatter tests (story 8-21, invariant I24 foundation) ------

func TestExtractFrontmatter_ReturnsBytesVerbatim(t *testing.T) {
	// Key ordering, quote styles, and whitespace must survive round-trip
	// with zero mutation. This is the anchor of I24 — no YAML
	// re-serialization may touch fmBytes.
	src := []byte("---\n" +
		"type: decision\n" +
		"date: \"2026-04-10\"\n" +
		"status: published\n" +
		"custom: 'single-quoted'\n" +
		"---\n" +
		"# Body\nContent here.\n")

	fmBytes, bodyBytes, err := ExtractFrontmatter(src)
	if err != nil {
		t.Fatalf("ExtractFrontmatter: %v", err)
	}

	wantFM := "---\n" +
		"type: decision\n" +
		"date: \"2026-04-10\"\n" +
		"status: published\n" +
		"custom: 'single-quoted'\n" +
		"---\n"
	if string(fmBytes) != wantFM {
		t.Errorf("fmBytes mismatch:\n got: %q\nwant: %q", string(fmBytes), wantFM)
	}

	wantBody := "# Body\nContent here.\n"
	if string(bodyBytes) != wantBody {
		t.Errorf("bodyBytes mismatch:\n got: %q\nwant: %q", string(bodyBytes), wantBody)
	}
}

func TestExtractFrontmatter_ConcatEqualsSource(t *testing.T) {
	// Postcondition: fmBytes + bodyBytes == normalized_input, byte-for-byte.
	// This is the concrete I24 identity property across varied body shapes.
	cases := []struct {
		name string
		src  string
	}{
		{"simple_body", "---\ntype: decision\ndate: \"2026-04-10\"\nstatus: published\n---\n## Body\n"},
		{"no_trailing_newline_on_body", "---\ntype: note\ndate: \"2026-01-01\"\nstatus: draft\n---\nbody"},
		{"empty_body", "---\ntype: note\ndate: \"2026-01-01\"\nstatus: draft\n---\n"},
		{"body_with_leading_blanks", "---\ntype: note\ndate: \"2026-01-01\"\nstatus: draft\n---\n\n\n# Heading\n"},
		{"body_with_code_fence_containing_dashes", "---\ntype: note\ndate: \"2026-01-01\"\nstatus: draft\n---\n```\n---\nnot a delimiter inside code\n```\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fmBytes, bodyBytes, err := ExtractFrontmatter([]byte(tc.src))
			if err != nil {
				t.Fatalf("ExtractFrontmatter: %v", err)
			}
			got := string(fmBytes) + string(bodyBytes)
			if got != tc.src {
				t.Errorf("concat mismatch:\n got: %q\nwant: %q", got, tc.src)
			}
		})
	}
}

func TestExtractFrontmatter_MissingFM_TypedError(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
	}{
		{"empty", []byte("")},
		{"plain_markdown", []byte("# Just a markdown file without front matter\n")},
		{"random_prose", []byte("some other stuff\n")},
		{"two_dashes_not_three", []byte("-- not quite three dashes\n")},
		{"three_dashes_no_newline", []byte("---not a proper delimiter\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ExtractFrontmatter(tc.src)
			if !errors.Is(err, ErrFrontmatterMissing) {
				t.Errorf("expected ErrFrontmatterMissing, got %v", err)
			}
		})
	}
}

func TestExtractFrontmatter_MalformedYAML_TypedError(t *testing.T) {
	cases := []struct {
		name string
		src  []byte
	}{
		{"unclosed_delimiter", []byte("---\ntype: decision\nbody without closing delimiter\n")},
		{"empty_yaml_block", []byte("---\n---\nbody\n")},
		{"opening_only", []byte("---\n")},
		{"yaml_parse_error", []byte("---\ntype: [unclosed\n---\nbody\n")},
		{"whitespace_only_yaml", []byte("---\n\n\n---\nbody\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ExtractFrontmatter(tc.src)
			if !errors.Is(err, ErrFrontmatterMalformed) {
				t.Errorf("expected ErrFrontmatterMalformed, got %v", err)
			}
		})
	}
}

func TestExtractFrontmatter_CRLFNormalized(t *testing.T) {
	srcCRLF := "---\r\ntype: note\r\ndate: \"2026-01-01\"\r\nstatus: draft\r\n---\r\nbody line\r\n"
	fmBytes, bodyBytes, err := ExtractFrontmatter([]byte(srcCRLF))
	if err != nil {
		t.Fatalf("ExtractFrontmatter: %v", err)
	}
	wantFM := "---\ntype: note\ndate: \"2026-01-01\"\nstatus: draft\n---\n"
	if string(fmBytes) != wantFM {
		t.Errorf("fmBytes:\n got: %q\nwant: %q", string(fmBytes), wantFM)
	}
	if string(bodyBytes) != "body line\n" {
		t.Errorf("bodyBytes: got %q, want %q", string(bodyBytes), "body line\n")
	}
	// Concat identity still holds on the normalized form.
	normalized := strings.ReplaceAll(srcCRLF, "\r\n", "\n")
	if string(fmBytes)+string(bodyBytes) != normalized {
		t.Errorf("concat does not equal normalized input")
	}
}

// TestExtractFrontmatter_UTF8BOM_Stripped covers the P0 regression:
// files saved with a UTF-8 BOM prefix (\xEF\xBB\xBF) previously
// failed `HasPrefix("---\n")`, returned ErrFrontmatterMissing, and
// downstream `doctor --fix` prepended a fresh FM ON TOP of the
// existing one (violating I31). The BOM must be stripped before
// delimiter detection so BOM+valid-FM parses correctly.
func TestExtractFrontmatter_UTF8BOM_Stripped(t *testing.T) {
	bom := "\xEF\xBB\xBF"
	src := bom + "---\ntype: decision\ndate: \"2026-04-19\"\nstatus: draft\n---\nbody\n"
	fmBytes, bodyBytes, err := ExtractFrontmatter([]byte(src))
	if err != nil {
		t.Fatalf("ExtractFrontmatter must accept BOM+FM, got err=%v", err)
	}
	wantFM := "---\ntype: decision\ndate: \"2026-04-19\"\nstatus: draft\n---\n"
	if string(fmBytes) != wantFM {
		t.Errorf("fmBytes:\n got: %q\nwant: %q", string(fmBytes), wantFM)
	}
	if string(bodyBytes) != "body\n" {
		t.Errorf("bodyBytes: got %q, want %q", string(bodyBytes), "body\n")
	}
}

// TestExtractFrontmatter_BOMPlusCRLF combines both normalizations —
// a legacy Windows editor (Notepad) emits BOM+CRLF. The strip of BOM
// must happen before the CRLF→LF conversion and before delimiter
// detection.
func TestExtractFrontmatter_BOMPlusCRLF(t *testing.T) {
	src := "\xEF\xBB\xBF---\r\ntype: note\r\ndate: \"2026-04-19\"\r\nstatus: draft\r\n---\r\nbody\r\n"
	fmBytes, bodyBytes, err := ExtractFrontmatter([]byte(src))
	if err != nil {
		t.Fatalf("ExtractFrontmatter must accept BOM+CRLF+FM, got err=%v", err)
	}
	wantFM := "---\ntype: note\ndate: \"2026-04-19\"\nstatus: draft\n---\n"
	if string(fmBytes) != wantFM {
		t.Errorf("fmBytes:\n got: %q\nwant: %q", string(fmBytes), wantFM)
	}
	if string(bodyBytes) != "body\n" {
		t.Errorf("bodyBytes: got %q, want %q", string(bodyBytes), "body\n")
	}
}

func TestExtractFrontmatter_PreservesInnerYAMLQuirks(t *testing.T) {
	// Sources with YAML comments, blank lines between keys, or tag-style
	// values must all survive untouched. This guards against a well-meaning
	// refactor that routes ExtractFrontmatter through yaml.Marshal.
	cases := []struct {
		name      string
		innerYAML string
	}{
		{"with_comment", "type: decision\n# this comment must survive\ndate: \"2026-04-10\"\nstatus: published\n"},
		{"with_blank_line", "type: decision\n\ndate: \"2026-04-10\"\nstatus: published\n"},
		{"unquoted_date", "type: decision\ndate: 2026-04-10\nstatus: published\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := "---\n" + tc.innerYAML + "---\nbody\n"
			fmBytes, _, err := ExtractFrontmatter([]byte(src))
			if err != nil {
				t.Fatalf("ExtractFrontmatter: %v", err)
			}
			// The inner YAML bytes must be identical to tc.innerYAML.
			wantFM := "---\n" + tc.innerYAML + "---\n"
			if string(fmBytes) != wantFM {
				t.Errorf("fmBytes:\n got: %q\nwant: %q", string(fmBytes), wantFM)
			}
		})
	}
}
