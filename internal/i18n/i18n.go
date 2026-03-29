// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package i18n provides internationalization for Lore CLI.
// All user-facing strings are centralized here. AI prompt directives
// and content signals remain in English (not part of this package).
//
// Architecture note: catalogs are Go code (catalog_en.go, catalog_fr.go).
// This is appropriate for 2-4 languages. If the project reaches 5+ languages,
// consider migrating to resource files (JSON/TOML) with code generation,
// to avoid requiring Go expertise from translators.
package i18n

import (
	"fmt"
	"os"
	"sync/atomic"
)

// Lang represents a supported language.
type Lang string

const (
	EN Lang = "en"
	FR Lang = "fr"
)

// current holds the active Messages catalog. Accessed via atomic.Value
// for thread safety. Pre-initialized to EN so T() never returns nil,
// even if called before Init().
var current atomic.Value

func init() {
	current.Store(catalogEN)
}

// Init sets the active language catalog. If lang is not supported,
// falls back silently to EN. The warning for unsupported languages
// is added in story 7b.4a (requires streams access).
func Init(lang string) {
	switch Lang(lang) {
	case EN:
		current.Store(catalogEN)
	case FR:
		current.Store(catalogFR)
	default:
		current.Store(catalogEN)
		if lang != "" {
			fmt.Fprintf(os.Stderr, "warning: unsupported language %q, falling back to English\n", lang)
		}
	}
}

// T returns the active Messages catalog. Never returns nil.
// Safe to call from any goroutine, including before Init().
func T() *Messages {
	v := current.Load()
	if v == nil {
		return catalogEN
	}
	return v.(*Messages)
}

// SupportedLanguages returns the list of languages with available catalogs.
// Phase A: EN only. Phase B adds FR.
func SupportedLanguages() []Lang {
	return []Lang{EN, FR}
}

// IsSupported checks if a language has a catalog available.
func IsSupported(lang string) bool {
	for _, l := range SupportedLanguages() {
		if string(l) == lang {
			return true
		}
	}
	return false
}
