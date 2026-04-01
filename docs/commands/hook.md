# lore hook

Manage the Git post-commit hook.

## Synopsis

```
lore hook <install|uninstall>
```

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `install` | Install the Lore post-commit hook |
| `uninstall` | Remove the Lore post-commit hook |

## Description

Manages the post-commit hook that triggers the documentation flow after each commit. The hook is installed in `.git/hooks/post-commit` (or the `core.hooksPath` location).

## Hook Markers

The hook uses markers for safe coexistence with other hooks:

```bash
# LORE-START
/path/to/lore _hook-post-commit
# LORE-END
```

If `core.hooksPath` is configured, Lore cannot auto-install. It provides the markers for manual insertion.

## Examples

```bash
# Install
lore hook install

# Uninstall
lore hook uninstall

# Check if installed
grep -q "LORE-START" .git/hooks/post-commit && echo "installed"
```

## Tips & Tricks

- `lore init` installs the hook automatically — you rarely need `lore hook install` directly.
- If you use Husky or pre-commit framework, add the Lore markers manually inside your existing hook.
- The hook calls `lore _hook-post-commit` (hidden command) — never call this directly.

## See Also

- [lore init](init.md) — Installs the hook automatically
- [Contextual Detection](../guides/contextual-detection.md) — How the hook decides what to do
