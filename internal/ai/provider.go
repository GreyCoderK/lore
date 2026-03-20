// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ai

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
)

// NewProvider creates an AIProvider based on configuration.
// Returns (nil, nil) when no provider is configured (zero-API mode).
// store may be nil (keychain unavailable). warnW receives plaintext warnings.
func NewProvider(cfg *config.Config, store credential.CredentialStore, warnW io.Writer) (domain.AIProvider, error) {
	if cfg.AI.Provider == "" {
		return nil, nil
	}

	// Validate endpoint URL scheme if user provided a custom endpoint.
	if cfg.AI.Endpoint != "" {
		if err := ValidateEndpoint(cfg.AI.Endpoint); err != nil {
			return nil, err
		}
	}

	// Resolve API key with priority: env > keychain > plaintext config
	apiKey, err := ResolveAPIKey(cfg.AI.Provider, cfg.AI.APIKey, store, warnW)
	if err != nil {
		return nil, err
	}

	// Build a config copy with the resolved key for provider constructors
	resolved := *cfg
	resolved.AI.APIKey = apiKey

	switch cfg.AI.Provider {
	case "anthropic":
		if apiKey == "" {
			return nil, fmt.Errorf("ai: provider: No API key found for anthropic. Run: lore config set-key anthropic")
		}
		return newAnthropicProvider(&resolved), nil
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("ai: provider: No API key found for openai. Run: lore config set-key openai")
		}
		return newOpenAIProvider(&resolved), nil
	case "ollama":
		return newOllamaProvider(&resolved), nil
	default:
		return nil, fmt.Errorf("ai: unknown provider %q, supported: anthropic, openai, ollama", cfg.AI.Provider)
	}
}

// ResolveAPIKey resolves an API key using the priority chain:
// 1. LORE_AI_API_KEY env var (no warning)
// 2. keychain via store.Get() if key is "" or "@keychain"
// 3. plaintext config value (emits warning)
// 4. empty string (caller decides if error)
func ResolveAPIKey(provider, configKey string, store credential.CredentialStore, warnW io.Writer) (string, error) {
	// 1. Environment variable — highest priority, no warning (CI use case)
	if envKey := os.Getenv("LORE_AI_API_KEY"); envKey != "" {
		return envKey, nil
	}

	// 2. Keychain — if config key is empty or "@keychain"
	if configKey == "" || configKey == "@keychain" {
		if store != nil {
			key, err := store.Get(provider)
			if err == nil {
				return string(key), nil
			}
			if !errors.Is(err, credential.ErrNotFound) && !errors.Is(err, credential.ErrKeychainNotAvailable) {
				return "", fmt.Errorf("ai: credential: %w", err)
			}
			// Not found in keychain — fall through to return empty
			if errors.Is(err, credential.ErrKeychainNotAvailable) && warnW != nil {
				_, _ = fmt.Fprintf(warnW, "Warning: System keychain not available. Using plaintext API key.\n")
			}
		}
		return "", nil
	}

	// 3. Plaintext config value — backward compatible, with warning
	if warnW != nil {
		_, _ = fmt.Fprintf(warnW, "Warning: API key stored in plaintext. Run: lore config set-key %s to use system keychain\n", provider)
	}
	return configKey, nil
}
