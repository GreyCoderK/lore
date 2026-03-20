// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build !darwin && !linux

package credential

type fallbackStore struct{}

func newPlatformStore() CredentialStore { return &fallbackStore{} }

func (f *fallbackStore) Set(provider string, secret []byte) error { return ErrKeychainNotAvailable }
func (f *fallbackStore) Get(provider string) ([]byte, error)     { return nil, ErrKeychainNotAvailable }
func (f *fallbackStore) Delete(provider string) error            { return ErrKeychainNotAvailable }
func (f *fallbackStore) List() ([]string, error)                 { return nil, ErrKeychainNotAvailable }
