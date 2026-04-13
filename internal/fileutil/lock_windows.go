//go:build windows

// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Windows implementation of FileLock. Advisory file locking via
// syscall.Flock is not available on Windows; this build uses only
// the in-process sync.Mutex. Cross-process mutual exclusion is not
// provided on Windows — acceptable for a developer CLI tool.

package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var inProcessLocks sync.Map

// FileLock holds an exclusive in-process lock backed by a sidecar
// `.lock` file. On Windows, only goroutine-level exclusion is
// provided; cross-process exclusion is not available.
type FileLock struct {
	fd       *os.File
	lockPath string
	mu       *sync.Mutex
	once     sync.Once
}

// NewFileLock creates (or opens) a `<target>.lock` file and acquires
// an in-process mutex. On Windows, no OS-level advisory lock is taken.
func NewFileLock(targetPath string) (*FileLock, error) {
	dir := filepath.Dir(targetPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("fileutil: lock mkdir: %w", err)
	}
	lockPath := targetPath + ".lock"

	val, _ := inProcessLocks.LoadOrStore(lockPath, &sync.Mutex{})
	mu := val.(*sync.Mutex)
	mu.Lock()

	fd, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		mu.Unlock()
		return nil, fmt.Errorf("fileutil: lock open: %w", err)
	}
	return &FileLock{fd: fd, lockPath: lockPath, mu: mu}, nil
}

// Unlock releases the lock. Safe to call multiple times.
func (l *FileLock) Unlock() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		if l.fd != nil {
			_ = l.fd.Close()
			l.fd = nil
		}
		if l.mu != nil {
			l.mu.Unlock()
			l.mu = nil
		}
	})
}
