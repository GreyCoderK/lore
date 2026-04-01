# Completions Shell

Activer la completion par tabulation pour les commandes Lore.

## Bash

```bash
# Ajouter a ~/.bashrc
eval "$(lore completion bash)"
```

## Zsh

```bash
# Ajouter a ~/.zshrc
eval "$(lore completion zsh)"
```

Ou générer un fichier :

```bash
lore completion zsh > "${fpath[1]}/_lore"
```

## Fish

```bash
lore completion fish | source
```

Ou sauvegarder de facon permanente :

```bash
lore completion fish > ~/.config/fish/completions/lore.fish
```

## PowerShell

```powershell
lore completion powershell | Out-String | Invoke-Expression
```

## Verifier

Apres avoir recharge votre shell, tapez `lore ` et appuyez sur Tab :

```
$ lore <TAB>
angela    config    delete    demo      doctor    hook
init      list      new       pending   release   show      status
```
