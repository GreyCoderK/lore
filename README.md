<p align="center">
  <img src="assets/logo.svg" alt="Lore" width="180">
</p>

<h1 align="center">Lore</h1>

<p align="center">
  <strong>Your code knows what. Lore knows why.</strong><br>
  <em>L'or de vos decisions techniques.</em>
</p>

<p align="center">
  <a href="https://github.com/GreyCoderK/lore/actions/workflows/ci.yml"><img src="https://github.com/GreyCoderK/lore/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/GreyCoderK/lore"><img src="https://img.shields.io/github/go-mod/go-version/GreyCoderK/lore" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-AGPL--3.0-blue.svg" alt="License: AGPL-3.0"></a>
  <a href="https://github.com/GreyCoderK/lore/releases"><img src="https://img.shields.io/github/v/release/GreyCoderK/lore" alt="GitHub Release"></a>
  <a href="https://app.codecov.io/gh/GreyCoderK/lore"><img src="https://img.shields.io/codecov/c/github/GreyCoderK/lore?label=coverage&color=d4a" alt="Coverage"></a>
  <a href="https://github.com/sponsors/GreyCoderK"><img src="https://img.shields.io/badge/Sponsor-%E2%9D%A4-ea4aaa?logo=github-sponsors" alt="Sponsor"></a>
  <a href="https://github.com/GreyCoderK/lore"><img src="https://img.shields.io/badge/lore-documented-d4a" alt="lore: documented"></a>
</p>

<p align="center">
  <img src="docs/assets/demo.gif" alt="Lore Demo" width="800">
</p>
<!-- Generate: vhs assets/demo.tape -->

---

## The Problem

You're 50 commits in. Six months later, someone asks: *"Why did we build it this way?"*

Git blame shows **who** changed **what** and **when**. But not **why**. The reasoning is gone — buried in a Slack thread, a PR comment, or the memory of a developer who left three months ago.

Every codebase has an invisible layer of decisions that code alone can't convey. And every day that passes, more of that knowledge evaporates.

## The Solution

Three questions. Ninety seconds. Done.

```
$ git commit -m "feat: add JWT auth middleware"
  [1/3] Type [feature]:
  [2/3] What [add JWT auth middleware]:
  [3/3] Why? Because stateless auth scales better than sessions
  Captured  feature-add-jwt-auth-middleware-2026-03-16.md
```

Lore hooks into your Git workflow and asks **3 questions** after every commit — Type, What, Why. The answers become a Markdown file living in your repo, searchable, versionable, portable. No wiki. No SaaS. No friction.

## Installation

```bash
# Homebrew (macOS / Linux)
brew install GreyCoderK/tap/lore

# Snap (Linux)
sudo snap install lore --classic

# Chocolatey (Windows) — package name is lore-cli (bare "lore" was taken)
choco install lore-cli

# Go (any platform)
go install github.com/greycoderk/lore@latest

# Pre-built binaries (macOS / Linux)
curl -sSL https://github.com/GreyCoderK/lore/releases/latest/download/install.sh | sh
```

Or download from [GitHub Releases](https://github.com/GreyCoderK/lore/releases) — binaries for macOS, Linux, and Windows.

### Optional (macOS only) — Notification icons

For Lore logo in notifications on macOS, install `terminal-notifier`:

```bash
brew install terminal-notifier
```

Without it, notifications fall back to `osascript display notification` which does not support custom icons (macOS limitation). Lore attempts to auto-install via Homebrew if available.

## Quickstart (5 minutes)

```bash
# 1. Initialize Lore in your project
lore init
# Creates .lore/ directory and installs the post-commit hook

# 2. Make a commit — Lore asks 3 questions automatically
git add . && git commit -m "Add rate limiting"
# → Type? feature
# → What? Add rate limiting (pre-filled from commit)
# → Why? API was getting hammered, 10K req/min from one client

# 3. See your captured decision
lore show
# → Displays the Markdown document with the full context

# 4. Check your documentation health
lore status
# → Shows coverage, pending commits, corpus stats

# Bonus: document a past commit retroactively
lore new --commit abc1234
```

### Interactive Mode

Use `lore new` for standalone documentation with the interactive type selector:

<p align="center">
  <img src="docs/assets/vhs/interactive.gif" alt="Lore Interactive" width="800">
</p>
<!-- Generate: vhs assets/vhs/interactive.tape -->

## How Lore Compares

| | **Lore** | Swimm | Confluence | GitBook | Nothing |
|---|---|---|---|---|---|
| **When** | Commit-time | After the fact | After the fact | After the fact | Never |
| **Where** | Local (`.lore/`) | SaaS | SaaS | SaaS | — |
| **Friction** | 90 seconds | 30 minutes | 30 minutes | 15 minutes | 0 |
| **AI** | Angela (opt-in) | Generic | Generic | Generic | — |
| **Lock-in** | Markdown | Proprietary | Proprietary | Mixed | — |
| **Price** | Free | $28/seat | $5.75/user | $8/user | Free |

Lore is also complementary to ADRs — it captures the daily *why* that feeds into bigger architectural decisions.

## Commands

| Command | Description |
|---------|-------------|
| `lore init` | Initialize Lore in the current repository |
| `lore new` | Create documentation on demand |
| `lore new --commit <hash>` | Document a past commit retroactively |
| `lore show [query]` | Search and display documents |
| `lore list` | List all documents in the corpus |
| `lore status` | Repository health dashboard |
| `lore status --badge` | Generate shields.io coverage badge |
| `lore delete <file>` | Delete a document with confirmation |
| `lore pending` | List undocumented commits |
| `lore pending resolve` | Resume interrupted documentation |
| `lore pending skip <hash>` | Skip a pending commit |
| `lore doctor` | Diagnose corpus inconsistencies |
| `lore doctor --fix` | Auto-repair fixable issues |
| `lore doctor --config` | Validate `.lorerc` configuration |
| `lore release [tag]` | Generate release notes from corpus |
| `lore demo` | Interactive demo of the workflow |
| `lore hook install` | Install the post-commit hook |
| `lore config` | Show current configuration |
| `lore angela draft` | Zero-API structural analysis |
| `lore angela draft --path ./docs` | Standalone mode — any Markdown directory, no `lore init` |
| `lore angela polish` | AI-assisted rewrite with diff review |
| `lore angela polish --for "CTO"` | Audience-adapted rewrite |
| `lore angela polish --auto` | Auto-accept additions, reject deletions |
| `lore angela review` | Corpus-wide coherence analysis |
| `lore angela review --filter "guides/.*"` | Review filtered subset |
| `lore angela review --all` | Review all docs (no sampling) |
| `lore decision` | Decision engine status and calibration |
| `lore completion <shell>` | Generate shell completions (bash/zsh/fish) |

### Angela in CI (Standalone)

Angela works as a **documentation quality gate** in any CI pipeline — no `lore init` required:

```yaml
# GitHub Actions — 3 lines
- uses: GreyCoderK/lore@v1
  with:
    path: ./docs
```

```bash
# Any CI — portable script (draft: offline, review: AI)
./scripts/angela-ci.sh --path docs --fail-on warning --install
./scripts/angela-ci.sh --mode review --path docs --all --install
```

Works on **any Markdown directory** — with or without YAML front matter. See the [Angela in CI guide](https://greycoderk.github.io/lore/guides/angela-ci/) for details.

## How It Works

### The Documentation Flow

1. **Type** — What kind of change? (feature, bugfix, decision, refactor, note)
2. **What** — Pre-filled from your commit message. Press Enter to confirm.
3. **Why** — The one question that matters. Why this approach?

If all 3 answers come in under 3 seconds, Lore enters **express mode** and skips optional questions.

### Contextual Detection

- **Merge commits** — Skipped automatically
- **Rebase** — Deferred to pending
- **Cherry-pick with doc** — Skipped
- **Amend** — Updates existing document
- **Non-TTY** (IDE, CI) — Deferred with OS notification (VS Code, dialog)
- **Ctrl+C** — Partial answers saved to pending (at any question level, including type selector and amend prompts)

### Document Format

```markdown
---
type: decision
date: 2026-03-16
status: published
commit: abc1234567890abcdef
generated_by: hook
---
# JWT Auth Middleware

## Why
Stateless authentication scales better than server-side sessions...

## Alternatives Considered
Session-based auth with Redis...

## Impact
Users can now authenticate without server-side state...
```

## Configuration

| File | Purpose | Git |
|------|---------|-----|
| `.lorerc` | Shared project config | Committed |
| `.lorerc.local` | Personal overrides (API keys) | Gitignored |
| `LORE_*` env vars | CI/automation overrides | — |

```yaml
# .lorerc
language: "en"           # "en" or "fr" — bilingual UI
ai:
  provider: ""            # "anthropic", "openai", "ollama", or "" (zero-API)
  model: ""               # e.g. "claude-sonnet-4-20250514", "gpt-4o"
  endpoint: ""            # custom endpoint (Groq, Together, Ollama, etc.)
  timeout: 60s
angela:
  max_tokens: 8192        # override auto-computed token limit
hooks:
  post_commit: true
  amend_prompt: true      # ask "Document this change?" on amend
```

## Architecture (for contributors)

```
cmd/           → Cobra commands (CLI entry points)
internal/
  domain/      → Interfaces, types, DTOs (no dependencies)
  config/      → Configuration cascade (.lorerc → .lorerc.local → env)
  git/         → Git adapter (hooks, log, diff)
  storage/     → Document storage, front matter, index, doctor
  workflow/    → Reactive (hook) and proactive (lore new) flows
  generator/   → Document generation pipeline
  angela/      → AI-assisted documentation (scoring, polish, review, personas)
  ai/          → AI provider implementations (Anthropic, OpenAI, Ollama)
  brand/       → Embedded assets (logo PNG via //go:embed)
  i18n/        → Bilingual message catalogs (EN/FR, 700+ strings)
  ui/          → Terminal UI (colors, progress spinners, lists)
.lore/
  docs/        → Documentation corpus (Markdown)
  pending/     → Interrupted/deferred commits
  store.db     → LKS index (SQLite, reconstructible)
```

**Principles:** Markdown is source of truth. Zero implicit network calls. Atomic writes. stderr for humans, stdout for machines.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

For security vulnerabilities, see [SECURITY.md](SECURITY.md).

## Community

- [GitHub Issues](https://github.com/GreyCoderK/lore/issues) — Bugs & feature requests
- [GitHub Discussions](https://github.com/GreyCoderK/lore/discussions) — Q&A, ideas, show & tell
- [SUPPORT.md](SUPPORT.md) — Where to get help
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) — Community guidelines

## Sponsor

If Lore helps you capture better decisions, consider [sponsoring the project](https://github.com/sponsors/GreyCoderK).

[![Sponsor](https://img.shields.io/badge/Sponsor-%E2%9D%A4-ea4aaa?logo=github-sponsors)](https://github.com/sponsors/GreyCoderK)

![lore-documented](assets/lore-documented.svg)

## Third-Party Notices

| Dependency | License |
|------------|---------|
| [cobra](https://github.com/spf13/cobra) | Apache-2.0 |
| [afero](https://github.com/spf13/afero) | Apache-2.0 |
| [mousetrap](https://github.com/inconshreveable/mousetrap) | Apache-2.0 |
| [pflag](https://github.com/spf13/pflag) | BSD-3-Clause |
| [fsnotify](https://github.com/fsnotify/fsnotify) | BSD-3-Clause |
| [x/term](https://pkg.go.dev/golang.org/x/term) | BSD-3-Clause |
| [x/sys](https://pkg.go.dev/golang.org/x/sys) | BSD-3-Clause |
| [x/text](https://pkg.go.dev/golang.org/x/text) | BSD-3-Clause |

## License

AGPL-3.0 — see [LICENSE](LICENSE). Commercial license available — see [LICENSING.md](../LICENSING.md).
