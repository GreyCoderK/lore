// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// defaultOllamaEndpoint is the Ollama default endpoint used by config/defaults.go.
// Providers check against this to avoid sending API calls to Ollama when they have
// their own default endpoint.
const defaultOllamaEndpoint = "http://localhost:11434"

// DefaultCallOptions returns sensible defaults for AI provider calls.
func DefaultCallOptions() domain.CallOptions {
	return domain.CallOptions{
		MaxTokens:   4096,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}
}

// ResolveOptions applies variadic options over defaults, returning final options.
func ResolveOptions(defaults domain.CallOptions, opts ...domain.Option) domain.CallOptions {
	for _, opt := range opts {
		opt(&defaults)
	}
	return defaults
}

// EnsureTimeout returns t if positive, otherwise the default 30s.
func EnsureTimeout(t time.Duration) time.Duration {
	if t > 0 {
		return t
	}
	return 30 * time.Second
}

// ValidateEndpoint checks that an endpoint URL uses http or https scheme only.
// Rejects file://, gopher://, javascript:, and other dangerous schemes.
// Also rejects plain http for non-localhost endpoints to prevent credential leakage.
func ValidateEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("ai: endpoint: invalid URL %q: %w", endpoint, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("ai: endpoint: scheme %q not allowed, use http or https", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("ai: endpoint: missing host in %q", endpoint)
	}
	if u.Scheme == "http" && !isLocalhost(u.Host) {
		return fmt.Errorf("ai: endpoint: http not allowed for remote hosts (use https), got %q", endpoint)
	}
	return nil
}

// isLocalhost returns true if host (with optional port) resolves to a loopback address.
func isLocalhost(host string) bool {
	h := strings.Split(host, ":")[0]
	return h == "localhost" || h == "127.0.0.1" || h == "::1"
}

// TruncateForError returns s truncated to maxLen bytes for safe inclusion in error messages.
// Prevents multi-MB response bodies from being propagated in error chains.
func TruncateForError(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// SafeHTTPClient returns an http.Client with connection limits, a global timeout,
// and redirect-following disabled (prevents SSRF via open redirects).
func SafeHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			MaxConnsPerHost:     5,
			IdleConnTimeout:     90 * time.Second,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
