---
type: reference
date: 2026-04-12
status: published
related:
  - init.md
  - ../guides/contextual-detection.md
  - doctor.md
---
# lore hook

Manage the Git post-commit hook that triggers Lore's documentation flow.

## Synopsis

```
lore hook <install|uninstall>
```

## What Does This Do?

After every `git commit`, the post-commit hook runs automatically and triggers the question flow. `lore hook` lets you install or remove this hook manually.

> **Analogy:** The hook is like a smoke detector — install it once and it activates when needed. `lore hook install` mounts it; `lore hook uninstall` takes it down.

Most users never need this command — `lore init` installs the hook automatically.

## Real World Scenario

> Your team uses Husky for pre-commit linting. You want to add Lore's post-commit hook without breaking the existing setup:
>
> ```bash
> lore hook install
> # Lore adds its section with markers — your Husky hooks stay untouched
> ```

![lore hook](../assets/vhs/hook.gif)
<!-- Generate: vhs assets/vhs/hook.tape -->

## Subcommands

### `lore hook install`

Installs the post-commit hook in `.git/hooks/post-commit` (or the `core.hooksPath` location).

**What it does:**

1. Checks if `.git/hooks/post-commit` exists
2. If it exists: adds Lore's section between `# LORE-START` and `# LORE-END` markers
3. If it doesn't exist: creates the file with Lore's hook
4. Makes the file executable (`chmod +x`)

**Coexistence with other hooks:**

```bash
#!/bin/bash
# Your existing hook code here
npm run lint-staged

# LORE-START
if (: < /dev/tty) 2>/dev/null; then
  exec lore _hook-post-commit < /dev/tty
else
  exec lore _hook-post-commit
fi
# LORE-END
```

The `< /dev/tty` redirect reconnects stdin from the terminal, which Git closes for hooks. The fallback ensures silent operation in CI/Docker environments where `/dev/tty` is unavailable.

The markers ensure Lore only modifies its own section — your other hooks are never touched.

### `lore hook uninstall`

Removes Lore's section from the hook file. If Lore was the only content, removes the file entirely.

## Edge Cases

### `core.hooksPath` configured

If Git is configured to use a custom hooks directory (common in monorepos), Lore can't auto-install:

```bash
lore hook install
# → Warning: core.hooksPath is set to /path/to/hooks
# → Add these lines to your post-commit hook manually:
# →   # LORE-START
# →   /usr/local/bin/lore _hook-post-commit
# →   # LORE-END
```

### Hook already installed

```bash
lore hook install
# → Hook already installed (idempotent — safe to run multiple times)
```

## Common Questions

### "What is `_hook-post-commit`?"

It is an internal command used by the hook file to run the Decision Engine, contextual detection, and question flow. Do not call it directly.

### "The hook isn't triggering after my commits"

Check these in order:

1. Is the hook installed? `grep "LORE" .git/hooks/post-commit`
2. Is the hook executable? `ls -la .git/hooks/post-commit` (should show `-rwx`)
3. Is `lore` in your PATH? `which lore`
4. Is `core.hooksPath` overriding? `git config core.hooksPath`

### "Can I temporarily disable the hook?"

Yes, three ways:

```bash
# 1. Skip one commit
git commit -m "quick fix [doc-skip]"

# 2. Uninstall and reinstall later
lore hook uninstall
# ... commits without Lore ...
lore hook install

# 3. Git's built-in skip (skips ALL hooks)
git commit --no-verify -m "emergency fix"
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (can't write to hooks directory) |

## Examples

```bash
# Install the hook
lore hook install
# → ✓ Post-commit hook installed

# Verify installation
cat .git/hooks/post-commit
# → #!/bin/bash
# → # LORE-START
# → /usr/local/bin/lore _hook-post-commit
# → # LORE-END

# Uninstall
lore hook uninstall
# → ✓ Post-commit hook removed

# Check if installed (scripting)
grep -q "LORE-START" .git/hooks/post-commit 2>/dev/null && echo "installed" || echo "not installed"
```

## Tips & Tricks

- **You rarely need this:** `lore init` installs the hook automatically.
- **Husky/pre-commit users:** Lore uses markers (`# LORE-START` / `# LORE-END`) and never touches other hooks.
- **Temporary disable:** `lore hook uninstall` then `lore hook install` when ready. Or use `[doc-skip]` in commit messages.
- **Monorepo tip:** If `core.hooksPath` is set, follow the manual instructions Lore provides.

## See Also

- [lore init](init.md) — Installs the hook automatically
- [Contextual Detection](../guides/contextual-detection.md) — How the hook decides what to do
- [lore doctor](doctor.md) — Diagnose hook issues
