# Types de Documents & Metadonnees

Reference complete des types, statuts et metadonnees front matter.

## Types de Documents

| Type | But | Quand l'utiliser |
|------|-----|------------------|
| **`decision`** | Decisions architecturales, choix de design | "Pourquoi X plutot que Y ?" — choix de base de donnees, framework, API |
| **`feature`** | Implementations de nouvelles fonctionnalites | "Que fait cette feature et pourquoi ?" — endpoints, composants, integrations |
| **`bugfix`** | Corrections de bugs | "Qu'est-ce qui etait casse et pourquoi ?" — race conditions, cas limites |
| **`refactor`** | Refactoring, optimisation | "Pourquoi restructurer ?" — extraction de packages, deduplication |
| **`release`** | Notes de version | Auto-genere par `lore release` |
| **`note`** | Notes generales, observations | "Bon a savoir" — notes de reunion, recherches |

## Flux de Questions par Type

```mermaid
graph TD
    A[Type selectionne] --> B{Type?}
    B -->|decision| C[Complet: What, Why, Alternatives, Impact]
    B -->|feature| D[Standard: What, Why]
    B -->|bugfix| E[Standard: What, Why]
    B -->|refactor| F[Standard: What, Why]
    B -->|note| G[Minimal: What, Why]
    B -->|release| H[Auto-genere — pas de questions]
```

Les documents **decision** ont des champs supplementaires (Alternatives, Impact) car les choix architecturaux necessitent plus de contexte.

## Statuts de Documents

| Statut | Signification | Defini par |
|--------|---------------|------------|
| **`draft`** | En cours | Par defaut a la creation |
| **`published`** | Final, revu | Manuel ou apres `angela polish` |
| **`archived`** | Obsolete, remplace | Manuel |
| **`demo`** | Cree par `lore demo` | `lore demo` uniquement |

## Reference Front Matter

```yaml
---
type: feature                         # REQUIS: decision|feature|bugfix|refactor|release|note
date: 2026-03-16                      # REQUIS: date de creation (YYYY-MM-DD)
status: draft                         # REQUIS: draft|published|archived|demo
commit: abc1234567890abcdef           # Optionnel: hash du commit git associe
tags: [auth, security, jwt]           # Optionnel: tags pour la recherche
related: [decision-auth-2026-03-07.md] # Optionnel: documents lies
generated_by: hook                    # Optionnel: hook|manual|lore
angela_mode: polish                   # Optionnel: draft|polish|review
---
```

## Tips & Tricks

- **Choisir le type :** Hesitation entre `decision` et `feature` ? "Est-ce un choix entre options ?" → `decision`. "Est-ce une construction ?" → `feature`.
- **Tags consultables :** Utilisez des tags coherents. `lore show --type decision` filtre par type ; les tags offrent une granularite plus fine.
- **Archiver plutot que supprimer :** Preferez `status: archived` a la suppression — conserve l'historique.

## Voir aussi

- [lore new](../commands/new.md) — Creer des documents
- [lore show](../commands/show.md) — Rechercher par type
