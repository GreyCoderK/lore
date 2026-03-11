package config

type AIConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	APIKey   string `yaml:"api_key"`
}

type AngelaConfig struct {
	Enabled bool   `yaml:"enabled"`
	Mode    string `yaml:"mode"`
}

type TemplatesConfig struct {
	Dir string `yaml:"dir"`
}

type HooksConfig struct {
	PostCommit bool `yaml:"post_commit"`
}

type OutputConfig struct {
	Dir    string `yaml:"dir"`
	Format string `yaml:"format"`
}

func defaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
		},
		Angela: AngelaConfig{
			Enabled: true,
			Mode:    "draft",
		},
		Templates: TemplatesConfig{
			Dir: ".lore/templates",
		},
		Hooks: HooksConfig{
			PostCommit: true,
		},
		Output: OutputConfig{
			Dir:    ".lore/docs",
			Format: "markdown",
		},
	}
}
