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
	report.Active["angela.mode"] = cfg.Angela.Mode
	report.Active["angela.max_tokens"] = fmt.Sprintf("%d", cfg.Angela.MaxTokens)
	report.Active["templates.dir"] = cfg.Templates.Dir
	report.Active["hooks.post_commit"] = fmt.Sprintf("%t", cfg.Hooks.PostCommit)
	report.Active["output.dir"] = cfg.Output.Dir
	report.Active["output.format"] = cfg.Output.Format

	report.Valid = len(report.Errors) == 0
	return report
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

