// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package notify

import "errors"

var (
	errLoreNotFound   = errors.New("notify: lore binary not found")
	errCLINotFound    = errors.New("notify: IDE CLI not found in PATH")
	errNoIPCSocket    = errors.New("notify: no live IPC socket found")
	errUnsupportedOS  = errors.New("notify: unsupported OS for dialog")
	errFallbackDialog = errors.New("notify: VS Code terminal failed, falling back to dialog")
)
