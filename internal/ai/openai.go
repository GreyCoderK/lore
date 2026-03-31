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
	"time"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

const (
	openaiDefaultEndpoint = "https://api.openai.com/v1/chat/completions"
	openaiDefaultModel    = "gpt-4o"
)

type openaiProvider struct {
	client   *http.Client
	apiKey   string
	model    string
	endpoint string
	timeout  time.Duration
}

func newOpenAIProvider(cfg *config.Config) *openaiProvider {
	model := cfg.AI.Model
	if model == "" {
		model = openaiDefaultModel
	}
	endpoint := cfg.AI.Endpoint
	if endpoint == "" || endpoint == defaultOllamaEndpoint {
		endpoint = openaiDefaultEndpoint
	}
	return &openaiProvider{
		client:   SafeHTTPClient(),
		apiKey:   cfg.AI.APIKey,
		model:    model,
		endpoint: endpoint,
		timeout:  EnsureTimeout(cfg.AI.Timeout),
	}
}

type openaiRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature float64          `json:"temperature,omitempty"`
	Messages    []openaiMessage  `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (p *openaiProvider) Complete(ctx context.Context, prompt string, opts ...domain.Option) (string, error) {
	defaults := DefaultCallOptions()
	defaults.Model = p.model
	defaults.Timeout = p.timeout
	resolved := ResolveOptions(defaults, opts...)

	ctx, cancel := context.WithTimeout(ctx, resolved.Timeout)
	defer cancel()

	var messages []openaiMessage
	if resolved.System != "" {
		messages = append(messages, openaiMessage{Role: "system", Content: resolved.System})
	}
	messages = append(messages, openaiMessage{Role: "user", Content: prompt})

	body := openaiRequest{
		Model:       resolved.Model,
		MaxTokens:   resolved.MaxTokens,
		Temperature: resolved.Temperature,
		Messages:    messages,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("ai: openai: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ai: openai: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai: openai: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "ai: openai: body close: %v\n", cerr)
		}
	}()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("ai: openai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai: openai: HTTP %d: %s", resp.StatusCode, scrubSensitive(TruncateForError(string(respBody), 512)))
	}

	var result openaiResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("ai: openai: unmarshal response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("ai: openai: empty response choices")
	}

	if result.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("ai: openai: empty response text")
	}

	return result.Choices[0].Message.Content, nil
}
