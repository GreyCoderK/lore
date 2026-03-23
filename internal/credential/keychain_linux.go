// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build linux

package credential

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type linuxStore struct{}

func newPlatformStore() CredentialStore {
	if _, err := exec.LookPath("secret-tool"); err != nil {
		return &fallbackLinuxStore{}
	}
	return &linuxStore{}
}

func (l *linuxStore) Set(provider string, secret []byte) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	cmd := exec.Command("secret-tool", "store",
		"--label="+ServiceName+"/"+provider,
		"service", ServiceName,
		"account", provider,
	)
	cmd.Stdin = bytes.NewReader(secret)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("credential: secret-tool: set %s: %s", provider, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (l *linuxStore) Get(provider string) ([]byte, error) {
	if err := ValidateProvider(provider); err != nil {
		return nil, err
	}
	cmd := exec.Command("secret-tool", "lookup",
		"service", ServiceName,
		"account", provider,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, ErrNotFound
	}
	result := bytes.TrimSpace(stdout.Bytes())
	if len(result) == 0 {
		return nil, ErrNotFound
	}
	return result, nil
}

func (l *linuxStore) Delete(provider string) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	cmd := exec.Command("secret-tool", "clear",
		"service", ServiceName,
		"account", provider,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// clear on absent = no-op
		return nil
	}
	return nil
}

func (l *linuxStore) List() ([]string, error) {
	var found []string
	for _, p := range KnownProviders {
		_, err := l.Get(p)
		if err == nil {
			found = append(found, p)
		} else if !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("credential: secret-tool: list: %w", err)
		}
	}
	return found, nil
}

// fallbackLinuxStore is used when secret-tool is not installed.
type fallbackLinuxStore struct{}

func (f *fallbackLinuxStore) Set(provider string, secret []byte) error { return ErrKeychainNotAvailable }
func (f *fallbackLinuxStore) Get(provider string) ([]byte, error)     { return nil, ErrKeychainNotAvailable }
func (f *fallbackLinuxStore) Delete(provider string) error            { return ErrKeychainNotAvailable }
func (f *fallbackLinuxStore) List() ([]string, error)                 { return nil, ErrKeychainNotAvailable }
