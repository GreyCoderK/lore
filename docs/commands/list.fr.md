# lore list

Lister tous les documents du corpus avec leurs métadonnées.

## Synopsis

```
lore list [flags]
```

## Description

Affiche un tableau de tous les documents triés par date (décroissant). Montre le type, le slug, la date et le nombre de tags pour chaque document.

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--type` | string | — | Filtrer par type de document (`decision`, `feature`, `bugfix`, `refactor`, `note`, `release`) |
| `--quiet` | bool | `false` | Sortie machine vers stdout |

## Format de sortie

```
TYPE       SLUG                              DATE        TAGS
decision   auth-strategy-2026-03-07          2026-03-07  3 tags
feature    add-rate-limiting-2026-03-16      2026-03-16  1 tag
refactor   extract-auth-middleware-2026-03-01 2026-03-01  0 tags
```

## Exemples

```bash
# Lister tous les documents
lore list

# Filtrer par type
lore list --type decision

# Compter les documents (compatible avec les pipes)
lore list --quiet | wc -l

# Combiner avec grep
lore list --quiet | grep "2026-03"
```

## Tips & Tricks

- Utilisez `lore list --type decision` avant les revues d'architecture pour retrouver toutes les décisions enregistrées.
- Combinez avec `lore show` pour un flux en deux étapes : lister → choisir → lire.
- La sortie `--quiet` est séparée par des tabulations, idéale pour le scripting : `lore list --quiet | cut -f2` extrait les slugs.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès (même si vide — affiche un message utile) |
| `1` | Erreur (`.lore/` introuvable) |

## Voir aussi

- [lore show](show.fr.md) — Rechercher et afficher un document spécifique
- [lore status](status.fr.md) — Vue d'ensemble de la santé du corpus
