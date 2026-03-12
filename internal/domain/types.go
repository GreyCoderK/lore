package domain

import (
	"io"
	"regexp"
	"strings"
	"time"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-friendly slug.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// DateString is a time.Time that serializes to YYYY-MM-DD in YAML.
type DateString struct {
	time.Time
}

// NewDateString creates a DateString from a time.Time.
func NewDateString(t time.Time) DateString {
	return DateString{Time: t}
}

func (d DateString) MarshalYAML() (interface{}, error) {
	return d.Format("2006-01-02"), nil
}

func (d *DateString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		// Fall back to RFC3339
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return err
		}
	}
	d.Time = t
	return nil
}

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
	Type        string     `yaml:"type"`
	Date        DateString `yaml:"date"`
	Commit      string     `yaml:"commit,omitempty"`
	Status      string     `yaml:"status"`
	Tags        []string   `yaml:"tags,omitempty"`
	Related     []string   `yaml:"related,omitempty"`
	GeneratedBy string     `yaml:"generated_by,omitempty"`
	AngelaMode  string     `yaml:"angela_mode,omitempty"`
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
