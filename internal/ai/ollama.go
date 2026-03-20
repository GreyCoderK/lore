// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

const ollamaDefaultModel = "llama3"

type ollamaProvider struct {
	client   *http.Client
	model    string
	endpoint string
	timeout  time.Duration
}

func newOllamaProvider(cfg *config.Config) *ollamaProvider {
	model := cfg.AI.Model
	if model == "" {
		model = ollamaDefaultModel
	}
	endpoint := cfg.AI.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	return &ollamaProvider{
		client:   SafeHTTPClient(),
		model:    model,
		endpoint: strings.TrimRight(endpoint, "/") + "/api/generate",
		timeout:  EnsureTimeout(cfg.AI.Timeout),
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func (p *ollamaProvider) Complete(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
	defaults := DefaultCallOptions()
	defaults.Model = p.model
	defaults.Timeout = p.timeout
	resolved := ResolveOptions(defaults, opts...)

	ctx, cancel := context.WithTimeout(ctx, resolved.Timeout)
	defer cancel()

	body := ollamaRequest{
		Model:  resolved.Model,
		Prompt: prompt,
		Stream: false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("ai: ollama: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ai: ollama: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai: ollama: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("ai: ollama: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: ollama: HTTP %d: %s", resp.StatusCode, TruncateForError(string(respBody), 512))
	}

	var result ollamaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ai: ollama: unmarshal response: %w", err)
	}

	return result.Response, nil
}
