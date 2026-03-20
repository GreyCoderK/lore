// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"context"
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

func TestAnthropicProvider_DefaultModel(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "anthropic", APIKey: "sk-test"}}
	p := newAnthropicProvider(cfg)
	if p.model != anthropicDefaultModel {
		t.Errorf("default model = %q, want %q", p.model, anthropicDefaultModel)
	}
}
