// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// SharedMessages holds strings used across multiple packages.
type SharedMessages struct {
	// config/config.go
	ConfigInsecurePerms string // args: path, mode, path
	ConfigUnknownWarn   string // arg: warning

	// config/validate.go
	ConfigNotSet            string
	ValidateUnknownDidYou   string // args: field, file, suggestion
	ValidateUnknownField    string // args: field, file

	// ai/provider.go
	AINoKeyAnthropic     string
	AINoKeyOpenAI        string
	AIUnknownProvider    string // args: provider, supported
	AIKeychainNotAvail   string
	AIPlaintextKeyWarn   string // arg: provider

	// storage/index.go
	IndexTitle           string
	IndexAutoGenNote     string
	IndexTableHeaderDoc  string
	IndexTableHeaderType string
	IndexTableHeaderDate string
	IndexTableHeaderStat string
	IndexTableHeaderTags string
	IndexDocSingular     string
	IndexDocPlural       string
	IndexTotalCount      string // args: count, unit

	// storage/release.go
	ReleaseSectionFeatures  string
	ReleaseSectionBugFixes  string
	ReleaseSectionRefactors string
	ReleaseSectionDecisions string
	ReleaseSectionNotes     string
	ReleaseTitle            string // arg: version
}
