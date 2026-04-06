// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// AssetName returns the expected archive filename for the current platform.
// Mirrors the GoReleaser name_template: lore_{{title .Os}}_{{arch}}.tar.gz
func AssetName() string {
	// Title-case the OS: "darwin" → "Darwin", "linux" → "Linux"
	titleOS := cases.Title(language.English).String(runtime.GOOS)

	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}

	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	return fmt.Sprintf("lore_%s_%s.%s", titleOS, arch, ext)
}

// DownloadAsset downloads a release asset from url into destDir.
// Returns the path to the downloaded file.
func DownloadAsset(ctx context.Context, client *http.Client, url string, destDir string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "lore-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: unexpected status %d", resp.StatusCode)
	}

	name := filepath.Base(url)
	dest := filepath.Join(destDir, name)

	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return "", err
	}
	return dest, f.Close()
}

// DownloadChecksum downloads checksums.txt and extracts the SHA256 hash
// for the given archive filename.
func DownloadChecksum(ctx context.Context, client *http.Client, checksumsURL string, archiveName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumsURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "lore-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksums: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hash>  <filename>"
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == archiveName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %s", archiveName)
}

// VerifySHA256 computes the SHA256 of the file at path and compares it
// to the expected hex-encoded hash.
func VerifySHA256(path string, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: got %s, want %s", actual, expected)
	}
	return nil
}

// ExtractBinary extracts the "lore" binary from a tar.gz or zip archive
// and returns the path to the extracted binary.
func ExtractBinary(archivePath string, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractFromZip(archivePath, destDir)
	}
	return extractFromTarGz(archivePath, destDir)
}

func extractFromTarGz(archivePath string, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	binaryName := "lore"
	if runtime.GOOS == "windows" {
		binaryName = "lore.exe"
	}

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			dest := filepath.Join(destDir, binaryName)
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0755)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return "", err
			}
			return dest, out.Close()
		}
	}
	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

func extractFromZip(archivePath string, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = r.Close() }()

	binaryName := "lore"
	if runtime.GOOS == "windows" {
		binaryName = "lore.exe"
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		dest := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = out.Close()
			_ = rc.Close()
			return "", err
		}
		_ = rc.Close()
		if err := out.Close(); err != nil {
			return "", err
		}
		return dest, nil
	}
	return "", fmt.Errorf("binary %q not found in archive", binaryName)
}

// ReplaceBinary atomically replaces the binary at targetPath with newBinaryPath.
// Preserves the original file permissions.
func ReplaceBinary(targetPath string, newBinaryPath string) error {
	info, err := os.Stat(targetPath)
	if err != nil {
		return err
	}
	perm := info.Mode().Perm()

	// Verify write access
	dir := filepath.Dir(targetPath)
	if err := checkWritable(dir); err != nil {
		return err
	}

	// Rename current binary to .bak (on Unix, running binary can be renamed)
	bakPath := targetPath + ".bak"
	if err := os.Rename(targetPath, bakPath); err != nil {
		return fmt.Errorf("rename old binary: %w", err)
	}

	// Copy new binary to target path
	if err := copyFile(newBinaryPath, targetPath, perm); err != nil {
		// Try to restore backup
		_ = os.Rename(bakPath, targetPath)
		return fmt.Errorf("install new binary: %w", err)
	}

	// Remove backup
	_ = os.Remove(bakPath)
	return nil
}

func checkWritable(dir string) error {
	tmp, err := os.CreateTemp(dir, ".lore-upgrade-check-*")
	if err != nil {
		return fmt.Errorf("no write permission: %w", err)
	}
	name := tmp.Name()
	_ = tmp.Close()
	_ = os.Remove(name)
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}

	if _, err = io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
