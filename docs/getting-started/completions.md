# Shell Completions

Enable tab completion for Lore commands.

## Bash

```bash
# Add to ~/.bashrc
eval "$(lore completion bash)"
```

## Zsh

```bash
# Add to ~/.zshrc
eval "$(lore completion zsh)"
```

Or generate a file:

```bash
lore completion zsh > "${fpath[1]}/_lore"
```

## Fish

```bash
lore completion fish | source
```

Or save permanently:

```bash
lore completion fish > ~/.config/fish/completions/lore.fish
```

## PowerShell

```powershell
lore completion powershell | Out-String | Invoke-Expression
```

## Verify

After reloading your shell, type `lore ` and press Tab:

```
$ lore <TAB>
angela    config    delete    demo      doctor    hook
init      list      new       pending   release   show      status
```
