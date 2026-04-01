# lore doctor

Diagnostiquer et réparer les incohérences du corpus.

## Synopsis

```
lore doctor [flags]
```

## Description

Exécute une suite de vérifications diagnostiques sur le corpus de documentation et la configuration. Peut corriger automatiquement la plupart des problèmes avec `--fix`.

## Flags

| Flag | Type | Défaut | Description |
|------|------|--------|-------------|
| `--fix` | bool | `false` | Corriger automatiquement les problèmes réparables |
| `--config` | bool | `false` | Valider `.lorerc` uniquement (ignore les vérifications du corpus) |
| `--rebuild-store` | bool | `false` | Reconstruire `store.db` à partir des documents et du log Git |
| `--quiet` | bool | `false` | Afficher uniquement le nombre de problèmes |

## Vérifications diagnostiques

| Vérification | Auto-réparable | Description |
|--------------|---------------|-------------|
| `orphan-tmp` | Oui | Fichiers `.tmp` restants d'écritures interrompues |
| `stale-index` | Oui | Fichier d'index désynchronisé par rapport aux documents |
| `stale-cache` | Oui | Cache de revue Angela obsolète |
| `broken-ref` | Non | Références vers des documents inexistants |
| `invalid-frontmatter` | Non | Erreurs d'analyse YAML dans les métadonnées des documents |
| `config` | Non | Validation de la configuration (fautes de frappe, clés inconnues) |

## Validation de la configuration (`--config`)

Détecte les erreurs courantes dans `.lorerc` :

- **Clés inconnues** — Suggère des corrections via la distance de Levenshtein : `ai.providr` → « Vouliez-vous dire `ai.provider` ? »
- **Valeurs invalides** — Types incorrects, nombres hors limites
- **Clés dépréciées** — Pointe vers les équivalents actuels

## Sortie

```
Docs Check:
  ✓ orphan-tmp         (none found)
  ✗ stale-index        .lore/docs/index.md (out of sync)
  ✓ broken-ref         (none found)
  ✓ stale-cache        (none found)
  ✓ invalid-frontmatter (none found)

Config Check:
  ✓ .lorerc            (valid)
  ✓ .lorerc.local      (valid, mode 0600)

1 issue found. Run: lore doctor --fix
```

## Exemples

```bash
# Lancer tous les diagnostics
lore doctor

# Réparation automatique
lore doctor --fix
# → Fixed: stale-index (rebuilt)
# → Manual: broken-ref in decision-2026-03-07.md (references removed document)

# Validation de la configuration uniquement
lore doctor --config

# Reconstruire le store LKS à partir de zéro
lore doctor --rebuild-store

# Porte CI : échouer si des problèmes existent
[ $(lore doctor --quiet) -eq 0 ] || exit 1
```

## Tips & Tricks

- Lancez `lore doctor` après avoir tiré les modifications d'une équipe — les commits des autres peuvent avoir créé des incohérences.
- Après un `lore delete`, lancez `lore doctor` pour trouver les références cassées.
- `--rebuild-store` est sans danger : il reconstruit `store.db` à partir des fichiers Markdown source de vérité. Utilisez-le après des migrations ou en cas de corruption.
- `--config` détecte les fautes de frappe qui retomberaient silencieusement sur les valeurs par défaut — lancez-le après avoir modifié `.lorerc`.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Aucun problème (ou tous corrigés) |
| `1` | Problèmes détectés (non corrigés) |
| `4` | Erreur de validation de la configuration |

## Voir aussi

- [lore status](status.fr.md) — Aperçu rapide de la santé
- [Configuration](../guides/configuration.md) — Référence de la configuration
