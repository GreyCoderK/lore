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
  provider: ""              # "anthropic", "openai", "ollama", ou "" (zéro-API)
  model: ""                 # Nom du modèle

angela:
  mode: draft               # Mode par défaut : "draft" (zéro-API) ou "polish" (1 appel API)
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
  api_key: "sk-ant-..."     # Stockée ici ou dans le trousseau OS
```

## Variables d'environnement

| Variable | Équivalent |
|----------|------------|
| `LORE_LANGUAGE` | `language` |
| `LORE_AI_PROVIDER` | `ai.provider` |
| `LORE_AI_API_KEY` | `ai.api_key` |

## Valider la configuration

```bash
lore doctor --config
```

Vérifie les fautes de frappe, clés inconnues, et suggère des corrections via distance de Levenshtein.

## Configurations typiques

### Développeur solo (minimaliste)

```yaml
# .lorerc — juste l'essentiel
hooks:
  post_commit: true
output:
  dir: .lore/docs
```

Pas d'IA, pas de config de langue. Anglais par défaut, mode zéro-API. Simplicité maximale.

### Projet open source

```yaml
# .lorerc — committé dans le repo
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

Le prompt star encourage les contributeurs à mettre une étoile. Le Decision Engine ignore les commits triviaux automatiquement.

### Équipe avec IA

```yaml
# .lorerc — config partagée (committée)
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
# .lorerc.local — personnel (gitignore, chmod 600)
ai:
  api_key: "sk-ant-..."
```

Chaque membre de l'équipe stocke sa propre clé API. La config partagée définit le fournisseur et le modèle.

### Projet bilingue (FR/EN)

```yaml
# .lorerc
language: "fr"
hooks:
  post_commit: true
```

Tous les messages UI, prompts, badges et messages de renforcement passent en français. "Lore" devient "L'or."

## Dépannage

### "Mon changement de config n'a aucun effet"

Vérifiez l'ordre de la cascade — une source de priorité plus élevée peut surcharger votre changement :

```
Flag CLI (--language fr)     ← priorité la plus haute
  ↓
Environnement (LORE_LANGUAGE)
  ↓
.lorerc.local
  ↓
.lorerc                      ← vous avez édité ici
  ↓
Défauts                      ← priorité la plus basse
```

Lancez `lore doctor --config` pour voir la configuration résolue.

### "Avertissement de clé inconnue"

```bash
lore doctor --config
# ✗ clé inconnue "ai.providr" — vouliez-vous dire "ai.provider" ?
```

Lore utilise la distance de Levenshtein pour suggérer des corrections de fautes de frappe.

## Voir aussi

- [`lore config`](../commands/config.md) — Voir et modifier la config
- [`lore doctor --config`](../commands/doctor.md) — Valider la config
