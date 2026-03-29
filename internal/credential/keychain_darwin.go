// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build darwin

package credential

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

type darwinStore struct {
	account string // OS username, used as the keychain account field
}

func newPlatformStore() CredentialStore {
	account := "lore"
	if u, err := user.Current(); err == nil {
		account = u.Username
	}
	return &darwinStore{account: account}
}

func (d *darwinStore) serviceName(provider string) string {
	return ServiceName + "/" + provider
}

func (d *darwinStore) Set(provider string, secret []byte) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	// Delete first, then add — avoids duplicate errors without -U.
	// We don't use -w <key> because that exposes the secret in process args (visible via ps).
	if err := d.Delete(provider); err != nil && !errors.Is(err, ErrNotFound) {
		fmt.Fprintf(os.Stderr, "credential: keychain: pre-delete %s: %v\n", provider, err)
	}

	cmd := exec.Command("security", "add-generic-password",
		"-s", d.serviceName(provider),
		"-a", d.account,
		"-w",
	)
	// Pipe secret via stdin — not visible in process args
	cmd.Stdin = bytes.NewReader(append(secret, '\n'))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("credential: keychain: set %s failed", provider)
	}
	return nil
}

func (d *darwinStore) Get(provider string) ([]byte, error) {
	if err := ValidateProvider(provider); err != nil {
		return nil, err
	}
	cmd := exec.Command("security", "find-generic-password",
		"-s", d.serviceName(provider),
		"-a", d.account,
		"-w",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "could not be found") || strings.Contains(errMsg, "SecKeychainSearchCopyNext") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("credential: keychain: get %s failed", provider)
	}

	return bytes.TrimSpace(stdout.Bytes()), nil
}

func (d *darwinStore) Delete(provider string) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	cmd := exec.Command("security", "delete-generic-password",
		"-s", d.serviceName(provider),
		"-a", d.account,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if strings.Contains(errMsg, "could not be found") || strings.Contains(errMsg, "SecKeychainSearchCopyNext") {
			return nil // delete absent = no-op
		}
		return fmt.Errorf("credential: keychain: delete %s failed", provider)
	}
	return nil
}

func (d *darwinStore) List() ([]string, error) {
	// Check each known provider rather than parsing dump-keychain
	var found []string

	for _, p := range KnownProviders {
		_, err := d.Get(p)
		if err == nil {
			found = append(found, p)
		} else if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("credential: keychain: list: %w", err)
		}
	}
	return found, nil
}
