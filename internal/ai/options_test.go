// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

func TestDefaultCallOptions(t *testing.T) {
	opts := DefaultCallOptions()
	if opts.MaxTokens != 4096 {
		t.Errorf("DefaultCallOptions: MaxTokens = %d, want 4096", opts.MaxTokens)
	}
	if opts.Temperature != 0.7 {
		t.Errorf("DefaultCallOptions: Temperature = %f, want 0.7", opts.Temperature)
	}
	if opts.Timeout != 30*time.Second {
		t.Errorf("DefaultCallOptions: Timeout = %v, want 30s", opts.Timeout)
	}
}

func TestWithModel(t *testing.T) {
	opts := DefaultCallOptions()
	resolved := ResolveOptions(opts, domain.WithModel("gpt-4o"))
	if resolved.Model != "gpt-4o" {
		t.Errorf("WithModel: Model = %q, want %q", resolved.Model, "gpt-4o")
	}
}

func TestWithMaxTokens(t *testing.T) {
	opts := DefaultCallOptions()
	resolved := ResolveOptions(opts, domain.WithMaxTokens(8192))
	if resolved.MaxTokens != 8192 {
		t.Errorf("WithMaxTokens: MaxTokens = %d, want 8192", resolved.MaxTokens)
	}
}

func TestWithTemperature(t *testing.T) {
	opts := DefaultCallOptions()
	resolved := ResolveOptions(opts, domain.WithTemperature(0.3))
	if resolved.Temperature != 0.3 {
		t.Errorf("WithTemperature: Temperature = %f, want 0.3", resolved.Temperature)
	}
}

func TestWithSystem(t *testing.T) {
	opts := DefaultCallOptions()
	resolved := ResolveOptions(opts, domain.WithSystem("You are Angela"))
	if resolved.System != "You are Angela" {
		t.Errorf("WithSystem: System = %q, want %q", resolved.System, "You are Angela")
	}
}

func TestResolveOptions_MultipleOverrides(t *testing.T) {
	opts := DefaultCallOptions()
	resolved := ResolveOptions(opts,
		domain.WithModel("claude-sonnet-4-20250514"),
		domain.WithMaxTokens(2048),
		domain.WithTemperature(0.0),
		domain.WithSystem("system prompt"),
	)
	if resolved.Model != "claude-sonnet-4-20250514" {
		t.Errorf("Model = %q, want claude-sonnet-4-20250514", resolved.Model)
	}
	if resolved.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, want 2048", resolved.MaxTokens)
	}
	if resolved.Temperature != 0.0 {
		t.Errorf("Temperature = %f, want 0.0", resolved.Temperature)
	}
	if resolved.System != "system prompt" {
		t.Errorf("System = %q, want %q", resolved.System, "system prompt")
	}
}

func TestValidateEndpoint_ValidHTTPS(t *testing.T) {
	if err := ValidateEndpoint("https://api.anthropic.com/v1/messages"); err != nil {
		t.Errorf("ValidateEndpoint HTTPS: %v", err)
	}
}

func TestValidateEndpoint_ValidHTTP(t *testing.T) {
	if err := ValidateEndpoint("http://localhost:11434"); err != nil {
		t.Errorf("ValidateEndpoint HTTP: %v", err)
	}
}

func TestValidateEndpoint_FileScheme_Rejected(t *testing.T) {
	err := ValidateEndpoint("file:///etc/passwd")
	if err == nil {
		t.Fatal("ValidateEndpoint file://: expected error")
	}
}

func TestValidateEndpoint_GopherScheme_Rejected(t *testing.T) {
	err := ValidateEndpoint("gopher://evil.com")
	if err == nil {
		t.Fatal("ValidateEndpoint gopher://: expected error")
	}
}

func TestValidateEndpoint_EmptyScheme_Rejected(t *testing.T) {
	err := ValidateEndpoint("not-a-url")
	if err == nil {
		t.Fatal("ValidateEndpoint no scheme: expected error")
	}
}

func TestValidateEndpoint_NoHost_Rejected(t *testing.T) {
	err := ValidateEndpoint("http://")
	if err == nil {
		t.Fatal("ValidateEndpoint no host: expected error")
	}
}

func TestSafeHTTPClient_NoRedirect(t *testing.T) {
	redirectCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		redirectCalled = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := SafeHTTPClient()
	resp, err := client.Get(srv.URL + "/redirect")
	if err != nil {
		t.Fatalf("SafeHTTPClient: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("StatusCode = %d, want 302 (redirect not followed)", resp.StatusCode)
	}
	if redirectCalled {
		t.Error("redirect was followed — SafeHTTPClient should block redirects")
	}
}

func TestTruncateForError_Short(t *testing.T) {
	result := TruncateForError("short", 512)
	if result != "short" {
		t.Errorf("TruncateForError short = %q, want %q", result, "short")
	}
}

func TestTruncateForError_Long(t *testing.T) {
	long := string(make([]byte, 1000))
	result := TruncateForError(long, 512)
	if len(result) > 530 {
		t.Errorf("TruncateForError long len = %d, want <= 530", len(result))
	}
}

func TestEnsureTimeout_Positive(t *testing.T) {
	d := EnsureTimeout(10 * time.Second)
	if d != 10*time.Second {
		t.Errorf("EnsureTimeout(10s) = %v, want 10s", d)
	}
}

func TestEnsureTimeout_Zero(t *testing.T) {
	d := EnsureTimeout(0)
	if d != 30*time.Second {
		t.Errorf("EnsureTimeout(0) = %v, want 30s", d)
	}
}

func TestResolveOptions_DefaultsPreserved(t *testing.T) {
	opts := DefaultCallOptions()
	resolved := ResolveOptions(opts, domain.WithModel("test"))
	if resolved.MaxTokens != 4096 {
		t.Errorf("MaxTokens changed to %d, want 4096 (preserved)", resolved.MaxTokens)
	}
	if resolved.Temperature != 0.7 {
		t.Errorf("Temperature changed to %f, want 0.7 (preserved)", resolved.Temperature)
	}
}
