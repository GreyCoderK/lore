# lore completion

Générer les scripts d'autocomplétion pour le shell.

## Synopsis

```
lore completion <bash|zsh|fish|powershell>
```

## Description

Génère les scripts d'autocomplétion pour le shell spécifié. Active la complétion par tabulation pour toutes les commandes, sous-commandes et flags de Lore.

## Shells supportés

| Shell | Commande |
|-------|----------|
| Bash | `eval "$(lore completion bash)"` |
| Zsh | `eval "$(lore completion zsh)"` |
| Fish | `lore completion fish \| source` |
| PowerShell | `lore completion powershell \| Out-String \| Invoke-Expression` |

## Installation permanente

### Bash
```bash
echo 'eval "$(lore completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh
```bash
echo 'eval "$(lore completion zsh)"' >> ~/.zshrc
# Ou générer vers fpath :
lore completion zsh > "${fpath[1]}/_lore"
```

### Fish
```bash
lore completion fish > ~/.config/fish/completions/lore.fish
```

## Voir aussi

- [Guide d'autocomplétion](../getting-started/completions.md) — Configuration détaillée
