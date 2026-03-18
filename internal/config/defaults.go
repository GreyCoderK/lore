// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import "github.com/spf13/viper"

type AIConfig struct {
	Provider string `yaml:"provider" mapstructure:"provider"`
	Model    string `yaml:"model" mapstructure:"model"`
	APIKey   string `yaml:"api_key" mapstructure:"api_key"`
}

type AngelaConfig struct {
	Mode      string `yaml:"mode" mapstructure:"mode"`
	MaxTokens int    `yaml:"max_tokens" mapstructure:"max_tokens"`
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

	v.SetDefault("angela.mode", "draft")
	v.SetDefault("angela.max_tokens", 2000)

	v.SetDefault("templates.dir", ".lore/templates")

	v.SetDefault("hooks.post_commit", true)

	v.SetDefault("output.format", "markdown")
	v.SetDefault("output.dir", ".lore/docs")
}
