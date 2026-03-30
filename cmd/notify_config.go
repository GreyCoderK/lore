// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/notify"
)

// notifyConfigFromApp builds a NotifyConfig from the app config.
// Returns nil if the config has default values (let the notify package use its own defaults).
func notifyConfigFromApp(cfg *config.Config) *notify.NotifyConfig {
	nc := cfg.Notification
	if nc.Mode == "" && len(nc.DisabledEnvs) == 0 {
		return nil // use notify.DefaultNotifyConfig()
	}
	mode := nc.Mode
	if mode == "" {
		mode = notify.ModeAuto
	}
	return &notify.NotifyConfig{
		Mode:         mode,
		DisabledEnvs: nc.DisabledEnvs,
	}
}
