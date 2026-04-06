# Changelog

All notable changes to Lore are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — Branch Awareness & Amend Workflow

### Added

- **Branch Awareness** — `Branch` and `Scope` fields propagated through the full
  pipeline: `CommitInfo` → `GenerateInput` → `TemplateContext` → `DocMeta` → Store.
  Documents now record which branch and scope they were captured on. 22 files modified.
  (`internal/domain/types.go`, `internal/git/adapter.go`, `internal/workflow/`)

- **Amend Workflow Improvements** — Question 0 ("Document this change? [Y/n]")
  before amend flow, [U]pdate/[C]reate/[S]kip choice when existing doc found,
  pre-fill from existing document (Type, Title, Why section). Configurable via
  `hooks.amend_prompt` in `.lorerc`. (`internal/workflow/reactive.go`, `proactive.go`)

- **Notification Branch Impact** — OS dialogs (macOS AppleScript, Linux zenity/kdialog,
  Windows WPF) now display branch and scope context. Zero API change to `NotifyNonTTY`
  via `I18nLabels` callback injection. (`internal/notify/dialog*.go`)

- **Preflight Check** — Validates `.lore/` exists, `docs/` is writable, and template
  engine initializes BEFORE asking questions. Saves to pending on failure.
  (`internal/workflow/preflight.go`)

- **Demo Branch Detection** — `demoBranch()` uses `CurrentBranch()` → `DefaultBranch()`
  (reads `origin/HEAD`) → fallback "main". No longer hardcodes "main".
  (`cmd/demo.go`)

- **Config fields** — `hooks.amend_prompt` (bool), `notification.mode/disabled_envs/amend`,
  `decision.*` thresholds/always_ask/always_skip/learning. (`internal/config/defaults.go`)

### Fixed

- **`CurrentBranch()` error swallowing** (HIGH) — Was returning `"", nil` for all
  errors. Now propagates real git errors, returns `"", nil` only for detached HEAD.
  (`internal/git/adapter.go`)

- **`ValidateMeta` hardcoded type list** (HIGH) — Replaced with dynamic
  `DocTypeNames()` to prevent desync when new doc types are added.
  (`internal/storage/frontmatter.go`, `internal/domain/types.go`)

- **Summary detection via filename prefix** (MEDIUM) — Replaced
  `strings.HasPrefix(f, "summary-")` with `signals.ScopeTypes[f] == "summary"`.
  (`internal/angela/corpus_signals.go`)

### Changed

- **Test coverage** — 74.2% → 82.0% global. 4 packages at 100%, 16 packages at 85%+.
  Added ~120 tests across 25 packages. Key technique: `httptest` + `redirectTransport`
  for upgrade/ (43.5% → 82.6%).

---

## [Pre-MVP Hardening]

### Summary

Full paranoid code review (6 sessions, 12 auditors, ~150 findings) followed by
comprehensive remediation across all 17 internal packages. Zero test regressions;
all 23 packages build and pass.

---

### Security

- **HTTP client hardened** — `SafeHTTPClient()` now sets `Transport` with
  `MaxIdleConns=10`, `MaxConnsPerHost=5`, `IdleConnTimeout=90s`, and a
  120 s global timeout. Prevents connection pool exhaustion under load.
  (`internal/ai/options.go`)

- **HTTPS enforced for remote endpoints** — `ValidateEndpoint()` rejects
  `http://` for non-localhost hosts. (`internal/ai/options.go`)

- **API key scrubbing in error responses** — New `scrubSensitive()` helper
  redacts authorization tokens, API keys, and secrets from AI provider error
  messages before they reach stderr. (`internal/ai/scrub.go`)

- **Permissions auto-fixed on .lorerc.local** — `enforceSecurePerms()` replaces
  the old warning-only check; it actively `chmod 600` files with world-readable
  bits. (`internal/config/config.go`)

- **Reserved filenames rejected** — `validateFilename()` now blocks `README.md`,
  `index.md`, and `.index.lock` to prevent index corruption or read confusion.
  (`internal/storage/reader.go`)

- **Case-insensitive symlink check on macOS** — `validateResolvedPath()` falls
  back to case-insensitive comparison on `darwin`, closing a path-escape vector on
  HFS+/APFS. (`internal/storage/reader.go`)

- **Document size limit** — `Unmarshal()` rejects files > 10 MB, preventing
  denial-of-service via oversized Markdown. (`internal/storage/frontmatter.go`)

- **Config file size limit** — Both `.lorerc` and `.lorerc.local` are rejected
  if > 1 MB, preventing YAML bomb attacks. (`internal/config/config.go`,
  `internal/config/validate.go`)

- **Prompt injection defense documented** — `sanitizePromptContent()` now carries
  a full defense-in-depth rationale comment. (`internal/angela/review.go`)

- **Git hook integrity check** — `HookExists()` validates both `LORE-START` and
  `LORE-END` markers; returns an explicit error for corrupted (single-marker)
  hooks. (`internal/git/hook.go`)

### Fixed

- **Panic on empty AI response** — `anthropic.go`, `openai.go`, and `ollama.go`
  now check for nil/empty `Content`, `Choices`, and `Response` fields before
  indexing, preventing runtime panics on malformed API responses.

- **Goroutine leak in template rendering** — `Render()` now allocates its
  `bytes.Buffer` inside the goroutine, so a timed-out template cannot share a
  buffer with the caller. (`internal/template/engine.go`)

- **Database rollback errors no longer silently swallowed** — All `tx.Rollback()`
  calls in `store.go` and `rebuild.go` now chain the rollback error into the
  returned error.

- **Nil store dereference in hook** — `hook_run.go` now checks
  `storePtr != nil && *storePtr != nil` before passing to the decision engine.

- **Index corruption on concurrent hooks** — `RegenerateIndex()` acquires an
  exclusive `.index.lock` file; concurrent callers skip silently instead of
  racing. (`internal/storage/index.go`)

- **Partial index guard** — If *all* documents fail to parse, the index is no
  longer written (avoids replacing a good index with an empty one).

- **Empty frontmatter rejected** — `Unmarshal()` returns an error for documents
  with `---\n---\n` (no YAML body). (`internal/storage/frontmatter.go`)

- **Empty git ref rejected** — `validateRef("")` now returns an error instead of
  silently accepting. (`internal/git/adapter.go`)

- **CommitRange returns `[]string{}` not `nil`** — Eliminates downstream nil-vs-
  empty ambiguity. (`internal/git/adapter.go`)

- **SavePending errors surfaced** — `proactive.go` now logs a warning to stderr
  when pending-answer save fails, instead of silently discarding.

- **Unsupported language fallback logged** — `i18n.Init("ja")` now prints a
  warning instead of silently falling back to English.

- **Defensive nil guard in `T()`** — Returns `catalogEN` if the atomic value is
  nil. (`internal/i18n/i18n.go`)

- **Subject length capped** — `ParseConventionalCommit` truncates subjects to
  200 characters, preventing unbounded metadata. (`internal/git/commit_info.go`)

- **Skipped git log entries reported** — `LogAll()` now counts and warns about
  unparseable entries. (`internal/git/adapter.go`)

- **Temp file cleanup logged** — `AtomicWrite` error paths now log `os.Remove`
  failures to stderr instead of discarding. (`internal/fileutil/atomic.go`)

- **Response body close errors logged** — All three AI providers now log
  `resp.Body.Close()` failures. (`internal/ai/anthropic.go`, `openai.go`,
  `ollama.go`)

- **Credential delete errors surfaced** — Darwin keychain `Set()` now logs
  non-`ErrNotFound` errors from the pre-delete step.
  (`internal/credential/keychain_darwin.go`)

### Performance

- **`HeadCommit()` batch method** — Single `git log -1 HEAD --format=%H%n%an%n%aI%n%P%n%B`
  replaces separate `HeadRef()` + `Log()` + `IsMergeCommit()` calls, cutting
  hook-path latency by ~10–15 ms. (`internal/git/adapter.go`)

- **`ScopeStats()` SQL aggregation** — Replaces loading all scope commits into
  memory with a single `SELECT COUNT(CASE WHEN …)` query, reducing the LKS
  history signal from O(n) record transfers to O(1).
  (`internal/store/commits.go`, `internal/workflow/decision/signals.go`)

- **Compiled regex for file patterns** — `testPatternRe` and `highValuePatternRe`
  replace nested-loop string matching in `FileValueSignal`.
  (`internal/workflow/decision/file_signals.go`)

- **Diff truncation without rejoin** — `ScanDiffContent` now uses a `nthIndex`
  helper to cap at 1000 lines without `SplitN` + `Join`.
  (`internal/workflow/decision/content_signals.go`)

- **Slice pre-allocation** — `signals` (cap 5), `files` (cap 16), `seen` (cap 16),
  `results` (cap 64) across `engine.go`, `reactive.go`, `commits.go`,
  `doc_index.go`.

- **String prefix micro-optimization** — `ExtractFilesFromDiff` uses direct byte
  comparison instead of `strings.HasPrefix` + `TrimPrefix`.

- **Composite database index** — New migration v2 adds
  `idx_commits_scope_date(scope, date DESC)` for the hot `CommitsByScope` query.

- **`PRAGMA optimize` on close** — `SQLiteStore.Close()` runs `PRAGMA optimize`
  before closing, improving long-lived database performance.

- **DB connection pool configured** — `SetMaxOpenConns(5)`, `SetMaxIdleConns(2)`
  prevent unbounded connection growth.

- **Default query timeout** — `queryDocsCtx` applies a 30 s timeout when the
  caller provides `context.Background()`.

- **Query result limits** — `CommitsByScope` now includes `LIMIT 10000`.

- **CRLF normalization skip** — `Unmarshal` skips `ReplaceAll` when no `\r\n` is
  present.

### Architecture

- **Service layer** — New `internal/service/` package with `PolishDocument()`,
  `ReviewCorpus()`, and `EngineConfigFromApp()`. Business logic extracted from
  `cmd/angela_polish.go`, `cmd/angela_review.go`, and `cmd/decision.go`, making
  orchestration testable without Cobra.

- **`handleReactiveWithOpts` decomposed** — Extracted `resolveHeadCommit()`,
  `buildSignalContext()`, and `handleDetectionResult()` helpers (~125 lines →
  4 focused functions). (`internal/workflow/reactive.go`)

- **Typed constants** — `DetectionAction`, `QuestionMode` (in `detection.go`),
  `DocStatus`, `Decision` (in `domain/types.go`) as type aliases for backward-
  compatible compile-time safety.

- **`DiffOptions` struct** — Replaces `dryRun bool, yesAll bool` parameters in
  `InteractiveDiff`, eliminating boolean blindness.
  (`internal/angela/diff.go`)

- **Path helper** — `domain.DocsPath(workDir)` centralizes
  `filepath.Join(workDir, ".lore", "docs")`. (`internal/domain/paths.go`)

- **Flag resolution helper** — `resolveDocTypeFlags()` deduplicates type-filter
  logic shared between `show` and `list`. (`cmd/flags.go`)

- **`CommitInfo` enriched** — `IsMerge` and `ParentCount` fields enable merge
  detection from a single `git log` call.

- **`ScopeStatsResult`** — New domain type for aggregated scope statistics,
  enabling SQL-level computation. (`internal/domain/types.go`)

### Code Quality

- **Consistent `_, _ = fmt.Fprintf`** — ~110 bare `fmt.Fprintf(streams.Err, …)`
  calls standardized across all `cmd/` files.

- **Dead comment removed** — `// newNoteCmd removed…` in `root.go`.

- **Unknown-field warning improved** — Config typos now suggest
  `lore doctor` for validation.

- **Design-choice comments** — `IsTerminal()`, `sanitizePromptContent()`,
  `ccTypeMap`, `IndexErr` (deprecated), `DeleteDoc` index behavior, and keychain
  label safety all carry explicit rationale comments.

### Tests

- **New tests added:**
  - `TestValidateFilename` — 10 subtests (empty, traversal, reserved, separator)
  - `TestUnmarshal_EmptyFrontmatter`
  - `TestWriteDoc_DoesNotRegenerateIndex`
  - `TestHookExists_CorruptedSingleMarker`
  - `TestHeadCommit`, `TestHeadCommit_NoCommits`, `TestHeadCommit_MergeCommit`
  - `TestScopeStats`
  - `TestCorpusStore_ListDocs_CombinedFilters` (5 filter combos)
  - `TestInstallHook_MultipleTypes`
  - `TestAtomicWrite_ReadOnlyDir`
  - `parseHeadCommitOutput` unit tests (valid, merge, bad format)
  - Signal name validation in `TestEvaluate_SignalCount`

- **Anti-patterns fixed:**
  - Path-traversal loop converted to `t.Run` subtests (`delete_test.go`)
  - Benchmark uses fixed data instead of changing per iteration
    (`hook_overhead_bench_test.go`)
  - Levenshtein test uses `fmt.Sprintf` for subtest names (`validate_test.go`)
  - Schema version assertion updated for migration v2 (`store_test.go`)
  - Delete-README assertion accepts "reserved" (`delete_test.go`)

- **`t.Parallel()` enabled** — `engine_test.go` (5 tests), `provider_test.go`
  (14 tests).

- **Integration test `testing.Short()` guards** — Already present in all
  integration tests (confirmed, no changes needed).
