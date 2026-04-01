# Configuration

Lore utilise un système de configuration en cascade.

## Fichiers de configuration

| Fichier | Rôle | Git |
|---------|------|-----|
| `.lorerc` | Configuration partagée du projet | Committé |
| `.lorerc.local` | Surcharges personnelles (clés API) | Gitignore (chmod 600) |
| `LORE_*` env vars | Surcharges CI/automation | — |
| `--language` flag | Surcharge CLI | — |

**Ordre de résolution** (priorité décroissante) : flags CLI > env vars > `.lorerc.local` > `.lorerc` > défauts.

## Référence complète

```yaml
# .lorerc — config partagée
language: "fr"              # "en" ou "fr" — langue de l'interface

ai:
  provider: ""              # "anthropic", "openai", "ollama", ou "" (zero-API)
  model: ""                 # Nom du modèle

angela:
  mode: draft               # Mode par défaut : "draft" (zero-API) ou "polish" (1 appel API)
  max_tokens: 2000

hooks:
  post_commit: true          # Activer le hook post-commit
  star_prompt_after: 5       # Afficher le prompt star après N commits documentés (0 = désactivé)

templates:
  dir: .lore/templates

output:
  format: markdown
  dir: .lore/docs
```

## Surcharges personnelles

```yaml
# .lorerc.local — personnel, gitignore, chmod 600
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
  api_key: "sk-ant-..."
```

## Variables d'environnement

| Variable | Equivalent |
|----------|------------|
| `LORE_LANGUAGE` | `language` |
| `LORE_AI_PROVIDER` | `ai.provider` |
| `LORE_AI_API_KEY` | `ai.api_key` |

## Valider la configuration

```bash
lore doctor --config
```

## Voir aussi

- [`lore config`](../commands/config.md) — Voir et modifier la config
- [`lore doctor --config`](../commands/doctor.md) — Valider la config
