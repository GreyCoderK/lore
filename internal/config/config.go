package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AI        AIConfig        `yaml:"ai"`
	Angela    AngelaConfig    `yaml:"angela"`
	Templates TemplatesConfig `yaml:"templates"`
	Hooks     HooksConfig     `yaml:"hooks"`
	Output    OutputConfig    `yaml:"output"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(".lorerc")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.SetEnvPrefix("LORE")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := defaultConfig()

	// Load .lorerc (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: read .lorerc: %w", err)
		}
	}

	// Merge .lorerc.local (optional override)
	v.SetConfigName(".lorerc.local")
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: read .lorerc.local: %w", err)
		}
	}

	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return cfg, nil
}
