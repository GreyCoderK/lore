# lore show

Rechercher et afficher des documents du corpus.

## Synopsis

```
lore show [keyword] [flags]
```

## Description

Recherche dans le corpus de documentation par mot-clé et affiche les documents correspondants. Le comportement s'adapte à l'environnement du terminal :

- **TTY** (terminal interactif) → Affiche une liste de sélection pour les résultats multiples
- **Non-TTY** (pipe, script) → Produit une liste tabulée lisible par machine
- **`--quiet`** → Sortie analysable uniquement, aucune invite

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `keyword` | Oui (sauf `--all`) | Terme de recherche à comparer aux titres et contenus des documents |

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--type` | string | — | Filtrer par type de document |
| `--after` | string | — | Afficher les documents après une date (`YYYY-MM` où `YYYY-MM-DD`) |
| `--all` | bool | `false` | Afficher tous les documents (déprécié — utilisez `lore list`) |
| `--quiet` | bool | `false` | Sortie machine : valeurs séparées par des tabulations vers stdout |
| `--feature` | bool | — | Raccourci pour `--type feature` |
| `--decision` | bool | — | Raccourci pour `--type decision` |
| `--bugfix` | bool | — | Raccourci pour `--type bugfix` |
| `--refactor` | bool | — | Raccourci pour `--type refactor` |
| `--note` | bool | — | Raccourci pour `--type note` |

Les raccourcis de type sont mutuellement exclusifs entre eux et avec `--type`.

## Modes de sortie

**Résultat unique** → Contenu complet du document vers stdout :
```bash
lore show "auth middleware"
# → Affiche le document Markdown complet
```

**Résultats multiples (TTY)** → Liste numérotée interactive :
```
  1  feature   Add JWT auth middleware       2026-02-15
  2  refactor  Extract auth middleware       2026-03-01
Select [1-2]:
```

**Résultats multiples (non-TTY où --quiet)** → Séparés par des tabulations :
```
feature\tAdd JWT auth middleware\t2026-02-15
refactor\tExtract auth middleware\t2026-03-01
```

## Exemples

```bash
# Rechercher par mot-clé
lore show "auth"

# Filtrer par type
lore show "middleware" --type decision
lore show "middleware" --décision  # raccourci

# Filtrer par date
lore show "api" --after 2026-03

# Envoyer vers grep
lore show "rate" --quiet | grep feature

# Tous les documents (préférer lore list)
lore show --all
```

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Correspondance trouvée et affichée |
| `2` | Aucune correspondance (ignoré) |
| `3` | Aucun mot-clé fourni (erreur utilisateur) |

## Voir aussi

- [lore list](list.fr.md) — Lister tous les documents avec métadonnées
- [lore status](status.fr.md) — Statistiques du corpus
