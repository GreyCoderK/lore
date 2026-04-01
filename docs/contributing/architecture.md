# Architecture (for Contributors)

A simplified overview of the Lore codebase. For contribution guidelines, see `CONTRIBUTING.md` at the project root.

## Project Structure

```
cmd/           Cobra commands — one file per CLI command
internal/
  domain/      Shared interfaces and types (no internal deps)
  config/      Configuration cascade (.lorerc → .lorerc.local → env)
  git/         Git adapter — hooks, log, diff, commit info
  storage/     Document storage, front matter, index, doctor
  workflow/    Reactive (hook) and proactive (lore new) flows
  generator/   Document generation pipeline
  angela/      AI-assisted documentation logic
  ai/          AI provider implementations (Anthropic, OpenAI, Ollama)
  i18n/        Bilingual message catalogs (EN/FR)
  ui/          Terminal UI — colors, progress, lists
  engagement/  Milestone messages, star prompt
  fileutil/    Atomic file operations
  notify/      IDE notification (non-TTY detection)
  status/      Repository health collector
  template/    Go template engine
.lore/
  docs/        Documentation corpus (Markdown with YAML front matter)
  pending/     Interrupted/deferred commits
  store.db     LKS index (SQLite — reconstructible from docs)
```

## Data Flow

```
commit → post-commit hook → workflow/reactive.go
  → questions (ui/) → template engine → generator/
  → atomic write to .lore/docs/ → index update (storage/)
```

## Key Patterns

- **Markdown is source of truth** — index, cache, LKS are all reconstructible
- **Atomic writes** — `.tmp` + `os.Rename()` prevents corruption
- **IOStreams** — `stderr` for humans, `stdout` for machines (`--quiet`)
- **Zero implicit network** — AI is opt-in, everything works offline
- **Front-matter-first** — YAML metadata in every document

## How to Contribute

1. Fork from `main`
2. Write tests (`go test ./...`)
3. Run `go vet ./...`
4. Open a PR — see the PR template in `.github/PULL_REQUEST_TEMPLATE.md`
