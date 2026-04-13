// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package fileutil — file locking primitive.
//
// Two concurrent `lore angela draft --all` or `lore angela review`
// invocations on the same repo used to race on the state file save.
// The atomic rename only protected each writer from itself; the second
// rename clobbered the first, and any incremental state
// (PruneMissingEntries, lifecycle updates) from the losing run silently
// disappeared.
//
// FileLock acquires an exclusive advisory lock on a companion `.lock`
// file so the critical section "load → mutate → save" runs serially
// across processes sharing the same filesystem. The underlying
// mechanism is syscall.Flock on Unix. Returns an error on platforms or
// filesystems that don't support advisory locking (e.g., NFS). The
// caller should handle the error gracefully.

package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

var inProcessLocks sync.Map

// FileLock holds an exclusive advisory lock on a side-car `.lock`
// file. Created via NewFileLock and released with Unlock. Not safe
// for concurrent use by multiple goroutines within a process; the
// caller should serialize access.
type FileLock struct {
	fd       *os.File
	lockPath string
	mu       *sync.Mutex
	once     sync.Once
}

// NewFileLock creates (or opens) a `<target>.lock` file next to the
// supplied target path and takes an exclusive advisory lock on it.
// Callers MUST defer Unlock to release the lock and close the fd.
//
// The lock file is never removed so concurrent processes that race
// to create it cannot accidentally delete a file another process
// holds. The lock file contains nothing; only its inode participates
// in the flock contract.
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
	if err := syscall.Flock(int(fd.Fd()), syscall.LOCK_EX); err != nil {
		_ = fd.Close()
		mu.Unlock()
		return nil, fmt.Errorf("fileutil: flock: %w", err)
	}
	return &FileLock{fd: fd, lockPath: lockPath, mu: mu}, nil
}

// Unlock releases the advisory lock and closes the file descriptor.
// Safe to call multiple times; the second call is a no-op via sync.Once.
func (l *FileLock) Unlock() {
	if l == nil {
		return
	}
	l.once.Do(func() {
		if l.fd != nil {
			_ = syscall.Flock(int(l.fd.Fd()), syscall.LOCK_UN)
			_ = l.fd.Close()
			l.fd = nil
		}
		if l.mu != nil {
			l.mu.Unlock()
			l.mu = nil
		}
	})
}
