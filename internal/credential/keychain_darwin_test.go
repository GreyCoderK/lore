// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build darwin

package credential

import (
	"errors"
	"testing"
)

func TestServiceName(t *testing.T) {
	d := &darwinStore{account: "testuser"}
	tests := []struct {
		provider string
		want     string
	}{
		{"anthropic", "lore/anthropic"},
		{"openai", "lore/openai"},
		{"ollama", "lore/ollama"},
		{"custom-provider", "lore/custom-provider"},
	}
	for _, tt := range tests {
		got := d.serviceName(tt.provider)
		if got != tt.want {
			t.Errorf("serviceName(%q) = %q, want %q", tt.provider, got, tt.want)
		}
	}
}

func TestDarwinStore_Set_InvalidProvider(t *testing.T) {
	d := &darwinStore{account: "testuser"}
	invalidProviders := []string{"", "has space", "semi;colon", "slash/path", "$(inject)"}
	for _, p := range invalidProviders {
		err := d.Set(p, []byte("secret"))
		if err == nil {
			t.Errorf("Set(%q) should have returned error for invalid provider", p)
		}
	}
}

func TestDarwinStore_Get_InvalidProvider(t *testing.T) {
	d := &darwinStore{account: "testuser"}
	invalidProviders := []string{"", "has space", "semi;colon", "a/b", "$(cmd)"}
	for _, p := range invalidProviders {
		_, err := d.Get(p)
		if err == nil {
			t.Errorf("Get(%q) should have returned error for invalid provider", p)
		}
	}
}

func TestDarwinStore_Delete_InvalidProvider(t *testing.T) {
	d := &darwinStore{account: "testuser"}
	invalidProviders := []string{"", "has space", "semi;colon", "a/b", "$(cmd)"}
	for _, p := range invalidProviders {
		err := d.Delete(p)
		if err == nil {
			t.Errorf("Delete(%q) should have returned error for invalid provider", p)
		}
	}
}

func TestDarwinStore_Get_NotFound(t *testing.T) {
	d := &darwinStore{account: "lore-test-nonexistent-user-xyzzy"}
	_, err := d.Get("anthropic")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get nonexistent: err = %v, want ErrNotFound", err)
	}
}

func TestDarwinStore_Delete_NotFound_IsNoop(t *testing.T) {
	d := &darwinStore{account: "lore-test-nonexistent-user-xyzzy"}
	err := d.Delete("anthropic")
	if err != nil {
		t.Errorf("Delete nonexistent: expected nil (no-op), got %v", err)
	}
}

func TestDarwinStore_List_Empty(t *testing.T) {
	// Use an account name that is extremely unlikely to have any entries.
	d := &darwinStore{account: "lore-test-nonexistent-user-xyzzy"}
	list, err := d.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List: got %d items, want 0", len(list))
	}
}

func TestDarwinStore_List_Caching(t *testing.T) {
	d := &darwinStore{account: "lore-test-nonexistent-user-xyzzy"}

	// First call populates cache.
	list1, err := d.List()
	if err != nil {
		t.Fatalf("List (1st): %v", err)
	}
	if !d.listCached {
		t.Error("listCached should be true after List()")
	}

	// Second call should use cache.
	list2, err := d.List()
	if err != nil {
		t.Fatalf("List (2nd): %v", err)
	}
	if len(list1) != len(list2) {
		t.Errorf("cached List mismatch: %d vs %d", len(list1), len(list2))
	}
}

func TestDarwinStore_Set_InvalidatesCacheFlag(t *testing.T) {
	d := &darwinStore{account: "lore-test-nonexistent-user-xyzzy"}
	d.listCached = true
	// Set with invalid provider will fail but still invalidates cache first.
	_ = d.Set("bad provider", []byte("x"))
	// The cache invalidation happens before validation in the Set method,
	// so listCached should be false.
	if d.listCached {
		t.Error("Set should invalidate listCached")
	}
}

func TestDarwinStore_Delete_InvalidatesCacheFlag(t *testing.T) {
	d := &darwinStore{account: "lore-test-nonexistent-user-xyzzy"}
	d.listCached = true
	// Delete with invalid provider will fail but still invalidates cache first.
	_ = d.Delete("bad provider")
	if d.listCached {
		t.Error("Delete should invalidate listCached")
	}
}

// Integration test: round-trip Set/Get/Delete using a test-only account.
// This actually writes to and reads from the macOS keychain.
func TestDarwinStore_Integration_SetGetDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	const testAccount = "lore-integration-test-account"
	const testProvider = "anthropic"
	const testSecret = "sk-test-integration-secret-12345"

	d := &darwinStore{account: testAccount}

	// Clean up before and after.
	cleanup := func() { _ = d.Delete(testProvider) }
	cleanup()
	t.Cleanup(cleanup)

	// Set
	if err := d.Set(testProvider, []byte(testSecret)); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Get
	got, err := d.Get(testProvider)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != testSecret {
		t.Errorf("Get = %q, want %q", got, testSecret)
	}

	// Delete
	if err := d.Delete(testProvider); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Get after delete should be ErrNotFound
	_, err = d.Get(testProvider)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete: err = %v, want ErrNotFound", err)
	}
}

// Integration test: List with a known entry.
func TestDarwinStore_Integration_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	const testAccount = "lore-integration-test-account"
	const testProvider = "anthropic"

	d := &darwinStore{account: testAccount}

	cleanup := func() { _ = d.Delete(testProvider) }
	cleanup()
	t.Cleanup(cleanup)

	if err := d.Set(testProvider, []byte("sk-list-test")); err != nil {
		t.Fatalf("Set: %v", err)
	}

	list, err := d.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	found := false
	for _, p := range list {
		if p == testProvider {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() = %v, expected to contain %q", list, testProvider)
	}
}

// Integration test: Set overwrites existing value.
func TestDarwinStore_Integration_SetOverwrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	const testAccount = "lore-integration-test-account"
	const testProvider = "openai"

	d := &darwinStore{account: testAccount}
	cleanup := func() { _ = d.Delete(testProvider) }
	cleanup()
	t.Cleanup(cleanup)

	// Set initial value.
	if err := d.Set(testProvider, []byte("first-value")); err != nil {
		t.Fatalf("Set (1st): %v", err)
	}

	// Overwrite with new value.
	if err := d.Set(testProvider, []byte("second-value")); err != nil {
		t.Fatalf("Set (2nd): %v", err)
	}

	got, err := d.Get(testProvider)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "second-value" {
		t.Errorf("Get after overwrite = %q, want %q", got, "second-value")
	}
}

func TestNewPlatformStore_ReturnsDarwinStore(t *testing.T) {
	s := newPlatformStore()
	ds, ok := s.(*darwinStore)
	if !ok {
		t.Fatalf("newPlatformStore() returned %T, want *darwinStore", s)
	}
	if ds.account == "" {
		t.Error("darwinStore.account should not be empty")
	}
}
