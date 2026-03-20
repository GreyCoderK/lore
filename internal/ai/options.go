// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

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
	return nil
}

// TruncateForError returns s truncated to maxLen bytes for safe inclusion in error messages.
// Prevents multi-MB response bodies from being propagated in error chains.
func TruncateForError(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

// SafeHTTPClient returns an http.Client that refuses to follow redirects.
// This prevents SSRF attacks via open redirects (e.g. 302 → internal services).
func SafeHTTPClient() *http.Client {
	return &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}
