---
type: guide
date: 2026-04-12
status: published
related:
  - ../commands/index.md
  - ../guides/configuration.md
  - ../commands/angela-draft.md
  - ../guides/angela-ci.md
angela_mode: polish
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

Le hook de Lore se déclenche automatiquement :

```
[1/3] Type [feature]:
[2/3] What [Add rate limiting to API]:
[3/3] Why? L'API recevait 10K req/min d'un seul client, causant de la latence pour tous
✓ Capture : feature-add-rate-limiting-2026-03-16.md
```

Trois questions. Quatre-vingt-dix secondes. C'est fait.

![lore interactif](../assets/vhs/interactive.gif)
<!-- Generate: vhs assets/vhs/interactive.tape -->

> **Que vient-il de se passer ?** Le hook post-commit de Lore a détecté votre commit, posé 3 questions, et enregistré un fichier Markdown dans `.lore/docs/`. Le fichier contient un en-tête YAML (type, date, hash du commit) et votre "pourquoi" — lié définitivement à ce commit.

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

```yaml
Documents : 1 | En attente : 0 | Couverture : 100%
```

| Métrique | Valeur | Signification |
|----------|--------|---------------|
| Documents | 1 | Total de fichiers `.lore/docs/*.md` |
| En attente | 0 | Commits sans documentation |
| Couverture | 100% | Commits documentés / total des commits |

> **Que vient-il de se passer ?** Lore a analysé vos commits et vos documents. 1 commit, 1 document = 100% de couverture. Au fil de vos commits, ce tableau de bord suit l'état de la documentation dans le temps.

## 5. Ajouter un badge de couverture

Montrez au monde que votre projet est documenté :

```bash
lore status --badge >> README.md
```

Cela génère un badge shields.io comme :

![lore | documented 85%](https://img.shields.io/badge/lore-documented%2085%25-d4a)

Les couleurs s'adaptent automatiquement : gris (< 50 %), vert (50-79 %), or (80 %+).

## 6. Explorer davantage

```bash
# Documenter un commit passé rétroactivement
lore new --commit abc1234

# Lister tous les documents
lore list

# Diagnostiquer
lore doctor

# Polish assisté par IA (avec clé API)
lore angela polish decision-database-2026-02-10.md

# Générer des exemples API depuis vos docs (hors ligne, gratuit)
lore angela polish api-spec.md --synthesize

# Consulter un expert spécifique sur votre doc
lore angela consult api-designer api-spec.md
```

## Et ensuite ?

- [Référence commandes](../commands/index.md) — Toutes les commandes en détail
- [Configuration](../guides/configuration.md) — Personnaliser Lore
- [Angela IA](../commands/angela-draft.md) — Documentation assistée par IA
- [Angela Consult](../commands/angela-consult.md) — Consultation ponctuelle d'un seul persona
- [Angela en CI](../guides/angela-ci.md) — Quality gate documentation dans tout pipeline CI (sans `lore init`)
- [Complétion shell](completions.md) — Complétion par tabulation pour toutes les commandes et flags
