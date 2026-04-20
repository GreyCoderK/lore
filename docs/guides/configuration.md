---
type: guide
date: 2026-04-12
status: published
related:
  - contextual-detection.md
  - ../faq.md
  - ../commands/config.md
  - ../commands/doctor.md
angela_mode: polish
---
# Configuration

Lore uses a cascading configuration system.

## Config Files

| File | Purpose | Git |
|------|---------|-----|
| `.lorerc` | Shared project settings | Committed |
| `.lorerc.local` | Personal overrides (API keys) | Gitignored (chmod 600) |
| `LORE_*` env vars | CI/automation overrides | — |
| `--language` flag | CLI override | — |

**Resolution order** (highest priority first): CLI flags > env vars > `.lorerc.local` > `.lorerc` > defaults.

## Full Config Reference

```yaml
# .lorerc — shared project config
language: "en"              # "en" or "fr" — UI language

ai:
  provider: ""              # "anthropic", "openai", "ollama", or "" (zero-API)
  model: ""                 # Model name (e.g., "claude-sonnet-4-20250514")
  # api_key: ""             # API key (prefer set-key or LORE_AI_API_KEY env var)
  # endpoint: ""            # Custom endpoint URL (for Ollama, Groq, Together, etc.)
  # timeout: 60s            # Timeout for AI API calls

angela:
  # mode: draft             # DEPRECATED — no runtime effect. Pick the mode via the sub-command: `lore angela draft|polish|review`
  # max_tokens: 8192         # Optional: override auto-computed max tokens (default: dynamic per mode)

  polish:
    backup:
      enabled: true          # Create a pre-polish backup of the source before each write
      path: polish-backups   # Subdirectory under state dir
      retention_days: 30     # Delete backups older than N days (0 = keep forever)
    log:
      retention_days: 30     # polish.log: keep entries from the last N days (0 = disable date filter)
      max_size_mb: 10        # polish.log: trim oldest entries to stay under this cap (0 = disable cap)

  gc:
    corrupt_quarantine:
      retention_days: 14     # Delete *.corrupt-<ts> state files older than N days (0 = keep forever)

hooks:
  post_commit: true          # Enable post-commit hook
  star_prompt: true          # Show star prompt
  star_prompt_after: 5       # Show star prompt after N documented commits (0 = disabled)
  amend_prompt: true         # Ask "Document this change?" on git commit --amend

notification:
  mode: auto                 # auto, terminal, dialog, notify, silent
  disabled_envs: []          # Environments to skip notification (e.g. ["vim"])
  amend: true                # Enable notifications for amend commits

decision:
  threshold_full: 60         # Score >= 60: full question flow
  threshold_reduced: 35      # Score 35-59: reduced questions
  threshold_suggest: 15      # Score 15-34: suggest skip (confirm)
  always_ask: [feat, breaking]  # Always ask for these commit types
  always_skip: [docs, style, ci, build]  # Auto-skip these commit types
  learning: true             # Enable LKS learning from past decisions
  learning_min_commits: 20   # Minimum commits before learning kicks in

templates:
  dir: .lore/templates       # Custom templates directory

output:
  format: markdown           # Output format
  dir: .lore/docs            # Documentation directory
```

## Branch Awareness

Since the Angela Enhancement Sprint, Lore captures the **git branch** and **conventional commit scope** at commit time and stores them in the document front matter:

```yaml
---
type: feature
date: 2026-04-01
commit: a1b2c3d
branch: feature/auth        # current git branch
scope: auth                 # parsed from "feat(auth): ..."
---
```

Both fields propagate through the full pipeline — hook → question flow → template → storage → LKS store — and appear in notification dialogs so you always know which branch a pending commit belongs to.

### Opting out

Branch and scope use `omitempty` in YAML output, so documents created on a detached HEAD or from commits without a conventional scope simply omit those fields. No configuration needed.

### Impact on the amend workflow

When you run `git commit --amend` and a document already exists for the pre-amend commit, Lore asks `Document this change? [Y/n]` (Question 0) and then offers `[U]pdate / [C]reate / [S]kip`. Configurable via `.lorerc`:

```yaml
hooks:
  amend_prompt: true       # Set to false to skip Question 0
notification:
  amend: true              # Enable notifications for amend commits
```

See [Contextual Detection](contextual-detection.md#amend-workflow) for the full behaviour.

## Personal Overrides

```yaml
# .lorerc.local — personal, gitignored, chmod 600
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
  api_key: "sk-ant-..."     # Stored here or in OS keychain
```

## Environment Variables

| Variable | Equivalent |
|----------|------------|
| `LORE_LANGUAGE` | `language` |
| `LORE_AI_PROVIDER` | `ai.provider` |
| `LORE_AI_API_KEY` | `ai.api_key` |

## Validate Configuration

```bash
lore doctor --config
```

Checks for typos and unknown keys, and suggests corrections using Levenshtein distance.

## Typical Configurations

### Solo Developer (Minimal)

```yaml
# .lorerc — just the essentials
hooks:
  post_commit: true
output:
  dir: .lore/docs
```

No AI, no language configuration. Defaults to English, zero-API mode.

### Open Source Project

```yaml
# .lorerc — committed to repo
language: "en"
hooks:
  post_commit: true
  star_prompt_after: 5
decision:
  always_ask: [feat, breaking]
  always_skip: [docs, style, ci]
output:
  dir: .lore/docs
```

The star prompt encourages contributors to star the repo. The Decision Engine skips trivial commits automatically.

### Team with AI

```yaml
# .lorerc — shared settings (committed)
language: "en"
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
hooks:
  post_commit: true
angela:
  max_tokens: 8192
```

```yaml
# .lorerc.local — personal (gitignored, chmod 600)
ai:
  api_key: "sk-ant-..."
```

Each team member stores their own API key locally. The shared config defines the provider and model.

> **`angela.max_tokens`** — When set, this value overrides the auto-computed limit. By default, Angela computes `max_tokens` dynamically based on document size (word_count × 1.3 × 1.8, capped at 8192, floor 512). If you set `angela.max_tokens: 10000` in `.lorerc`, that value is always used instead. Increase this if Angela warns that "input exceeds max output" or if responses are being truncated.

### Polish safety & retention keys

| Key | Default | What it controls |
|-----|---------|------------------|
| `angela.polish.backup.enabled` | `true` | Whether `polish` writes a pre-polish backup to `polish-backups/` before touching the source. Disabling trades safety for speed; a first-time-disabled warning is printed once per state dir. |
| `angela.polish.backup.path` | `polish-backups` | Subdirectory under the state dir where backups land. |
| `angela.polish.backup.retention_days` | `30` | Backups older than N days are deleted by `lore doctor --prune` (and as a side-effect of each polish run). `0` = keep forever. |
| `angela.polish.log.retention_days` | `30` | `polish.log` entries older than N days are dropped on prune. `0` = disable date filter. |
| `angela.polish.log.max_size_mb` | `10` | After the date filter, if `polish.log` still exceeds this cap, oldest entries are trimmed until it fits. `0` = disable cap. |
| `angela.gc.corrupt_quarantine.retention_days` | `14` | `*.corrupt-<ts>` quarantined state files older than N days are deleted on prune. Symlinks and non-regular files are always skipped. `0` = keep forever. |

Run `lore doctor --prune --dry-run` to preview the effect of your retention policy before running it. See [`lore doctor`](../commands/doctor.md#prune-generated-artifacts) for the full command reference.

### Bilingual Project (FR/EN)

```yaml
# .lorerc
language: "fr"
hooks:
  post_commit: true
```

All UI messages, prompts, badges, and reinforcement messages switch to French. The product name becomes "L'or."

## AI Provider Setup

Angela's `polish` and `review` commands require an AI provider. Three providers are supported, each with different trade-offs:

### Anthropic (Claude)

Best quality for technical documentation. Requires API credits purchased separately — distinct from a Claude.ai chat subscription.

**Step 1 — Get an API key:**

1. Go to [console.anthropic.com](https://console.anthropic.com) and sign up (or log in)
2. Navigate to **Settings → API Keys → Create Key**
3. Copy the key (starts with `sk-ant-...`)
4. Add billing credits: **Settings → Plans & Billing → Add Credits** (minimum $5)

> **Important:** A Claude.ai chat subscription (Pro, Team) does NOT include API credits. The API is a separate product billed at [console.anthropic.com](https://console.anthropic.com). You need credits even if you pay for Claude.ai.

**Step 2 — Configure Lore:**

```yaml
# .lorerc
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"  # or claude-haiku-4-5-20251001 (cheaper)
```

```bash
# Store API key securely in OS keychain
lore config set-key anthropic
# → Enter API key: sk-ant-...
```

**Step 3 — Test:**

```bash
lore angela draft --all                                  # free, no API
lore angela polish <your-doc>.md --dry-run               # 1 API call, preview only
lore angela review                                       # 1 API call, corpus analysis
```

| Item | Detail |
|------|--------|
| **Sign up** | [console.anthropic.com](https://console.anthropic.com) |
| **API keys** | Settings → API Keys → Create Key |
| **Add credits** | Settings → Plans & Billing → Add Credits ($5 minimum) |
| **Cost per polish** | ~$0.01–0.05 (Sonnet), ~$0.001 (Haiku) |
| **Endpoint** | `https://api.anthropic.com/v1/messages` (automatic) |
| **Models** | `claude-sonnet-4-20250514` (recommended), `claude-haiku-4-5-20251001` (cheapest) |

### OpenAI (GPT)

**Step 1 — Get an API key:**

1. Go to [platform.openai.com/api-keys](https://platform.openai.com/api-keys) and sign up (or log in)
2. Click **Create new secret key**, name it (e.g., "lore")
3. Copy the key (starts with `sk-...`)
4. Add billing credits: **Settings → Billing → Add payment method** then **Add credits** ($5 minimum)

> **Note:** An OpenAI API account is separate from a ChatGPT subscription. The API uses prepaid credits — no recurring billing unless you enable auto-recharge.

**Step 2 — Configure Lore:**

```yaml
# .lorerc
ai:
  provider: "openai"
  model: "gpt-4o-mini"  # cheapest, or gpt-4o for best quality
```

```bash
# Store API key securely in OS keychain
lore config set-key openai
# → Enter API key: sk-...
```

**Step 3 — Test:**

```bash
lore angela polish <your-doc>.md --dry-run               # preview changes
lore angela review                                       # corpus analysis
```

| Item | Detail |
|------|--------|
| **Sign up** | [platform.openai.com](https://platform.openai.com) |
| **API keys** | [platform.openai.com/api-keys](https://platform.openai.com/api-keys) |
| **Add credits** | Settings → Billing → Add credits ($5 minimum) |
| **Cost per polish** | ~$0.001 (gpt-4o-mini), ~$0.01–0.05 (gpt-4o) |
| **Endpoint** | `https://api.openai.com/v1/chat/completions` (automatic) |
| **Custom endpoint** | Set `ai.endpoint` for compatible APIs (Azure OpenAI, Ollama — see below) |
| **Models** | `gpt-4o-mini` (cheapest), `gpt-4o` (best quality), `gpt-4.1-mini`, `gpt-4.1` |

### Ollama (Local — Free)

Runs entirely on your machine. No API key, no cost, no data leaves your system.

**Step 1 — Install Ollama:**

=== "macOS"

    ```bash
    brew install ollama
    ```

=== "Linux"

    ```bash
    curl -fsSL https://ollama.com/install.sh | sh
    ```

=== "Windows"

    Download the installer from [ollama.com/download](https://ollama.com/download)

**Step 2 — Download a model and start:**

```bash
ollama serve &            # start the server (runs on port 11434)
ollama pull llama3.2      # download a model (~2GB)
```

Other recommended models:

| Model | Size | Quality | Speed |
|-------|------|---------|-------|
| `llama3.2` | 2GB | Good for short docs | Fast |
| `llama3.1:8b` | 4.7GB | Better quality | Medium |
| `llama3.1:70b` | 40GB | Near GPT-4o quality | Slow (needs 64GB RAM) |
| `mistral` | 4.1GB | Good all-around | Fast |
| `codellama` | 3.8GB | Best for code-heavy docs | Fast |
| `gemma2` | 5.4GB | Good for technical writing | Medium |

**Step 3 — Configure Lore:**

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3.2"       # or any model from `ollama list`
```

No `lore config set-key` step needed — Ollama requires no authentication.

**Step 4 — Test:**

```bash
ollama list                                               # verify model is installed
lore doctor --config                                     # verify provider detected
lore angela polish <your-doc>.md --dry-run               # test polish
lore angela review                                       # test review
```

| Item | Detail |
|------|--------|
| **Download** | [ollama.com/download](https://ollama.com/download) or `brew install ollama` |
| **Cost** | Free (runs on your hardware) |
| **Endpoint** | `http://localhost:11434` (automatic) |
| **Browse models** | [ollama.com/library](https://ollama.com/library) |
| **List installed** | `ollama list` |
| **Pull new model** | `ollama pull <model-name>` |

> **Quality tip:** Small models (llama3.2, phi3) may hallucinate or produce generic filler text. For best results, use a model with at least 8B parameters (llama3.1:8b, mistral) and write detailed first drafts before polishing.

### Testing OpenAI code path via Ollama (free)

Ollama exposes an OpenAI-compatible API at `/v1/chat/completions`, letting you test the `openai` provider without spending OpenAI credits:

```yaml
# .lorerc.local
ai:
  provider: "openai"
  model: "llama3.2"
  endpoint: "http://localhost:11434/v1/chat/completions"
  api_key: "unused"       # Ollama ignores API keys, but the field must be non-empty
```

```bash
# Verify it works
ollama serve &
lore angela polish <your-doc>.md --dry-run
```

> **Note:** This only works for the `openai` provider. The `anthropic` provider uses a different request format that Ollama does not support.

### Provider comparison

| | **Anthropic** | **OpenAI** | **Ollama** |
|---|---|---|---|
| **Quality** | Best for technical docs | Very good | Depends on model size |
| **Cost** | ~$0.01–0.05/call | ~$0.001–0.01/call | Free |
| **Privacy** | Data sent to API | Data sent to API | 100% local |
| **Setup time** | 5 min (sign up + credits) | 5 min (sign up + credits) | 2 min (install + pull) |
| **Offline** | No | No | Yes |
| **Speed** | Fast (~3s) | Fast (~3s) | Depends on hardware (5-30s) |
| **Sign up** | [console.anthropic.com](https://console.anthropic.com) | [platform.openai.com](https://platform.openai.com) | No account needed |

### No AI? No problem

`lore angela draft` and `lore angela draft --all` work **100% offline** with zero configuration. They analyze document structure, missing sections, style consistency, and cross-references — entirely locally.

To use polish/review without API credits, see the [manual workflow via Claude.ai chat](../faq.md#i-have-a-claudeai-subscription-but-no-api-credits-can-i-use-angela) in the FAQ.

## Troubleshooting

### "My config change has no effect"

Check the cascade order — a higher-priority source may be overriding your change:

```
CLI flag (--language fr)     ← highest priority
  ↓
Environment (LORE_LANGUAGE)
  ↓
.lorerc.local
  ↓
.lorerc                      ← you edited this
  ↓
Defaults                     ← lowest priority
```

Run `lore doctor --config` to see the resolved configuration.

### "Unknown key warning"

```bash
lore doctor --config
# ✗ unknown key "ai.providr" — did you mean "ai.provider"?
```

Lore uses Levenshtein distance to suggest corrections for typos.

## See Also

- [`lore config`](../commands/config.md) — View and set config
- [`lore doctor --config`](../commands/doctor.md) — Validate config
