# lore delete

Supprimer un document du corpus.

## Synopsis

```
lore delete <fichier> [flags]
```

## Qu'est-ce que ça fait ?

Supprime un fichier de documentation de `.lore/docs/` avec confirmation interactive. Avertit des références entrantes depuis d'autres documents.

> **Analogie :** Arracher une page de votre journal de projet. Lore s'assure que vous le voulez vraiment, car une fois parti, le "pourquoi" est perdu.

## Scénario concret

> Vous avez refactoré complètement le système d'authentification. L'ancien document "session-based-auth-2026-01.md" est maintenant trompeur. Temps de nettoyer :
>
> ```bash
> lore delete session-based-auth-2026-01.md
> ```
>
> Lore avertit qu'un autre document le référence, demande confirmation.

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `fichier` | Oui | Nom exact du fichier (ex : `decision-auth-strategy-2026-03-07.md`) |

> **Comment trouver le nom :** Lancez `lore list` pour voir tous les documents avec leurs noms de fichiers.

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--force` | bool | `false` | Supprimer sans demander (pour scripts) |

## Comportement

- **Interactif (TTY) :** Montre le résumé du document, avertit des références, demande `Supprimer ? [o/N]`
- **Non-TTY sans `--force` :** Erreur (sécurité — jamais d'auto-suppression dans les pipes)
- **Documents démo :** Pas de confirmation nécessaire
- **Fichier introuvable :** Erreur avec suggestion de vérifier `lore list`

## Exemples

```bash
# Trouver d'abord le nom de fichier
lore list
# → decision  database-selection-2026-02-10.md  2026-02-10

# Supprimer avec confirmation
lore delete database-selection-2026-02-10.md
# → decision — Database Selection (2026-02-10)
# → ⚠ Référencé par : feature-add-user-model-2026-02-12.md
# → Supprimer ? [o/N] o
# → Supprimé

# Force (scripts)
lore delete old-doc-2025-01-01.md --force
```

## Questions fréquentes

### "Archiver ou supprimer ?"

**Préférez l'archivage.** Éditez le document et changez `status: active` en `status: archived`. Ça préserve l'historique. Ne supprimez que quand le document est vraiment faux ou nuisible.

### "J'ai supprimé un document mais un autre le référence"

Lancez `lore doctor` — il signalera la référence cassée. Éditez ensuite le document référençant pour supprimer ou mettre à jour le lien.

### "Puis-je annuler une suppression ?"

Si pas encore committé : `git checkout -- .lore/docs/fichier.md`. Si committé : `git show HEAD~1:.lore/docs/fichier.md > .lore/docs/fichier.md`. Le document est un simple fichier — Git est votre bouton d'annulation.

## Tips & Tricks

- **Préférez archiver :** `status: archived` au lieu de supprimer — préserve l'historique.
- **Après suppression :** Lancez `lore doctor` pour vérifier les références cassées.
- **Git est votre filet :** Tout document supprimé est récupérable via `git checkout`.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Document supprimé |
| `1` | Erreur (non trouvé, non-TTY sans `--force`) |

## Voir aussi

- [lore list](list.md) — Trouver le nom de fichier à supprimer
- [lore doctor](doctor.md) — Vérifier les références après suppression
