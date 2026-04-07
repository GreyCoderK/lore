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
	account    string // OS username, used as the keychain account field
	cachedList []string
	listCached bool
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
	d.listCached = false // Invalidate cache
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	// Delete first, then add — avoids duplicate errors without -U.
	// We don't use -w <key> because that exposes the secret in process args (visible via ps).
	if err := d.Delete(provider); err != nil && !errors.Is(err, ErrNotFound) {
		fmt.Fprintf(os.Stderr, "credential: keychain: pre-delete %s: %v\n", provider, err)
	}

	// Pass secret as -w argument. While visible in /proc on Linux, macOS
	// security(1) is the only reliable way — stdin piping with -w (no arg)
	// stores empty values on some macOS versions.
	cmd := exec.Command("security", "add-generic-password",
		"-s", d.serviceName(provider),
		"-a", d.account,
		"-w", string(secret),
	)
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

	val := bytes.TrimSpace(stdout.Bytes())
	if len(val) == 0 {
		return nil, ErrNotFound
	}
	return val, nil
}

func (d *darwinStore) Delete(provider string) error {
	d.listCached = false // Invalidate cache
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
	if d.listCached {
		return d.cachedList, nil
	}
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
	d.cachedList = found
	d.listCached = true
	return found, nil
}
