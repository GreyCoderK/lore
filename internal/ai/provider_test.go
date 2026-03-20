// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
)

// mockStore implements credential.CredentialStore for testing.
type mockStore struct {
	SetFunc    func(provider string, secret []byte) error
	GetFunc    func(provider string) ([]byte, error)
	DeleteFunc func(provider string) error
	ListFunc   func() ([]string, error)
}

func (m *mockStore) Set(provider string, secret []byte) error { return m.SetFunc(provider, secret) }
func (m *mockStore) Get(provider string) ([]byte, error)      { return m.GetFunc(provider) }
func (m *mockStore) Delete(provider string) error             { return m.DeleteFunc(provider) }
func (m *mockStore) List() ([]string, error)                  { return m.ListFunc() }

func noopStore() *mockStore {
	return &mockStore{
		GetFunc:    func(string) ([]byte, error) { return nil, credential.ErrNotFound },
		SetFunc:    func(string, []byte) error { return nil },
		DeleteFunc: func(string) error { return nil },
		ListFunc:   func() ([]string, error) { return nil, nil },
	}
}

func TestNewProvider_Empty_ReturnsNilNil(t *testing.T) {
	cfg := &config.Config{}
	p, err := NewProvider(cfg, noopStore(), io.Discard)
	if err != nil {
		t.Fatalf("NewProvider empty: unexpected error: %v", err)
	}
	if p != nil {
		t.Errorf("NewProvider empty: expected nil provider, got %T", p)
	}
}

func TestNewProvider_Unknown_ReturnsError(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "xyz"}}
	_, err := NewProvider(cfg, noopStore(), io.Discard)
	if err == nil {
		t.Fatal("NewProvider unknown: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("NewProvider unknown: error = %q, want 'unknown provider'", err)
	}
}

func TestNewProvider_Anthropic_NoKey_ReturnsError(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "anthropic"}}
	_, err := NewProvider(cfg, noopStore(), io.Discard)
	if err == nil {
		t.Fatal("NewProvider anthropic no key: expected error")
	}
	if !strings.Contains(err.Error(), "No API key found") {
		t.Errorf("error = %q, want 'No API key found'", err)
	}
}

func TestNewProvider_OpenAI_NoKey_ReturnsError(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "openai"}}
	_, err := NewProvider(cfg, noopStore(), io.Discard)
	if err == nil {
		t.Fatal("NewProvider openai no key: expected error")
	}
	if !strings.Contains(err.Error(), "No API key found") {
		t.Errorf("error = %q, want 'No API key found'", err)
	}
}

func TestNewProvider_Ollama_NoKey_OK(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "ollama"}}
	p, err := NewProvider(cfg, noopStore(), io.Discard)
	if err != nil {
		t.Fatalf("NewProvider ollama no key: %v", err)
	}
	if p == nil {
		t.Error("NewProvider ollama: expected non-nil provider")
	}
}

func TestNewProvider_Anthropic_WithKey_Plaintext(t *testing.T) {
	var buf bytes.Buffer
	cfg := &config.Config{AI: config.AIConfig{Provider: "anthropic", APIKey: "sk-test"}}
	p, err := NewProvider(cfg, noopStore(), &buf)
	if err != nil {
		t.Fatalf("NewProvider anthropic: %v", err)
	}
	if p == nil {
		t.Error("NewProvider anthropic: expected non-nil provider")
	}
	// AC-5: plaintext warning
	if !strings.Contains(buf.String(), "plaintext") {
		t.Errorf("expected plaintext warning, got %q", buf.String())
	}
}

func TestNewProvider_OpenAI_WithKey_Plaintext(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "openai", APIKey: "sk-test"}}
	p, err := NewProvider(cfg, noopStore(), io.Discard)
	if err != nil {
		t.Fatalf("NewProvider openai: %v", err)
	}
	if p == nil {
		t.Error("NewProvider openai: expected non-nil provider")
	}
}

// --- Credential Resolution Tests ---

func TestResolveAPIKey_EnvVar_Priority(t *testing.T) {
	t.Setenv("LORE_AI_API_KEY", "env-key-123")
	store := &mockStore{
		GetFunc: func(string) ([]byte, error) { return []byte("keychain-key"), nil },
	}
	key, err := ResolveAPIKey("anthropic", "config-key", store, io.Discard)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "env-key-123" {
		t.Errorf("key = %q, want env-key-123 (env var should win)", key)
	}
}

func TestResolveAPIKey_Keychain_OK(t *testing.T) {
	store := &mockStore{
		GetFunc: func(provider string) ([]byte, error) {
			if provider == "anthropic" {
				return []byte("keychain-secret"), nil
			}
			return nil, credential.ErrNotFound
		},
	}
	key, err := ResolveAPIKey("anthropic", "", store, io.Discard)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "keychain-secret" {
		t.Errorf("key = %q, want keychain-secret", key)
	}
}

func TestResolveAPIKey_Keychain_AtKeychain(t *testing.T) {
	store := &mockStore{
		GetFunc: func(string) ([]byte, error) { return []byte("kc-val"), nil },
	}
	key, err := ResolveAPIKey("openai", "@keychain", store, io.Discard)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "kc-val" {
		t.Errorf("key = %q, want kc-val", key)
	}
}

func TestResolveAPIKey_Keychain_Miss(t *testing.T) {
	store := &mockStore{
		GetFunc: func(string) ([]byte, error) { return nil, credential.ErrNotFound },
	}
	key, err := ResolveAPIKey("anthropic", "", store, io.Discard)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "" {
		t.Errorf("key = %q, want empty (not found anywhere)", key)
	}
}

func TestResolveAPIKey_Plaintext_Warning(t *testing.T) {
	var buf bytes.Buffer
	store := noopStore()
	key, err := ResolveAPIKey("anthropic", "sk-plain", store, &buf)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "sk-plain" {
		t.Errorf("key = %q, want sk-plain", key)
	}
	if !strings.Contains(buf.String(), "plaintext") {
		t.Errorf("expected plaintext warning, got %q", buf.String())
	}
}

func TestResolveAPIKey_KeychainUnavailable_Warning(t *testing.T) {
	var buf bytes.Buffer
	store := &mockStore{
		GetFunc: func(string) ([]byte, error) { return nil, credential.ErrKeychainNotAvailable },
	}
	key, err := ResolveAPIKey("anthropic", "", store, &buf)
	if err != nil {
		t.Fatalf("ResolveAPIKey: %v", err)
	}
	if key != "" {
		t.Errorf("key = %q, want empty", key)
	}
	if !strings.Contains(buf.String(), "keychain not available") {
		t.Errorf("expected keychain warning, got %q", buf.String())
	}
}

func TestResolveAPIKey_KeychainError_Propagated(t *testing.T) {
	store := &mockStore{
		GetFunc: func(string) ([]byte, error) { return nil, errors.New("keychain locked") },
	}
	_, err := ResolveAPIKey("anthropic", "", store, io.Discard)
	if err == nil {
		t.Fatal("expected error from keychain failure")
	}
	if !strings.Contains(err.Error(), "keychain locked") {
		t.Errorf("error = %q, want 'keychain locked'", err)
	}
}

func TestResolveAPIKey_NilStore_EmptyKey(t *testing.T) {
	key, err := ResolveAPIKey("anthropic", "", nil, io.Discard)
	if err != nil {
		t.Fatalf("ResolveAPIKey nil store: %v", err)
	}
	if key != "" {
		t.Errorf("key = %q, want empty with nil store", key)
	}
}
