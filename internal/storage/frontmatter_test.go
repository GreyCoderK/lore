// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package storage

import (
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
