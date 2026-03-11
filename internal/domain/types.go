package domain

import (
	"io"
	"time"
)

type IOStreams struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
}

type CommitInfo struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
	Type    string
	Scope   string
	Subject string
}

type DocMeta struct {
	Type        string    `yaml:"type"`
	Date        time.Time `yaml:"date"`
	Commit      string    `yaml:"commit,omitempty"`
	Status      string    `yaml:"status"`
	Tags        []string  `yaml:"tags,omitempty"`
	Related     []string  `yaml:"related,omitempty"`
	GeneratedBy string    `yaml:"generated_by,omitempty"`
	AngelaMode  string    `yaml:"angela_mode,omitempty"`
}

type DocFilter struct {
	Type    string
	Keyword string
	After   time.Time
	Before  time.Time
}

type Option func(*CallOptions)

type CallOptions struct {
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}