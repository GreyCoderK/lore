package storage

import (
	"strings"
	"testing"
	"time"

	"github.com/museigen/lore/internal/domain"
)

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	meta := domain.DocMeta{
		Type:        "decision",
		Date:        domain.NewDateString(time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)),
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
	if gotMeta.Status != "demo" {
		t.Errorf("Status: got %q, want 'demo'", gotMeta.Status)
	}
	if gotMeta.Commit != "abc1234" {
		t.Errorf("Commit: got %q, want 'abc1234'", gotMeta.Commit)
	}
	if len(gotMeta.Tags) != 2 {
		t.Errorf("Tags: got %d, want 2", len(gotMeta.Tags))
	}
	if gotBody != body {
		t.Errorf("Body: got %q, want %q", gotBody, body)
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
		Date:   domain.NewDateString(time.Date(2026, 3, 7, 0, 0, 0, 0, time.UTC)),
		Status: "demo",
	}
	data, err := Marshal(meta, "body")
	if err != nil {
		t.Fatalf("storage: marshal: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "\"2026-03-07\"") && !strings.Contains(content, "2026-03-07") {
		t.Errorf("expected date in YYYY-MM-DD format, got:\n%s", content)
	}
	if strings.Contains(content, "T00:00:00") {
		t.Errorf("expected no RFC3339 timestamp, got:\n%s", content)
	}
}
