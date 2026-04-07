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

angela:
  mode: draft               # Default mode: "draft" (zero-API) or "polish" (1 API call)
  max_tokens: 2000           # Max tokens for AI responses

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

Checks for typos, unknown keys, and suggests corrections via Levenshtein distance.

## Typical Configurations

### Solo Developer (Minimal)

```yaml
# .lorerc — just the essentials
hooks:
  post_commit: true
output:
  dir: .lore/docs
```

No AI, no language config. Defaults to English, zero-API mode. Maximum simplicity.

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

Star prompt encourages contributors to star the repo. Decision engine skips trivial commits automatically.

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
  mode: draft
  max_tokens: 2000
```

```yaml
# .lorerc.local — personal (gitignored, chmod 600)
ai:
  api_key: "sk-ant-..."
```

Each team member stores their own API key. The shared config defines the provider and model.

### Bilingual Project (FR/EN)

```yaml
# .lorerc
language: "fr"
hooks:
  post_commit: true
```

All UI messages, prompts, badges, and reinforcement messages switch to French. "Lore" becomes "L'or."

## AI Provider Setup

Angela's `polish` and `review` commands require an AI provider. Three providers are supported, each with different trade-offs.

### Anthropic (Claude)

Best quality for technical documentation. Requires API credits purchased separately from a Claude.ai chat subscription.

```yaml
# .lorerc
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"  # or claude-haiku-4-5-20251001 (cheaper)
```

```bash
# Store API key securely in OS keychain
lore config set-key anthropic
```

> **Important:** A Claude.ai chat subscription (Pro, Team) does NOT include API credits. The API is billed separately at [console.anthropic.com](https://console.anthropic.com) → Plans & Billing. Minimum $5 credits.

| Item | Detail |
|------|--------|
| **Get a key** | [console.anthropic.com](https://console.anthropic.com) → API Keys |
| **Cost per call** | ~$0.01–0.05 (Sonnet), ~$0.001 (Haiku) |
| **Endpoint** | `https://api.anthropic.com/v1/messages` (default, automatic) |
| **Format** | Anthropic-specific (not OpenAI-compatible) |

### OpenAI (GPT)

```yaml
# .lorerc
ai:
  provider: "openai"
  model: "gpt-4o-mini"  # cheapest, or gpt-4o for best quality
```

```bash
lore config set-key openai
```

| Item | Detail |
|------|--------|
| **Get a key** | [platform.openai.com](https://platform.openai.com) → API Keys |
| **Cost per call** | ~$0.001 (gpt-4o-mini), ~$0.01 (gpt-4o) |
| **Endpoint** | `https://api.openai.com/v1/chat/completions` (default) |
| **Custom endpoint** | Set `ai.endpoint` for compatible APIs (e.g., Ollama in OpenAI mode, Azure OpenAI — untested, contributions welcome) |

### Ollama (Local — Free)

Runs entirely on your machine. No API key, no cost, no data sent anywhere.

```yaml
# .lorerc
ai:
  provider: "ollama"
  model: "llama3.2"  # or any model: mistral, codellama, gemma2, etc.
```

```bash
# Install and start Ollama
brew install ollama    # macOS
ollama serve &         # start the server
ollama pull llama3.2   # download a model
```

No `lore config set-key` needed — Ollama has no authentication.

| Item | Detail |
|------|--------|
| **Install** | [ollama.com](https://ollama.com) or `brew install ollama` |
| **Cost** | Free (runs on your hardware) |
| **Endpoint** | `http://localhost:11434` (default, automatic) |
| **Models** | Any Ollama model — `ollama list` to see installed |

### Testing OpenAI provider without OpenAI credits

Ollama exposes an OpenAI-compatible API. You can validate the `openai` provider against Ollama:

```yaml
# .lorerc
ai:
  provider: "openai"
  model: "llama3.2"
  endpoint: "http://localhost:11434/v1/chat/completions"
```

```yaml
# .lorerc.local
ai:
  api_key: "any-value"  # Ollama ignores API keys
```

> **Note:** This only works for `openai` provider. The `anthropic` provider uses a different request format that Ollama does not support.

### Provider comparison

| | **Anthropic** | **OpenAI** | **Ollama** |
|---|---|---|---|
| **Quality** | Best for technical docs | Very good | Depends on model |
| **Cost** | ~$0.01–0.05/call | ~$0.001–0.01/call | Free |
| **Privacy** | Data sent to API | Data sent to API | 100% local |
| **Setup** | API key + credits | API key + credits | Install + pull model |
| **Offline** | No | No | Yes |
| **Speed** | Fast | Fast | Depends on hardware |

### No AI? No problem

`lore angela draft` and `lore angela draft --all` work **100% offline** with zero configuration. They analyze document structure, missing sections, style consistency, and cross-references — all locally.

For polish/review without API credits, see the [manual workflow via Claude.ai chat](../faq.md#i-have-a-claudeai-subscription-but-no-api-credits-can-i-use-angela) in the FAQ.

## Troubleshooting

### "My config change has no effect"

Check the cascade order — a higher-priority source may override your change:

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
