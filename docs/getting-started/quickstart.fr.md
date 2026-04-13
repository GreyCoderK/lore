---
type: guide
date: 2026-04-12
status: published
related:
  - ../commands/index.md
  - ../guides/configuration.md
  - ../commands/angela-draft.md
  - ../guides/angela-ci.md
---
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

Le hook de lore se déclenche automatiquement :

```
[1/3] Type [feature]:
[2/3] What [Add rate limiting to API]:
[3/3] Why? L'API recevait 10K req/min d'un seul client, causant de la latence pour tous
✓ Capture : feature-add-rate-limiting-2026-03-16.md
```

Trois questions. Quatre-vingt-dix secondes. C'est fait.

![lore interactif](../assets/vhs/interactive.gif)
<!-- Generate: vhs assets/vhs/interactive.tape -->

> **Que vient-il de se passer ?** Le hook post-commit de lore a détecté votre commit, posé 3 questions, et enregistré un fichier Markdown dans `.lore/docs/`. Le fichier contient un en-tête YAML (type, date, hash du commit) et votre "pourquoi" — lié définitivement à ce commit.

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

> **Que vient-il de se passer ?** lore a analysé vos commits et vos documents. 1 commit, 1 document = 100% de couverture. Au fil de vos commits, ce tableau de bord suit l'état de la documentation dans le temps.

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
- [Angela en CI](../guides/angela-ci.md) — Quality gate documentation dans tout pipeline CI (sans `lore init`)
