// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"io"
	"net/url"
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

// maxConfigSize is the maximum allowed size for a config file (1 MB).
const maxConfigSize = 1 << 20

type Config struct {
	Language     string             `yaml:"language" mapstructure:"language"`
	AI           AIConfig           `yaml:"ai"`
	Angela       AngelaConfig       `yaml:"angela"`
	Decision     DecisionConfig     `yaml:"decision"`
	Notification NotificationConfig `yaml:"notification"`
	Templates    TemplatesConfig    `yaml:"templates"`
	Hooks        HooksConfig        `yaml:"hooks"`
	Output       OutputConfig       `yaml:"output"`
}

type DecisionConfig struct {
	ThresholdFull      int      `yaml:"threshold_full" mapstructure:"threshold_full"`
	ThresholdReduced   int      `yaml:"threshold_reduced" mapstructure:"threshold_reduced"`
	ThresholdSuggest   int      `yaml:"threshold_suggest" mapstructure:"threshold_suggest"`
	AlwaysAsk          []string `yaml:"always_ask" mapstructure:"always_ask"`
	AlwaysSkip         []string `yaml:"always_skip" mapstructure:"always_skip"`
	CriticalScopes     []string `yaml:"critical_scopes" mapstructure:"critical_scopes"`
	Learning           bool     `yaml:"learning" mapstructure:"learning"`
	LearningMinCommits int      `yaml:"learning_min_commits" mapstructure:"learning_min_commits"`
}

// enforceSecurePerms checks and fixes permissions on .lorerc.local if they're
// broader than owner-only (0600). This file may contain API keys.
func enforceSecurePerms(dir string) {
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
	if info.Mode().Perm()&0o077 != 0 {
		if chmodErr := os.Chmod(path, 0o600); chmodErr != nil {
			_, _ = fmt.Fprintf(WarnWriter, "Warning: could not fix permissions on %s: %v\n", filepath.Base(path), chmodErr)
		} else {
			_, _ = fmt.Fprintf(WarnWriter, "Notice: fixed %s permissions to 0600\n", filepath.Base(path))
		}
	}
}

// Load is a convenience wrapper that loads config from the current working directory.
func Load() (*Config, error) {
	return LoadFromDir(".")
}

// checkConfigFileSize checks that a config file does not exceed maxConfigSize.
// It tries .yaml, .yml, and bare extensions in order.
func checkConfigFileSize(dir, name string) error {
	for _, ext := range []string{".yaml", ".yml", ""} {
		candidate := filepath.Join(dir, name+ext)
		info, err := os.Stat(candidate)
		if err != nil {
			continue
		}
		if info.Size() > maxConfigSize {
			return fmt.Errorf("config: %s is too large (%d bytes, max %d)", filepath.Base(candidate), info.Size(), maxConfigSize)
		}
		return nil
	}
	return nil // file not found — not an error
}

// warnConfigVariants warns if multiple config file variants exist in dir,
// which would make the effective config ambiguous.
func warnConfigVariants(dir string, w io.Writer) {
	variants := []string{".lorerc", ".lorerc.yaml", ".lorerc.yml"}
	var found []string
	for _, v := range variants {
		path := filepath.Join(dir, v)
		if _, err := os.Stat(path); err == nil {
			found = append(found, v)
		}
	}
	if len(found) > 1 {
		_, _ = fmt.Fprintf(w, "Warning: multiple config files found (%s) — using %s\n",
			strings.Join(found, ", "), found[0])
	}
}

// loadViper builds a Viper instance with defaults, config files, and env vars.
func loadViper(dir string) (*viper.Viper, error) {
	v := viper.New()

	setDefaults(v)

	// Warn if ambiguous config file variants exist.
	warnConfigVariants(dir, WarnWriter)

	// Pre-check config file sizes before parsing.
	if err := checkConfigFileSize(dir, ".lorerc"); err != nil {
		return nil, err
	}
	if err := checkConfigFileSize(dir, ".lorerc.local"); err != nil {
		return nil, err
	}

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
		// Enforce secure permissions on .lorerc.local (may contain API keys).
		enforceSecurePerms(dir)

		// Warn if .lorerc.local overrides ai.endpoint to a non-HTTPS remote host
		if endpoint := v.GetString("ai.endpoint"); endpoint != "" {
			if u, err := url.Parse(endpoint); err == nil {
				host := strings.Split(u.Host, ":")[0]
				isLocal := host == "localhost" || host == "127.0.0.1" || host == "::1"
				if u.Scheme == "http" && !isLocal {
					_, _ = fmt.Fprintf(WarnWriter, "Warning: ai.endpoint uses insecure http:// for remote host %q — consider https://\n", u.Host)
				}
			}
		}
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
		_, _ = fmt.Fprintf(WarnWriter, "Warning: %s\nRun 'lore doctor' to validate configuration.\n", strictErr)
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
	cmd.PersistentFlags().String("language", "", "Override display language (en, fr)")
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
	if err := v.BindPFlag("language", cmd.Flags().Lookup("language")); err != nil {
		return fmt.Errorf("config: bind flag language: %w", err)
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
