---
type: reference
date: 2026-04-12
status: published
related:
  - ../getting-started/completions.md
  - index.md
angela_mode: polish
---
# lore completion

Générer les scripts de complétion shell pour l'auto-complétion de toutes les commandes et flags.

## Synopsis

```
lore completion <bash|zsh|fish|powershell>
```

## Qu'est-ce que ça fait ?

Après configuration, appuyer sur Tab après `lore ` affiche toutes les commandes disponibles. `lore an<TAB>` complète en `lore angela`, et `lore angela <TAB>` montre `draft`, `polish`, `review`. La complétion par tab économise des frappes et fait découvrir des commandes inconnues.

## Scénario concret

> Vous en avez marre de taper `lore angela polish` à chaque fois. La complétion par tab rend ça `lore an<TAB> po<TAB>` — 6 frappes au lieu de 20 :
>
> ```bash
> eval "$(lore completion zsh)"
> ```
>
> 15 secondes de configuration, des heures de frappes économisées.

![lore completion](../assets/vhs/completion.gif)
<!-- Generate: vhs assets/vhs/completion.tape -->

## Shells supportés

### Bash

```bash
# Temporaire (session courante)
eval "$(lore completion bash)"

# Permanent
echo 'eval "$(lore completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh

```bash
# Temporaire
eval "$(lore completion zsh)"

# Permanent (option 1 — eval dans .zshrc)
echo 'eval "$(lore completion zsh)"' >> ~/.zshrc

# Permanent (option 2 — générer dans fpath)
lore completion zsh > "${fpath[1]}/_lore"
```

### Fish

```bash
# Temporaire
lore completion fish | source

# Permanent
lore completion fish > ~/.config/fish/completions/lore.fish
```

### PowerShell

```powershell
# Temporaire
lore completion powershell | Out-String | Invoke-Expression

# Permanent — ajouter à votre profil
Add-Content $PROFILE 'lore completion powershell | Out-String | Invoke-Expression'
```

## Vérifier que ça fonctionne

Après rechargement de votre shell, tapez `lore ` et appuyez sur Tab :

```
$ lore <TAB>
angela      check-update  completion  config      decision    delete
demo        doctor        hook        init        list        new
pending     release       show        status      upgrade
```

Tapez `lore show --<TAB>` pour voir les flags :

```
$ lore show --<TAB>
--all       --after     --bugfix    --decision  --feature
--note      --quiet     --refactor  --type
```

## Conseils avancés

- **Aliases :** Combinez avec des aliases shell pour aller encore plus vite :
  ```bash
  alias ls='lore show'
  alias ll='lore list'
  alias ld='lore doctor'
  alias la='lore angela'
  ```
- **Fish est le plus simple :** Fish charge les complétions depuis `~/.config/fish/completions/` automatiquement — pas besoin de `source`.
- **La méthode Zsh fpath** est plus rapide que `eval` — la complétion est compilée une fois, pas interprétée à chaque démarrage.

## Flags

Cette commande ne prend pas de flags. Le nom du shell est un argument positionnel requis.

**Arguments valides :** `bash`, `zsh`, `fish`, `powershell`

## Exemples

```bash
# Générer pour votre shell et évaluer immédiatement
eval "$(lore completion zsh)"

# Sauvegarder dans un fichier pour une configuration permanente
lore completion bash > /etc/bash_completion.d/lore
lore completion fish > ~/.config/fish/completions/lore.fish

# Quel shell utilisez-vous ?
echo $SHELL
# → /bin/zsh → utilisez : lore completion zsh
```

## Questions fréquentes

### "Quel shell j'utilise ?"

```bash
echo $SHELL
# → /bin/zsh    → lore completion zsh
# → /bin/bash   → lore completion bash
# → /usr/bin/fish → lore completion fish
```

Sur macOS, le shell par défaut est Zsh depuis Catalina. Sur la plupart des distros Linux, c'est Bash.

### "Dois-je relancer après une mise à jour de lore ?"

Uniquement si de nouvelles commandes ont été ajoutées. Le script de complétion reflète la liste de commandes de lore au moment de la génération. Après `lore upgrade`, régénérez :

```bash
eval "$(lore completion zsh)"
```

### "'command not found: compinit'"

Problème spécifique à Zsh. Ajoutez ceci à `~/.zshrc` avant l'eval de complétion :

```bash
autoload -Uz compinit && compinit
eval "$(lore completion zsh)"
```

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Script de complétion généré |
| `1` | Erreur (nom de shell invalide) |

## Voir aussi

- [Guide complétion shell](../getting-started/completions.md) — Configuration détaillée avec dépannage
- [Vue d'ensemble commandes](index.md) — Toutes les commandes disponibles
