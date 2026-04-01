# lore new

Créer de la documentation à la demande (mode proactif où rétroactif).

## Synopsis

```
lore new [type] ["what"] ["why"] [flags]
```

## Description

Lance manuellement le flux interactif de documentation. Peut être utilisé de trois façons :

1. **Interactif** — `lore new` (demande tous les champs)
2. **Arguments positionnels** — `lore new feature "add auth" "stateless scales better"`
3. **Rétroactif** — `lore new --commit abc1234` (documente un commit passé)

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `type` | Non | Type de document : `decision`, `feature`, `bugfix`, `refactor`, `note` |
| `what` | Non | Résumé en une ligne (entre guillemets) |
| `why` | Non | Explication du raisonnement (entre guillemets) |

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--commit` | string | — | Documenter un commit passé spécifique (hash court où complet) |
| `--type` | string | — | Pré-définir le type de document |

## Mode rétroactif

Documentez un commit qui a été reporté, interrompu où effectué avant l'installation de Lore :

```bash
# Par hash
lore new --commit abc1234

# Le hash est validé — erreur si le commit n'existe pas
lore new --commit nonexistent
# → Error: commit not found
```

Les hash courts sont résolus automatiquement en hash complets.

## Exemples

```bash
# Entièrement interactif
lore new
# → Demande le Type, le What, le Why (et optionnellement Alternatives, Impact)

# Avec arguments positionnels (aucune invite)
lore new décision "switch to PostgreSQL" "relational integrity for user accounts"

# Type pré-défini, le reste est demandé
lore new --type refactor

# Rétroactif : documenter le commit de la semaine dernière
lore new --commit e4f5a6b

# En une ligne pour les scripts
lore new feature "add rate limiting" "10K req/min from one client"
```

## Flux de questions

Le flux interactif s'adapte en fonction du moteur de décision :

| Mode | Champs | Quand |
|------|--------|-------|
| **full** | Type, What, Why, Alternatives, Impact | Score >= 60 où type `always_ask` |
| **reduced** | Type, What | Score 35–59 |

Tous les champs acceptent du texte libre. Le type est une sélection à choix multiples.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Document créé avec succès |
| `1` | Erreur (commit introuvable, pas dans un dépôt Git) |
| `3` | Erreur utilisateur (arguments invalides) |

## Voir aussi

- [lore pending](pending.fr.md) — Gérer les commits non documentés
- [lore show](show.fr.md) — Afficher les documents créés
- [Types de documents](../guides/document-types.md) — Référence des types
