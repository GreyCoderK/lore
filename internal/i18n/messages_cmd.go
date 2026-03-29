// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package i18n

// CmdMessages holds strings for CLI commands (cmd/ package).
// Each field documents its format args in a comment when applicable.
type CmdMessages struct {
	// root.go
	RootShort         string
	RootConfigErrHint string // arg: error
	StoreUnavailWarn  string // arg: error

	// init.go
	InitShort              string
	InitNotGitRepo         string
	InitNotGitRepoHint     string
	InitAlreadyInitialized string
	InitCreatedLore        string
	InitCreatedLorerc      string
	InitCreatedLorercLocal string
	InitModifiedGitignore  string
	InitWarningPrefix      string
	InitHooksPathWarn      string // arg: path
	InitInstalledHook      string
	InitNotInPathWarn      string
	InitInstallHint        string
	InitAddToPathHint      string
	InitBashPathHint       string
	InitZshPathHint        string
	InitFishPathHint       string
	InitPowerShellHint     string
	InitTagline            string
	InitDemoPrompt         string
	InitDemoWarning        string

	// new.go
	NewUse            string
	NewShort          string
	NewCommitNotFound string // arg: hash
	NewCommitFlagDesc string

	// list.go
	ListShort         string
	ListLong          string
	ListParseWarning  string // arg: error
	ListNoDocsOfType  string // arg: type
	ListNoDocsYet     string
	ListTagSingular   string
	ListTagPlural     string

	// show.go
	ShowUse             string
	ShowShort           string
	ShowLong            string
	ShowTypeExclusive   string
	ShowUsageHint       string
	ShowTryHint         string
	ShowNoMatchKeyword  string // arg: keyword
	ShowTryAll          string
	ShowNoDocsFound     string
	ShowSelectPrompt    string

	// delete.go
	DeleteUse            string
	DeleteShort          string
	DeleteInvalidName    string // arg: filename
	DeleteNotFound       string // arg: filename
	DeleteRefWarning     string
	DeleteRefNotUpdated  string
	DeleteForceRequired  string
	DeleteConfirmPrompt  string // arg: filename
	DeleteCancelled      string

	// note.go
	NoteShort string

	// pending.go
	PendingShort          string
	PendingLong           string
	PendingListShort      string
	PendingResolveShort   string
	PendingSkipShort      string
	PendingNoPending      string
	PendingResolveHint    string
	PendingSkipHint       string
	PendingNoPendingRes   string
	PendingInvalidSel     string // args: input, max
	PendingSelectPrompt   string // arg: max
	PendingInvalidSelIn   string // args: input, max
	PendingSkipError      string // arg: error
	PendingListHeader     string

	// status.go
	StatusShort              string
	StatusLong               string
	StatusHeader             string // arg: branch/path
	StatusHookNotInstalled   string
	StatusHookNotInstHint    string
	StatusHookInstalled      string
	StatusHookLabel          string
	StatusDocsDocumented     string // arg: count
	StatusDocsPending        string // arg: count
	StatusDocsLabel          string
	StatusExpressLine        string // args: pct, express, total, altPct
	StatusExpressUnreadable  string // arg: count
	StatusExpressLabel       string
	StatusAngelaMode         string // arg: mode
	StatusAngelaNoApiKey     string
	StatusAngelaProvider     string // arg: provider
	StatusAngelaDocsReview   string // arg: count
	StatusAngelaAllClean     string
	StatusAngelaLabel        string
	StatusReviewNoIssues     string
	StatusReviewFindings     string // args: count, age
	StatusReviewLabel        string
	StatusHealthAllGood      string
	StatusHealthIssues       string // arg: count
	StatusHealthLabel        string
	StatusTagline            string
	StatusReviewAgeJustNow   string
	StatusReviewAgeHours     string // arg: hours
	StatusReviewAgeDays      string // arg: days

	// hook.go
	HookShort               string
	HookInstallShort        string
	HookInstallHooksPathW   string // arg: path
	HookInstallCannotAuto   string
	HookInstallManualHint   string
	HookInstallVerb         string
	HookUninstallShort      string
	HookUninstallVerb       string
	HookPostCommitShort     string

	// demo.go
	DemoShort          string
	DemoConsentPrompt  string
	DemoSimCommit      string // arg: message
	DemoTypePrompt     string // arg: value
	DemoTypeConfirm    string // arg: value
	DemoWhatPrompt     string // arg: value
	DemoWhatConfirm    string // arg: value
	DemoWhyPrompt      string // arg: value
	DemoIndexWarning   string // arg: error
	DemoSimShow        string
	DemoShowType       string // arg: type
	DemoShowWhat       string // arg: what
	DemoShowWhy        string // arg: why
	DemoShowDate       string // arg: date
	DemoShowCommit     string // arg: hash
	DemoShowStatus     string
	DemoTagline        string

	// release.go
	ReleaseShort            string
	ReleaseNotInitMsg       string
	ReleaseNotInitHint      string
	ReleaseNoTagsError      string
	ReleaseNoTagsHint       string
	ReleaseParseWarning     string // arg: error
	ReleaseNoChanges        string
	ReleaseNoChangesHint    string
	ReleaseChangelogHdrWarn string
	ReleaseIndexRegenWarn   string // arg: error

	// doctor.go
	DoctorShort           string
	DoctorConfigCheck     string
	DoctorConfigOK        string
	DoctorActiveValues    string
	DoctorValueNotSet     string
	DoctorDocsCheck       string
	DoctorNoneFound       string
	DoctorConfigOKInline  string
	DoctorHealthAllGood   string
	DoctorIssuesFound     string // arg: count
	DoctorManualFix       string // args: issue, hint
	DoctorFixSummary      string // args: fixed, remaining
	DoctorStoreRebuilt    string // arg: docCount
	DoctorStoreSkipped    string // arg: count
	DoctorStoreCommits    string // arg: count

	// config_cmd.go
	ConfigShort          string
	SetKeyShort          string
	SetKeyUnknownProv    string // args: provider, supported
	SetKeyPrompt         string // arg: provider
	SetKeyStored         string // arg: provider
	DeleteKeyShort       string
	DeleteKeyUnknownProv string // args: provider, supported
	DeleteKeyDeleted     string // arg: provider
	ListKeysShort        string
	ListKeysStored       string // arg: provider
	ListKeysNotSet       string // arg: provider

	// lore_check.go
	LoreCheckNotInit     string
	LoreCheckNotInitHint string

	// angela.go
	AngelaShort              string
	AngelaDraftShort         string
	AngelaDraftNoFile        string
	AngelaDraftNotFound      string // arg: filename
	AngelaDraftCorpusWarn    string // arg: error
	AngelaDraftNoSuggestions string
	AngelaDraftHeader        string // arg: filename
	AngelaDraftScoreLine     string // args: score, label
	AngelaDraftSuggCount     string // arg: count
	AngelaDraftAllNoDocs     string
	AngelaDraftAllHeader     string // arg: count
	AngelaDraftAllSugg       string // args: count, score
	AngelaDraftAllSuggWarn   string // args: count, warnings, score
	AngelaDraftAllSummary    string // args: needAttention, total, suggestions

	// angela_polish.go
	AngelaPolishShort      string
	AngelaPolishNotFound   string // arg: filename
	AngelaPolishNoProvider string
	AngelaPolishNoChanges  string
	AngelaPolishNoneApplied string
	AngelaPolishIndexWarn  string // arg: error
	AngelaPolishVerb       string // arg: filename

	// angela_review.go
	AngelaReviewShort       string
	AngelaReviewNoProvider  string
	AngelaReviewCorpusNote  string // arg: count
	AngelaReviewCacheWarn   string // arg: error
	AngelaReviewHdrPartial  string // args: analyzed, total
	AngelaReviewHdrFull     string // arg: count
	AngelaReviewCoherent    string
	AngelaReviewFindingSum  string // args: count, severity
	AngelaReviewMinDocs    string // args: minRequired, currentCount
	AngelaReviewParseErr   string
	AngelaPolishModified   string

	// decision.go
	DecisionShort          string
	DecisionDiffWarn       string // arg: error
	DecisionCommitLabel    string // arg: hash
	DecisionSubjectLabel   string // arg: subject
	DecisionScoreLabel     string // arg: score
	DecisionActionLabel    string // arg: action
	DecisionConfidenceLabel string // arg: confidence
	DecisionSignalsHeader  string
	DecisionPrefillHeader  string
	DecisionPrefillWhat    string // arg: what
	DecisionPrefillWhy     string // args: why, confidence
	DecisionStoreUnavail   string

	// completion.go
	CompletionShort        string
	CompletionLong         string

	// root.go — language warnings
	UnsupportedLangWarn    string // arg: lang

	// show.go — deprecation
	ShowAllDeprecated      string
}
