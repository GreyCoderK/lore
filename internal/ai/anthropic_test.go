// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

func TestAnthropicProvider_Complete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "sk-test" {
			t.Errorf("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != anthropicAPIVersion {
			t.Errorf("missing anthropic-version header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"text":"hello world"}]}`))
	}))
	defer srv.Close()

	p := &anthropicProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "test-model",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	result, err := p.Complete(context.Background(), "test prompt")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "hello world" {
		t.Errorf("Complete = %q, want %q", result, "hello world")
	}
}

func TestAnthropicProvider_Complete_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer srv.Close()

	p := &anthropicProvider{
		client:   srv.Client(),
		apiKey:   "bad-key",
		model:    "test-model",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("Complete: expected error for HTTP 401")
	}
	if !strings.Contains(err.Error(), "HTTP 401") {
		t.Errorf("error = %q, want HTTP 401", err)
	}
}

func TestAnthropicProvider_Complete_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"text":"late"}]}`))
	}))
	defer srv.Close()

	p := &anthropicProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "test-model",
		endpoint: srv.URL,
		timeout:  50 * time.Millisecond,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("Complete: expected timeout error")
	}
}

func TestAnthropicProvider_Complete_WithModelOverride(t *testing.T) {
	var receivedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req anthropicRequest
		if err := decodeJSON(r.Body, &req); err == nil {
			receivedModel = req.Model
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"text":"ok"}]}`))
	}))
	defer srv.Close()

	p := &anthropicProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "default-model",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test", domain.WithModel("override-model"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if receivedModel != "override-model" {
		t.Errorf("model = %q, want %q", receivedModel, "override-model")
	}
}

func TestAnthropicProvider_Complete_WithSystemPrompt(t *testing.T) {
	var captured anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decodeJSON(r.Body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"text":"ok"}]}`))
	}))
	defer srv.Close()

	p := &anthropicProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "test-model",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "user content", domain.WithSystem("You are Angela"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// System must be an array of content blocks with type "text"
	if len(captured.System) != 1 {
		t.Fatalf("system blocks = %d, want 1", len(captured.System))
	}
	if captured.System[0].Type != "text" {
		t.Errorf("system[0].type = %q, want %q", captured.System[0].Type, "text")
	}
	if captured.System[0].Text != "You are Angela" {
		t.Errorf("system[0].text = %q, want %q", captured.System[0].Text, "You are Angela")
	}

	// cache_control must be present with type "ephemeral"
	if captured.System[0].CacheControl == nil {
		t.Fatal("system[0].cache_control is nil, want ephemeral")
	}
	if captured.System[0].CacheControl.Type != "ephemeral" {
		t.Errorf("cache_control.type = %q, want %q", captured.System[0].CacheControl.Type, "ephemeral")
	}

	// User content must be in messages, not system
	if len(captured.Messages) != 1 || captured.Messages[0].Content != "user content" {
		t.Errorf("messages = %+v, want single user message with 'user content'", captured.Messages)
	}
}

func TestAnthropicProvider_Complete_WithoutSystemPrompt(t *testing.T) {
	var captured anthropicRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decodeJSON(r.Body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":[{"text":"ok"}]}`))
	}))
	defer srv.Close()

	p := &anthropicProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "test-model",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "just a prompt")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// No system block when no system prompt provided
	if len(captured.System) != 0 {
		t.Errorf("system blocks = %d, want 0 (no system prompt)", len(captured.System))
	}
}

func TestAnthropicRequest_CacheControl_JSONSerialization(t *testing.T) {
	req := anthropicRequest{
		Model:     "test-model",
		MaxTokens: 1024,
		System: []anthropicSystemBlock{
			{
				Type:         "text",
				Text:         "You are Angela",
				CacheControl: &anthropicCacheControl{Type: "ephemeral"},
			},
		},
		Messages: []anthropicMessage{{Role: "user", Content: "hello"}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	s := string(data)
	if !strings.Contains(s, `"cache_control"`) {
		t.Errorf("JSON missing cache_control field: %s", s)
	}
	if !strings.Contains(s, `"type":"ephemeral"`) {
		t.Errorf("JSON missing ephemeral type: %s", s)
	}

	// Verify omitempty: no cache_control when nil
	reqNoCache := anthropicRequest{
		Model:     "test-model",
		MaxTokens: 1024,
		System:    []anthropicSystemBlock{{Type: "text", Text: "test"}},
		Messages:  []anthropicMessage{{Role: "user", Content: "hello"}},
	}
	data2, _ := json.Marshal(reqNoCache)
	if strings.Contains(string(data2), `"cache_control"`) {
		t.Errorf("JSON should not contain cache_control when nil: %s", string(data2))
	}
}

func TestAnthropicProvider_DefaultModel(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "anthropic", APIKey: "sk-test"}}
	p := newAnthropicProvider(cfg)
	if p.model != anthropicDefaultModel {
		t.Errorf("default model = %q, want %q", p.model, anthropicDefaultModel)
	}
}
