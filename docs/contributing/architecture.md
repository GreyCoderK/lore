---
type: reference
date: 2026-04-12
status: published
related:
  - architecture.fr.md
---
# Architecture (for Contributors)

A concise overview of the lore codebase. For contribution guidelines, see `CONTRIBUTING.md` at the project root.

## Project Structure

```text
cmd/           Cobra commands — one file per CLI command (the "what")
internal/
  domain/      Shared interfaces and types — the contract between packages (no deps)
  config/      Configuration cascade — why: 5-level override system for flexibility
  git/         Git adapter — why: abstract Git so we never shell out unsafely
  storage/     Document storage — why: Markdown is source of truth, everything derives from it
                 plain_reader.go — PlainCorpusStore for standalone mode (any Markdown dir, no front matter required)
  workflow/    Reactive (hook) + proactive (lore new) — why: two entry points, same pipeline
  generator/   Document generation — why: decouple template rendering from storage
  angela/      AI logic — why: keep AI separate from core (opt-in, not required)
                 langdetect.go   — 24-language detection (including VHS tape syntax)
                 vhs_signals.go  — cross-check tape↔doc↔GIF↔CLI commands
                 multipass.go    — split large docs into sections for sequential polishing
                 preflight.go    — token/cost/timeout estimation before API calls
                 postprocess.go  — auto-tag code fences, normalize Mermaid indent
  ai/          AI providers — why: interface-based, swap Anthropic/OpenAI/Ollama freely
  i18n/        Bilingual catalogs — why: EN/FR from day one, not bolted on later
  ui/          Terminal UI — why: IOStreams pattern (stderr=human, stdout=machine)
  engagement/  Milestones, star prompt — why: behavioral hooks to build documentation habit
  fileutil/    Atomic writes — why: .tmp + rename prevents corruption on Ctrl+C
  notify/      IDE notification — why: non-TTY commits need visibility
  status/      Health collector — why: one place to gather all metrics
  template/    Go templates — why: stdlib, no external engine dependency
.lore/
  docs/        The corpus — THE source of truth. Delete everything else, rebuild from here.
  pending/     Deferred commits — why: never lose a commit, even on Ctrl+C
  store.db     LKS index — reconstructible. If corrupted: lore doctor --rebuild-store
```

## Data Flow

```mermaid
graph LR
    A[git commit] --> B[post-commit hook]
    B --> C{Decision Engine}
    C -->|score ≥ 60| D[Full questions]
    C -->|score 35-59| E[Reduced questions]
    C -->|score < 15| F[Auto-skip]
    D --> G[Template engine]
    E --> G
    G --> H[Generator pipeline]
    H --> I[Atomic write .lore/docs/]
    I --> J[Index update]
```

**In words:**

```text
commit → hook → Decision Engine scores → questions (if needed)
  → template → generator → atomic write → index update
```

## What is LKS?

**LKS** (Lore Knowledge Store) is the SQLite database at `.lore/store.db`. It is a **derived index** — a search and query layer built on top of the Markdown corpus in `.lore/docs/`.

| Property | Value |
|----------|-------|
| Format | SQLite (`.lore/store.db`) |
| Reconstructible | Yes — `lore doctor --rebuild-store` rebuilds from `.lore/docs/` |
| What it stores | Document metadata, tags, commit associations, scope/branch info |
| Why it exists | Fast lookups without parsing every Markdown file on every query |

The LKS is **never the source of truth**. When the database and the Markdown files disagree, Markdown wins. Treat `store.db` as a build artifact.

## Key Patterns

- **Markdown is source of truth** — the index, cache, and LKS are all reconstructible from `.lore/docs/`
- **Atomic writes** — `.tmp` + `os.Rename()` prevents corruption on `Ctrl+C`
- **IOStreams** — `stderr` for human output, `stdout` for machine output (`--quiet`)
- **Zero implicit network** — AI is opt-in; everything works offline
- **Front-matter-first** — every document carries YAML metadata

## Decision Engine Scores

The Decision Engine applies three thresholds to determine how many questions to ask:

| Score range | Behavior | Default threshold |
|-------------|----------|------------------|
| ≥ 60 | Full questions (What + Why + Alternatives + Impact) | `threshold_full: 60` |
| 35 – 59 | Reduced questions (What + Why only) | `threshold_reduced: 35` |
| 15 – 34 | Suggest only — minimal prompt | `threshold_suggest: 15` |
| < 15 | Auto-skip — no questions | — |

All thresholds are configurable in `.lorerc`. See [Contextual Detection](../guides/contextual-detection.md) for the 7 scoring signals.

## How to Contribute

1. Fork from `main`
2. Write tests (`go test ./...`)
3. Run `go vet ./...`
4. Open a PR — see the PR template in `.github/PULL_REQUEST_TEMPLATE.md`
