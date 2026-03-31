// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ReadLanguageOnly reads only the language field from .lorerc without full
// config loading. This is used before Cobra command construction so that
// i18n.T() returns the correct language for command Short/Long strings.
// Returns "" if the file is absent, unreadable, or has no language field.
// No validation, no Viper, no error surfacing — silent fallback.
func ReadLanguageOnly(dir string) string {
	for _, name := range []string{".lorerc.yaml", ".lorerc.yml", ".lorerc"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		var minimal struct {
			Language string `yaml:"language"`
		}
		if err := yaml.Unmarshal(data, &minimal); err != nil {
			return ""
		}
		return minimal.Language
	}
	return ""
}
