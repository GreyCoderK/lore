// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import (
	"io"
	"regexp"
	"sort"
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
	Hash        string
	Author      string
	Date        time.Time
	Message     string
	Type        string
	Scope       string
	Subject     string
	Branch      string // current branch at commit time; "" if detached HEAD
	IsMerge     bool   // true when the commit has more than one parent
	ParentCount int    // number of parent commits (0 for root, 1 for normal, 2+ for merge)
}

// InstallResult describes the outcome of a hook install operation.
type InstallResult struct {
	Installed     bool   // true if the hook was installed
	HooksPathWarn string // non-empty if core.hooksPath is configured
}

// DocStatus represents the lifecycle state of a document.
type DocStatus = string

const (
	StatusDraft     DocStatus = "draft"
	StatusPublished DocStatus = "published"
	StatusArchived  DocStatus = "archived"
)

// DocType represents the category of a document.
type DocType = string

// Document type constants.
const (
	DocTypeDecision DocType = "decision"
	DocTypeFeature  DocType = "feature"
	DocTypeBugfix   DocType = "bugfix"
	DocTypeRefactor DocType = "refactor"
	DocTypeRelease  DocType = "release"
	DocTypeNote     DocType = "note"
	DocTypeSummary  DocType = "summary"
)

// Decision represents the outcome of the documentation decision engine.
type Decision = string

const (
	DecisionDocumented   Decision = "documented"
	DecisionSkipped      Decision = "skipped"
	DecisionAutoSkipped  Decision = "auto-skipped"
	DecisionMergeSkipped Decision = "merge-skipped"
	DecisionPending      Decision = "pending"
	DecisionUnknown      Decision = "unknown"
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
	DocTypeSummary:  true,
}

// ValidDocType reports whether t is a recognized document type.
// This is the single source of truth for accepted types.
func ValidDocType(t string) bool {
	return validDocTypes[t]
}

// DocTypeNames returns sorted list of valid document type names.
func DocTypeNames() []string {
	names := make([]string, 0, len(validDocTypes))
	for k := range validDocTypes {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

type DocMeta struct {
	Type        string   `yaml:"type"`
	Date        string   `yaml:"date"`
	Commit      string   `yaml:"commit,omitempty"`
	Branch      string   `yaml:"branch,omitempty"`
	Scope       string   `yaml:"scope,omitempty"`
	Status      string   `yaml:"status"`
	Tags        []string `yaml:"tags,omitempty"`
	Related     []string `yaml:"related,omitempty"`
	GeneratedBy string   `yaml:"generated_by,omitempty"`
	AngelaMode  string   `yaml:"angela_mode,omitempty"`

	// Synthesized records the signatures of every Example Synthesizer
	// that has produced output for this doc. The map key is
	// the synthesizer Name (e.g. "api-postman"), and the value carries the
	// fields defined by synthesizer.Signature (hash, at, version, sections,
	// evidence_count, warnings).
	//
	// Stored as map[string]map[string]any to keep the domain package free
	// of dependencies on internal/angela. Converted to/from typed
	// synthesizer.Signature inside the synthesizer package.
	Synthesized map[string]map[string]any `yaml:"synthesized,omitempty"`

	Filename string `yaml:"-"` // populated at runtime by ListDocs, not serialized
}

type DocFilter struct {
	Type   string
	After  string // YYYY-MM-DD, inclusive
	Before string // YYYY-MM-DD, inclusive
	Tags   []string
	Status string
	Text   string // case-insensitive search in body and filename
	Branch string // filter by branch name
	Scope  string // filter by scope
}

type Option func(*CallOptions)

type CallOptions struct {
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
	System      string
}

// CommitRecord is the store-layer view of a commit (what we persist).
// Not a replacement for CommitInfo which is the git-layer view.
type CommitRecord struct {
	Hash               string
	Date               time.Time
	Branch             string
	Scope              string
	ConvType           string
	Subject            string
	Message            string
	FilesChanged       int
	LinesAdded         int
	LinesDeleted       int
	DocID              string  // nullable — filename of generated doc
	Decision           string  // documented|skipped|pending|auto-skipped|merge-skipped|unknown
	DecisionScore      int     // nullable — score 0-100
	DecisionConfidence float64 // nullable
	SkipReason         string  // nullable
	QuestionMode       string  // full|reduced|confirm|none
}

// DocIndexEntry is the store-layer view of a document's metadata.
type DocIndexEntry struct {
	Filename         string
	Type             string
	Date             string
	CommitHash       string
	Branch           string
	Scope            string
	Status           string
	Tags             []string // stored as comma-separated in DB
	Related          []string // stored as comma-separated in DB
	GeneratedBy      string
	AngelaMode       string
	ConsolidatedInto string
	ContentHash      string // SHA-256 of body
	SummaryWhy       string
	SummaryWhat      string
	TitleExtracted   string
	WordCount        int
	UpdatedAt        time.Time
}

type CodeSignature struct {
	CommitHash string
	FilePath   string
	EntityName string
	EntityType string // func|method|type|struct|interface|class|trait|enum|const_block
	SigHash    string // SHA-256 of normalized body
	Lang       string
	LineStart  int
	ChangeType string // added|deleted|modified|moved|context
}

type AIUsageRecord struct {
	Timestamp     time.Time
	Mode          string // polish|review|render|ask|consult|merge
	Provider      string
	Model         string
	TokensIn      int
	TokensOut     int
	CachedIn      int
	CostUSD       float64
	LatencyMS     int
	CommitHash    string
	DocID         string
	PromptVersion int
}

type AIStatsAggregate struct {
	TotalCalls     int
	TotalTokensIn  int
	TotalTokensOut int
	TotalCachedIn  int
	TotalCostUSD   float64
	AvgLatencyMS   int
	CacheHitRate   float64
	ByMode         map[string]AIStatsAggregate
}

type DailyAIStats struct {
	Date      string
	Calls     int
	TokensIn  int
	TokensOut int
	CostUSD   float64
}

// ScopeStatsResult holds aggregated statistics for a scope,
// computed via SQL instead of loading all records into memory.
type ScopeStatsResult struct {
	TotalCommits    int
	DocumentedCount int
	SkippedCount    int
	LastDocDate     int64
	LastCommitDate  int64
}

type CommitPattern struct {
	ConvType         string
	Scope            string
	TotalCount       int
	DocumentedCount  int
	SkippedCount     int
	AutoSkippedCount int
	AvgDiffLines     int
	AvgScore         int
	DocRate          float64 // DocumentedCount / TotalCount
	SkipRate         float64 // SkippedCount / TotalCount
}

type ReviewCacheEntry struct {
	ReviewDate  time.Time
	CorpusHash  string
	CorpusCount int
	FindingsJSON string
	TokensIn    int
	TokensOut   int
	Provider    string
	Model       string
}

func WithModel(m string) Option        { return func(o *CallOptions) { o.Model = m } }
func WithMaxTokens(n int) Option       { return func(o *CallOptions) { o.MaxTokens = n } }
func WithTemperature(t float64) Option { return func(o *CallOptions) { o.Temperature = t } }
func WithTimeout(d time.Duration) Option { return func(o *CallOptions) { o.Timeout = d } }
func WithSystem(s string) Option       { return func(o *CallOptions) { o.System = s } }
