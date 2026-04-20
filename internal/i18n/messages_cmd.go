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
	PendingFlagQuiet      string
	PendingFlagCommit     string
	PendingFlagType       string
	PendingFlagWhat       string
	PendingFlagWhy        string
	PendingFlagAlt        string
	PendingFlagImpact     string
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
	StatusFlagQuiet          string
	StatusFlagBadge          string
	StatusCollecting         string // spinner label

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
	ReleaseCollecting       string // spinner label
	ReleaseGenerating       string // spinner label

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
	DoctorScanning        string // spinner label
	DoctorScanned         string // args: docCount, checks
	DoctorFixing          string // spinner label
	DoctorRebuilding      string // spinner label

	// Story 8-22 — malformed-frontmatter suggestion block.
	DoctorSuggestedActions    string // no args
	DoctorMalformedRestore    string // no args — "Restore from a polish backup:"
	DoctorMalformedEditManual string // no args — "Edit the file manually to repair the YAML block."

	// Story 8-23 — prune output.
	DoctorPruneHeader      string // no args — "Pruning generated artifacts:"
	DoctorPruneRowOK       string // args: feature (str), removed (int), kept (int), human-bytes (str)
	DoctorPruneRowErr      string // args: feature (str), err (str)
	DoctorPruneTotal       string // arg: human-bytes (str)
	DoctorPruneDryRunFoot  string // no args — "(dry-run: no files changed)"

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
	AngelaDraftAllVerboseHint string // hint about --verbose flag when warnings exist
	AngelaDraftAllWarningsHeader string // shown before inline warning list
	AngelaDraftDiffSummary    string // args: new, persisting, resolved — differential mode summary line
	AngelaDraftResolvedHeader string // header shown before the resolved-since-last-run list

	// angela_polish.go
	AngelaPolishShort       string
	AngelaPolishNotFound    string // arg: filename
	AngelaPolishNoProvider  string
	AngelaPolishNoChanges   string
	AngelaPolishNoneApplied string
	AngelaPolishIndexWarn   string // arg: error
	AngelaPolishVerb        string // arg: filename
	AngelaPolishProgress    string // arg: filename — shown before API call
	AngelaPolishDone        string // shown after API call completes
	AngelaPolishStep1       string // arg: filename — step 1/3 preparing
	AngelaPolishStep2       string // arg: filename — step 2/3 calling AI (spinner)
	AngelaPolishStep2Done   string // step 2/3 done
	AngelaPolishStep3       string // step 3/3 computing diff

	// Polish safety nets (backup + dry-run)
	AngelaPolishBackupCreated   string // arg: path
	AngelaPolishBackupPruneWarn string // arg: error
	AngelaPolishBackupDisabled  string
	AngelaPolishRestoreShort    string
	AngelaPolishRestoreNoBackup string // arg: filename
	AngelaPolishRestoreOK       string // args: filename, stamp
	AngelaPolishRestoreListHdr  string // arg: filename
	AngelaPolishRestoreListRow  string // args: stamp, pretty time
	AngelaPolishRestoreUnknown  string // arg: stamp

	// Story 8-21 — structural integrity messages (cmd-level, user-facing).
	AngelaPolishCorruptSource        string // no args — title of refusal
	AngelaPolishCorruptSourceHint    string // arg: filename — pointer to doctor/restore
	AngelaPolishLeakedFMStripped     string // args: bytes (int), line (int)
	AngelaPolishDryRunDuplicates     string // arg: group count
	AngelaPolishDuplicateHeadingRow  string // args: heading (%q), count (int)
	AngelaPolishArbitrateAbortMsg    string // arg: group count
	AngelaPolishArbitrateRefusedMsg  string // arg: group count
	AngelaPolishArbitrateRefusedHint string // no args — "Re-run in a TTY, or use --arbitrate-rule=…"

	// angela_review.go
	AngelaReviewShort       string
	AngelaReviewNoProvider  string
	AngelaReviewCorpusNote  string // arg: count
	AngelaReviewCacheWarn   string // arg: error
	AngelaReviewHdrPartial  string // args: analyzed, total
	AngelaReviewHdrFull     string // arg: count
	AngelaReviewCoherent    string
	AngelaReviewFindingSum  string // args: count, severity
	AngelaReviewMinDocs     string // args: minRequired, currentCount
	AngelaReviewParseErr    string
	AngelaReviewProgress    string // arg: count — shown before API call
	AngelaReviewDone        string // shown after API call completes
	AngelaReviewStep1       string // arg: count — step 1/2 preparing summaries
	AngelaReviewStep2       string // arg: count — step 2/2 calling AI (spinner)
	AngelaReviewStep2Done   string // step 2/2 done

	// Evidence-required findings rejection output
	AngelaReviewRejectedCount string // arg: count
	AngelaReviewRejectedLine  string // args: title, reason
	AngelaPolishModified    string

	// Error messages for angela review command
	AngelaReviewErrFormatRequiresPreview    string // static error, no args
	AngelaReviewErrMutuallyExclusive        string // arg: comma-separated flag names
	AngelaReviewErrUnknownPersonas          string // arg: comma-separated names
	AngelaReviewErrUnknownConfiguredPersona string // arg: comma-separated names
	AngelaReviewErrUseConfiguredNoManual    string // static error, no args
	AngelaReviewErrUnknownFormat            string // arg: format value

	// Badge hints (status, init, draft)
	BadgeHintStatus        string // shown at end of lore status dashboard
	BadgeHintInit          string // shown after lore init
	BadgeHintDraftClean    string // shown when draft --all finds zero suggestions

	// Synthesizer cmd strings
	SynthDryRunFooter      string // --synthesizer-dry-run footer
	SynthApplyDone         string // args: count, filename
	SynthApplyStatusChange string // args: old, new
	SynthApplyNone         string // args: filename
	SynthDryRunHeader      string // args: filename, count
	SynthDryRunProposal    string // args: i, total, name, key
	ConsultHeader          string // args: file, icon, name, expertise
	ConsultNoSuggestion    string // arg: name
	ConsultListTitle       string
	ConsultListUsage       string
	StatusCoverage         string // arg: percent

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

	// upgrade.go
	UpgradeShort         string
	UpgradeChecking      string
	UpgradeAlreadyLatest string // arg: version
	UpgradeNewVersion    string // args: current, latest
	UpgradeDownloading   string // arg: version
	UpgradeVerifying     string
	UpgradeChecksumFail  string
	UpgradeInstalling    string
	UpgradeSuccess       string // arg: version
	UpgradeHomebrew      string // arg: command
	UpgradeGoInstall     string // arg: command
	UpgradePermissionErr string // arg: path
	UpgradeNetworkErr    string
	UpgradeAPIErr        string // arg: error
	UpgradeNoRelease     string
	UpgradeSkipDevBuild  string
	UpgradeVersionFlag   string
	UpgradeVersionNotFnd string // arg: version

	// check_update.go
	CheckUpdateShort       string
	CheckUpdateUpToDate    string // arg: version
	CheckUpdateAvail       string // args: current, latest
	CheckUpdateHint        string
	CheckUpdatePreRelease  string
	CheckUpdateChecking    string // spinner label

	// ─── Flag help strings (Task 9 i18n sweep) ────────────────────────
	// Naming convention: <Command><Name>Flag for each flag. All flag
	// help strings MUST carry the same positional %s/%d/%q specifiers
	// in EN and FR — enforced by TestI11_AllCatalogsFormatArgParity.

	// angela draft root (cmd/angela.go)
	AngelaDraftFlagAll             string
	AngelaDraftFlagVerbose         string
	AngelaDraftFlagFormat          string
	AngelaDraftFlagFailOn          string
	AngelaDraftFlagStrict          string
	AngelaDraftFlagSeverity        string
	AngelaDraftFlagDiffOnly        string
	AngelaDraftFlagResetState      string
	AngelaDraftFlagPersonasMode    string
	AngelaDraftFlagManualPersonas  string
	AngelaDraftFlagPersona         string
	AngelaDraftFlagInteractive     string
	AngelaDraftFlagAutofix         string
	AngelaDraftFlagDryRun          string
	AngelaDraftFlagSynthesizers    string
	AngelaDraftFlagNoSynthesizers  string

	// angela polish (cmd/angela_polish.go)
	AngelaPolishFlagDryRun             string
	AngelaPolishFlagYes                string
	AngelaPolishFlagFor                string
	AngelaPolishFlagAuto               string
	AngelaPolishFlagIncremental        string
	AngelaPolishFlagFull               string
	AngelaPolishFlagHalluStrict        string
	AngelaPolishFlagForce              string
	AngelaPolishFlagInteractive        string
	AngelaPolishFlagSynthesizers       string
	AngelaPolishFlagNoSynthesizers     string
	AngelaPolishFlagSynthDryRun        string
	AngelaPolishFlagSynthesize         string
	AngelaPolishFlagSetStatus          string
	AngelaPolishFlagPersona            string
	AngelaPolishFlagArbitrateRule      string
	AngelaPolishFlagVerbose            string

	// angela polish restore (cmd/angela_polish_restore.go)
	AngelaPolishRestoreFlagTimestamp string
	AngelaPolishRestoreFlagList      string

	// angela review (cmd/angela_review.go)
	AngelaReviewFlagQuiet                 string
	AngelaReviewFlagVerbose               string
	AngelaReviewFlagFor                   string
	AngelaReviewFlagFilter                string
	AngelaReviewFlagAll                   string
	AngelaReviewFlagDiffOnly              string
	AngelaReviewFlagInteractive           string
	AngelaReviewFlagSynthesizers          string
	AngelaReviewFlagNoSynthesizers        string
	AngelaReviewFlagPersona               string
	AngelaReviewFlagNoPersonas            string
	AngelaReviewFlagUseConfiguredPersonas string
	AngelaReviewFlagPreview               string
	AngelaReviewFlagFormat                string

	// angela review ignore / log (cmd/angela_review_ignore.go + log.go)
	AngelaReviewIgnoreFlagReason string
	AngelaReviewLogFlagFormat    string

	// decision (cmd/decision.go)
	DecisionFlagExplain     string
	DecisionFlagCalibration string

	// delete (cmd/delete.go)
	DeleteFlagForce string

	// doctor (cmd/doctor.go)
	DoctorFlagFix          string
	DoctorFlagQuiet        string
	DoctorFlagConfig       string
	DoctorFlagRebuildStore string
	DoctorFlagPrune        string
	DoctorFlagDryRun       string

	// init (cmd/init.go)
	InitFlagNoDemo string

	// list (cmd/list.go)
	ListFlagType  string
	ListFlagQuiet string

	// release (cmd/release.go)
	ReleaseFlagFrom    string
	ReleaseFlagTo      string
	ReleaseFlagVersion string
	ReleaseFlagQuiet   string

	// show (cmd/show.go)
	ShowFlagType     string
	ShowFlagAfter    string
	ShowFlagAll      string
	ShowFlagQuiet    string
	ShowFlagFeature  string
	ShowFlagDecision string
	ShowFlagBugfix   string
	ShowFlagRefactor string
	ShowFlagNote     string
}
