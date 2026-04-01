# Completions Shell

Activer la completion par tabulation pour les commandes Lore.

## Bash

```bash
# Ajouter à ~/.bashrc
eval "$(lore completion bash)"
```

## Zsh

```bash
# Ajouter à ~/.zshrc
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

Ou sauvegarder de façon permanente :

```bash
lore completion fish > ~/.config/fish/completions/lore.fish
```

## PowerShell

```powershell
lore completion powershell | Out-String | Invoke-Expression
```

## Vérifier

Après avoir rechargé votre shell, tapez `lore ` et appuyez sur Tab :

```
$ lore <TAB>
angela    config    delete    démo      doctor    hook
init      list      new       pending   release   show      status
```
