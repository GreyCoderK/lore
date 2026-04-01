# lore config

Manage API credentials and view configuration.

## Synopsis

```
lore config <set-key|delete-key|list-keys>
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `set-key <provider>` | Store an API key securely |
| `delete-key <provider>` | Remove a stored API key |
| `list-keys` | Show status of all providers |

**Known providers:** `anthropic`, `openai`, `ollama`

## Description

Manages API credentials for Angela AI features. Keys are stored in the OS credential manager (macOS Keychain, Linux secret-service, Windows Credential Manager) or in `.lorerc.local` as fallback.

## `lore config set-key`

```bash
lore config set-key anthropic
# → Enter API key: [hidden input]
# → ✓ Key stored for anthropic
```

Reads the key from stdin with no echo (secure input).

## `lore config delete-key`

```bash
lore config delete-key anthropic
# → ✓ Key removed for anthropic
```

## `lore config list-keys`

```bash
lore config list-keys
# anthropic     stored
# openai        not set
# ollama        stored
```

## Tips & Tricks

- Store keys via `lore config set-key` rather than editing `.lorerc.local` manually — it uses the OS keychain when available.
- For CI, use env vars: `LORE_AI_API_KEY=sk-...` (no keychain needed).
- Ollama doesn't need an API key (local models), but you can set the endpoint in `.lorerc`.

## See Also

- [Configuration guide](../guides/configuration.md) — Full config reference
- [lore angela draft](angela-draft.md) — Uses the configured provider
- [lore doctor --config](doctor.md) — Validate configuration
