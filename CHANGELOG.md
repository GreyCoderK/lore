# Changelog

All notable changes to Lore are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] — Angela Enhancement Sprint, Branch Awareness & Amend Workflow

### Added

- **Angela Quality Score** — 0-100 scoring with 11 criteria (Why section, diagrams,
  tables, code blocks, structure, front matter, references, density, style) and
  letter grades A-F. Before/after comparison with colored delta on `angela polish`.
  (`internal/angela/score.go`)

- **Angela `--auto` mode** — Classifies each diff hunk (pure addition, pure deletion,
  cosmetic, major deletion, modification) and auto-decides: accept additions,
  reject deletions, ask only for modifications. Summary at end.
  (`internal/angela/diff.go`, `cmd/angela_polish.go`)

- **Angela `--for` audience rewrite** — `angela polish --for "CTO"` rewrites for a
  target audience with persona boost (+20 for matching personas). Interactive choice:
  new file (original unchanged) or overwrite original.
  (`cmd/angela_polish.go`, `internal/angela/persona.go`)

- **Angela `[b]oth` diff option** — Keep original lines AND add new lines. Only
  shown when hunk has both deletions and additions.
  (`internal/angela/diff.go`)

- **Angela Hunk Warnings** — Warns before destructive changes: net loss >15 lines,
  section heading deletions, code block removals, table row deletions.
  (`internal/angela/diff.go`)

- **Angela Post-Processing** — Local transforms after AI response: heading number
  restoration, code fence language detection (25+ languages), mermaid indent
  normalization. (`internal/angela/postprocess.go`)

- **Angela Token Stats** — Real-time display of tokens sent/received, model, speed
  (tok/s), cost after each API call. `UsageTracker` interface with `sync.Mutex` in
  all 3 providers. (`internal/domain/interfaces.go`, `internal/ai/*.go`)

- **Angela i18n UI** — ~30 new i18n keys for all runtime messages (preflight, token
  stats, timeout, cost, quality, warnings). Bilingual diff prompts accept both
  EN (y/n/b/q) and FR (o/n/l/q) input. (`internal/i18n/catalog_en.go`, `catalog_fr.go`)

- **Angela Hunk Location** — Each diff hunk shows `@@ line X (N lines) @@` header
  for document position context. (`internal/angela/diff.go`)

- **Angela Standalone Mode** — `--path` flag on `angela draft` and `angela review`
  enables analysis of any Markdown directory without `lore init`. `PlainCorpusStore`
  gracefully handles files with or without YAML front matter.
  (`internal/storage/plain_reader.go`, `cmd/angela.go`)

- **Angela CI Quality Gate** — GitHub Action composite (`action.yml`) and portable
  shell script (`scripts/angela-ci.sh`) for GitHub Actions, GitLab CI, Jenkins,
  Bitbucket. Supports `--path`, `--fail-on`, `--install`, `--quiet`.

- **VHS Cross-Check** — Detects orphan tapes (output GIF not referenced in docs),
  orphan GIF references (docs referencing non-existent tape output), and CLI command
  mismatches in `.tape` files. Integrated into `draft --all` output and review AI prompt.
  (`internal/angela/vhs_signals.go`)

- **Language Detection** — 24 programming languages including VHS tape syntax.
  Auto-tags bare code fences during `angela polish` post-processing.
  (`internal/angela/langdetect.go`)

- **Multi-Pass Polishing** — Documents exceeding single-pass token limits are
  automatically split into sections and polished sequentially.
  (`internal/angela/multipass.go`)

- **Audience Rewrite** — `angela polish --for <audience>` rewrites documents for a
  specific audience (CTO, new developer, commercial team). Saves as separate file.
  (`cmd/angela_polish.go`)

- **Preflight & Cost Estimation** — Token estimation, cost warnings, abort-if-too-large,
  timeout prediction before API calls. Supports known model cost maps.
  (`internal/angela/preflight.go`)

- **Progress Spinners** — Visual feedback with elapsed time on all long-running commands
  (review, polish, doctor, release, status, upgrade, check-update).
  (`internal/ui/progress.go`)

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

- **Angela `--filter` and `--all` for review** — `--filter <regex>` filters documents
  by filename, `--all` disables 25+25 sampling on large corpora. Both combine freely
  with `--for`, `--path`, `--quiet`. (`cmd/angela_review.go`, `internal/angela/review.go`)

- **Embedded Logo (brand package)** — Lore logo PNG embedded via `//go:embed` and
  cached to temp dir. Eliminates filesystem lookups for notification icons.
  (`internal/brand/brand.go`)

- **Notification Icons** — macOS (AppleScript), Linux (zenity `--window-icon`,
  kdialog `--icon`), and Windows (WPF `Icon`, NotifyIcon) now display the Lore
  logo in dialogs and toast notifications via the brand package.
  (`internal/notify/dialog_darwin.go`, `dialog_linux.go`, `dialog_windows.go`, `simple.go`)

- **Graceful Signal Handling** — `Ctrl+C` (SIGINT/SIGTERM) cancels the command
  context via `signal.NotifyContext`, giving `SavePending` a chance to persist
  partial answers before exit. (`cmd/root.go`)

- **Chocolatey Distribution** — Enabled in GoReleaser config + Chocolatey CLI
  installed in release workflow. (`/.goreleaser.yaml`, `.github/workflows/release.yml`)

- **Recursive PlainCorpusStore** — `--path` now scans subdirectories recursively
  (was flat-only). `ReadDoc` accepts relative paths with subdirectories.
  (`internal/storage/plain_reader.go`)

- **`angela draft --all --verbose`** — New `--verbose`/`-v` flag prints every
  suggestion inline instead of just a count. By default, the command already
  prints warning-severity suggestions inline (they are blockers), and displays
  a hint at the end inviting the user to re-run with `-v` for the full detail.
  Previously, the only output was `5 suggestions (2 warnings)` with no way to
  see what the suggestions actually were without re-running `draft` per-file.
  (`cmd/angela.go`, `internal/i18n/catalog_{en,fr}.go`)

- **Dual scoring profile: strict vs free-form** — `ScoreDocument` now picks
  a scoring profile based on document type. Strict types (`decision`,
  `feature`, `bugfix`, `refactor`) keep the original lore scoring with
  heavy weight on `## Why`, related refs, and `status`. Free-form types
  (notes, guides, tutorials, blog posts, concept pages, any unknown type)
  use a rebalanced profile that drops lore-specific criteria (Why, related,
  status) and redistributes points into structure, code, density, and
  paragraph quality. A well-written tutorial now reaches 95/100 (A);
  before it plateaued at F. (`internal/angela/score.go`)

- **Free-form type detection by exclusion** — `isFreeFormType` now
  whitelists the 4 strict lore types and treats everything else as
  free-form. This means arbitrary types from external sites
  (`blog-post`, `howto`, `explanation`, `concept`, `landing`, `faq`, …)
  are recognized automatically without a code change. Structure/
  completeness/persona/scoring behavior all branch on this predicate.
  (`internal/angela/draft.go`)

- **Permissive front matter parsing in standalone mode** — New
  `storage.UnmarshalPermissive` skips `ValidateMeta`, so external docs
  with partial front matter (e.g. just `type` and `date`, or arbitrary
  custom types) are preserved instead of silently downgraded to synthetic
  metadata. `PlainCorpusStore` now fills gaps (status="published", tags
  inferred from filename) only when a field is missing, not when the
  whole parse fails. (`internal/storage/frontmatter.go`,
  `internal/storage/plain_reader.go`, `cmd/angela.go`)

- **Translation pair detection** — `isTranslationPair` recognizes 13
  language codes (fr, en, es, de, it, pt, zh, ja, ko, ru, ar, nl, pl)
  and skips duplicate/cross-reference/body-mention checks between
  translation pairs. No more "Possible duplicate: foo.fr.md" warnings
  on bilingual mkdocs sites. (`internal/angela/coherence.go`)

### Deprecated

- **`angela.mode` config field** — Had no runtime effect: the mode is selected
  by the sub-command (`lore angela draft | polish | review`). The field is
  retained in the struct for backward compatibility so existing `.lorerc`
  files parse without error, but:
  - `doctor --config` emits a deprecation warning when the field is present
  - the field no longer has a default value (was `"draft"`)
  - it is no longer shown in the "Active values" table
  - the `init` template and documentation no longer mention it

  The field will be removed entirely in v2.
  (`internal/config/defaults.go`, `internal/config/validate.go`, `cmd/init.go`)

- **`hooks.post_commit: false` was silently ignored** (HIGH, now also fixed) —
  The hook runner `_hook-post-commit` in `cmd/hook_run.go` never read the
  flag. Setting `hooks.post_commit: false` in `.lorerc` did not prevent the
  question flow from running on subsequent commits. The runner now returns
  early when the flag is false; the installed `.git/hooks/post-commit`
  script becomes a no-op instead of a full dispatch.
  (`cmd/hook_run.go`)

### Fixed

- **macOS notifications have no icon** (LOW) — `display notification` (osascript)
  does not support custom icons on macOS. Lore now auto-installs `terminal-notifier`
  via Homebrew when available (`brew install --quiet terminal-notifier`) to enable
  Lore logo in toast notifications. Manual install: `brew install terminal-notifier`.
  (`internal/notify/simple.go`)

- **Ctrl+C before first question loses commit** (HIGH) — `RegisterInterruptState`
  is now called BEFORE `PreflightCheck` in `HandleProactive`, so the signal handler
  can save pending even if the user interrupts before any question is asked.
  (`internal/workflow/proactive.go`)

- **Ctrl+C loses partial answers** (HIGH) — Pressing Ctrl+C during any question
  (type selector, What, Why, amend Question 0, [U]/[C]/[S]) now saves partial
  answers to `.lore/pending/` before exiting. Uses `RegisterInterruptState` +
  `FlushOnInterrupt` called directly from the signal handler — works even when
  stdin read is blocked. Second Ctrl+C force-exits with code 130.
  (`internal/workflow/interrupt.go`, `internal/workflow/questions.go`,
  `internal/workflow/reactive.go`, `internal/workflow/proactive.go`, `cmd/root.go`)

- **`selectType` Ctrl+C silently continues** (HIGH) — Was returning `(defaultType, nil)`,
  causing the flow to proceed as if the user chose the default type. Now returns
  `ErrInterrupted` which propagates through the entire question chain.
  (`internal/workflow/type_select.go`, `internal/workflow/questions.go`)

- **`readAmendAnswer` ignores context** (MEDIUM) — Was blocking indefinitely on
  stdin read without respecting context cancellation. Now accepts `ctx` parameter
  and uses goroutine + select for interruptible reads. Detects byte `0x03` (Ctrl+C
  in raw mode) and returns `ErrInterrupted`.
  (`internal/workflow/reactive.go`)

- **Amend pre-fill missing QuestionMode** (MEDIUM) — Amend workflow now sets
  `QuestionMode: "reduced"` and `PrefilledWhyConfidence: 0.9` when pre-filling
  from existing document. (`internal/workflow/reactive.go`)

- **`angela-ci.sh --fail-on warning` never detected warnings** (HIGH) — The
  grep pattern `^  warning` (2 spaces) matched neither the single-file draft
  output (also 2 spaces but different column) nor the `draft --all` inline
  format (9 spaces indent). External CI pipelines using `--fail-on warning`
  silently passed on every run regardless of the corpus state. Now uses
  `^[[:space:]]+(warning|gap|obsolete|style)[[:space:]]` to match any
  indentation, with a dedicated test suite (`scripts/angela-ci_test.sh`)
  to prevent regression. (`scripts/angela-ci.sh`)

- **`action.yml` severity counting broken and missing review mode** (HIGH) —
  The GitHub Action composite had two bugs: (a) same broken grep pattern as
  `angela-ci.sh`, and (b) `mode: review` completely ignored `fail_on` — the
  output was captured with `|| true` and no exit logic was applied. Fixed
  both: unified severity counting for draft and review modes, explicit
  `fail_on: none` handling, and invalid `fail_on` values now exit 2 with
  an `::error::` annotation. (`action.yml`)

- **False "## What / ## Why missing" warnings on free-form docs** (HIGH) —
  The structure check emitted warnings for any document type, including
  notes, tutorials, guides, blog posts, and external docs with arbitrary
  types. A typical mkdocs site of 68 docs produced ~136 false warnings,
  making CI gates unusable. Fixed by branching on `isFreeFormType(meta.Type)`
  in `checkStructure` — What/Why/Alternatives/Impact requirements now only
  apply to the 4 strict lore types. (`internal/angela/draft.go`)

- **False "Possible duplicate" warnings on FR/EN translation pairs** (MEDIUM) —
  Bilingual mkdocs sites where `foo.md` and `foo.fr.md` shared inferred tags
  were flagged as duplicates. Added `isTranslationPair` detection that
  recognizes 13 language codes (including `fr/en/es/de/it/pt/zh/ja/ko/ru/
  ar/nl/pl`) and skips dupe/cross-ref/body-mention checks between pairs.
  (`internal/angela/coherence.go`)

- **Body-too-short was warning on narrative docs** (MEDIUM) — Landing pages,
  section index stubs, and short tutorial intros are legitimately under 50
  characters. The "body too short" check now emits `info` (not `warning`)
  for free-form types, so `--fail-on warning` doesn't fail external CI
  pipelines on legitimately short pages. Stays `warning` for strict lore
  types where a short body signals a missing explanation.
  (`internal/angela/draft.go`)

- **VHS orphan-check severity too strict** (MEDIUM) — The VHS tape/doc
  cross-check emitted `warning` for orphan tapes, orphan GIFs, and CLI
  command mismatches. External users with an unrelated `assets/vhs/`
  directory saw their CI fail. Downgraded to `info` — findings are still
  visible in `-v` mode but never block `--fail-on warning`. (`cmd/angela.go`)

- **Quality score plateaued at F for free-form docs** (MEDIUM) — The original
  `ScoreDocument` allocated 15 pts to `## Why`, 5 pts to `related:`, and
  4 pts to `status:` — all lore-specific. A perfect tutorial could never
  exceed ~65/100. New `scoreFreeForm` profile redistributes these 24 pts
  into density (20), structure (20), code (15), and paragraph quality (9).
  A well-written tutorial now reaches 95/100 (A) legitimately.
  (`internal/angela/score.go`)

- **`PlainCorpusStore` silently downgraded partial front matter** (MEDIUM) —
  `Unmarshal` called `ValidateMeta` which required `type + date + status`.
  An external doc with `type: decision` + `date` but no `status` was
  rejected and fell back to `buildSyntheticMeta` → `type: "note"` → lost
  all strict checks the author wanted. New `UnmarshalPermissive` is used
  by standalone mode; missing fields are backfilled with defaults instead
  of rejecting the whole parse.
  (`internal/storage/frontmatter.go`, `internal/storage/plain_reader.go`)

- **Non-deterministic language detection on tied votes** (MEDIUM) —
  `DetectLanguageMultiLine` used a map iteration to pick the winner on
  tied vote counts, causing the same input to return different languages
  on Linux/darwin vs. Windows runners. Now ties break by first-vote line
  index (earliest wins), making the output fully deterministic.
  (`internal/angela/langdetect.go`)

- **`angela.max_tokens` config ignored** (HIGH) — User-configured `angela.max_tokens: 10000`
  in `.lorerc` was overridden by computed value (2756). Now config always wins via
  `ResolveMaxTokens(..., configMaxTokens ...int)` variadic parameter.
  (`internal/angela/tokens.go`)

- **Angela adds English headings to French docs** (HIGH) — AI inserted `## Why`,
  `## Impact` in French documents. Fixed with explicit LANGUAGE RULE in prompt +
  translation table. (`internal/angela/polish.go`)

- **Angela deletes content silently** (HIGH) — Sections 4-8 deleted when max_tokens
  too low. Fixed with preflight abort (input > max_output), truncation guard (rejects
  diff if output = max_tokens), and PRESERVE CONTENT rules in prompt.
  (`internal/angela/preflight.go`, `cmd/angela_polish.go`)

- **DiffBoth deduplication dropped lines** (MEDIUM) — `containsLine()` failed on
  identical lines (empty lines). Replaced with `Lines[].Kind == '+'` approach using
  LCS edit ops. (`internal/angela/diff.go`)

- **Spinner "1m60s" at minute boundary** (LOW) — Float rounding produced "1m60s".
  Fixed with integer modulo `int(totalSec) % 60`. (`cmd/angela_polish.go`)

- **`sanitizeAudience` strips accents** (LOW) — Changed from `r >= 'a' && r <= 'z'`
  to `unicode.IsLetter()` for French filenames. (`cmd/angela_polish.go`)

- **Ollama ignores max_tokens** (MEDIUM) — Added `NumPredict` field mapped to
  `num_predict` in Ollama API. (`internal/ai/ollama.go`)

- **Race on `Spinner.warned`** (MEDIUM) — Fixed with `atomic.Bool`.
  (`internal/ui/progress.go`)

- **Race on `lastUsage` in 3 providers** (MEDIUM) — Fixed with `sync.Mutex`.
  (`internal/ai/anthropic.go`, `openai.go`, `ollama.go`)

- **`SplitSections` splits on `##` inside code blocks** (LOW) — Added code fence
  state tracking. (`internal/angela/multipass.go`)

- **`countCodeFences` wrong formula** (LOW) — Rewritten with proper open/close
  state machine. (`internal/angela/score.go`)

- **No context cancellation in multi-pass** (LOW) — Added `ctx.Err()` check between
  sections. (`internal/angela/multipass.go`)

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
