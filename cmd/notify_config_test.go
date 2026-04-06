// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/notify"
)

func TestNotifyConfigFromApp_Default(t *testing.T) {
	cfg := &config.Config{}
	result := notifyConfigFromApp(cfg)
	if result != nil {
		t.Error("default config should return nil (use package defaults)")
	}
}

func TestNotifyConfigFromApp_WithMode(t *testing.T) {
	cfg := &config.Config{
		Notification: config.NotificationConfig{
			Mode: "dialog",
		},
	}
	result := notifyConfigFromApp(cfg)
	if result == nil {
		t.Fatal("expected non-nil config")
	}
	if result.Mode != notify.ModeDialog {
		t.Errorf("Mode = %q, want %q", result.Mode, notify.ModeDialog)
	}
}

func TestNotifyConfigFromApp_WithDisabledEnvs(t *testing.T) {
	cfg := &config.Config{
		Notification: config.NotificationConfig{
			DisabledEnvs: []string{"ci"},
		},
	}
	result := notifyConfigFromApp(cfg)
	if result == nil {
		t.Fatal("expected non-nil config")
	}
	if result.Mode != notify.ModeAuto {
		t.Errorf("Mode = %q, want %q (auto fallback)", result.Mode, notify.ModeAuto)
	}
	if len(result.DisabledEnvs) != 1 || result.DisabledEnvs[0] != "ci" {
		t.Errorf("DisabledEnvs = %v, want [ci]", result.DisabledEnvs)
	}
}
