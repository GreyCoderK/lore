// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// WarnWriter is the destination for config warnings (e.g. unknown fields).
// It defaults to os.Stderr but can be overridden in tests.
var WarnWriter io.Writer = os.Stderr

type Config struct {
	AI        AIConfig        `yaml:"ai"`
	Angela    AngelaConfig    `yaml:"angela"`
	Templates TemplatesConfig `yaml:"templates"`
	Hooks     HooksConfig     `yaml:"hooks"`
	Output    OutputConfig    `yaml:"output"`
}

// warnIfInsecurePerms warns to WarnWriter if .lorerc.local has permissions
// broader than owner-only (0600). This file may contain API keys.
func warnIfInsecurePerms(dir string) {
	path := filepath.Join(dir, ".lorerc.local.yaml")
	info, err := os.Stat(path)
	if err != nil {
		// Try without .yaml extension
		path = filepath.Join(dir, ".lorerc.local.yml")
		info, err = os.Stat(path)
		if err != nil {
			path = filepath.Join(dir, ".lorerc.local")
			info, err = os.Stat(path)
			if err != nil {
				return
			}
		}
	}
	mode := info.Mode().Perm()
	if mode&0o077 != 0 {
		_, _ = fmt.Fprintf(WarnWriter, "Warning: %s is readable by others (mode %04o). Consider: chmod 600 %s\n", filepath.Base(path), mode, filepath.Base(path))
	}
}

// Load is a convenience wrapper that loads config from the current working directory.
func Load() (*Config, error) {
	return LoadFromDir(".")
}

// loadViper builds a Viper instance with defaults, config files, and env vars.
func loadViper(dir string) (*viper.Viper, error) {
	v := viper.New()

	setDefaults(v)

	// Read .lorerc (shared, version-controlled)
	v.SetConfigName(".lorerc")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: read .lorerc: %w", err)
		}
	}

	// Merge .lorerc.local (personal, gitignored)
	v.SetConfigName(".lorerc.local")
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: read .lorerc.local: %w", err)
		}
	} else {
		// Warn if .lorerc.local is world-readable (may contain API keys).
		warnIfInsecurePerms(dir)
	}

	// Environment variables
	v.SetEnvPrefix("LORE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return v, nil
}

// unmarshalConfig converts a Viper instance to a *Config struct.
// It warns on stderr if the config contains unknown fields (e.g. typos).
func unmarshalConfig(v *viper.Viper) (*Config, error) {
	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.TextUnmarshallerHookFunc(),
	)

	// First pass: detect unknown fields via ErrorUnused.
	var probe Config
	strictErr := v.Unmarshal(&probe, func(dc *mapstructure.DecoderConfig) {
		dc.ErrorUnused = true
		dc.DecodeHook = decodeHook
	})
	if strictErr != nil {
		_, _ = fmt.Fprintf(WarnWriter, "Warning: %s\n", strictErr)
	}

	// Second pass: always succeed so unknown fields don't block the user.
	var cfg Config
	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.DecodeHook = decodeHook
	}); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	return &cfg, nil
}

// LoadFromDir loads configuration from a specific directory.
func LoadFromDir(dir string) (*Config, error) {
	v, err := loadViper(dir)
	if err != nil {
		return nil, err
	}
	return unmarshalConfig(v)
}

// RegisterFlags declares persistent CLI flags on the given command.
// Called by cmd/root.go — no Viper dependency is exposed.
func RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("ai-provider", "", "AI provider (anthropic, openai, ollama)")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output")
	cmd.PersistentFlags().Bool("verbose", false, "Show detailed output")
	cmd.PersistentFlags().Bool("no-color", false, "Disable color output")
}

// bindFlags binds Cobra flags to Viper keys internally.
// Only flags that map to Config struct fields are bound here.
// --quiet, --verbose, --no-color are handled by IOStreams in root.go, not via Viper.
func bindFlags(v *viper.Viper, cmd *cobra.Command) error {
	if err := v.BindPFlag("ai.provider", cmd.Flags().Lookup("ai-provider")); err != nil {
		return fmt.Errorf("config: bind flag ai-provider: %w", err)
	}
	return nil
}

// LoadFromDirWithFlags loads config with the full cascade including CLI flag overrides.
// Called by cmd/root.go in PersistentPreRunE after RegisterFlags.
func LoadFromDirWithFlags(dir string, cmd *cobra.Command) (*Config, error) {
	v, err := loadViper(dir)
	if err != nil {
		return nil, err
	}
	if err := bindFlags(v, cmd); err != nil {
		return nil, err
	}
	return unmarshalConfig(v)
}
