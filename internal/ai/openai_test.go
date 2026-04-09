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

func TestOpenAIProvider_Complete_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer sk-test" {
			t.Errorf("Authorization = %q, want Bearer sk-test", auth)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hello openai"}}]}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	result, err := p.Complete(context.Background(), "test")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "hello openai" {
		t.Errorf("Complete = %q, want %q", result, "hello openai")
	}
}

func TestOpenAIProvider_Complete_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("expected error for HTTP 429")
	}
	if !strings.Contains(err.Error(), "HTTP 429") {
		t.Errorf("error = %q, want HTTP 429", err)
	}
}

func TestOpenAIProvider_Complete_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"late"}}]}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
		endpoint: srv.URL,
		timeout:  50 * time.Millisecond,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("Complete: expected timeout error")
	}
}

func TestOpenAIProvider_Complete_WithModelOverride(t *testing.T) {
	var receivedModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openaiRequest
		if err := decodeJSON(r.Body, &req); err == nil {
			receivedModel = req.Model
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
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

func TestOpenAIProvider_Complete_WithSystemPrompt(t *testing.T) {
	var captured openaiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decodeJSON(r.Body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "user content", domain.WithSystem("You are Angela"))
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Should have system message first, then user message
	if len(captured.Messages) != 2 {
		t.Fatalf("messages count = %d, want 2", len(captured.Messages))
	}
	if captured.Messages[0].Role != "system" || captured.Messages[0].Content != "You are Angela" {
		t.Errorf("messages[0] = %+v, want system 'You are Angela'", captured.Messages[0])
	}
	if captured.Messages[1].Role != "user" || captured.Messages[1].Content != "user content" {
		t.Errorf("messages[1] = %+v, want user 'user content'", captured.Messages[1])
	}
}

func TestOpenAIProvider_Complete_WithoutSystemPrompt(t *testing.T) {
	var captured openaiRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decodeJSON(r.Body, &captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}]}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "just a prompt")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// Only user message, no system
	if len(captured.Messages) != 1 {
		t.Fatalf("messages count = %d, want 1", len(captured.Messages))
	}
	if captured.Messages[0].Role != "user" {
		t.Errorf("messages[0].role = %q, want user", captured.Messages[0].Role)
	}
}

func TestOpenAIProvider_Complete_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
		endpoint: srv.URL,
		timeout:  5 * time.Second,
	}

	_, err := p.Complete(context.Background(), "test")
	if err == nil {
		t.Fatal("Complete: expected error for empty choices")
	}
	if !strings.Contains(err.Error(), "empty response choices") {
		t.Errorf("error = %q, want 'empty response choices'", err)
	}
}

func TestOpenAIProvider_LastUsage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":200,"completion_tokens":100},"model":"gpt-4o"}`))
	}))
	defer srv.Close()

	p := &openaiProvider{
		client:   srv.Client(),
		apiKey:   "sk-test",
		model:    "gpt-4o",
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
	if u.InputTokens != 200 || u.OutputTokens != 100 {
		t.Errorf("usage = %d/%d, want 200/100", u.InputTokens, u.OutputTokens)
	}
	if u.Model != "gpt-4o" {
		t.Errorf("Model = %q, want gpt-4o", u.Model)
	}
}

func TestOpenAIProvider_DefaultModel(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "openai", APIKey: "sk-test"}}
	p := newOpenAIProvider(cfg)
	if p.model != openaiDefaultModel {
		t.Errorf("default model = %q, want %q", p.model, openaiDefaultModel)
	}
}
