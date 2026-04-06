# lore completion

Générer les scripts de complétion shell pour l'auto-complétion de toutes les commandes et flags.

## Synopsis

```
lore completion <bash|zsh|fish|powershell>
```

## Qu'est-ce que ça fait ?

Après configuration, taper `lore ` et appuyer sur Tab affiche toutes les commandes disponibles. Taper `lore an<TAB>` complète en `lore angela`. Taper `lore angela <TAB>` montre `draft`, `polish`, `review`. Ça économise des frappes et aide à découvrir des commandes.

> **Analogie :** C'est comme l'autocomplétion de votre clavier de téléphone — mais pour les commandes Lore dans votre terminal.

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

## Flags

Cette commande ne prend pas de flags. Le nom du shell est un argument positionnel requis.

**Arguments valides :** `bash`, `zsh`, `fish`, `powershell`

## Shells supportés

### Bash

```bash
# Temporaire (session courante)
eval "$(lore completion bash)"

# Permanent
echo 'eval "$(lore completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh (défaut macOS)

```bash
# Temporaire
eval "$(lore completion zsh)"

# Permanent (option 1 — eval dans .zshrc)
echo 'eval "$(lore completion zsh)"' >> ~/.zshrc

# Permanent (option 2 — compilé, démarrage plus rapide)
lore completion zsh > "${fpath[1]}/_lore"
autoload -Uz compinit && compinit
```

> **Astuce :** L'option 2 (fpath) est plus rapide car la complétion est compilée une fois, pas interprétée à chaque démarrage.

### Fish

```bash
# Temporaire
lore completion fish | source

# Permanent (Fish charge automatiquement)
lore completion fish > ~/.config/fish/completions/lore.fish
```

### PowerShell

```powershell
# Temporaire
lore completion powershell | Out-String | Invoke-Expression

# Permanent
Add-Content $PROFILE 'lore completion powershell | Out-String | Invoke-Expression'
```

## Vérifier

```
$ lore <TAB>
angela        check-update  completion    config        decision      delete
demo          doctor        hook          init          list          new
pending       release       show          status        upgrade
```

## Exemples

```bash
# Générer pour votre shell et évaluer immédiatement
eval "$(lore completion zsh)"

# Sauvegarder dans un fichier
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
```

Sur macOS, le défaut est Zsh depuis Catalina. Sur la plupart des distros Linux, c'est Bash.

### "Dois-je relancer après une mise à jour de Lore ?"

Uniquement si de nouvelles commandes ont été ajoutées. Après `lore upgrade`, régénérez :

```bash
eval "$(lore completion zsh)"
```

### "compinit: command not found"

Problème spécifique à Zsh. Ajoutez ceci à `~/.zshrc` avant l'eval :

```bash
autoload -Uz compinit && compinit
eval "$(lore completion zsh)"
```

## Tips & Tricks

- **Aliases :** `alias ls='lore show'`, `alias ll='lore list'`, `alias ld='lore doctor'`
- **Fish est le plus simple :** charge automatiquement depuis `~/.config/fish/completions/`
- **Zsh fpath** est plus rapide que `eval` — compilé une fois, pas interprété à chaque démarrage

## Codes de sortie

| Code | Signification |
|------|---------------|
| `0` | Script de complétion généré |
| `1` | Erreur (nom de shell invalide) |

## Voir aussi

- [Guide complétion shell](../getting-started/completions.md) — Configuration détaillée avec dépannage
- [Vue d'ensemble commandes](index.md) — Toutes les commandes disponibles
