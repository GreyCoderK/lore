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

## See Also

- [`lore config`](../commands/config.md) — View and set config
- [`lore doctor --config`](../commands/doctor.md) — Validate config
