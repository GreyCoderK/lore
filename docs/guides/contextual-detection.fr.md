# Détection Contextuelle

Comment le hook post-commit de Lore décide quoi faire avec chaque commit.

## Vue d'ensemble

Quand le hook se déclenche après un commit, Lore évalue une chaîne de règles. La première règle qui correspond l'emporte.

## Chaîne de Détection

```mermaid
flowchart TD
    A[Hook post-commit] --> B{doc-skip dans le message?}
    B -->|Oui| C[Ignorer silencieusement]
    B -->|Non| D{Non-TTY ou TERM=dumb?}
    D -->|Oui| E[Differer vers pending]
    D -->|Non| F{Rebase en cours?}
    F -->|Oui| E
    F -->|Non| G{Commit de merge?}
    G -->|Oui| H[Ignorer — message 1 ligne]
    G -->|Non| I{Cherry-pick + doc existe?}
    I -->|Oui| C
    I -->|Non| J{Amend + doc existe?}
    J -->|Oui| K[Proposer édition du doc existant]
    J -->|Non| L[Scoring Decision Engine]
    L --> M{Score?}
    M -->|>=60| N[Questions complètes]
    M -->|35-59| O[Questions réduites]
    M -->|15-34| P[Suggerer d'ignorer — confirmer]
    M -->|<15| Q[Auto-ignorer silencieusement]
```

## Règles de Détection (Ordre de Priorité)

| # | Règle | Action | Raison |
|---|-------|--------|--------|
| 1 | `[doc-skip]` dans le message | Ignorer | Intention explicite du dev |
| 2 | Non-TTY ou `TERM=dumb` | Différer | CI/pipes ne doivent jamais bloquer |
| 3 | Rebase en cours | Différer | Éviter les prompts pendant le replay |
| 4 | Commit de merge (2+ parents) | Ignorer | Commits d'infrastructure |
| 5 | Cherry-pick + doc source existe | Ignorer | Déjà documenté |
| 6 | Amend + doc existant | Proposer modification | L'utilisateur édite du travail précédent |
| 7 | Score Decision Engine | Action basée sur le score | Analyse multi-signaux |

## Notifications IDE

Quand un commit est différé dans un IDE non-TTY :

1. **VS Code IPC** — Notification native (multi-instance)
2. **Dialog OS** — `osascript` (macOS), `zenity`/`kdialog` (Linux)
3. **Fallback** — Fichier lock (`~/.lore/notify.lock`)

## Tips & Tricks

- Utilisez `[doc-skip]` pour les commits triviaux (typos, config CI, bump de deps).
- Vérifiez le scoring : `lore décision --explain HEAD` montre le détail.
- Après un rebase, vérifiez `lore pending` — les commits rebasés ont été différés.
- Ctrl+C pendant les questions sauvegarde les réponses partielles. `lore pending resolve` reprend.

## Voir aussi

- [lore decision](../commands/decision.md) — Inspecter le scoring
- [lore pending](../commands/pending.md) — Gérer les commits différés
- [Configuration](configuration.md) — Ajuster les seuils
