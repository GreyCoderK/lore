// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"errors"
	"fmt"
	"regexp"
)

// ServiceName is the keychain service label prefix for Lore credentials.
const ServiceName = "lore"

// KnownProviders is the single source of truth for supported AI providers.
var KnownProviders = []string{"anthropic", "openai", "ollama"}

var validProviderRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateProvider rejects provider names with special characters.
// Prevents injection into exec.Command arguments and keychain service names.
func validateProvider(name string) error {
	if !validProviderRe.MatchString(name) {
		return fmt.Errorf("credential: invalid provider name %q", name)
	}
	return nil
}

// ErrKeychainNotAvailable signals that no system keychain is available.
var ErrKeychainNotAvailable = errors.New("credential: system keychain not available")

// ErrNotFound signals that the requested credential does not exist.
var ErrNotFound = errors.New("credential: key not found")

// CredentialStore abstracts OS-level credential storage.
type CredentialStore interface {
	Set(provider string, secret []byte) error
	Get(provider string) ([]byte, error)
	Delete(provider string) error
	List() ([]string, error)
}

// NewStore returns a platform-appropriate CredentialStore.
// On macOS: uses security CLI (Keychain).
// On Linux: uses secret-tool CLI (secret-service/libsecret).
// On other platforms: returns a fallback that always returns ErrKeychainNotAvailable.
func NewStore() CredentialStore {
	return newPlatformStore()
}
