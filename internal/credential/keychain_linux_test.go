// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build linux

package credential

import (
	"errors"
	"testing"
)

// ---------------------------------------------------------------------------
// fallbackLinuxStore — all methods return ErrKeychainNotAvailable
// ---------------------------------------------------------------------------

func TestFallbackLinuxStore_Set(t *testing.T) {
	f := &fallbackLinuxStore{}
	err := f.Set("anthropic", []byte("sk-test"))
	if !errors.Is(err, ErrKeychainNotAvailable) {
		t.Errorf("Set: got %v, want ErrKeychainNotAvailable", err)
	}
}

func TestFallbackLinuxStore_Get(t *testing.T) {
	f := &fallbackLinuxStore{}
	val, err := f.Get("anthropic")
	if !errors.Is(err, ErrKeychainNotAvailable) {
		t.Errorf("Get: got %v, want ErrKeychainNotAvailable", err)
	}
	if val != nil {
		t.Errorf("Get: got %v, want nil", val)
	}
}

func TestFallbackLinuxStore_Delete(t *testing.T) {
	f := &fallbackLinuxStore{}
	err := f.Delete("anthropic")
	if !errors.Is(err, ErrKeychainNotAvailable) {
		t.Errorf("Delete: got %v, want ErrKeychainNotAvailable", err)
	}
}

func TestFallbackLinuxStore_List(t *testing.T) {
	f := &fallbackLinuxStore{}
	list, err := f.List()
	if !errors.Is(err, ErrKeychainNotAvailable) {
		t.Errorf("List: got %v, want ErrKeychainNotAvailable", err)
	}
	if list != nil {
		t.Errorf("List: got %v, want nil", list)
	}
}

// ---------------------------------------------------------------------------
// ValidateProvider — valid and invalid inputs
// ---------------------------------------------------------------------------

func TestValidateProvider_Valid(t *testing.T) {
	valid := []string{"anthropic", "openai", "ollama", "my-provider", "test_123", "ProviderX"}
	for _, p := range valid {
		if err := ValidateProvider(p); err != nil {
			t.Errorf("ValidateProvider(%q) unexpected error: %v", p, err)
		}
	}
}

func TestValidateProvider_Invalid(t *testing.T) {
	invalid := []string{"", "has space", "semi;colon", "slash/path", "$(inject)", "a\nb"}
	for _, p := range invalid {
		if err := ValidateProvider(p); err == nil {
			t.Errorf("ValidateProvider(%q) should have returned error", p)
		}
	}
}

// ---------------------------------------------------------------------------
// newPlatformStore — returns fallbackLinuxStore when secret-tool is absent
// ---------------------------------------------------------------------------

func TestNewPlatformStore_FallbackWhenNoSecretTool(t *testing.T) {
	// On CI (ubuntu-latest) secret-tool is typically not installed,
	// so newPlatformStore should return *fallbackLinuxStore.
	// If secret-tool happens to be present, it returns *linuxStore which
	// is also valid — we just verify a non-nil CredentialStore is returned.
	s := newPlatformStore()
	if s == nil {
		t.Fatal("newPlatformStore returned nil")
	}

	// Try to identify which type was returned.
	switch s.(type) {
	case *fallbackLinuxStore:
		// Expected on most CI runners.
	case *linuxStore:
		// Acceptable — secret-tool is installed.
	default:
		t.Fatalf("newPlatformStore returned unexpected type %T", s)
	}
}

// ---------------------------------------------------------------------------
// linuxStore — validation errors (Set/Get/Delete with invalid providers)
// ---------------------------------------------------------------------------

func TestLinuxStore_Set_InvalidProvider(t *testing.T) {
	l := &linuxStore{}
	invalidProviders := []string{"", "has space", "semi;colon", "slash/path", "$(inject)"}
	for _, p := range invalidProviders {
		err := l.Set(p, []byte("secret"))
		if err == nil {
			t.Errorf("Set(%q) should have returned error for invalid provider", p)
		}
	}
}

func TestLinuxStore_Get_InvalidProvider(t *testing.T) {
	l := &linuxStore{}
	invalidProviders := []string{"", "has space", "semi;colon", "a/b", "$(cmd)"}
	for _, p := range invalidProviders {
		_, err := l.Get(p)
		if err == nil {
			t.Errorf("Get(%q) should have returned error for invalid provider", p)
		}
	}
}

func TestLinuxStore_Delete_InvalidProvider(t *testing.T) {
	l := &linuxStore{}
	invalidProviders := []string{"", "has space", "semi;colon", "a/b", "$(cmd)"}
	for _, p := range invalidProviders {
		err := l.Delete(p)
		if err == nil {
			t.Errorf("Delete(%q) should have returned error for invalid provider", p)
		}
	}
}

// ---------------------------------------------------------------------------
// linuxStore — cache invalidation on Set and Delete
// ---------------------------------------------------------------------------

func TestLinuxStore_Set_InvalidatesCacheFlag(t *testing.T) {
	l := &linuxStore{listCached: true}
	// Set with an invalid provider will fail but should still invalidate the cache.
	_ = l.Set("bad provider", []byte("x"))
	if l.listCached {
		t.Error("Set should invalidate listCached")
	}
}

func TestLinuxStore_Delete_InvalidatesCacheFlag(t *testing.T) {
	l := &linuxStore{listCached: true}
	// Delete with an invalid provider will fail but should still invalidate the cache.
	_ = l.Delete("bad provider")
	if l.listCached {
		t.Error("Delete should invalidate listCached")
	}
}
