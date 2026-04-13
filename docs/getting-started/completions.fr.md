---
type: guide
date: 2026-04-12
status: published
related:
  - ../commands/completion.md
  - installation.md
---
# Complétions Shell

Activez la complétion par tabulation pour toutes les commandes, sous-commandes et flags de lore.

## Pourquoi configurer ceci ?

Sans complétions, vous tapez les commandes complètes de mémoire. Avec les complétions, vous tapez quelques lettres et appuyez sur Tab :

```
lore an<TAB>     → lore angela
lore angela <TAB> → draft  polish  review
lore show --<TAB> → --all  --after  --type  --quiet  --feature ...
```

Les complétions économisent des frappes, évitent les fautes de frappe, et révèlent des commandes que vous ne connaissiez pas. Configuration en 15 secondes.

## Configuration par shell

### Bash

```bash
# Temporaire (session courante)
eval "$(lore completion bash)"

# Permanent — ajouter à votre profil
echo 'eval "$(lore completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh (défaut macOS)

```bash
# Temporaire
eval "$(lore completion zsh)"

# Permanent (option 1 — eval dans .zshrc)
echo 'eval "$(lore completion zsh)"' >> ~/.zshrc
source ~/.zshrc

# Permanent (option 2 — compilé, démarrage plus rapide)
lore completion zsh > "${fpath[1]}/_lore"
autoload -Uz compinit && compinit
```

> **Astuce :** L'option 2 (fpath) est plus rapide car la complétion est compilée une fois, pas interprétée à chaque démarrage du shell. Recommandé pour les utilisateurs Zsh avancés.

### Fish

```bash
# Temporaire
lore completion fish | source

# Permanent (Fish charge automatiquement depuis le dossier completions)
lore completion fish > ~/.config/fish/completions/lore.fish
```

Fish est le plus simple — pas besoin de commande `source` pour la configuration permanente.

### PowerShell

```powershell
# Temporaire
lore completion powershell | Out-String | Invoke-Expression

# Permanent — ajouter à votre profil
Add-Content $PROFILE 'lore completion powershell | Out-String | Invoke-Expression'
```

## Vérifier le fonctionnement

Après avoir rechargé votre shell, tapez `lore` suivi d'un espace et appuyez sur Tab :

```
$ lore <TAB>
angela        check-update  completion    config        decision      delete
demo          doctor        hook          init          list          new
pending       release       show          status        upgrade
```

Tapez `lore show --<TAB>` pour voir les flags :

```
$ lore show --<TAB>
--all       --after     --bugfix    --decision  --feature
--note      --quiet     --refactor  --type
```

## Tips & Tricks

### Aliases utiles

Combinez les complétions avec des aliases pour un maximum de vitesse :

```bash
# Ajouter à votre profil shell
alias ls='lore show'
alias ll='lore list'
alias ld='lore doctor'
alias ln='lore new'
alias la='lore angela'
alias lp='lore pending'
```

### Dépannage

**"Les complétions n'apparaissent pas"**

1. Avez-vous rechargé votre shell ? (`source ~/.bashrc` ou ouvrir un nouveau terminal)
2. `lore` est-il dans votre PATH ? (`which lore`)
3. Pour Zsh : `compinit` est-il appelé après l'ajout de la complétion ?

**"Les complétions sont lentes"**

Utilisez la méthode fpath (Zsh) ou fichier (Fish) au lieu de `eval`. `eval` régénère les complétions à chaque démarrage du shell.

## Voir aussi

- [Commande lore completion](../commands/completion.md) — Référence technique
- [Installation](installation.md) — Vérifiez que lore est installé en premier
