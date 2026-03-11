package testutil

import (
	"bytes"

	"github.com/museigen/lore/internal/config"
	"github.com/museigen/lore/internal/domain"
)

func TestConfig() *config.Config {
	return &config.Config{
		AI: config.AIConfig{
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
		},
		Angela: config.AngelaConfig{
			Enabled: true,
			Mode:    "draft",
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
