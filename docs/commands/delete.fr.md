---
type: reference
date: 2026-04-12
status: published
related:
  - list.md
  - doctor.md
---
# lore delete

Supprimer un document du corpus.

## Synopsis

```
lore delete <fichier> [flags]
```

## Qu'est-ce que ça fait ?

Supprime un fichier de documentation de `.lore/docs/`. Lore demande une confirmation avant de supprimer.

## Scénario concret

> Vous avez refactoré complètement le système d'authentification. L'ancien document "session-based-auth-2026-01.md" est maintenant trompeur — il décrit une approche abandonnée. Temps de nettoyer :
>
> ```bash
> lore delete session-based-auth-2026-01.md
> ```
>
> Lore avertit qu'un autre document le référence, demande confirmation. Vous supprimez puis lancez `lore doctor` pour nettoyer.

![lore delete](../assets/vhs/delete.gif)
<!-- Generate: vhs assets/vhs/delete.tape -->

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `fichier` | Oui | Nom exact du fichier (ex : `decision-auth-strategy-2026-03-07.md`) |

> **Comment trouver le nom :** Lancez `lore list` pour voir tous les documents avec leurs noms de fichiers.

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--force` | bool | `false` | Supprimer sans demander (pour scripts) |

## Ce qui se passe

### Interactif (par défaut)

```bash
lore delete decision-auth-strategy-2026-03-07.md
```

```
  decision — Switch to PostgreSQL (2026-03-07)
  ⚠ Référencé par : feature-add-user-model-2026-03-08.md

  Supprimer ce document ? [o/N] o
  ✓ Supprimé
```

Lore affiche :
1. **Ce que vous supprimez** — type, titre, date
2. **Références** — autres docs qui le mentionnent (pour savoir ce qui pourrait casser)
3. **Confirmation** — vous devez taper `o` pour continuer

### Mode force (scripts/CI)

```bash
lore delete decision-auth-strategy-2026-03-07.md --force
# → Supprimé immédiatement, sans questions
```

### Règles de sécurité

| Scénario | Comportement |
|----------|-------------|
| **Normal (TTY)** | Demande confirmation |
| **Pipe/Non-TTY (sans `--force`)** | Erreur — jamais d'auto-suppression dans les scripts |
| **Documents démo** | Pas de confirmation nécessaire (ce sont juste des démos) |
| **Fichier introuvable** | Erreur explicite avec suggestion |

## Tips & Tricks

- **Préférez l'archivage à la suppression :** Éditez le fichier et définissez `status: archived`. Ça préserve l'historique.
- **Nettoyage en lot :** `lore list --type note --quiet | xargs -I{} lore delete {} --force` (attention !).
- **Après suppression :** Lancez `lore doctor` pour vérifier les références cassées.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Document supprimé |
| `1` | Erreur (non trouvé, non-TTY sans `--force`) |

## Exemples

```bash
# Trouver d'abord le nom de fichier
lore list
# → decision  database-selection-2026-02-10.md  2026-02-10

# Supprimer avec confirmation
lore delete database-selection-2026-02-10.md
# → decision — Database Selection (2026-02-10)
# → Supprimer ? [o/N] o
# → Supprimé

# Force (scripts)
lore delete old-doc-2025-01-01.md --force
```

## Questions fréquentes

### "Archiver ou supprimer ?"

**Préférez l'archivage.** Changez `status: active` en `status: archived` dans le document. Ça préserve l'historique. Ne supprimez que quand le document est activement trompeur ou nuisible.

### "J'ai supprimé un document mais un autre le référence"

Lancez `lore doctor` — il signalera la référence cassée. Éditez ensuite le document référençant pour supprimer ou mettre à jour le lien.

### "Puis-je annuler une suppression ?"

Si pas encore committé : `git checkout -- .lore/docs/fichier.md`. Si committé : `git show HEAD~1:.lore/docs/fichier.md > .lore/docs/fichier.md`. Les documents sont des fichiers simples — Git est votre bouton d'annulation.

## Voir aussi

- [lore list](list.md) — Trouver le nom de fichier à supprimer
- [lore doctor](doctor.md) — Vérifier les références après suppression
