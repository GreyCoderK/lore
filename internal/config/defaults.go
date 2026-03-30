// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/spf13/viper"
)

// Decision Engine defaults — duplicated here to break the config/ → decision/
// dependency cycle. decision.DefaultConfig() remains the canonical source for
// the Engine itself; these values are only used for Viper defaults.
const (
	defaultThresholdFull      = 60
	defaultThresholdReduced   = 35
	defaultThresholdSuggest   = 15
	defaultLearningMinCommits = 20
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

type NotificationConfig struct {
	Mode         string   `yaml:"mode" mapstructure:"mode"`
	DisabledEnvs []string `yaml:"disabled_envs" mapstructure:"disabled_envs"`
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("language", "en")

	v.SetDefault("ai.provider", "")
	v.SetDefault("ai.model", "")
	v.SetDefault("ai.api_key", "")
	v.SetDefault("ai.endpoint", "http://localhost:11434")
	v.SetDefault("ai.timeout", "30s")

	v.SetDefault("angela.mode", "draft")
	v.SetDefault("angela.max_tokens", 2000)

	v.SetDefault("templates.dir", filepath.Join(domain.LoreDir, domain.TemplatesDir))

	v.SetDefault("hooks.post_commit", true)

	v.SetDefault("output.format", "markdown")
	v.SetDefault("output.dir", filepath.Join(domain.LoreDir, domain.DocsDir))

	v.SetDefault("notification.mode", "auto")
	v.SetDefault("notification.disabled_envs", []string{})

	v.SetDefault("decision.threshold_full", defaultThresholdFull)
	v.SetDefault("decision.threshold_reduced", defaultThresholdReduced)
	v.SetDefault("decision.threshold_suggest", defaultThresholdSuggest)
	v.SetDefault("decision.always_ask", []string{"feat", "breaking"})
	v.SetDefault("decision.always_skip", []string{"docs", "style", "ci", "build"})
	v.SetDefault("decision.critical_scopes", []string{})
	v.SetDefault("decision.learning", true)
	v.SetDefault("decision.learning_min_commits", defaultLearningMinCommits)
}
