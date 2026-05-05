---
type: reference
date: 2026-05-03
status: published
related:
  - commands/index.md
  - guides/configuration.md
angela_mode: polish
---
# Lore CLI — Cheat Sheet

A one-page reference of every command. Pin this in your terminal split.

## 1. Setup (one-time per repo)

| Command | What it does |
|---|---|
| `lore init` | Create `.lore/`, install post-commit hook |
| `lore init --no-demo` | Same, but skip the seed demo doc |
| `lore hook install` | Reinstall the hook (idempotent) |
| `lore hook uninstall` | Remove the hook |
| `lore config` | Show effective config (cascade `~/.lorerc` → repo → env → flags) |
| `lore doctor` | Diagnose corpus, hook, config |
| `lore doctor --fix` | Auto-repair fixable issues |
| `lore doctor --config` | Validate `.lorerc` only |

## 2. Capture (the everyday flow)

The post-commit hook handles capture automatically. These commands are for explicit / retroactive captures.

| Command | What it does |
|---|---|
| `git commit -m "..."` | Hook fires → 3 essential questions (+ 2 optional for higher-stakes) |
| `git commit -m "[doc-skip] ..."` | Skip silently (typo fixes, dep bumps) |
| `git commit --amend` | Asks "Document this change? [Y/n]" then `[U]pdate` / `[C]reate` / `[S]kip` |
| `lore new` | Interactive type selector + question flow, no commit needed |
| `lore new --type feature` | Skip the type selector |
| `lore new --commit <hash>` | Document a past commit retroactively |
| `lore pending` | List undocumented commits (deferred from non-TTY) |
| `lore pending resolve` | Resume an interrupted question flow |
| `lore pending skip <hash>` | Drop a deferred commit without documenting |

## 3. Search & review

| Command | What it does |
|---|---|
| `lore show` | Open the latest doc |
| `lore show <query>` | Fuzzy search the corpus |
| `lore show --type decision` | Filter by document type |
| `lore show --tag <tag>` | Filter by tag |
| `lore list` | List all docs (date desc) |
| `lore status` | Coverage + corpus stats dashboard |
| `lore status --badge` | Generate a shields.io coverage badge |
| `lore decision --explain HEAD` | Show why HEAD scored full / reduced / suggested skip |
| `lore release [tag]` | Generate release notes from the corpus |

## 4. Maintenance

| Command | What it does |
|---|---|
| `lore delete <file>` | Delete a doc with confirmation |
| `lore upgrade` | Self-upgrade to the latest release |
| `lore check-update` | Tell me if a new version is out — no install |
| `lore demo` | Walk through the workflow interactively |
| `lore completion bash\|zsh\|fish\|powershell` | Generate shell completions |

## 5. Angela — the embedded AI reviewer (opt-in)

Angela has 3 modes. **`draft` is 100% offline, no API key.** `polish` and `review` need an AI provider.

| Command | What it does |
|---|---|
| `lore angela draft <file>` | Offline structural analysis: missing sections, style, related docs |
| `lore angela draft --all` | Run draft on every doc in the corpus |
| `lore angela draft --path ./docs` | Standalone — any Markdown directory, no `lore init` needed |
| `lore angela polish <file>` | AI rewrite + interactive `[y/n/b/q]` diff review |
| `lore angela polish --auto` | Auto-accept additions, auto-reject deletions, ask only modifications |
| `lore angela polish --for "CTO"` | Audience-adapted rewrite, saved as a separate file |
| `lore angela polish --persona <id>` | Steer the rewrite by a specific persona lens |
| `lore angela polish --dry-run` | Preview changes without applying |
| `lore angela polish --synthesize` | Apply offline Example Synthesizer (Postman, SQL…) — no AI call |
| `lore angela review` | Corpus-wide coherence: contradictions, isolated docs, gaps |
| `lore angela review --filter "guides/.*"` | Restrict to a subset |
| `lore angela review --all` | Disable 25+25 sampling on large corpora |
| `lore angela review --persona <id>` | Multi-persona coherence — combine `--persona` flags |
| `lore angela consult <persona> <file>` | Single-persona offline draft-check on one doc |
| `lore angela personas` | List the 7 persona IDs and aliases |

### Persona quick-pick

| ID | Name | Best for |
|---|---|---|
| `tech-writer` | Salou | Editorial polish, prose quality |
| `ux-designer` | Gougou | User mental models, accessibility |
| `api-designer` | Ouattara | API contracts, HTTP semantics |
| `qa-reviewer` | Kouamé | Validation criteria, edge cases |
| `architect` | Doumbia | System design, trade-offs |
| `business-analyst` | Béda | Requirements traceability |
| `storyteller` | Affoué | Narrative clarity, onboarding |

## 6. Angela in CI (no `lore init`)

```yaml
# GitHub Actions — 3 lines
- uses: GreyCoderK/lore@v1
  with:
    path: ./docs
    fail_on: warning   # or "error" or "none"
```

```bash
# Any CI — portable script
./scripts/angela-ci.sh --path docs --fail-on warning --install
./scripts/angela-ci.sh --mode review --path docs --all --install
```

## 7. Configuration (`.lorerc`)

Keys you'll actually tune:

```yaml
language: "en"                 # or "fr"
ai:
  provider: "anthropic"        # anthropic | openai | ollama | "" (zero-API)
  model: "claude-sonnet-4-6"
  timeout: 120s
angela:
  max_tokens: 16000            # override auto-computed token cap
hooks:
  post_commit: true
  amend_prompt: true
decision:
  threshold_full: 60           # ≥ ask full questions
  threshold_reduced: 35        # 35–59 ask reduced
  threshold_suggest: 15        # 15–34 suggest skip
  always_ask: [feat, breaking]
  always_skip: [docs, style, ci, build]
```

`.lorerc` is committed (team config). `.lorerc.local` is gitignored (API keys).

## 8. Distribution / install

| Channel | Command |
|---|---|
| Homebrew (macOS / Linux) | `brew install GreyCoderK/tap/lore` |
| Go (any platform) | `go install github.com/greycoderk/lore@latest` |
| curl install (macOS / Linux) | `curl -sSfL https://raw.githubusercontent.com/GreyCoderK/lore/main/install.sh \| sh` |
| Chocolatey (Windows) | `choco install lore-cli` *(pending moderation — fall back to GitHub Releases)* |
| Direct binary | [GitHub Releases](https://github.com/GreyCoderK/lore/releases) |

## See also

- [Full command reference](commands/index.md)
- [Configuration guide](guides/configuration.md)
- [Contextual Detection](guides/contextual-detection.md)
- [Angela personas](commands/angela-personas.md)
