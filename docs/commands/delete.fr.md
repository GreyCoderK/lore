# lore delete

Supprimer un document du corpus.

## Synopsis

```
lore delete <filename> [flags]
```

## Description

Supprime un document de `.lore/docs/` avec confirmation interactive. Signale les références entrantes depuis d'autres documents avant de confirmer. Les documents de démo ne nécessitent pas de confirmation.

## Arguments

| Argument | Requis | Description |
|----------|--------|-------------|
| `filename` | Oui | Nom de fichier exact (ex. `decision-auth-strategy-2026-03-07.md`) |

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--force` | bool | `false` | Passer la confirmation (pour scripts/CI) |

## Details de comportement

- **Interactif (TTY) :** Affiche un résumé du document, signale les références, demande `Delete? [y/N]`
- **Non-TTY sans `--force` :** Erreur, code de sortie 1 (sécurité — jamais de suppression automatique dans les pipes)
- **Documents de démo :** Confirmation non requise (toujours supprimables sans invite)
- **Fichier absent :** Erreur conviviale avec suggestion de vérifier `lore list`

## Exemples

```bash
# Supprimer avec confirmation
lore delete feature-add-auth-2026-03-16.md
# → Affiche : "feature — Add JWT auth middleware (2026-03-16)"
# → "⚠ Referenced by: decision-auth-strategy-2026-03-07.md"
# → "Delete? [y/N]"

# Suppression forcée (scripts/CI)
lore delete feature-add-auth-2026-03-16.md --force

# Supprimer des documents de démo (aucune confirmation requise)
lore delete demo-example-2026-03-16.md
```

## Tips & Tricks

- Lancez toujours `lore list` d'abord pour obtenir le nom de fichier exact.
- Vérifiez les références avant de supprimer : les documents liés pourraient devenir incohérents.
- Utilisez `lore doctor` après des suppressions en masse pour vérifier l'intégrité du corpus.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Document supprimé |
| `1` | Erreur (fichier introuvable, non-TTY sans `--force`) |

## Voir aussi

- [lore list](list.fr.md) — Trouver les noms de fichiers des documents
- [lore doctor](doctor.fr.md) — Vérifier l'intégrité du corpus après suppression
