---
type: guide
date: 2026-04-12
status: published
related:
  - ../commands/completion.md
  - installation.md
---
# Shell Completions

Enable tab completion for all lore commands, subcommands, and flags.

## Why Set This Up?

Without completions, you must type commands from memory. With completions, a few letters and Tab are enough:

```
lore an<TAB>      → lore angela
lore angela <TAB> → draft  polish  review
lore show --<TAB> → --all  --after  --type  --quiet  --feature ...
```

Completions save keystrokes, prevent typos, and surface commands you might not know exist. Setup takes about 15 seconds.

## Setup by Shell

### Bash

```bash
# Temporary (current session)
eval "$(lore completion bash)"

# Permanent — add to your profile
echo 'eval "$(lore completion bash)"' >> ~/.bashrc
source ~/.bashrc
```

### Zsh (macOS default)

```bash
# Temporary
eval "$(lore completion zsh)"

# Permanent (option 1 — eval in .zshrc)
echo 'eval "$(lore completion zsh)"' >> ~/.zshrc
source ~/.zshrc

# Permanent (option 2 — compiled, faster startup)
lore completion zsh > "${fpath[1]}/_lore"
autoload -Uz compinit && compinit
```

> **Tip:** Option 2 (fpath) is faster because the completion script is compiled once rather than interpreted on every shell startup. Recommended for Zsh power users.

### Fish

```bash
# Temporary
lore completion fish | source

# Permanent (Fish auto-loads from completions directory)
lore completion fish > ~/.config/fish/completions/lore.fish
```

Fish is the simplest shell to configure — no `source` command is needed for the permanent setup.

### PowerShell

```powershell
# Temporary
lore completion powershell | Out-String | Invoke-Expression

# Permanent — add to your profile
Add-Content $PROFILE 'lore completion powershell | Out-String | Invoke-Expression'
```

## Verify It Works

After reloading your shell, type `lore` followed by a space and press Tab:

```
$ lore <TAB>
angela        check-update  completion    config        decision      delete
demo          doctor        hook          init          list          new
pending       release       show          status        upgrade
```

Type `lore show --<TAB>` to see all flags:

```
$ lore show --<TAB>
--all       --after     --bugfix    --decision  --feature
--note      --quiet     --refactor  --type
```

## Pro Tips

### Useful Aliases

Pair completions with short aliases for maximum speed:

```bash
# Add to your shell profile
alias ls='lore show'
alias ll='lore list'
alias ld='lore doctor'
alias ln='lore new'
alias la='lore angela'
alias lp='lore pending'
```

### Troubleshooting

**"Completions don't appear"**

1. Did you reload your shell? (`source ~/.bashrc` or open a new terminal)
2. Is `lore` in your PATH? (`which lore`)
3. For Zsh: is `compinit` called after adding the completion?

**"Completions are slow"**

Use the fpath method (Zsh) or the file method (Fish) instead of `eval`. `eval` regenerates completions on every shell startup, which adds latency.

## See Also

- [lore completion command](../commands/completion.md) — Technical reference
- [Installation](installation.md) — Make sure lore is installed first
