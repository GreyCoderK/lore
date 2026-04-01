# lore hook

Gérer le hook Git post-commit.

## Synopsis

```
lore hook <install|uninstall>
```

## Sous-commandes

| Sous-commande | Description |
|---------------|-------------|
| `install` | Installer le hook post-commit de Lore |
| `uninstall` | Supprimer le hook post-commit de Lore |

## Description

Gère le hook post-commit qui déclenche le flux de documentation après chaque commit. Le hook est installé dans `.git/hooks/post-commit` (ou à l'emplacement de `core.hooksPath`).

## Marqueurs du hook

Le hook utilise des marqueurs pour coexister sans risque avec d'autres hooks :

```bash
# LORE-START
/path/to/lore _hook-post-commit
# LORE-END
```

Si `core.hooksPath` est configuré, Lore ne peut pas installer le hook automatiquement. Il fournit les marqueurs pour une insertion manuelle.

## Exemples

```bash
# Installer
lore hook install

# Désinstaller
lore hook uninstall

# Vérifier si le hook est installé
grep -q "LORE-START" .git/hooks/post-commit && echo "installed"
```

## Tips & Tricks

- `lore init` installe le hook automatiquement — vous avez rarement besoin de `lore hook install` directement.
- Si vous utilisez Husky ou un framework pre-commit, ajoutez les marqueurs Lore manuellement dans votre hook existant.
- Le hook appelle `lore _hook-post-commit` (commande cachée) — ne l'appelez jamais directement.

## Voir aussi

- [lore init](init.fr.md) — Installe le hook automatiquement
- [Détection contextuelle](../guides/contextual-detection.md) — Comment le hook décide quoi faire
