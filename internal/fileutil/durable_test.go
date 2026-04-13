// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package fileutil_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/fileutil"
)

func TestAtomicWriteJSON_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	v := payload{Name: "lore", Count: 42}

	if err := fileutil.AtomicWriteJSON(path, v, 0o644); err != nil {
		t.Fatalf("AtomicWriteJSON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got payload
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Name != v.Name || got.Count != v.Count {
		t.Errorf("decoded = %+v, want %+v", got, v)
	}
}

func TestAtomicWriteJSON_IsIndented(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := fileutil.AtomicWriteJSON(path, map[string]int{"a": 1}, 0o644); err != nil {
		t.Fatalf("AtomicWriteJSON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "\n") {
		t.Error("expected indented JSON (with newlines)")
	}
}

func TestAtomicWriteJSON_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := fileutil.AtomicWriteJSON(path, map[string]string{"k": "first"}, 0o644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := fileutil.AtomicWriteJSON(path, map[string]string{"k": "second"}, 0o644); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "second") {
		t.Errorf("expected 'second' in file, got: %s", data)
	}
	if strings.Contains(string(data), "first") {
		t.Errorf("old content 'first' still present: %s", data)
	}
}

func TestAtomicWriteJSON_BadDir(t *testing.T) {
	err := fileutil.AtomicWriteJSON("/nonexistent/dir/state.json", map[string]int{"x": 1}, 0o644)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestAtomicWriteJSON_UnmarshalableValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	ch := make(chan int)
	err := fileutil.AtomicWriteJSON(path, ch, 0o644)
	if err == nil {
		t.Fatal("expected marshal error for channel value")
	}
	if !strings.Contains(err.Error(), "fileutil:") {
		t.Errorf("expected 'fileutil:' prefix, got: %v", err)
	}
}

func TestAtomicWriteJSON_NoTempLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := fileutil.AtomicWriteJSON(path, map[string]int{"a": 1}, 0o644); err != nil {
		t.Fatalf("AtomicWriteJSON: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "state.json" {
			t.Errorf("unexpected temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWriteStream_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.dat")

	content := []byte("stream content here")
	r := bytes.NewReader(content)

	if err := fileutil.AtomicWriteStream(path, r, 0o644); err != nil {
		t.Fatalf("AtomicWriteStream: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content = %q, want %q", data, content)
	}
}

func TestAtomicWriteStream_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.dat")

	if err := fileutil.AtomicWriteStream(path, bytes.NewReader([]byte("first")), 0o644); err != nil {
		t.Fatalf("first write: %v", err)
	}
	if err := fileutil.AtomicWriteStream(path, bytes.NewReader([]byte("second")), 0o644); err != nil {
		t.Fatalf("second write: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "second" {
		t.Errorf("content = %q, want %q", string(data), "second")
	}
}

func TestAtomicWriteStream_EmptyReader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.dat")

	if err := fileutil.AtomicWriteStream(path, bytes.NewReader(nil), 0o644); err != nil {
		t.Fatalf("AtomicWriteStream empty: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected empty file, got %d bytes", len(data))
	}
}

func TestAtomicWriteStream_BadDir(t *testing.T) {
	r := bytes.NewReader([]byte("data"))
	err := fileutil.AtomicWriteStream("/nonexistent/dir/backup.dat", r, 0o644)
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestAtomicWriteStream_NoTempLeftOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup.dat")

	if err := fileutil.AtomicWriteStream(path, bytes.NewReader([]byte("data")), 0o644); err != nil {
		t.Fatalf("AtomicWriteStream: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "backup.dat" {
			t.Errorf("unexpected temp file left behind: %s", e.Name())
		}
	}
}

func TestAtomicWriteStream_LargeData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.dat")

	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = byte(i % 256)
	}

	if err := fileutil.AtomicWriteStream(path, bytes.NewReader(big), 0o644); err != nil {
		t.Fatalf("AtomicWriteStream large: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(data, big) {
		t.Errorf("large stream content mismatch (size got=%d want=%d)", len(data), len(big))
	}
}
