// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package synthesizer is Angela's doc-enrichment framework.
//
// A Synthesizer proposes a new block of content (an HTTP+JSON example, a SQL
// query, an env-var template...) assembled from information already present in
// a documentation file. It never invents content: every field in every output
// traces back to a literal character span in a source doc via an Evidence.
//
// Five invariants govern the framework:
//
//   - I4  zero-hallucination: every output field has >=1 Evidence with
//     Rule == "literal" pointing to an exact span in the source.
//   - I5  security-first: fields declared as server-injected in the doc's
//     Security section are excluded by construction.
//   - I5-bis fail-safe on missing Security: degraded mode filters a
//     configurable well-known list and emits a mandatory review finding.
//   - I6  idempotency: two runs on an unchanged doc produce byte-identical
//     output; the frontmatter signature is a function of the source spans.
//   - I7  no silent merge: changes to previously accepted output flow through
//     the interactive diff review. The framework never overwrites user edits.
//
// Concrete implementations live under impls/<name>/. Ships the
// first member (api-postman).
package synthesizer

import (
	"strings"

	"github.com/greycoderk/lore/internal/domain"
)

// Synthesizer is a doc-enrichment unit. Implementations are registered via
// Registry.Register and activated per-run through AngelaConfig.
type Synthesizer interface {
	// Name is the stable identifier used in config, frontmatter signatures and
	// CLI flags (e.g., "api-postman", "sql-query"). Must be lowercase kebab-case.
	Name() string

	// Applies is a fast, side-effect-free gate. It returns true when the doc
	// looks like a candidate for this synthesizer (frontmatter type, presence
	// of characteristic sections, regex hits). It must NOT parse the body in
	// depth - Detect does that.
	Applies(doc *Doc) bool

	// Detect enumerates synthesis opportunities inside the doc. Each Candidate
	// carries a primary anchor (evidence span) and any synthesizer-specific
	// extras needed to later synthesize without re-parsing. Detect may return
	// an empty slice and nil error when the doc applies but has no candidate
	// (e.g., endpoints section present but empty).
	Detect(doc *Doc) ([]Candidate, error)

	// Synthesize turns a Candidate into a ready-to-insert Block plus the list
	// of Evidences supporting every field in the block's output and any
	// non-fatal Warnings (degraded mode, fuzzy heading match, etc.).
	//
	// The contract for I4 lives here: every output key produced by the
	// synthesizer must appear at least once in the returned []Evidence with
	// Rule == "literal" and Snippet equal to the source file content at the
	// declared span.
	Synthesize(c Candidate, cfg Config) (Block, []Evidence, []Warning, error)
}

// Doc is the framework's pre-parsed representation of a markdown file. It is
// built by the registry hook (review/polish/draft pipelines) from the raw
// bytes and handed to every synthesizer that Applies to it.
type Doc struct {
	// Path is the source file path, relative to the corpus root when known,
	// absolute otherwise. Used for evidence File references.
	Path string

	// Meta is the parsed YAML frontmatter. Synthesized (from the frontmatter
	// "synthesized" key) is surfaced separately via Signatures because it is
	// not part of domain.DocMeta yet.
	Meta domain.DocMeta

	// Body is the markdown content AFTER the frontmatter terminator.
	Body string

	// Lines is Body split on "\n", 1-indexed at Lines[1]. Lines[0] is empty
	// to make line numbers cited in Evidence directly indexable.
	Lines []string

	// Sections is the flat list of top-level and sub-level headings found in
	// Body, in source order. Synthesizers use FuzzyFindSection (task 5) on
	// this slice rather than re-parsing.
	Sections []Section

	// Signatures holds previously-recorded synthesizer signatures, keyed by
	// synthesizer Name. Empty when the doc has never been synthesized. The
	// registry compares current evidence hashes against these to decide
	// skip vs regenerate (I6).
	Signatures map[string]Signature
}

// Section is one markdown heading with its content span in the Body.
type Section struct {
	// Heading is the heading text WITH leading hashes, e.g. "### Endpoints".
	Heading string

	// Level is the heading depth (1 for #, 2 for ##, ...).
	Level int

	// Title is Heading without the leading "#+ " prefix.
	Title string

	// StartLine is the 1-based line number of the heading itself.
	StartLine int

	// EndLine is the 1-based line number of the last line BEFORE the next
	// heading at the same or shallower level (exclusive upper bound
	// rendered as inclusive line number - the line is part of this section).
	EndLine int

	// Content is the raw content between the heading and EndLine (excluding
	// the heading line). Trailing newline trimmed.
	Content string
}

// Candidate is a synthesis opportunity detected in a doc. The shape is
// deliberately opaque: Extra lets synthesizers carry implementation-specific
// context from Detect to Synthesize without a second parse pass.
type Candidate struct {
	// Key is a stable, unique-per-doc identifier for this candidate (e.g.
	// "POST /api/account-statement/search"). Used for diff-per-section
	// routing and idempotency signatures.
	Key string

	// Anchor is the primary evidence span - typically the endpoint declaration
	// line for api-postman, the schema table row for sql-query, etc.
	Anchor Evidence

	// Extra carries synthesizer-specific data from Detect to Synthesize. The
	// framework treats it as opaque; concrete synthesizers own its schema.
	Extra map[string]any
}

// MetaFieldPrefix is reserved for Evidence.Field values that do NOT map
// to a JSON output key (anchors, requiredness proofs, etc.). Using a
// single reserved prefix makes it trivial to filter pseudo-evidences out
// of a downstream view that only cares about the emitted output keys
// (code review finding #9 - previously these used ad-hoc conventions
// like "__endpoint__" and "<name>#required").
const (
	MetaFieldPrefix       = "_meta."
	MetaFieldEndpoint     = MetaFieldPrefix + "endpoint"
	MetaFieldRequiredSuffix = ".required"
)

// RequiredFieldKey returns the reserved evidence Field value that
// documents the requiredness claim for an output field.
func RequiredFieldKey(fieldName string) string {
	return MetaFieldPrefix + fieldName + MetaFieldRequiredSuffix
}

// IsMetaField reports whether an Evidence.Field value targets framework
// metadata (anchors, requiredness) rather than a real JSON output key.
func IsMetaField(field string) bool {
	return strings.HasPrefix(field, MetaFieldPrefix)
}

// Evidence anchors one piece of synthesized content to a literal character
// span in a source doc.
//
// Invariant I4 (zero-hallucination) requires: for every key in a Block's
// generated output, at least one Evidence exists in the Synthesize return
// with Field equal to that key, Rule == "literal", and Snippet byte-equal
// to doc.Lines[Line][ColStart:ColEnd].
type Evidence struct {
	// Field is the output key this evidence supports (e.g. "month",
	// "creditAmountMin"). Empty for anchor-only evidences attached to
	// Candidates and Warnings.
	Field string

	// File is the source file path, matching Doc.Path when the evidence
	// comes from the doc being synthesized. Different for cross-file
	// evidence (not used in MVP v1).
	File string

	// Line is the 1-based line number. ColStart/ColEnd are byte offsets
	// into that line.
	Line     int
	ColStart int
	ColEnd   int

	// Snippet is the literal source text at File:Line[ColStart:ColEnd]. The
	// framework validates Snippet == doc.Lines[Line][ColStart:ColEnd] before
	// accepting the evidence.
	Snippet string

	// Rule is the reason the evidence is valid. MVP v1 accepts only
	// "literal". Future relaxations would introduce
	// alternative rules with their own invariants.
	Rule string
}

// Block is the output produced by a Synthesizer for one Candidate. The
// framework inserts it into the doc AFTER the heading matched by
// InsertAfterHeading when polish accepts the proposal.
type Block struct {
	// Title is the subsection heading the framework will generate for the
	// block (e.g. "Example d'appel (Postman) - minimal"). The framework
	// prepends the heading level matching the parent section.
	Title string

	// Language is the fenced-code info string used in the generated markdown
	// fence (e.g. "http+json", "sql", "bash", "mermaid"). Empty for prose
	// blocks (not used in MVP v1).
	Language string

	// Content is the block's body WITHOUT the fence. The framework wraps it
	// with the fence markers when rendering. Must be canonical and
	// deterministic: byte-identical for equal inputs across runs (I6).
	Content string

	// Notes is an ordered list of caveat lines appended under the fenced
	// block as a bullet list ("- note"). Typical content: server-injected
	// fields, degraded mode disclosure, variable convention reminders.
	Notes []string

	// InsertAfterHeading identifies where in the doc the block belongs.
	// Value matches a Section.Heading verbatim (same text, same hashes).
	// An empty value means "end of doc" - never used in MVP v1.
	InsertAfterHeading string
}

// Warning is a non-fatal signal emitted during Detect or Synthesize. It flows
// into the review pipeline as a ReviewFinding and into the
// frontmatter signature (see Signature.Warnings).
type Warning struct {
	// Code is a short, stable identifier ("missing-security-section",
	// "fuzzy-heading-match", "endpoints-without-field-source"). Used as the
	// finding category and as a lookup key in severity_override.
	Code string

	// Message is a human-readable explanation localized by the framework at
	// render time. Concrete synthesizers emit the CODE only; the
	// framework's i18n layer translates.
	Message string

	// Line is the 1-based line number the warning attaches to, or 0 when it
	// is doc-scoped (e.g. missing-security-section is doc-scoped).
	Line int
}

// Signature is what a synthesizer persists into the doc's frontmatter under
// the "synthesized.<name>" key. See Task 4.
type Signature struct {
	// Hash is a hex-encoded sha256 over the canonical list of evidence
	// spans (File, Line, ColStart, ColEnd, Snippet) the synthesizer used.
	// Evidence-based, not output-based: the hash remains stable across
	// cosmetic output tweaks and shifts only when source spans change.
	Hash string `yaml:"hash"`

	// At is the RFC3339 timestamp of the last successful Synthesize call
	// that produced output matching this signature.
	At string `yaml:"at"`

	// Version is the synthesizer's own output-format version. Bumped when a
	// backward-incompatible output change ships; forces regeneration even
	// when the source evidences are unchanged.
	Version string `yaml:"version"`

	// Sections is the ordered list of source section titles consulted for
	// this synthesis. Purely informational; not part of Hash.
	Sections []string `yaml:"sections,omitempty"`

	// EvidenceCount is the number of Evidence records backing the output.
	// Informational, for audit visibility in frontmatter diffs.
	EvidenceCount int `yaml:"evidence_count"`

	// Warnings is the list of Warning.Code values emitted for this run. Used
	// by the review hook to avoid re-emitting findings already present in
	// the frontmatter and by audits to spot docs shipped in degraded mode.
	Warnings []string `yaml:"warnings,omitempty"`
}

// Config carries per-run settings passed to Synthesize. It is derived from
// AngelaConfig.Synthesizers at command entry and holds the minimal subset
// concrete synthesizers need (avoids leaking the whole AngelaConfig surface
// to each implementation).
type Config struct {
	// WellKnownServerFields is the list used by synthesizers to filter
	// likely server-injected fields when the source doc has no Security
	// section (I5-bis). Editable in AngelaConfig.Synthesizers.
	WellKnownServerFields []string

	// PerSynthesizer is a bag of synthesizer-specific options keyed by
	// synthesizer Name. Implementations look up their own key; unknown
	// keys are ignored by design (allows shipping options before the
	// synthesizer that reads them).
	PerSynthesizer map[string]map[string]any
}
