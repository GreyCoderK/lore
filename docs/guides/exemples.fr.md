# Exemples

## Dépôt de démonstration

Un dépôt démo pré-configuré est disponible dans [`examples/demo-repo/`](https://github.com/greycoderk/lore/tree/main/examples/demo-repo) :

- `.lorerc` — Configuration minimale
- `.lore/docs/` — 3-5 documents réels générés par Lore (dogfooding)
- `README.md` — Comment reproduire le setup

## Cas d'usage courants

### Développeur solo

```yaml
# .lorerc — config minimale, zero friction
hooks:
  post_commit: true
output:
  dir: .lore/docs
```

Chaque commit déclenche 3 questions. Consultez vos décisions avec `lore show` quand vous revisitez le code des mois plus tard.

### Projet open source

```yaml
# .lorerc — partage, committé dans le repo
language: "en"
hooks:
  post_commit: true
  star_prompt_after: 5
output:
  dir: .lore/docs
```

Les contributeurs documentent leurs changements. Le corpus devient un journal de design vivant.

### Équipe avec IA

```yaml
# .lorerc — partage
hooks:
  post_commit: true

# .lorerc.local — personnel, gitignore
ai:
  provider: "anthropic"
  model: "claude-sonnet-4-20250514"
  api_key: "sk-ant-..."
```

Utilisez `lore angela draft` pour l'analyse sans API et `lore angela polish` pour la réécriture assistée.

## Voir aussi

- [Quickstart](../getting-started/quickstart.md) — Guide pratique 5 minutes
- [Configuration](configuration.md) — Référence complète
