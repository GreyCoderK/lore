// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"time"

	"github.com/spf13/viper"
)

type AIConfig struct {
	Provider string        `yaml:"provider" mapstructure:"provider"`
	Model    string        `yaml:"model" mapstructure:"model"`
	APIKey   string        `yaml:"api_key" mapstructure:"api_key"`
	Endpoint string        `yaml:"endpoint" mapstructure:"endpoint"`
	Timeout  time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

type AngelaConfig struct {
	Mode       string                 `yaml:"mode" mapstructure:"mode"`
	MaxTokens  int                    `yaml:"max_tokens" mapstructure:"max_tokens"`
	StyleGuide map[string]interface{} `yaml:"style_guide" mapstructure:"style_guide"`
}

type TemplatesConfig struct {
	Dir string `yaml:"dir" mapstructure:"dir"`
}

type HooksConfig struct {
	PostCommit bool `yaml:"post_commit" mapstructure:"post_commit"`
}

type OutputConfig struct {
	Dir    string `yaml:"dir" mapstructure:"dir"`
	Format string `yaml:"format" mapstructure:"format"`
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("ai.provider", "")
	v.SetDefault("ai.model", "")
	v.SetDefault("ai.api_key", "")
	v.SetDefault("ai.endpoint", "http://localhost:11434")
	v.SetDefault("ai.timeout", "30s")

	v.SetDefault("angela.mode", "draft")
	v.SetDefault("angela.max_tokens", 2000)

	v.SetDefault("templates.dir", ".lore/templates")

	v.SetDefault("hooks.post_commit", true)

	v.SetDefault("output.format", "markdown")
	v.SetDefault("output.dir", ".lore/docs")
}
