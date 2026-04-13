// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

// TestDetectInstallMethod_GOPATHBin verifies that GOPATH/bin detection works
// when GOPATH env is explicitly set.
func TestDetectInstallMethod_GOPATHBin(t *testing.T) {
	fakeGOPATH := "/home/user/mygopath"
	t.Setenv("GOPATH", fakeGOPATH)

	path := fakeGOPATH + "/bin/lore"
	method, hint := DetectInstallMethod(path)
	if method != InstallGoInstall {
		t.Errorf("DetectInstallMethod(%q) method = %d, want InstallGoInstall (%d)", path, method, InstallGoInstall)
	}
	if hint == "" {
		t.Errorf("DetectInstallMethod(%q) expected non-empty hint for GOPATH/bin", path)
	}
}

// TestDetectInstallMethod_EmptyGOPATH verifies that empty GOPATH falls through to binary.
func TestDetectInstallMethod_EmptyGOPATH(t *testing.T) {
	t.Setenv("GOPATH", "")

	path := "/usr/bin/lore"
	method, hint := DetectInstallMethod(path)
	if method != InstallBinary {
		t.Errorf("DetectInstallMethod(%q) method = %d, want InstallBinary", path, method)
	}
	if hint != "" {
		t.Errorf("expected empty hint, got %q", hint)
	}
}

// TestInstallMethodConstants checks the constants are the expected iota values.
func TestInstallMethodConstants(t *testing.T) {
	if InstallBinary != 0 {
		t.Errorf("InstallBinary = %d, want 0", InstallBinary)
	}
	if InstallHomebrew != 1 {
		t.Errorf("InstallHomebrew = %d, want 1", InstallHomebrew)
	}
	if InstallGoInstall != 2 {
		t.Errorf("InstallGoInstall = %d, want 2", InstallGoInstall)
	}
}

// TestListNewerReleases_MoreThan20 verifies the 20-entry cap is enforced.
func TestListNewerReleases_MoreThan20(t *testing.T) {
	releases := make([]ReleaseInfo, 25)
	for i := 0; i < 25; i++ {
		releases[i] = ReleaseInfo{TagName: "v2." + string(rune('0'+i%10)) + ".0"}
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	newer, err := ListNewerReleases(ctx, client, "test/repo", "v0.0.0")
	if err != nil {
		t.Fatalf("ListNewerReleases: %v", err)
	}
	if len(newer) > 20 {
		t.Errorf("expected at most 20 newer releases, got %d", len(newer))
	}
}

// TestFetchReleases_InvalidJSON tests the JSON decode error path.
func TestFetchReleases_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not valid json {{{"))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	_, err := CheckLatestRelease(ctx, client, "test/repo")
	if err == nil {
		t.Error("expected error for invalid JSON response")
	}
}

// TestListNewerReleases_FetchError tests error propagation from fetchReleases.
func TestListNewerReleases_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	_, err := ListNewerReleases(ctx, client, "test/repo", "v1.0.0")
	if err == nil {
		t.Error("expected error for 503 status in ListNewerReleases")
	}
}

// TestFindRelease_FetchError tests error propagation in FindRelease.
func TestFindRelease_FetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	_, err := FindRelease(ctx, client, "test/repo", "v1.0.0")
	if err == nil {
		t.Error("expected error for 500 status in FindRelease")
	}
}

// TestVerifySHA256_MissingFile tests the file-open error path in VerifySHA256.
func TestVerifySHA256_MissingFile(t *testing.T) {
	err := VerifySHA256("/nonexistent/path/file.bin", "abc123")
	if err == nil {
		t.Error("expected error for missing file in VerifySHA256")
	}
}

// TestCopyFile_MissingSource tests copyFile with a missing source file.
func TestCopyFile_MissingSource(t *testing.T) {
	tmp := t.TempDir()
	err := copyFile("/nonexistent/src.bin", tmp+"/dst.bin", 0644)
	if err == nil {
		t.Error("expected error for missing source in copyFile")
	}
}

// TestCheckWritable_NonExistentDir tests checkWritable on a non-existent directory.
func TestCheckWritable_NonExistentDir(t *testing.T) {
	err := checkWritable("/nonexistent/dir/path")
	if err == nil {
		t.Error("expected error for non-existent directory in checkWritable")
	}
}

// TestExtractBinary_TarGz_InvalidGzip tests extractFromTarGz on corrupt data.
func TestExtractBinary_TarGz_InvalidGzip(t *testing.T) {
	tmp := t.TempDir()
	archivePath := tmp + "/bad.tar.gz"
	if err := os.WriteFile(archivePath, []byte("not a valid gzip content"), 0644); err != nil {
		t.Fatal(err)
	}

	destDir := tmp + "/out"
	os.MkdirAll(destDir, 0755)

	_, err := ExtractBinary(archivePath, destDir)
	if err == nil {
		t.Error("expected error for invalid gzip data")
	}
}

// TestReplaceBinary_WriteFailure tests that ReplaceBinary fails when the target
// directory is read-only (simulated with chmod).
func TestReplaceBinary_WriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod does not restrict writes on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root, chmod 0555 does not restrict root")
	}

	tmp := t.TempDir()

	// Create old binary in a read-only subdirectory
	roDir := tmp + "/ro"
	os.MkdirAll(roDir, 0755)
	roPath := roDir + "/lore"
	os.WriteFile(roPath, []byte("old"), 0755)

	newPath := tmp + "/lore.new"
	os.WriteFile(newPath, []byte("new"), 0755)

	os.Chmod(roDir, 0555) // read-only dir
	defer os.Chmod(roDir, 0755)

	err := ReplaceBinary(roPath, newPath)
	if err == nil {
		t.Error("expected error when target directory is read-only")
	}
}
