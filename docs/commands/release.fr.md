# lore release

Générer des notes de version à partir des commits documentés.

## Synopsis

```
lore release [flags]
```

## Description

Rassemble tous les documents entre deux références Git et génère des notes de version regroupées par type. Met à jour `CHANGELOG.md` et sauvegarde un document de release dans `.lore/docs/`.

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--from` | string | Dernier tag | Début de la plage de commits (tag où SHA) |
| `--to` | string | HEAD | Fin de la plage de commits |
| `--version` | string | — | Libellé de version pour les notes de release |
| `--quiet` | bool | `false` | Afficher uniquement le chemin du fichier |

## Sortie

Génère `release-<version>-<date>.md` dans `.lore/docs/` avec :

```markdown
# Release v1.2.0 (2026-03-16)

## Features
- Add rate limiting to API endpoints
- Add user authentication middleware

## Bug Fixes
- Fix token refresh race condition

## Decisions
- Switch to PostgreSQL for data persistence
```

Met également à jour :
- `CHANGELOG.md` avec l'en-tête de la nouvelle version
- `.lore/releases.json` avec les métadonnées

## Exemples

```bash
# Notes de release depuis le dernier tag
lore release --version v1.2.0

# Entre deux tags
lore release --version v1.2.0 --from v1.1.0

# Silencieux : afficher uniquement le chemin du fichier
lore release --version v1.2.0 --quiet
# → .lore/docs/release-v1.2.0-2026-03-16.md
```

## Tips & Tricks

- Lancez avant `git tag` pour inclure les notes de release dans le commit taggé.
- Le document de release fait partie du corpus — recherchable avec `lore show --type release`.
- Compatible avec GoReleaser : le `CHANGELOG.md` généré alimente `goreleaser release`.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Notes de release générées |
| `1` | Erreur (aucun document dans la plage, aucun tag trouvé) |

## Voir aussi

- [lore list](list.fr.md) — Voir tous les documents
- [lore angela review](angela-review.fr.md) — Vérification de cohérence du corpus avant une release
