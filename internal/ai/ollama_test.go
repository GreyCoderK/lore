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

func TestOllamaProvider_Complete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"hello ollama"}`))
	}))
	defer srv.Close()

	p := &ollamaProvider{
		client:   srv.Client(),
		model:    "llama3",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	result, err := p.Complete(context.Background(), "test")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "hello ollama" {
		t.Errorf("Complete = %q, want %q", result, "hello ollama")
	}
}

func TestOllamaProvider_Complete_ConnectionRefused(t *testing.T) {
	p := &ollamaProvider{
		client:   &http.Client{},
		model:    "llama3",
		endpoint: "http://127.0.0.1:1", // port 1 - will refuse
		timeout:  2 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "ai: ollama:") {
		t.Errorf("error = %q, want ai: ollama: prefix", err)
	}
}

func TestOllamaProvider_CustomEndpoint(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{
		Provider: "ollama",
		Endpoint: "http://custom-host:9999",
	}}
	p := newOllamaProvider(cfg)
	if p.endpoint != "http://custom-host:9999/api/generate" {
		t.Errorf("endpoint = %q, want http://custom-host:9999/api/generate", p.endpoint)
	}
}

func TestOllamaProvider_Complete_WithSystemPrompt(t *testing.T) {
	var captured ollamaRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decodeJSON(r.Body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"ok"}`))
	}))
	defer srv.Close()

	p := &ollamaProvider{
		client:   srv.Client(),
		model:    "llama3",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "user content", domain.WithSystem("You are Angela"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// System prompt should be in the native system field
	if captured.System != "You are Angela" {
		t.Errorf("system = %q, want 'You are Angela'", captured.System)
	}
	// User content should be in the prompt field (not mixed with system)
	if captured.Prompt != "user content" {
		t.Errorf("prompt = %q, want 'user content'", captured.Prompt)
	}
}

func TestOllamaProvider_Complete_WithoutSystemPrompt(t *testing.T) {
	var captured ollamaRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decodeJSON(r.Body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"ok"}`))
	}))
	defer srv.Close()

	p := &ollamaProvider{
		client:   srv.Client(),
		model:    "llama3",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "just a prompt")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// No system prompt, so prompt should be exactly what was passed
	if captured.Prompt != "just a prompt" {
		t.Errorf("prompt = %q, want %q", captured.Prompt, "just a prompt")
	}
}

func TestOllamaProvider_Complete_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":""}`))
	}))
	defer srv.Close()

	p := &ollamaProvider{
		client:   srv.Client(),
		model:    "llama3",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("Complete: expected error for empty response")
	}
	if !strings.Contains(err.Error(), "empty response") {
		t.Errorf("error = %q, want 'empty response'", err)
	}
}

func TestOllamaProvider_LastUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"response":"ok","model":"llama3.2","prompt_eval_count":300,"eval_count":150}`))
	}))
	defer srv.Close()

	p := &ollamaProvider{
		client:   srv.Client(),
		model:    "llama3.2",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	u := p.LastUsage()
	if u == nil {
		t.Fatal("LastUsage: expected non-nil")
	}
	if u.InputTokens != 300 || u.OutputTokens != 150 {
		t.Errorf("usage = %d/%d, want 300/150", u.InputTokens, u.OutputTokens)
	}
	if u.Model != "llama3.2" {
		t.Errorf("Model = %q, want llama3.2", u.Model)
	}
}

func TestOllamaProvider_DefaultModel(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "ollama"}}
	p := newOllamaProvider(cfg)
	if p.model != ollamaDefaultModel {
		t.Errorf("default model = %q, want %q", p.model, ollamaDefaultModel)
	}
}
