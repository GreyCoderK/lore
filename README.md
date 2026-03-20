# Lore

[![CI](https://github.com/GreyCoderK/lore/actions/workflows/ci.yml/badge.svg)](https://github.com/GreyCoderK/lore/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/greycoderk/lore)](https://goreportcard.com/report/github.com/greycoderk/lore)
[![Go Reference](https://pkg.go.dev/badge/github.com/greycoderk/lore.svg)](https://pkg.go.dev/github.com/greycoderk/lore)
[![Go Version](https://img.shields.io/github/go-mod/go-version/GreyCoderK/lore)](https://github.com/GreyCoderK/lore)
[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL--3.0-blue.svg)](LICENSE)
[![GitHub Release](https://img.shields.io/github/v/release/GreyCoderK/lore)](https://github.com/GreyCoderK/lore/releases)
[![GitHub Stars](https://img.shields.io/github/stars/GreyCoderK/lore)](https://github.com/GreyCoderK/lore/stargazers)
[![GitHub Forks](https://img.shields.io/github/forks/GreyCoderK/lore)](https://github.com/GreyCoderK/lore/network/members)
[![GitHub Issues](https://img.shields.io/github/issues/GreyCoderK/lore)](https://github.com/GreyCoderK/lore/issues)
[![Sponsor](https://img.shields.io/badge/Sponsor-GreyCoderK-ea4aaa?logo=github-sponsors)](https://github.com/sponsors/GreyCoderK)

**A CLI tool that captures the *why* behind your code, one commit at a time.**

Lore hooks into your Git workflow to document decisions, trade-offs, and context that code alone can't convey. Every commit becomes an opportunity to build a searchable knowledge base — no wiki, no Confluence, just Markdown files living next to your code.

```
$ git commit -m "feat: add JWT auth middleware"
  [1/3] Type [feature]:
  [2/3] What [add JWT auth middleware]:
  [3/3] Why? Because stateless auth scales better than sessions
  Captured  feature-add-jwt-auth-middleware-2026-03-16.md
```

*Your code knows what. Lore knows why.*

## Why Lore?

Code captures **what** you built. Git captures **when** you changed it. But neither captures **why** you made the decisions you made.

Six months later, you're staring at a piece of code wondering: *Why did we choose JWT over sessions? What alternatives did we consider? What was the impact?* The answer is lost — buried in a Slack thread, a PR comment, or someone's memory.

Lore fixes this by making documentation a natural part of your commit workflow:

- **3 questions, <10 seconds** — Type, What, Why. That's it. Express mode skips optional questions when you're fast.
- **Zero friction** — Post-commit hook asks automatically. No context switch, no separate tool.
- **Markdown files in your repo** — Searchable, versionable, portable. No external service required.
- **Works offline** — Everything is local. No network calls, no API keys (unless you opt in to Angela).

## Installation

```bash
# Homebrew (macOS / Linux)
brew install GreyCoderK/lore/lore

# Snap (Linux)
sudo snap install lore --classic

# Go
go install github.com/greycoderk/lore@latest
```

Or download pre-built binaries from [GitHub Releases](https://github.com/GreyCoderK/lore/releases).

## Quick Start

```bash
# Initialize Lore in your project
lore init

# That's it. Next time you commit, Lore will ask 3 questions.
# Or create documentation on demand:
lore new

# Retroactively document a past commit:
lore new --commit abc1234
```

## Commands

| Command | Description |
|---------|-------------|
| `lore init` | Initialize Lore in the current repository |
| `lore new` | Create documentation on demand (manual mode) |
| `lore new --commit <hash>` | Document a past commit retroactively |
| `lore show <query>` | Search and display a document |
| `lore list` | List all documents in the corpus |
| `lore status` | Repository health dashboard |
| `lore delete <filename>` | Delete a document with confirmation |
| `lore pending` | List commits with incomplete documentation |
| `lore pending resolve` | Resume interrupted documentation |
| `lore pending skip <hash>` | Skip a pending commit |
| `lore doctor` | Diagnose corpus inconsistencies |
| `lore doctor --fix` | Automatically repair fixable issues |
| `lore demo` | Interactive demo of the documentation flow |

## How It Works

### The Documentation Flow

When you commit, Lore's post-commit hook triggers an interactive flow:

1. **Type** — What kind of change? (feature, bugfix, decision, refactor, note)
2. **What** — Pre-filled from your commit message. Press Enter to confirm.
3. **Why** — The one question that matters. Why this approach?

If you're fast (all 3 answers in <3 seconds), Lore enters **express mode** and skips the optional questions (Alternatives, Impact). If you take your time, it asks all 5.

### Contextual Detection

Lore intelligently handles Git contexts:

- **Merge commits** — Skipped automatically (no documentation needed)
- **Rebase** — Deferred to pending (you can resolve later)
- **Cherry-pick with existing doc** — Skipped (already documented)
- **Amend** — Updates the existing document instead of creating a new one
- **Non-TTY** (CI, scripts) — Deferred to pending silently
- **Ctrl+C** — Partial answers saved to pending (nothing lost)

### Document Format

Documents are Markdown files with YAML front matter:

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

### Corpus Health

```bash
$ lore doctor
  ✓  orphan-tmp          (none found)
  ✗  stale-index         README.md (out of sync)
  ✓  broken-ref          (none found)
  ✓  stale-cache         (none found)
  ✓  invalid-frontmatter (none found)

1 issue found. Run: lore doctor --fix
```

## Configuration

Lore uses a cascading config system:

| File | Purpose | Git |
|------|---------|-----|
| `.lorerc` | Shared project config | Committed |
| `.lorerc.local` | Personal overrides (API keys) | Gitignored |
| `LORE_*` env vars | CI/automation overrides | — |

```yaml
# .lorerc
ai:
  provider: ""          # "anthropic", "openai", "ollama", or "" (zero-API)

hooks:
  post_commit: true

templates:
  dir: .lore/templates
```

## Architecture

```
lore_cli/
├── cmd/                    # Cobra commands (CLI layer)
├── internal/
│   ├── ai/                 # AI provider implementations (Epic 6)
│   ├── angela/             # AI-assisted documentation logic (Epic 6)
│   ├── cli/                # Exit codes, CLI utilities
│   ├── config/             # Configuration cascade (.lorerc)
│   ├── domain/             # Interfaces, types, DTOs
│   ├── engagement/         # Milestone messages
│   ├── fileutil/           # Atomic file operations
│   ├── generator/          # Document generation pipeline
│   ├── git/                # Git adapter (hooks, log, diff)
│   ├── status/             # Repository status collector
│   ├── storage/            # Document storage, front matter, index, doctor
│   ├── template/           # Go template engine
│   ├── ui/                 # Terminal UI (colors, verbs, lists)
│   └── workflow/           # Reactive/proactive documentation flows
└── .lore/                  # Lore data directory
    ├── docs/               # Documentation corpus (Markdown)
    ├── pending/            # Interrupted/deferred commits
    └── templates/          # Custom templates (optional)
```

### Design Principles

- **Markdown is the source of truth** — Everything else (index, cache) is derived and reconstructible
- **Zero implicit network calls** — AI features are opt-in, everything works offline
- **Atomic writes** — `.tmp` + `os.Rename()` prevents corruption
- **stderr for humans, stdout for machines** — `--quiet` outputs machine-parseable data on stdout
- **No external dependencies for core** — stdlib `net/http` for AI providers, no SDKs

## Project Roadmap

Lore is developed in epics, each delivering a cohesive set of features:

| Epic | Name | Status | Stories |
|------|------|--------|---------|
| 1 | Project Setup & First Impression | Done | 4 stories |
| 2 | Document Capture at Commit | Done | 9 stories |
| 3 | Search & Retrieval | Done | 3 stories |
| 4 | Documentation Lifecycle | Done | 3 stories |
| 5 | Maintenance & Resilience | Done | 2 stories |
| 6 | Angela — AI-Assisted Documentation | **Next** | 3 stories |
| 7 | Release & Advanced Review | Backlog | 2 stories |
| 8 | User Notes | Backlog | 3 stories |
| 9 | References & Resolution | Backlog | 5 stories |
| 10 | Interactive Selection & Flow | Backlog | 4 stories |
| 11 | Reference Health & Diagnostics | Backlog | 3 stories |
| 12 | Cross-Cutting Reference Integration | Backlog | 3 stories |

### Current Status: Epic 6 — Angela

The next milestone introduces **Angela**, an AI-assisted documentation companion:

- **Story 6-1**: AI provider abstraction (Anthropic, OpenAI, Ollama) with factory pattern
- **Story 6-2**: `lore angela draft` — zero-API structural analysis, style guide alignment, corpus coherence checks
- **Story 6-3**: `lore angela polish` — single API call with interactive diff review

Angela operates in two modes:
- **Draft mode** (zero-API) — Local analysis: missing sections, style guide compliance, related documents. No network calls.
- **Polish mode** (1 API call) — AI rewrites with interactive diff. Accept/reject each change individually. Atomic writes preserve originals.

## Third-Party Notices

The following dependencies require attribution under their respective licenses:

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

Full license texts are available in each dependency's repository.

## License

AGPL-3.0 — see [LICENSE](LICENSE). A commercial license is available for proprietary use — see [LICENSING.md](../LICENSING.md).
