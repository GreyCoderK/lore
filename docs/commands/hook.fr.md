# lore hook

Gérer le hook Git post-commit qui déclenche le flux de documentation Lore.

## Synopsis

```
lore hook <install|uninstall>
```

## Qu'est-ce que ça fait ?

Le hook est le moteur invisible de Lore. Après chaque `git commit`, il se déclenche automatiquement et lance le flux de questions. `lore hook` permet d'installer ou de retirer ce hook manuellement.

> **Analogie :** Le hook est comme un détecteur de fumée. Vous l'installez une fois, vous l'oubliez, et il s'active quand il le faut. `lore hook install` monte le détecteur. `lore hook uninstall` le retire.

La plupart des utilisateurs n'ont jamais besoin de cette commande — `lore init` installe le hook automatiquement.

## Scénario concret

> Votre équipe utilise Husky pour le linting pre-commit. Vous voulez ajouter le hook post-commit de Lore sans casser la configuration existante :
>
> ```bash
> lore hook install
> # Lore ajoute sa section avec des marqueurs — vos hooks Husky restent intacts
> ```

![lore hook](../assets/vhs/hook.gif)
<!-- Generate: vhs assets/vhs/hook.tape -->

## Sous-commandes

### `lore hook install`

Installe le hook post-commit dans `.git/hooks/post-commit` (ou à l'emplacement `core.hooksPath`).

**Ce qu'il fait :**

1. Vérifie si `.git/hooks/post-commit` existe
2. S'il existe : ajoute la section Lore entre les marqueurs `# LORE-START` et `# LORE-END`
3. S'il n'existe pas : crée le fichier avec le hook Lore
4. Rend le fichier exécutable (`chmod +x`)

**Coexistence avec d'autres hooks :**

```bash
#!/bin/bash
# Votre code de hook existant ici
npm run lint-staged

# LORE-START
/usr/local/bin/lore _hook-post-commit
# LORE-END
```

Les marqueurs garantissent que Lore ne touche qu'à sa propre section.

### `lore hook uninstall`

Supprime la section Lore du fichier hook. Si Lore était le seul contenu, supprime le fichier entièrement.

## Cas limites

### `core.hooksPath` configuré

Si Git est configuré pour utiliser un dossier de hooks personnalisé (courant en monorepo), Lore ne peut pas s'auto-installer :

```bash
lore hook install
# → Attention : core.hooksPath est défini sur /path/to/hooks
# → Ajoutez ces lignes à votre hook post-commit manuellement :
# →   # LORE-START
# →   /usr/local/bin/lore _hook-post-commit
# →   # LORE-END
```

### Hook déjà installé

```bash
lore hook install
# → Hook déjà installé (idempotent — sûr de relancer)
```

## Exemples

```bash
# Installer le hook
lore hook install
# → ✓ Hook post-commit installé

# Vérifier l'installation
cat .git/hooks/post-commit
# → #!/bin/bash
# → # LORE-START
# → /usr/local/bin/lore _hook-post-commit
# → # LORE-END

# Désinstaller
lore hook uninstall
# → ✓ Hook post-commit supprimé

# Vérifier par script
grep -q "LORE-START" .git/hooks/post-commit 2>/dev/null && echo "installé" || echo "pas installé"
```

## Questions fréquentes

### "C'est quoi `_hook-post-commit` ?"

Une commande interne cachée. Le fichier hook appelle `lore _hook-post-commit` qui exécute le Decision Engine, la détection contextuelle et le flux de questions. Ne l'appelez jamais directement.

### "Le hook ne se déclenche pas après mes commits"

Vérifiez dans l'ordre :

1. Hook installé ? `grep "LORE" .git/hooks/post-commit`
2. Hook exécutable ? `ls -la .git/hooks/post-commit` (devrait montrer `-rwx`)
3. `lore` dans le PATH ? `which lore`
4. `core.hooksPath` qui surcharge ? `git config core.hooksPath`

### "Puis-je désactiver le hook temporairement ?"

```bash
# 1. Ignorer un commit
git commit -m "quick fix [doc-skip]"

# 2. Désinstaller et réinstaller plus tard
lore hook uninstall
# ... commits sans Lore ...
lore hook install

# 3. Skip intégré Git (saute TOUS les hooks)
git commit --no-verify -m "urgence"
```

## Tips & Tricks

- **Rarement nécessaire :** `lore init` installe le hook automatiquement.
- **Utilisateurs Husky/pre-commit :** Lore utilise des marqueurs et ne touche jamais vos autres hooks.
- **Désactivation temporaire :** `lore hook uninstall` puis `lore hook install` quand prêt. Ou `[doc-skip]`.
- **Monorepo :** Si `core.hooksPath` est défini, suivez les instructions manuelles que Lore fournit.

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Succès |
| `1` | Erreur (impossible d'écrire dans le dossier hooks) |

## Voir aussi

- [lore init](init.md) — Installe le hook automatiquement
- [Détection contextuelle](../guides/contextual-detection.md) — Comment le hook décide quoi faire
- [lore doctor](doctor.md) — Diagnostiquer les problèmes de hook
