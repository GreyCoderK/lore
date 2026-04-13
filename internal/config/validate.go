// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigReport holds the results of config validation.
type ConfigReport struct {
	Valid    bool              // true if no errors (warnings are OK)
	Warnings []string         // unknown fields, suggestions
	Errors   []string         // invalid values, parse errors
	Active   map[string]string // resolved values after cascade
}

// OK returns true when validation found no errors and no warnings.
func (r *ConfigReport) OK() bool {
	return len(r.Warnings) == 0 && len(r.Errors) == 0
}

// maskSecret returns "****" if s is non-empty, or "(not set)" if empty.
func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	return "****"
}

// validFields is the set of all known dot-separated config keys,
// auto-generated from the Config struct via reflection on yaml tags.
// This eliminates manual synchronization when adding new config fields.
var validFields = buildValidFields(reflect.TypeOf(Config{}), "")

// validPrefixes is the set of config key prefixes corresponding to map
// fields. Any user-supplied key that starts with one of these prefixes
// is considered valid even if it is not in validFields, because map
// sub-keys are dynamic and cannot be enumerated at compile time.
var validPrefixes = buildValidPrefixes(reflect.TypeOf(Config{}), "")

// deprecatedFields maps dot-separated config keys to a deprecation message.
// A present-but-ignored field emits a warning at doctor --config time so the
// user discovers that the knob they thought they were turning is inert.
// Keep the message short — it is shown inline in the doctor report.
var deprecatedFields = map[string]string{
	"angela.mode": "angela.mode is ignored — choose the mode via the sub-command (lore angela draft / polish / review)",
}

// buildValidFields recursively extracts yaml tag names from a struct type
// to build the set of valid dot-separated config keys.
func buildValidFields(t reflect.Type, prefix string) map[string]bool {
	fields := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		// yaml tag may contain options like "omitempty" — take only the name
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		fields[key] = true
		// Recurse into struct fields
		ft := f.Type
		if ft.Kind() == reflect.Struct {
			for k, v := range buildValidFields(ft, key) {
				fields[k] = v
			}
		}
	}
	return fields
}

// buildValidPrefixes recursively finds map-typed fields and returns their
// dot-separated key paths. Any user config key that starts with
// "<prefix>." is accepted without a warning, because map sub-keys are
// dynamic and cannot be enumerated at build time.
func buildValidPrefixes(t reflect.Type, prefix string) map[string]bool {
	prefixes := make(map[string]bool)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		key := name
		if prefix != "" {
			key = prefix + "." + name
		}
		ft := f.Type
		if ft.Kind() == reflect.Map {
			prefixes[key] = true
		} else if ft.Kind() == reflect.Struct {
			for k, v := range buildValidPrefixes(ft, key) {
				prefixes[k] = v
			}
		}
	}
	return prefixes
}

// hasValidPrefix reports whether key is a sub-key of any registered map
// prefix (e.g. "angela.style_guide.foo" is valid when "angela.style_guide"
// is a prefix).
func hasValidPrefix(key string) bool {
	for pfx := range validPrefixes {
		if strings.HasPrefix(key, pfx+".") {
			return true
		}
	}
	return false
}

// ValidateConfig validates .lorerc and .lorerc.local files in dir.
// It checks for unknown fields (with "did you mean?" suggestions) and
// collects the active resolved values.
func ValidateConfig(dir string) *ConfigReport {
	report := &ConfigReport{
		Active: make(map[string]string),
	}

	// Check each config file for unknown fields.
	for _, name := range []string{".lorerc", ".lorerc.local"} {
		checkConfigFile(dir, name, report)
	}

	// Collect active values by loading the full cascade.
	cfg, err := LoadFromDir(dir)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("load error: %v", err))
		report.Valid = false
		return report
	}

	report.Active["ai.provider"] = cfg.AI.Provider
	report.Active["ai.model"] = cfg.AI.Model
	report.Active["ai.api_key"] = maskSecret(cfg.AI.APIKey)
	report.Active["ai.endpoint"] = cfg.AI.Endpoint
	report.Active["ai.timeout"] = cfg.AI.Timeout.String()
	// angela.mode is intentionally NOT reported: it is deprecated and has
	// no runtime effect. See deprecatedFields above.
	report.Active["angela.max_tokens"] = fmt.Sprintf("%d", cfg.Angela.MaxTokens)
	report.Active["templates.dir"] = cfg.Templates.Dir
	report.Active["hooks.post_commit"] = fmt.Sprintf("%t", cfg.Hooks.PostCommit)
	report.Active["output.dir"] = cfg.Output.Dir
	report.Active["output.format"] = cfg.Output.Format

	// Invariant I1: draft must stay offline. A programming error that adds
	// an AI field to DraftConfig would make draft silently depend on a
	// provider. Check at load time so regressions fail loud.
	if err := ValidateDraftOfflineInvariant(); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("invariant violation: %v", err))
	}

	report.Valid = len(report.Errors) == 0
	return report
}

// forbiddenDraftFieldSegments lists CamelCase-word segments that indicate AI /
// network dependency. A DraftConfig field whose name (split on CamelCase
// boundaries) contains any of these exact segments violates Invariant I1
// (draft is offline forever).
//
// These are matched against individual segments (not substrings) to avoid
// false positives like "FailOn" accidentally matching "ai" inside "fail".
var forbiddenDraftFieldSegments = map[string]bool{
	"ai":         true,
	"model":      true,
	"models":     true,
	"provider":   true,
	"escalate":   true,
	"escalation": true,
	"apikey":     true,
	"endpoint":   true,
	"llm":        true,
	"openai":     true,
	"anthropic":  true,
	"ollama":     true,
	"token":      true,
	"tokens":     true,
}

// ValidateDraftOfflineInvariant enforces Invariant I1: DraftConfig and its
// nested sub-configs must not contain any field whose CamelCase-split name
// hints at AI or network dependency. The invariant is structural — the type
// system already prevents most violations because DraftConfig has no provider
// fields — but this check adds a runtime guard against accidents during
// refactors.
//
// Called from ValidateConfig() at config load time and from the unit test
// TestDraftConfig_NoAIFields.
func ValidateDraftOfflineInvariant() error {
	draftType := reflect.TypeOf(DraftConfig{})
	return walkStructForbiddenSegments(draftType, "DraftConfig", forbiddenDraftFieldSegments)
}

// walkStructForbiddenSegments recursively inspects a struct type. For every
// field (and every nested struct field), it splits the field name on
// CamelCase word boundaries and checks whether any segment (lowercased)
// matches the forbidden set. Returns the first violation found or nil when
// the type is clean.
func walkStructForbiddenSegments(t reflect.Type, path string, forbidden map[string]bool) error {
	if t.Kind() != reflect.Struct {
		return nil
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		for _, seg := range splitCamelCase(f.Name) {
			if forbidden[strings.ToLower(seg)] {
				return fmt.Errorf("%s.%s contains forbidden segment %q — draft must stay offline (I1)", path, f.Name, seg)
			}
		}
		// Recurse into nested struct fields (common Go idiom, not pointer).
		if f.Type.Kind() == reflect.Struct {
			if err := walkStructForbiddenSegments(f.Type, path+"."+f.Name, forbidden); err != nil {
				return err
			}
		}
	}
	return nil
}

// splitCamelCase splits a Go identifier into its CamelCase word segments.
// Example: "MaxOutputTokens" → ["Max", "Output", "Tokens"].
// Consecutive uppercase letters are treated as an acronym segment:
// "HTTPServer" → ["HTTP", "Server"], "APIKey" → ["API", "Key"].
func splitCamelCase(name string) []string {
	var segments []string
	var current []rune
	runes := []rune(name)
	for i, r := range runes {
		if i > 0 && isUpper(r) {
			// Boundary: Upper after lower (Max|Output) OR
			// last Upper in an acronym run followed by lower (HTTP|Server).
			prev := runes[i-1]
			next := rune(0)
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			if !isUpper(prev) || (next != 0 && !isUpper(next)) {
				if len(current) > 0 {
					segments = append(segments, string(current))
					current = current[:0]
				}
			}
		}
		current = append(current, r)
	}
	if len(current) > 0 {
		segments = append(segments, string(current))
	}
	return segments
}

func isUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// checkConfigFile parses a single YAML config file and reports unknown keys.
func checkConfigFile(dir, name string, report *ConfigReport) {
	// Try extensions in order: .yaml, .yml, bare
	var path string
	for _, ext := range []string{".yaml", ".yml", ""} {
		candidate := filepath.Join(dir, name+ext)
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
	}
	if path == "" {
		return // file not found — not an error
	}

	data, err := os.ReadFile(path)
	if err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s: read error: %v", filepath.Base(path), err))
		return
	}
	if len(data) > 1<<20 {
		report.Errors = append(report.Errors, fmt.Sprintf("%s: file too large (%d bytes, max 1MB)", filepath.Base(path), len(data)))
		return
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("%s: YAML parse error: %v", filepath.Base(path), err))
		return
	}

	// Flatten nested map to dot-separated keys.
	flat := flattenMap("", raw)
	for _, key := range flat {
		if msg, deprecated := deprecatedFields[key]; deprecated {
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("deprecated field %q in %s — %s", key, filepath.Base(path), msg))
			continue
		}
		if !validFields[key] && hasValidPrefix(key) {
			continue
		}
		if !validFields[key] {
			suggestion := suggestField(key)
			if suggestion != "" {
				report.Warnings = append(report.Warnings,
					fmt.Sprintf("unknown field %q in %s, did you mean %q?", key, filepath.Base(path), suggestion))
			} else {
				report.Warnings = append(report.Warnings,
					fmt.Sprintf("unknown field %q in %s", key, filepath.Base(path)))
			}
		}
	}
}

// flattenMap returns all dot-separated key paths from a nested map, sorted.
func flattenMap(prefix string, m map[string]interface{}) []string {
	var keys []string
	for k, v := range m {
		full := k
		if prefix != "" {
			full = prefix + "." + k
		}
		keys = append(keys, full)
		if sub, ok := v.(map[string]interface{}); ok {
			keys = append(keys, flattenMap(full, sub)...)
		}
	}
	sort.Strings(keys)
	return keys
}

// suggestField returns the closest valid field name if Levenshtein distance ≤ 2,
// or empty string if no close match exists.
func suggestField(unknown string) string {
	best := ""
	bestDist := 3 // threshold: distance must be < 3
	for field := range validFields {
		// Only compare leaf fields (containing a dot).
		if !strings.Contains(field, ".") {
			continue
		}
		d := levenshtein(unknown, field)
		if d < bestDist {
			bestDist = d
			best = field
		}
	}
	return best
}

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use single-row optimization.
	prev := make([]int, lb+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr := make([]int, lb+1)
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev = curr
	}
	return prev[lb]
}

