// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"reflect"
	"testing"

	"github.com/greycoderk/lore/internal/config"
)

func TestApplySynthesizerFlags_NoFlagWins(t *testing.T) {
	cfg := &config.Config{}
	cfg.Angela.Synthesizers.Enabled = []string{"api-postman", "sql-query"}

	applySynthesizerFlags(cfg, []string{"new-list"}, true /*noFlag*/, true)

	if cfg.Angela.Synthesizers.Enabled != nil {
		t.Fatalf("--no-synthesizers must clear Enabled, got %v", cfg.Angela.Synthesizers.Enabled)
	}
}

func TestApplySynthesizerFlags_SynthesizersListReplacesEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Angela.Synthesizers.Enabled = []string{"old"}

	applySynthesizerFlags(cfg, []string{"a", "b"}, false, true /*flag changed*/)

	if !reflect.DeepEqual(cfg.Angela.Synthesizers.Enabled, []string{"a", "b"}) {
		t.Fatalf("flag did not replace Enabled: %v", cfg.Angela.Synthesizers.Enabled)
	}
}

func TestApplySynthesizerFlags_NoChangePreservesConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Angela.Synthesizers.Enabled = []string{"from-config"}

	applySynthesizerFlags(cfg, nil, false, false /*flag NOT changed*/)

	if !reflect.DeepEqual(cfg.Angela.Synthesizers.Enabled, []string{"from-config"}) {
		t.Fatalf("config value lost: %v", cfg.Angela.Synthesizers.Enabled)
	}
}

func TestApplySynthesizerFlags_EmptyListExplicitlyClears(t *testing.T) {
	cfg := &config.Config{}
	cfg.Angela.Synthesizers.Enabled = []string{"from-config"}

	// User passed `--synthesizers ""` (empty list, but flag was changed).
	applySynthesizerFlags(cfg, []string{}, false, true)

	if len(cfg.Angela.Synthesizers.Enabled) != 0 {
		t.Fatalf("explicit empty list should clear, got %v", cfg.Angela.Synthesizers.Enabled)
	}
}

func TestApplySynthesizerFlags_NilCfgIsNoOp(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nil cfg must not panic: %v", r)
		}
	}()
	applySynthesizerFlags(nil, []string{"x"}, true, true)
}
