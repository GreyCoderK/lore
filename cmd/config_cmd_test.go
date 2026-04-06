// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/credential"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/ui"
)

// mockCredStore is an in-memory credential store for testing.
type mockCredStore struct {
	data map[string][]byte
}

func newMockCredStore() *mockCredStore {
	return &mockCredStore{data: make(map[string][]byte)}
}

func (m *mockCredStore) Set(provider string, secret []byte) error {
	m.data[provider] = append([]byte(nil), secret...)
	return nil
}
func (m *mockCredStore) Get(provider string) ([]byte, error) {
	v, ok := m.data[provider]
	if !ok {
		return nil, credential.ErrNotFound
	}
	return v, nil
}
func (m *mockCredStore) Delete(provider string) error {
	delete(m.data, provider)
	return nil
}
func (m *mockCredStore) List() ([]string, error) {
	var keys []string
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func runConfigCmd(t *testing.T, store credential.CredentialStore, stdinInput string, args ...string) (stdout, stderr string, exitErr error) {
	t.Helper()
	restore := ui.SaveAndDisableColor()
	defer restore()

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		In:  strings.NewReader(stdinInput),
		Out: &out,
		Err: &errBuf,
	}

	cmd := newSetKeyCmd(store, streams)
	if len(args) > 0 && args[0] == "delete-key" {
		cmd = newDeleteKeyCmd(store, streams)
		args = args[1:]
	} else if len(args) > 0 && args[0] == "list-keys" {
		cmd = newListKeysCmd(store, streams)
		args = args[1:]
	}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

func TestConfigSetKey_OK(t *testing.T) {
	store := newMockCredStore()
	_, stderr, err := runConfigCmd(t, store, "sk-secret-123\n", "anthropic")
	if err != nil {
		t.Fatalf("set-key: %v", err)
	}
	if !strings.Contains(stderr, "Stored") {
		t.Errorf("expected 'Stored' message, got: %s", stderr)
	}
	key, _ := store.Get("anthropic")
	if string(key) != "sk-secret-123" {
		t.Errorf("stored key = %q, want sk-secret-123", key)
	}
}

func TestConfigSetKey_UnknownProvider(t *testing.T) {
	store := newMockCredStore()
	_, _, err := runConfigCmd(t, store, "key\n", "gemini")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %q, want 'unknown provider'", err)
	}
}

func TestConfigDeleteKey_OK(t *testing.T) {
	store := newMockCredStore()
	_ = store.Set("anthropic", []byte("old-key"))
	_, stderr, err := runConfigCmd(t, store, "", "delete-key", "anthropic")
	if err != nil {
		t.Fatalf("delete-key: %v", err)
	}
	if !strings.Contains(stderr, "Deleted") {
		t.Errorf("expected 'Deleted' message, got: %s", stderr)
	}
	_, getErr := store.Get("anthropic")
	if getErr != credential.ErrNotFound {
		t.Error("key should be deleted")
	}
}

func TestConfigDeleteKey_UnknownProvider(t *testing.T) {
	store := newMockCredStore()
	_, _, err := runConfigCmd(t, store, "", "delete-key", "gemini")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !strings.Contains(err.Error(), "unknown provider") {
		t.Errorf("error = %q, want 'unknown provider'", err)
	}
}

func TestConfigSetKey_EmptyInput(t *testing.T) {
	store := newMockCredStore()
	_, _, err := runConfigCmd(t, store, "\n", "anthropic")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "empty key") {
		t.Errorf("error = %q, want 'empty key'", err)
	}
}

func TestConfigSetKey_NoInput(t *testing.T) {
	store := newMockCredStore()
	_, _, err := runConfigCmd(t, store, "", "anthropic")
	if err == nil {
		t.Fatal("expected error for no input")
	}
	if !strings.Contains(err.Error(), "no input") {
		t.Errorf("error = %q, want 'no input'", err)
	}
}

func TestConfigListKeys_Masked(t *testing.T) {
	store := newMockCredStore()
	_ = store.Set("anthropic", []byte("secret"))

	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{In: strings.NewReader(""), Out: &out, Err: &errBuf}
	cmd := newListKeysCmd(store, streams)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list-keys: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "anthropic: ****") {
		t.Errorf("expected masked 'anthropic: ****', got: %s", output)
	}
	if !strings.Contains(output, "openai: (not set)") {
		t.Errorf("expected 'openai: (not set)', got: %s", output)
	}
	if strings.Contains(output, "secret") {
		t.Error("API key value should NOT appear in list output")
	}
}
