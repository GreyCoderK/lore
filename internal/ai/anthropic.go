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
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

const (
	anthropicDefaultEndpoint = "https://api.anthropic.com/v1/messages"
	anthropicDefaultModel    = "claude-sonnet-4-20250514"
	anthropicAPIVersion      = "2023-06-01"
	maxResponseBytes         = 10 << 20 // 10 MB — guards against unbounded responses
)

type anthropicProvider struct {
	client   *http.Client
	apiKey   string
	model    string
	endpoint string
	timeout  time.Duration
}

func newAnthropicProvider(cfg *config.Config) *anthropicProvider {
	model := cfg.AI.Model
	if model == "" {
		model = anthropicDefaultModel
	}
	endpoint := cfg.AI.Endpoint
	if endpoint == "" || endpoint == "http://localhost:11434" {
		endpoint = anthropicDefaultEndpoint
	}
	return &anthropicProvider{
		client:   SafeHTTPClient(),
		apiKey:   cfg.AI.APIKey,
		model:    model,
		endpoint: endpoint,
		timeout:  EnsureTimeout(cfg.AI.Timeout),
	}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *anthropicProvider) Complete(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
	defaults := DefaultCallOptions()
	defaults.Model = p.model
	defaults.Timeout = p.timeout
	resolved := ResolveOptions(defaults, opts...)

	ctx, cancel := context.WithTimeout(ctx, resolved.Timeout)
	defer cancel()

	body := anthropicRequest{
		Model:     resolved.Model,
		MaxTokens: resolved.MaxTokens,
		Messages:  []anthropicMessage{{Role: "user", Content: prompt}},
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("ai: anthropic: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ai: anthropic: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai: anthropic: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("ai: anthropic: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: anthropic: HTTP %d: %s", resp.StatusCode, TruncateForError(string(respBody), 512))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ai: anthropic: unmarshal response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("ai: anthropic: API error: %s", result.Error.Message)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("ai: anthropic: empty response content")
	}

	return result.Content[0].Text, nil
}
