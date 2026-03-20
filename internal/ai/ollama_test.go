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

func TestOllamaProvider_DefaultModel(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "ollama"}}
	p := newOllamaProvider(cfg)
	if p.model != ollamaDefaultModel {
		t.Errorf("default model = %q, want %q", p.model, ollamaDefaultModel)
	}
}
