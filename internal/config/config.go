package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Config struct {
	AI        AIConfig        `yaml:"ai"`
	Angela    AngelaConfig    `yaml:"angela"`
	Templates TemplatesConfig `yaml:"templates"`
	Hooks     HooksConfig     `yaml:"hooks"`
	Output    OutputConfig    `yaml:"output"`
}

// Load is a convenience wrapper that loads config from the current working directory.
func Load() (*Config, error) {
	return LoadFromDir(".")
}

// LoadFromDir loads configuration from a specific directory.
func LoadFromDir(dir string) (*Config, error) {
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
	}

	// Environment variables
	v.SetEnvPrefix("LORE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return &cfg, nil
}

// RegisterFlags declares persistent CLI flags on the given command.
// Called by cmd/root.go in init() — no Viper dependency is exposed.
func RegisterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("ai-provider", "", "AI provider (anthropic, openai, ollama)")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress non-essential output")
	cmd.PersistentFlags().Bool("verbose", false, "Show detailed output")
	cmd.PersistentFlags().Bool("no-color", false, "Disable color output")
}

// bindFlags binds Cobra flags to Viper keys internally.
func bindFlags(v *viper.Viper, cmd *cobra.Command) {
	v.BindPFlag("ai.provider", cmd.PersistentFlags().Lookup("ai-provider"))
	v.BindPFlag("output.quiet", cmd.PersistentFlags().Lookup("quiet"))
	v.BindPFlag("output.verbose", cmd.PersistentFlags().Lookup("verbose"))
}

// LoadFromDirWithFlags loads config with the full cascade including CLI flag overrides.
// Called by cmd/root.go in PersistentPreRunE after RegisterFlags.
func LoadFromDirWithFlags(dir string, cmd *cobra.Command) (*Config, error) {
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
	}

	// Environment variables
	v.SetEnvPrefix("LORE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// CLI flags (highest priority)
	bindFlags(v, cmd)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}

	return &cfg, nil
}
