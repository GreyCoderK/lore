# lore completion

Generate shell completion scripts.

## Synopsis

```
lore completion <bash|zsh|fish|powershell>
```

## Description

Generates auto-completion scripts for the specified shell. Enables tab-completion for all Lore commands, subcommands, and flags.

## Supported Shells

| Shell | Command |
|-------|---------|
| Bash | `eval "$(lore completion bash)"` |
| Zsh | `eval "$(lore completion zsh)"` |
| Fish | `lore completion fish \| source` |
| PowerShell | `lore completion powershell \| Out-String \| Invoke-Expression` |

## Permanent Installation

### Bash
```bash
echo 'eval "$(lore completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh
```bash
echo 'eval "$(lore completion zsh)"' >> ~/.zshrc
# Or generate to fpath:
lore completion zsh > "${fpath[1]}/_lore"
```

### Fish
```bash
lore completion fish > ~/.config/fish/completions/lore.fish
```

## See Also

- [Shell Completions guide](../getting-started/completions.md) — Detailed setup
