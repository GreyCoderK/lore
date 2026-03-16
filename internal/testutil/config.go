package testutil

import (
	"bytes"

	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
)

func TestConfig() *config.Config {
	return &config.Config{
		AI: config.AIConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
		},
		Angela: config.AngelaConfig{
			Mode:      "draft",
			MaxTokens: 2000,
		},
		Templates: config.TemplatesConfig{
			Dir: ".lore/templates",
		},
		Hooks: config.HooksConfig{
			PostCommit: true,
		},
		Output: config.OutputConfig{
			Dir:    ".lore/docs",
			Format: "markdown",
		},
	}
}

func TestStreams() (domain.IOStreams, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	streams := domain.IOStreams{
		Out: &out,
		Err: &errBuf,
		In:  &bytes.Buffer{},
	}
	return streams, &out, &errBuf
}
