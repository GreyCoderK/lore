// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"path/filepath"
	"time"

	"github.com/greycoderk/lore/internal/domain"
	"github.com/spf13/viper"
)

// Decision Engine defaults — duplicated here to break the config/ → decision/
// dependency cycle. decision.DefaultConfig() remains the canonical source for
// the Engine itself; these values are only used for Viper defaults.
const (
	defaultThresholdFull      = 60
	defaultThresholdReduced   = 35
	defaultThresholdSuggest   = 15
	defaultLearningMinCommits = 20
)

type AIConfig struct {
	Provider string        `yaml:"provider" mapstructure:"provider"`
	Model    string        `yaml:"model" mapstructure:"model"`
	APIKey   string        `yaml:"api_key" mapstructure:"api_key"`
	Endpoint string        `yaml:"endpoint" mapstructure:"endpoint"`
	Timeout  time.Duration `yaml:"timeout" mapstructure:"timeout"`
}

// AngelaConfig groups all configuration for the `lore angela` sub-commands
// (draft / polish / review). Legacy fields (Mode, MaxTokens, StyleGuide) are
// retained for backward compatibility with existing .lorerc files. Per-command
// sub-configs (Draft, Review, Polish) give each sub-command its own namespace.
//
// Invariant I1 (draft is offline forever): DraftConfig and its sub-configs
// MUST NOT contain any AI-related fields. This invariant is enforced by
// ValidateDraftOfflineInvariant() at config load time using reflection.
type AngelaConfig struct {
	// Mode is retained for backward compatibility with existing .lorerc
	// files but has no runtime effect. The mode is selected by the sub-
	// command: `lore angela draft`, `polish`, or `review`.
	//
	// Deprecated: unused. ValidateConfig emits a deprecation warning when
	// this field is set in a config file. Will be removed in v2.
	Mode       string                 `yaml:"mode" mapstructure:"mode"`
	MaxTokens  int                    `yaml:"max_tokens" mapstructure:"max_tokens"`
	StyleGuide map[string]interface{} `yaml:"style_guide" mapstructure:"style_guide"`

	// ModeDetection drives how AngelaConfig resolves the operating mode
	// (lore-native / hybrid / standalone). Valid values: "auto" (default,
	// probes the filesystem), "lore-native", "hybrid", "standalone".
	// The detection logic probes the filesystem at runtime.
	ModeDetection string `yaml:"mode_detection" mapstructure:"mode_detection"`

	// StateDir overrides the default state directory location. Empty means
	// "resolve from mode" — `.lore/angela/` for lore-native, `.angela-state/`
	// otherwise. Relative paths are interpreted relative to the working dir.
	StateDir string `yaml:"state_dir" mapstructure:"state_dir"`

	// I18n controls suggestion language selection.
	I18n I18nConfig `yaml:"i18n" mapstructure:"i18n"`

	// Personas holds the GLOBAL defaults for persona selection across
	// all three sub-commands. Per-command configs (Draft.Personas, etc.)
	// can override individual fields at the YAML level; the global values
	// serve as the fallback when per-command fields are unset.
	Personas PersonasConfig `yaml:"personas" mapstructure:"personas"`

	// Per-command sub-configs.
	Draft  DraftConfig  `yaml:"draft" mapstructure:"draft"`
	Review ReviewConfig `yaml:"review" mapstructure:"review"`
	Polish PolishConfig `yaml:"polish" mapstructure:"polish"`

	// Synthesizers controls the Example Synthesizer family.
	// Synthesizers enrich docs with literal recompositions of information
	// already present (Postman examples, SQL queries, env templates). The
	// framework enforces invariants I4 (zero-hallucination), I5 (security-
	// first projection), I6 (idempotency), I7 (no silent merge).
	Synthesizers SynthesizersConfig `yaml:"synthesizers" mapstructure:"synthesizers"`
}

// SynthesizersConfig activates and configures the Example Synthesizer family.
type SynthesizersConfig struct {
	// Enabled is the ordered list of synthesizer names to activate. Empty
	// means the framework runs zero synthesizers - safe default pre-8-18.
	// Post-8-18 default becomes ["api-postman"].
	Enabled []string `yaml:"enabled" mapstructure:"enabled"`

	// WellKnownServerFields is the list of field names treated as server-
	// injected by default when a doc's Security section is missing (I5-bis
	// degraded mode). Editable per-project; additions tighten the fail-safe
	// filter.
	//
	// Matching is EXACT-STRING only - a listed name like "tenantId" does
	// not filter a close-but-different field like "tenantIdFormatted". The
	// design choice favors predictability (users see exactly what is
	// filtered, no surprise) over recall. Projects that need substring
	// filtering can either add every variant explicitly or declare a
	// Security section in the doc and rely on I5 strict projection.
	WellKnownServerFields []string `yaml:"well_known_server_fields" mapstructure:"well_known_server_fields"`

	// PerSynthesizer carries synthesizer-specific options keyed by
	// synthesizer Name. Unknown keys are ignored by design - lets config
	// reference options for synthesizers that ship in later binaries.
	PerSynthesizer map[string]map[string]any `yaml:"per_synthesizer" mapstructure:"per_synthesizer"`
}

// ───────────────────────────────────────────────────────────────
// Shared building-block types (used by multiple sub-configs)
// ───────────────────────────────────────────────────────────────

// I18nConfig controls how suggestion messages are localized.
type I18nConfig struct {
	// SuggestionsLanguage selects the language for locally-generated
	// suggestions. "auto" follows the top-level `language` config.
	// Valid values: "auto", "fr", "en".
	SuggestionsLanguage string `yaml:"suggestions_language" mapstructure:"suggestions_language"`
}

// PersonasConfig controls persona selection for draft / review / polish.
// Used both globally (angela.personas) and per-command. Empty fields in
// a per-command config inherit from the global config via ResolvePersonasConfig.
type PersonasConfig struct {
	// Selection mode: "auto" (smart per doc type), "manual" (use ManualList),
	// "all" (all 6 personas), "none" (no persona checks).
	Selection string `yaml:"selection" mapstructure:"selection"`

	// Max caps the number of active personas. Max <= 0 means use default (3).
	// To select all matching personas, use Selection: "all".
	Max int `yaml:"max" mapstructure:"max"`

	// ManualList is consulted when Selection == "manual". Contains persona
	// names (e.g., "tech-writer", "architect"). Empty otherwise.
	ManualList []string `yaml:"manual_list" mapstructure:"manual_list"`

	// FreeFormMode controls persona activation for free-form doc types
	// (tutorial, guide, landing, etc.). Valid values:
	//   "none"    — no personas for free-form docs (current behavior before 8-11)
	//   "minimal" — only tech-writer (default)
	//   "full"    — smart per-type selection
	FreeFormMode string `yaml:"free_form_mode" mapstructure:"free_form_mode"`
}

// DraftDifferentialConfig controls hash/state-based incremental
// analysis for `lore angela draft`. Draft is strictly
// offline and per-machine so this struct intentionally OMITS the
// `state_shared` review-future field — users who set that key under
// `angela.draft.differential` will get a "strict unmarshal" warning
// instead of a silent no-op that implies a feature that doesn't exist.
//
// Draft and review use separate config types so `state_shared` (a
// review-only placeholder) does not pollute draft's schema.
type DraftDifferentialConfig struct {
	// Enabled turns differential mode on or off.
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// StateFile is the filename (not path) where state is persisted,
	// relative to AngelaConfig.StateDir. Example: "draft-state.json".
	StateFile string `yaml:"state_file" mapstructure:"state_file"`

	// DiffOnly hides PERSISTING findings in the report, showing only
	// NEW and RESOLVED. Ideal for CI pipelines.
	DiffOnly bool `yaml:"diff_only" mapstructure:"diff_only"`
}

// ReviewDifferentialConfig controls hash/state-based incremental
// analysis for `lore angela review`. Differs from the
// draft variant by the presence of `state_shared` (placeholder for
// v1.1 shared-team review state) and by supporting the REGRESSED
// lifecycle status in its diff surface.
type ReviewDifferentialConfig struct {
	// Enabled turns differential mode on or off.
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`

	// StateFile is the filename (not path) where state is persisted,
	// relative to AngelaConfig.StateDir. Example: "review-state.json".
	StateFile string `yaml:"state_file" mapstructure:"state_file"`

	// DiffOnly hides PERSISTING findings in the report, showing only
	// NEW, REGRESSED, and RESOLVED. Ideal for CI pipelines.
	DiffOnly bool `yaml:"diff_only" mapstructure:"diff_only"`

	// StateShared is a placeholder field. It is
	// reserved for a future v1.1 feature where teams could share
	// review state across machines (with merge-conflict handling). In
	// MVP v1 it is a no-op — left in the schema so users can set it
	// without errors and so the YAML key stays stable across versions.
	StateShared bool `yaml:"state_shared" mapstructure:"state_shared"`
}

// OutputFormatConfig controls the CLI output rendering for a sub-command.
type OutputFormatConfig struct {
	// Format: "human" (colored terminal output) | "json" (machine-parseable).
	// In standalone + non-TTY mode, mode detection auto-promotes
	// "human" → "json" when the user hasn't made an explicit choice.
	Format string `yaml:"format" mapstructure:"format"`

	// Color: "auto" (colorize only when TTY) | "always" | "never".
	Color string `yaml:"color" mapstructure:"color"`

	// Progress: "auto" | "always" | "never".
	Progress string `yaml:"progress" mapstructure:"progress"`

	// Verbose enables detailed output (breakdowns, rejected findings, etc.).
	Verbose bool `yaml:"verbose" mapstructure:"verbose"`
}

// ExitCodeConfig controls shell exit code resolution for a sub-command.
type ExitCodeConfig struct {
	// FailOn: minimum severity that triggers a non-zero exit code.
	// Valid values: "error" | "warning" | "info" | "never".
	FailOn string `yaml:"fail_on" mapstructure:"fail_on"`

	// Strict promotes all warnings to errors for exit-code purposes.
	Strict bool `yaml:"strict" mapstructure:"strict"`
}

// ───────────────────────────────────────────────────────────────
// DraftConfig — OFFLINE ONLY (Invariant I1)
//
// This struct MUST NOT contain any AI-related fields. Enforced at
// config load time by ValidateDraftOfflineInvariant() using reflection.
// ───────────────────────────────────────────────────────────────

// DraftConfig holds all configuration for the `lore angela draft` command.
// Draft is guaranteed offline: no field in this struct (or its sub-configs)
// triggers network I/O. This invariant is structural — do not add any field
// whose name contains ai/model/provider/escalate/token.
type DraftConfig struct {
	Checks           DraftChecksConfig      `yaml:"checks" mapstructure:"checks"`
	Scoring          ScoringConfig          `yaml:"scoring" mapstructure:"scoring"`
	SeverityOverride map[string]string      `yaml:"severity_override" mapstructure:"severity_override"`
	Differential     DraftDifferentialConfig `yaml:"differential" mapstructure:"differential"`
	Autofix          DraftAutofixConfig      `yaml:"autofix" mapstructure:"autofix"`
	Interactive      DraftInteractiveConfig `yaml:"interactive" mapstructure:"interactive"`
	Personas         PersonasConfig         `yaml:"personas" mapstructure:"personas"`
	Output           OutputFormatConfig     `yaml:"output" mapstructure:"output"`
	ExitCode         ExitCodeConfig         `yaml:"exit_code" mapstructure:"exit_code"`
	IgnoreFile       string                 `yaml:"ignore_file" mapstructure:"ignore_file"`
}

// DraftChecksConfig toggles individual analyzer categories in draft.
type DraftChecksConfig struct {
	Structure    bool `yaml:"structure" mapstructure:"structure"`
	Completeness bool `yaml:"completeness" mapstructure:"completeness"`
	Coherence    bool `yaml:"coherence" mapstructure:"coherence"`
	Personas     bool `yaml:"personas" mapstructure:"personas"`
	Scoring      bool `yaml:"scoring" mapstructure:"scoring"`
}

// ScoringConfig tunes the scoring profile and failure threshold.
type ScoringConfig struct {
	// Profile: "auto" (select by doc type), "strict", "free-form".
	Profile string `yaml:"profile" mapstructure:"profile"`

	// ShowBreakdown prints the per-category point breakdown in verbose mode.
	ShowBreakdown bool `yaml:"show_breakdown" mapstructure:"show_breakdown"`

	// FailBelow sets a minimum score threshold. Documents scoring below this
	// cause draft to exit non-zero. 0 means "never fail on score".
	FailBelow int `yaml:"fail_below" mapstructure:"fail_below"`
}

// DraftAutofixConfig controls the autofix engine.
// Autofix is 100% local — it uses existing helpers like DetectLanguage().
// No AI calls, no network. Invariant I1 holds.
type DraftAutofixConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Mode    string `yaml:"mode" mapstructure:"mode"` // "safe" | "aggressive"
	Backup  bool   `yaml:"backup" mapstructure:"backup"`
}

// DraftInteractiveConfig controls the draft fix-it TUI.
type DraftInteractiveConfig struct {
	// Enabled toggles the TUI mode on. Auto-downgraded to printf output
	// when stdout is not a TTY.
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

// ───────────────────────────────────────────────────────────────
// ReviewConfig — can use AI (single-shot in MVP)
// ───────────────────────────────────────────────────────────────

// ReviewConfig holds all configuration for the `lore angela review` command.
type ReviewConfig struct {
	Evidence     EvidenceConfig          `yaml:"evidence" mapstructure:"evidence"`
	Differential ReviewDifferentialConfig `yaml:"differential" mapstructure:"differential"`
	Sampling     SamplingConfig          `yaml:"sampling" mapstructure:"sampling"`
	Interactive  ReviewInteractiveConfig `yaml:"interactive" mapstructure:"interactive"`
	Personas     PersonasConfig          `yaml:"personas" mapstructure:"personas"`
	Output       OutputFormatConfig      `yaml:"output" mapstructure:"output"`
	ExitCode     ReviewExitCodeConfig    `yaml:"exit_code" mapstructure:"exit_code"`
}

// EvidenceConfig controls finding evidence validation.
type EvidenceConfig struct {
	// Required: when true, findings without verifiable quotes are rejected.
	Required bool `yaml:"required" mapstructure:"required"`

	// MinConfidence filters findings whose AI-reported confidence is below
	// this threshold (0.0 - 1.0).
	MinConfidence float64 `yaml:"min_confidence" mapstructure:"min_confidence"`

	// Validation strictness: "strict" | "lenient" | "off".
	Validation string `yaml:"validation" mapstructure:"validation"`
}

// SamplingConfig controls how many docs are passed to the AI per review call.
type SamplingConfig struct {
	// Mode: "smart" (25 recent + 25 oldest), "all", "recent".
	Mode string `yaml:"mode" mapstructure:"mode"`

	// Limit caps the number of docs sent to the AI in a single call.
	Limit int `yaml:"limit" mapstructure:"limit"`
}

// ReviewInteractiveConfig controls the review TUI.
type ReviewInteractiveConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Editor  string `yaml:"editor" mapstructure:"editor"` // "" means use $EDITOR
}

// ReviewExitCodeConfig controls exit codes specific to review findings.
// Review uses severity levels from the AI ("contradiction" > "gap" > ...)
// so its threshold semantics differ from draft's ExitCodeConfig.
type ReviewExitCodeConfig struct {
	// FailOnSeverity: minimum severity for non-zero exit.
	// Valid values: "contradiction" | "gap" | "obsolete" | "style" | "never".
	FailOnSeverity string `yaml:"fail_on_severity" mapstructure:"fail_on_severity"`
}

// ───────────────────────────────────────────────────────────────
// PolishConfig — uses AI, ships with safety nets
// ───────────────────────────────────────────────────────────────

// PolishConfig holds all configuration for the `lore angela polish` command.
type PolishConfig struct {
	DryRun             bool                     `yaml:"dry_run" mapstructure:"dry_run"`
	Backup             PolishBackupConfig       `yaml:"backup" mapstructure:"backup"`
	HallucinationCheck HallucinationCheckConfig `yaml:"hallucination_check" mapstructure:"hallucination_check"`
	Incremental        PolishIncrementalConfig  `yaml:"incremental" mapstructure:"incremental"`
	Interactive        PolishInteractiveConfig  `yaml:"interactive" mapstructure:"interactive"`
	Personas           PersonasConfig           `yaml:"personas" mapstructure:"personas"`
	Audience           PolishAudienceConfig     `yaml:"audience" mapstructure:"audience"`
}

// PolishBackupConfig controls automatic backup of files before polish writes.
type PolishBackupConfig struct {
	Enabled       bool   `yaml:"enabled" mapstructure:"enabled"`
	Path          string `yaml:"path" mapstructure:"path"`                     // relative to StateDir
	RetentionDays int    `yaml:"retention_days" mapstructure:"retention_days"` // 0 = forever
}

// HallucinationCheckConfig controls post-polish claim verification.
type HallucinationCheckConfig struct {
	Enabled    bool   `yaml:"enabled" mapstructure:"enabled"`
	Strictness string `yaml:"strictness" mapstructure:"strictness"` // "warn" | "reject" | "off"
}

// PolishIncrementalConfig enables section-level re-polish.
type PolishIncrementalConfig struct {
	Enabled        bool `yaml:"enabled" mapstructure:"enabled"`
	MinChangeLines int  `yaml:"min_change_lines" mapstructure:"min_change_lines"`
}

// PolishInteractiveConfig controls the polish diff-per-section TUI.
type PolishInteractiveConfig struct {
	Enabled     bool   `yaml:"enabled" mapstructure:"enabled"`
	Granularity string `yaml:"granularity" mapstructure:"granularity"` // "section" | "paragraph"
}

// PolishAudienceConfig controls audience-targeted polish rewrites.
type PolishAudienceConfig struct {
	Default string   `yaml:"default" mapstructure:"default"`
	Allowed []string `yaml:"allowed" mapstructure:"allowed"` // empty = any audience allowed
}

type TemplatesConfig struct {
	Dir string `yaml:"dir" mapstructure:"dir"`
}

type HooksConfig struct {
	PostCommit      bool `yaml:"post_commit" mapstructure:"post_commit"`
	StarPrompt      bool `yaml:"star_prompt" mapstructure:"star_prompt"`
	StarPromptAfter int  `yaml:"star_prompt_after" mapstructure:"star_prompt_after"`
	AmendPrompt     bool `yaml:"amend_prompt" mapstructure:"amend_prompt"`
}

type OutputConfig struct {
	Dir    string `yaml:"dir" mapstructure:"dir"`
	Format string `yaml:"format" mapstructure:"format"`
}

type NotificationConfig struct {
	Mode         string   `yaml:"mode" mapstructure:"mode"`
	DisabledEnvs []string `yaml:"disabled_envs" mapstructure:"disabled_envs"`
	Amend        bool     `yaml:"amend" mapstructure:"amend"`
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("language", "en")

	v.SetDefault("ai.provider", "")
	v.SetDefault("ai.model", "")
	v.SetDefault("ai.api_key", "")
	v.SetDefault("ai.endpoint", "http://localhost:11434")
	v.SetDefault("ai.timeout", "30s")

	setAngelaDefaults(v)

	v.SetDefault("templates.dir", filepath.Join(domain.LoreDir, domain.TemplatesDir))

	v.SetDefault("hooks.post_commit", true)
	v.SetDefault("hooks.star_prompt", true)
	v.SetDefault("hooks.star_prompt_after", 5)
	v.SetDefault("hooks.amend_prompt", true)

	v.SetDefault("output.format", "markdown")
	v.SetDefault("output.dir", filepath.Join(domain.LoreDir, domain.DocsDir))

	v.SetDefault("notification.mode", "auto")
	v.SetDefault("notification.disabled_envs", []string{})
	v.SetDefault("notification.amend", true)

	v.SetDefault("decision.threshold_full", defaultThresholdFull)
	v.SetDefault("decision.threshold_reduced", defaultThresholdReduced)
	v.SetDefault("decision.threshold_suggest", defaultThresholdSuggest)
	v.SetDefault("decision.always_ask", []string{"feat", "breaking"})
	v.SetDefault("decision.always_skip", []string{"docs", "style", "ci", "build"})
	v.SetDefault("decision.critical_scopes", []string{})
	v.SetDefault("decision.learning", true)
	v.SetDefault("decision.learning_min_commits", defaultLearningMinCommits)
}

// setAngelaDefaults registers all Viper defaults for AngelaConfig and its
// per-command sub-configs. Each default is chosen so that running
// `lore angela draft ./docs` in a brand-new directory with no .lorerc
// produces a meaningful result (invariant I3: zero-config works).
//
// This helper keeps setDefaults() readable when the angela schema is large.
func setAngelaDefaults(v *viper.Viper) {
	// ─── Legacy fields (backward compat) ───
	// angela.mode is deprecated (no runtime effect). Viper still needs the
	// key registered so that existing .lorerc files and LORE_ANGELA_MODE
	// env vars can parse without YAML decode errors. Empty default lets
	// tooling distinguish "user set it" from "inherited default".
	v.SetDefault("angela.mode", "")
	v.SetDefault("angela.max_tokens", 2000)

	// ─── Global angela settings ───
	v.SetDefault("angela.mode_detection", "auto")
	v.SetDefault("angela.state_dir", "") // resolved at runtime by mode detection

	v.SetDefault("angela.i18n.suggestions_language", "auto")

	// Global persona defaults (per-command configs inherit these).
	v.SetDefault("angela.personas.selection", "auto")
	v.SetDefault("angela.personas.max", 3)
	v.SetDefault("angela.personas.manual_list", []string{})
	v.SetDefault("angela.personas.free_form_mode", "minimal")

	// ─── Draft (offline, CI-friendly) ───
	v.SetDefault("angela.draft.checks.structure", true)
	v.SetDefault("angela.draft.checks.completeness", true)
	v.SetDefault("angela.draft.checks.coherence", true)
	v.SetDefault("angela.draft.checks.personas", true)
	v.SetDefault("angela.draft.checks.scoring", true)

	v.SetDefault("angela.draft.scoring.profile", "auto")
	v.SetDefault("angela.draft.scoring.show_breakdown", false)
	v.SetDefault("angela.draft.scoring.fail_below", 0)

	v.SetDefault("angela.draft.severity_override", map[string]string{})

	v.SetDefault("angela.draft.differential.enabled", true)
	v.SetDefault("angela.draft.differential.state_file", "draft-state.json")
	v.SetDefault("angela.draft.differential.diff_only", false)

	v.SetDefault("angela.draft.autofix.enabled", false)
	v.SetDefault("angela.draft.autofix.mode", "safe")
	v.SetDefault("angela.draft.autofix.backup", true)

	v.SetDefault("angela.draft.interactive.enabled", false)

	// Per-command persona override defaults were removed as dead weight.
	// The global `angela.personas` defaults remain.

	v.SetDefault("angela.draft.output.format", "human")
	v.SetDefault("angela.draft.output.color", "auto")
	v.SetDefault("angela.draft.output.progress", "auto")
	v.SetDefault("angela.draft.output.verbose", false)

	v.SetDefault("angela.draft.exit_code.fail_on", "error")
	v.SetDefault("angela.draft.exit_code.strict", false)

	v.SetDefault("angela.draft.ignore_file", ".angela-ignore")

	// ─── Review (single-shot AI, evidence-grounded by default) ───
	v.SetDefault("angela.review.evidence.required", true)
	v.SetDefault("angela.review.evidence.min_confidence", 0.4)
	v.SetDefault("angela.review.evidence.validation", "strict")

	v.SetDefault("angela.review.differential.enabled", true)
	v.SetDefault("angela.review.differential.state_file", "review-state.json")
	v.SetDefault("angela.review.differential.diff_only", false)

	v.SetDefault("angela.review.sampling.mode", "smart")
	v.SetDefault("angela.review.sampling.limit", 50)

	v.SetDefault("angela.review.interactive.enabled", false)
	v.SetDefault("angela.review.interactive.editor", "")

	// Per-command persona override defaults removed — see draft block above.

	v.SetDefault("angela.review.output.format", "human")
	v.SetDefault("angela.review.output.color", "auto")
	v.SetDefault("angela.review.output.progress", "auto")
	v.SetDefault("angela.review.output.verbose", false)

	v.SetDefault("angela.review.exit_code.fail_on_severity", "contradiction")

	// ─── Polish (AI, ships with safety nets) ───
	v.SetDefault("angela.polish.dry_run", false)

	v.SetDefault("angela.polish.backup.enabled", true)
	v.SetDefault("angela.polish.backup.path", "polish-backups")
	v.SetDefault("angela.polish.backup.retention_days", 30)

	v.SetDefault("angela.polish.hallucination_check.enabled", true)
	v.SetDefault("angela.polish.hallucination_check.strictness", "warn")

	v.SetDefault("angela.polish.incremental.enabled", false)
	v.SetDefault("angela.polish.incremental.min_change_lines", 3)

	v.SetDefault("angela.polish.interactive.enabled", false)
	v.SetDefault("angela.polish.interactive.granularity", "section")

	// Per-command persona override defaults removed — see draft block above.

	v.SetDefault("angela.polish.audience.default", "")
	v.SetDefault("angela.polish.audience.allowed", []string{})

	// ─── Synthesizers  ───
	// api-postman enabled by default (went green against the
	// dogfood fixture with zero I4/I5 violations). Users opt out via
	// --no-synthesizers or by clearing this list in .lorerc.
	v.SetDefault("angela.synthesizers.enabled", []string{"api-postman"})

	// Default well-known list per 2026-04-15 session decision. Extended per
	// project via .lorerc; tightened globally in a future patch if new
	// canonical names emerge.
	v.SetDefault("angela.synthesizers.well_known_server_fields", []string{
		"tenantId",
		"authenticatedUsername",
		"principalId",
	})

	v.SetDefault("angela.synthesizers.per_synthesizer", map[string]map[string]any{})
}
