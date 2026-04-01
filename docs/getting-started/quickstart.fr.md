# Quickstart (5 minutes)

De zéro à votre premier "pourquoi" capturé en 5 minutes.

## 1. Initialiser Lore

```bash
cd votre-projet
lore init
```

Crée le dossier `.lore/` et installe le hook post-commit.

## 2. Faire un commit

```bash
git add .
git commit -m "Add rate limiting to API"
```

Le hook Lore se déclenche automatiquement :

```
[1/3] Type [feature]:
[2/3] What [Add rate limiting to API]:
[3/3] Why? L'API recevait 10K req/min d'un seul client, causant de la latence pour tous
✓ Capture : feature-add-rate-limiting-2026-03-16.md
```

Trois questions. Quatre-vingt-dix secondes. C'est fait.

## 3. Consulter votre document

```bash
lore show
```

```markdown
---
type: feature
date: 2026-03-16
commit: e4f5a6b
---
# Add rate limiting to API

## Why
L'API recevait 10K req/min d'un seul client, causant de la latence pour tous.
```

## 4. Vérifier la santé du corpus

```bash
lore status
```

```
Documents : 1 | En attente : 0 | Couverture : 100%
```

## 5. Explorer davantage

```bash
# Documenter un commit passé rétroactivement
lore new --commit abc1234

# Lister tous les documents
lore list

# Diagnostiquer
lore doctor
```

## Et ensuite ?

- [Référence commandes](../commands/index.md) — Toutes les commandes en détail
- [Configuration](../guides/configuration.md) — Personnaliser Lore
- [Angela IA](../commands/angela-draft.md) — Documentation assistée par IA
