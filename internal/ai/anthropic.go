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
	"os"
	"sync"
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
	client    *http.Client
	apiKey    string
	model     string
	endpoint  string
	timeout   time.Duration
	mu        sync.Mutex
	lastUsage *domain.AIUsage
}

func newAnthropicProvider(cfg *config.Config) *anthropicProvider {
	model := cfg.AI.Model
	if model == "" {
		model = anthropicDefaultModel
	}
	endpoint := cfg.AI.Endpoint
	if endpoint == "" || endpoint == defaultOllamaEndpoint {
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
	Model       string                  `json:"model"`
	MaxTokens   int                     `json:"max_tokens"`
	Temperature float64                 `json:"temperature,omitempty"`
	System      []anthropicSystemBlock  `json:"system,omitempty"`
	Messages    []anthropicMessage      `json:"messages"`
}

type anthropicSystemBlock struct {
	Type         string                `json:"type"`
	Text         string                `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicCacheControl struct {
	Type string `json:"type"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
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
		Model:       resolved.Model,
		MaxTokens:   resolved.MaxTokens,
		Temperature: resolved.Temperature,
		Messages:    []anthropicMessage{{Role: "user", Content: prompt}},
	}
	if resolved.System != "" {
		body.System = []anthropicSystemBlock{
			{
				Type:         "text",
				Text:         resolved.System,
				CacheControl: &anthropicCacheControl{Type: "ephemeral"},
			},
		}
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
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "ai: anthropic: body close: %v\n", cerr)
		}
	}()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("ai: anthropic: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: anthropic: HTTP %d: %s", resp.StatusCode, scrubSensitive(TruncateForError(string(respBody), 512)))
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

	if result.Content[0].Text == "" {
		return "", fmt.Errorf("ai: anthropic: empty response text")
	}

	p.mu.Lock()
	p.lastUsage = &domain.AIUsage{
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		Model:        result.Model,
	}
	p.mu.Unlock()

	return result.Content[0].Text, nil
}

func (p *anthropicProvider) LastUsage() *domain.AIUsage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastUsage
}
