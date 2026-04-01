// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import "testing"

func TestDetectInstallMethod(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantMethod InstallMethod
		wantHint   bool
	}{
		{
			name:       "homebrew cellar",
			path:       "/opt/homebrew/Cellar/lore/1.0.0/bin/lore",
			wantMethod: InstallHomebrew,
			wantHint:   true,
		},
		{
			name:       "homebrew bin",
			path:       "/opt/homebrew/bin/lore",
			wantMethod: InstallHomebrew,
			wantHint:   true,
		},
		{
			name:       "linuxbrew",
			path:       "/home/linuxbrew/.linuxbrew/bin/lore",
			wantMethod: InstallHomebrew,
			wantHint:   true,
		},
		{
			name:       "go bin default",
			path:       "/home/user/go/bin/lore",
			wantMethod: InstallGoInstall,
			wantHint:   true,
		},
		{
			name:       "usr local bin",
			path:       "/usr/local/bin/lore",
			wantMethod: InstallBinary,
			wantHint:   false,
		},
		{
			name:       "local bin",
			path:       "/home/user/.local/bin/lore",
			wantMethod: InstallBinary,
			wantHint:   false,
		},
		{
			name:       "tmp directory",
			path:       "/tmp/lore",
			wantMethod: InstallBinary,
			wantHint:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, hint := DetectInstallMethod(tt.path)
			if method != tt.wantMethod {
				t.Errorf("DetectInstallMethod(%q) method = %d, want %d", tt.path, method, tt.wantMethod)
			}
			if tt.wantHint && hint == "" {
				t.Errorf("DetectInstallMethod(%q) expected non-empty hint", tt.path)
			}
			if !tt.wantHint && hint != "" {
				t.Errorf("DetectInstallMethod(%q) expected empty hint, got %q", tt.path, hint)
			}
		})
	}
}
