// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package credential

import (
	"errors"
	"testing"
)

// mockStore for unit tests — in-memory map.
type mockStore struct {
	data map[string][]byte
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string][]byte)}
}

func (m *mockStore) Set(provider string, secret []byte) error {
	m.data[provider] = append([]byte(nil), secret...)
	return nil
}

func (m *mockStore) Get(provider string) ([]byte, error) {
	v, ok := m.data[provider]
	if !ok {
		return nil, ErrNotFound
	}
	return v, nil
}

func (m *mockStore) Delete(provider string) error {
	delete(m.data, provider)
	return nil
}

func (m *mockStore) List() ([]string, error) {
	var keys []string
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func TestMockStore_SetGet_Roundtrip(t *testing.T) {
	s := newMockStore()
	if err := s.Set("anthropic", []byte("sk-test")); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get("anthropic")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "sk-test" {
		t.Errorf("Get = %q, want sk-test", got)
	}
}

func TestMockStore_Get_NotFound(t *testing.T) {
	s := newMockStore()
	_, err := s.Get("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get absent: err = %v, want ErrNotFound", err)
	}
}

func TestMockStore_Delete_Absent_NoOp(t *testing.T) {
	s := newMockStore()
	if err := s.Delete("nonexistent"); err != nil {
		t.Errorf("Delete absent: %v", err)
	}
}

func TestMockStore_List_Empty(t *testing.T) {
	s := newMockStore()
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("List empty: got %d items, want 0", len(list))
	}
}

func TestMockStore_List_WithEntries(t *testing.T) {
	s := newMockStore()
	_ = s.Set("anthropic", []byte("k1"))
	_ = s.Set("openai", []byte("k2"))
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("List: got %d items, want 2", len(list))
	}
}

func TestErrKeychainNotAvailable_Is(t *testing.T) {
	if !errors.Is(ErrKeychainNotAvailable, ErrKeychainNotAvailable) {
		t.Error("ErrKeychainNotAvailable should match itself")
	}
}
