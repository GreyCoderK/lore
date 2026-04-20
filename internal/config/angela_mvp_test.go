// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// ═══════════════════════════════════════════════════════════════
// Story 8.2 tests: AngelaConfig pivot
// ═══════════════════════════════════════════════════════════════

// TestDraftConfig_NoAIFields is the runtime guard for Invariant I1 (draft
// is offline forever). It walks DraftConfig via reflection and fails if any
// field name contains a forbidden segment. Paired with ValidateDraftOfflineInvariant
// which runs at config load time.
func TestDraftConfig_NoAIFields(t *testing.T) {
	if err := ValidateDraftOfflineInvariant(); err != nil {
		t.Errorf("Invariant I1 violated: %v", err)
	}
}

// TestSplitCamelCase verifies the CamelCase splitter used by the invariant
// check. This is critical because a buggy splitter would either let real
// violations through OR produce false positives on innocent field names.
func TestSplitCamelCase(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "MaxOutputTokens", []string{"Max", "Output", "Tokens"}},
		{"single", "Provider", []string{"Provider"}},
		{"acronym_prefix", "HTTPServer", []string{"HTTP", "Server"}},
		{"acronym_suffix", "ServerHTTP", []string{"Server", "HTTP"}},
		{"all_lower", "model", []string{"model"}},
		{"failon_not_ai", "FailOn", []string{"Fail", "On"}},
		{"apikey_segment", "APIKey", []string{"API", "Key"}},
		{"empty", "", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCamelCase(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitCamelCase(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestValidateDraftOfflineInvariant_DetectsViolation constructs an adversarial
// DraftConfig-like struct that contains a forbidden field and asserts the
// walker catches it. Ensures the check actually enforces the invariant.
func TestValidateDraftOfflineInvariant_DetectsViolation(t *testing.T) {
	type BadConfig struct {
		Provider string // forbidden segment "Provider"
	}
	err := walkStructForbiddenSegments(reflect.TypeOf(BadConfig{}), "BadConfig", forbiddenDraftFieldSegments)
	if err == nil {
		t.Error("expected violation for field containing 'Provider', got nil")
	}
}

// TestValidateDraftOfflineInvariant_DetectsNestedViolation ensures the
// recursive walker catches violations in nested struct fields, not just
// top-level ones.
func TestValidateDraftOfflineInvariant_DetectsNestedViolation(t *testing.T) {
	type Inner struct {
		Endpoint string // forbidden segment "Endpoint"
	}
	type Outer struct {
		Inner Inner
	}
	err := walkStructForbiddenSegments(reflect.TypeOf(Outer{}), "Outer", forbiddenDraftFieldSegments)
	if err == nil {
		t.Error("expected violation in nested Inner.Endpoint, got nil")
	}
}

// TestLoadFromDir_AngelaDefaults_ZeroConfig verifies that running
// `lore angela draft` in a directory with NO .lorerc file produces
// sensible defaults (invariant I3: zero-config works).
func TestLoadFromDir_AngelaDefaults_ZeroConfig(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Global angela defaults.
	if cfg.Angela.ModeDetection != "auto" {
		t.Errorf("ModeDetection default: got %q, want %q", cfg.Angela.ModeDetection, "auto")
	}
	if cfg.Angela.I18n.SuggestionsLanguage != "auto" {
		t.Errorf("I18n.SuggestionsLanguage default: got %q", cfg.Angela.I18n.SuggestionsLanguage)
	}
	if cfg.Angela.Personas.Selection != "auto" {
		t.Errorf("Personas.Selection default: got %q", cfg.Angela.Personas.Selection)
	}
	if cfg.Angela.Personas.Max != 3 {
		t.Errorf("Personas.Max default: got %d, want 3", cfg.Angela.Personas.Max)
	}
	if cfg.Angela.Personas.FreeFormMode != "minimal" {
		t.Errorf("Personas.FreeFormMode default: got %q", cfg.Angela.Personas.FreeFormMode)
	}

	// Draft defaults.
	if !cfg.Angela.Draft.Checks.Structure {
		t.Error("Draft.Checks.Structure should default to true")
	}
	if !cfg.Angela.Draft.Differential.Enabled {
		t.Error("Draft.Differential.Enabled should default to true")
	}
	if cfg.Angela.Draft.Differential.StateFile != "draft-state.json" {
		t.Errorf("Draft.Differential.StateFile: got %q", cfg.Angela.Draft.Differential.StateFile)
	}
	if cfg.Angela.Draft.Output.Format != "human" {
		t.Errorf("Draft.Output.Format: got %q", cfg.Angela.Draft.Output.Format)
	}
	if cfg.Angela.Draft.ExitCode.FailOn != "error" {
		t.Errorf("Draft.ExitCode.FailOn: got %q", cfg.Angela.Draft.ExitCode.FailOn)
	}
	if cfg.Angela.Draft.IgnoreFile != ".angela-ignore" {
		t.Errorf("Draft.IgnoreFile: got %q", cfg.Angela.Draft.IgnoreFile)
	}
	if cfg.Angela.Draft.Autofix.Enabled {
		t.Error("Draft.Autofix.Enabled should default to false (opt-in)")
	}

	// Review defaults.
	if !cfg.Angela.Review.Evidence.Required {
		t.Error("Review.Evidence.Required should default to true")
	}
	if cfg.Angela.Review.Evidence.MinConfidence != 0.4 {
		t.Errorf("Review.Evidence.MinConfidence: got %f", cfg.Angela.Review.Evidence.MinConfidence)
	}
	if cfg.Angela.Review.Evidence.Validation != "strict" {
		t.Errorf("Review.Evidence.Validation: got %q", cfg.Angela.Review.Evidence.Validation)
	}
	if cfg.Angela.Review.Sampling.Limit != 50 {
		t.Errorf("Review.Sampling.Limit: got %d", cfg.Angela.Review.Sampling.Limit)
	}
	if cfg.Angela.Review.ExitCode.FailOnSeverity != "contradiction" {
		t.Errorf("Review.ExitCode.FailOnSeverity: got %q", cfg.Angela.Review.ExitCode.FailOnSeverity)
	}

	// Polish defaults.
	if cfg.Angela.Polish.DryRun {
		t.Error("Polish.DryRun should default to false")
	}
	if !cfg.Angela.Polish.Backup.Enabled {
		t.Error("Polish.Backup.Enabled should default to true (safety net)")
	}
	if cfg.Angela.Polish.Backup.RetentionDays != 30 {
		t.Errorf("Polish.Backup.RetentionDays: got %d", cfg.Angela.Polish.Backup.RetentionDays)
	}
	if !cfg.Angela.Polish.HallucinationCheck.Enabled {
		t.Error("Polish.HallucinationCheck.Enabled should default to true")
	}
	if cfg.Angela.Polish.HallucinationCheck.Strictness != "warn" {
		t.Errorf("Polish.HallucinationCheck.Strictness: got %q", cfg.Angela.Polish.HallucinationCheck.Strictness)
	}
	if cfg.Angela.Polish.Incremental.Enabled {
		t.Error("Polish.Incremental.Enabled should default to false (opt-in)")
	}
	// Story 8-21: polish.log schema keys (consumed by story 8-23 pruner).
	if cfg.Angela.Polish.Log.RetentionDays != 30 {
		t.Errorf("Polish.Log.RetentionDays: got %d, want 30", cfg.Angela.Polish.Log.RetentionDays)
	}
	if cfg.Angela.Polish.Log.MaxSizeMB != 10 {
		t.Errorf("Polish.Log.MaxSizeMB: got %d, want 10", cfg.Angela.Polish.Log.MaxSizeMB)
	}
}

// TestLoadFromDir_ExtendedAngelaConfigFullyPopulated writes a .lorerc with
// every new field set to a non-default value and verifies each one round-trips
// correctly through Viper + mapstructure. This protects the per-command
// sub-config mapping from silent breakage.
func TestLoadFromDir_ExtendedAngelaConfigFullyPopulated(t *testing.T) {
	dir := t.TempDir()
	lorerc := `
language: fr

angela:
  mode: draft  # legacy, should still load
  max_tokens: 4000

  mode_detection: standalone
  state_dir: custom-state

  i18n:
    suggestions_language: fr

  personas:
    selection: manual
    max: 5
    manual_list: [architect, tech-writer]
    free_form_mode: full

  draft:
    checks:
      structure: false
      completeness: true
      coherence: false
      personas: true
      scoring: false
    scoring:
      profile: strict
      show_breakdown: true
      fail_below: 60
    severity_override:
      coherence: info
      style: 'off'
    differential:
      enabled: false
      state_file: my-draft.json
      diff_only: true
    autofix:
      enabled: true
      mode: aggressive
      backup: false
    interactive:
      enabled: true
    personas:
      selection: all
      max: 6
      manual_list: []
      free_form_mode: full
    output:
      format: json
      color: never
      progress: never
      verbose: true
    exit_code:
      fail_on: warning
      strict: true
    ignore_file: custom-ignore.yaml

  review:
    evidence:
      required: false
      min_confidence: 0.7
      validation: lenient
    differential:
      enabled: false
      state_file: my-review.json
      diff_only: true
    sampling:
      mode: all
      limit: 100
    interactive:
      enabled: true
      editor: nvim
    personas:
      selection: manual
      max: 2
      manual_list: [qa-reviewer]
      free_form_mode: none
    output:
      format: json
    exit_code:
      fail_on_severity: gap

  polish:
    dry_run: true
    backup:
      enabled: false
      path: my-backups
      retention_days: 7
    hallucination_check:
      enabled: false
      strictness: reject
    incremental:
      enabled: true
      min_change_lines: 5
    interactive:
      enabled: true
      granularity: paragraph
    personas:
      selection: none
      max: 0
    audience:
      default: cto
      allowed: [cto, developer]
`
	if err := os.WriteFile(filepath.Join(dir, ".lorerc"), []byte(lorerc), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadFromDir(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	// Legacy fields still work.
	if cfg.Angela.MaxTokens != 4000 {
		t.Errorf("Angela.MaxTokens: got %d, want 4000", cfg.Angela.MaxTokens)
	}

	// Global angela fields.
	if cfg.Angela.ModeDetection != "standalone" {
		t.Errorf("ModeDetection: got %q", cfg.Angela.ModeDetection)
	}
	if cfg.Angela.StateDir != "custom-state" {
		t.Errorf("StateDir: got %q", cfg.Angela.StateDir)
	}
	if cfg.Angela.I18n.SuggestionsLanguage != "fr" {
		t.Errorf("I18n.SuggestionsLanguage: got %q", cfg.Angela.I18n.SuggestionsLanguage)
	}
	if cfg.Angela.Personas.Selection != "manual" {
		t.Errorf("Personas.Selection: got %q", cfg.Angela.Personas.Selection)
	}
	if cfg.Angela.Personas.Max != 5 {
		t.Errorf("Personas.Max: got %d", cfg.Angela.Personas.Max)
	}
	if len(cfg.Angela.Personas.ManualList) != 2 || cfg.Angela.Personas.ManualList[0] != "architect" {
		t.Errorf("Personas.ManualList: got %v", cfg.Angela.Personas.ManualList)
	}
	if cfg.Angela.Personas.FreeFormMode != "full" {
		t.Errorf("Personas.FreeFormMode: got %q", cfg.Angela.Personas.FreeFormMode)
	}

	// Draft fields.
	if cfg.Angela.Draft.Checks.Structure {
		t.Error("Draft.Checks.Structure should be false")
	}
	if cfg.Angela.Draft.Scoring.Profile != "strict" {
		t.Errorf("Draft.Scoring.Profile: got %q", cfg.Angela.Draft.Scoring.Profile)
	}
	if cfg.Angela.Draft.Scoring.FailBelow != 60 {
		t.Errorf("Draft.Scoring.FailBelow: got %d", cfg.Angela.Draft.Scoring.FailBelow)
	}
	if cfg.Angela.Draft.SeverityOverride["coherence"] != "info" {
		t.Errorf("Draft.SeverityOverride[coherence]: got %q", cfg.Angela.Draft.SeverityOverride["coherence"])
	}
	if cfg.Angela.Draft.SeverityOverride["style"] != "off" {
		t.Errorf("Draft.SeverityOverride[style]: got %q", cfg.Angela.Draft.SeverityOverride["style"])
	}
	if cfg.Angela.Draft.Differential.Enabled {
		t.Error("Draft.Differential.Enabled should be false")
	}
	if cfg.Angela.Draft.Differential.StateFile != "my-draft.json" {
		t.Errorf("Draft.Differential.StateFile: got %q", cfg.Angela.Draft.Differential.StateFile)
	}
	if !cfg.Angela.Draft.Autofix.Enabled {
		t.Error("Draft.Autofix.Enabled should be true")
	}
	if cfg.Angela.Draft.Autofix.Mode != "aggressive" {
		t.Errorf("Draft.Autofix.Mode: got %q", cfg.Angela.Draft.Autofix.Mode)
	}
	if !cfg.Angela.Draft.Interactive.Enabled {
		t.Error("Draft.Interactive.Enabled should be true")
	}
	if cfg.Angela.Draft.Personas.Selection != "all" {
		t.Errorf("Draft.Personas.Selection: got %q", cfg.Angela.Draft.Personas.Selection)
	}
	if cfg.Angela.Draft.Output.Format != "json" {
		t.Errorf("Draft.Output.Format: got %q", cfg.Angela.Draft.Output.Format)
	}
	if cfg.Angela.Draft.ExitCode.FailOn != "warning" {
		t.Errorf("Draft.ExitCode.FailOn: got %q", cfg.Angela.Draft.ExitCode.FailOn)
	}
	if !cfg.Angela.Draft.ExitCode.Strict {
		t.Error("Draft.ExitCode.Strict should be true")
	}
	if cfg.Angela.Draft.IgnoreFile != "custom-ignore.yaml" {
		t.Errorf("Draft.IgnoreFile: got %q", cfg.Angela.Draft.IgnoreFile)
	}

	// Review fields.
	if cfg.Angela.Review.Evidence.Required {
		t.Error("Review.Evidence.Required should be false")
	}
	if cfg.Angela.Review.Evidence.MinConfidence != 0.7 {
		t.Errorf("Review.Evidence.MinConfidence: got %f", cfg.Angela.Review.Evidence.MinConfidence)
	}
	if cfg.Angela.Review.Evidence.Validation != "lenient" {
		t.Errorf("Review.Evidence.Validation: got %q", cfg.Angela.Review.Evidence.Validation)
	}
	if cfg.Angela.Review.Sampling.Mode != "all" {
		t.Errorf("Review.Sampling.Mode: got %q", cfg.Angela.Review.Sampling.Mode)
	}
	if cfg.Angela.Review.Sampling.Limit != 100 {
		t.Errorf("Review.Sampling.Limit: got %d", cfg.Angela.Review.Sampling.Limit)
	}
	if !cfg.Angela.Review.Interactive.Enabled {
		t.Error("Review.Interactive.Enabled should be true")
	}
	if cfg.Angela.Review.Interactive.Editor != "nvim" {
		t.Errorf("Review.Interactive.Editor: got %q", cfg.Angela.Review.Interactive.Editor)
	}
	if cfg.Angela.Review.ExitCode.FailOnSeverity != "gap" {
		t.Errorf("Review.ExitCode.FailOnSeverity: got %q", cfg.Angela.Review.ExitCode.FailOnSeverity)
	}

	// Polish fields.
	if !cfg.Angela.Polish.DryRun {
		t.Error("Polish.DryRun should be true")
	}
	if cfg.Angela.Polish.Backup.Enabled {
		t.Error("Polish.Backup.Enabled should be false")
	}
	if cfg.Angela.Polish.Backup.Path != "my-backups" {
		t.Errorf("Polish.Backup.Path: got %q", cfg.Angela.Polish.Backup.Path)
	}
	if cfg.Angela.Polish.Backup.RetentionDays != 7 {
		t.Errorf("Polish.Backup.RetentionDays: got %d", cfg.Angela.Polish.Backup.RetentionDays)
	}
	if cfg.Angela.Polish.HallucinationCheck.Enabled {
		t.Error("Polish.HallucinationCheck.Enabled should be false")
	}
	if cfg.Angela.Polish.HallucinationCheck.Strictness != "reject" {
		t.Errorf("Polish.HallucinationCheck.Strictness: got %q", cfg.Angela.Polish.HallucinationCheck.Strictness)
	}
	if !cfg.Angela.Polish.Incremental.Enabled {
		t.Error("Polish.Incremental.Enabled should be true")
	}
	if cfg.Angela.Polish.Incremental.MinChangeLines != 5 {
		t.Errorf("Polish.Incremental.MinChangeLines: got %d", cfg.Angela.Polish.Incremental.MinChangeLines)
	}
	if !cfg.Angela.Polish.Interactive.Enabled {
		t.Error("Polish.Interactive.Enabled should be true")
	}
	if cfg.Angela.Polish.Interactive.Granularity != "paragraph" {
		t.Errorf("Polish.Interactive.Granularity: got %q", cfg.Angela.Polish.Interactive.Granularity)
	}
	if cfg.Angela.Polish.Audience.Default != "cto" {
		t.Errorf("Polish.Audience.Default: got %q", cfg.Angela.Polish.Audience.Default)
	}
	if len(cfg.Angela.Polish.Audience.Allowed) != 2 {
		t.Errorf("Polish.Audience.Allowed: got %v", cfg.Angela.Polish.Audience.Allowed)
	}
}

// TestResolvePersonasConfig_* used to live here. Paranoid-review fix
// (2026-04-11 architecture): the ResolvePersonasConfig API they
// exercised was never called from production — the per-command
// Personas overrides are read nowhere, so the resolver was a 50-line
// dead-code appendage with its own tests. The resolver and its
// tests have been removed; the PersonasConfig struct itself stays
// for schema stability.
