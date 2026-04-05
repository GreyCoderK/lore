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
  star_prompt_after: 5       # Show star prompt after N documented commits (0 = disabled)

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
