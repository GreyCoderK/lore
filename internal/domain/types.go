package domain

import (
	"io"
	"regexp"
	"strings"
	"time"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-friendly slug (display purposes).
// For filename generation, use storage.slugify which also handles accents and length limits.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
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

// InstallResult describes the outcome of a hook install operation.
type InstallResult struct {
	Installed     bool   // true if the hook was installed
	HooksPathWarn string // non-empty if core.hooksPath is configured
}

// Document type constants.
const (
	DocTypeDecision = "decision"
	DocTypeFeature  = "feature"
	DocTypeBugfix   = "bugfix"
	DocTypeRefactor = "refactor"
	DocTypeRelease  = "release"
	DocTypeNote     = "note"
)

// validDocTypes is the single source of truth for accepted document types.
// Access via ValidDocType() — do not mutate directly.
var validDocTypes = map[string]bool{
	DocTypeDecision: true,
	DocTypeFeature:  true,
	DocTypeBugfix:   true,
	DocTypeRefactor: true,
	DocTypeRelease:  true,
	DocTypeNote:     true,
}

// ValidDocType reports whether t is a recognized document type.
// This is the single source of truth for accepted types.
func ValidDocType(t string) bool {
	return validDocTypes[t]
}

type DocMeta struct {
	Type        string   `yaml:"type"`
	Date        string   `yaml:"date"`
	Commit      string   `yaml:"commit,omitempty"`
	Status      string   `yaml:"status"`
	Tags        []string `yaml:"tags,omitempty"`
	Related     []string `yaml:"related,omitempty"`
	GeneratedBy string   `yaml:"generated_by,omitempty"`
	AngelaMode  string   `yaml:"angela_mode,omitempty"`

	Filename string `yaml:"-"` // populated at runtime by ListDocs, not serialized
}

type DocFilter struct {
	Type   string
	After  string // YYYY-MM-DD, inclusive
	Before string // YYYY-MM-DD, inclusive
	Tags   []string
	Status string
	Text   string // case-insensitive search in body and filename
}

type Option func(*CallOptions)

type CallOptions struct {
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}
