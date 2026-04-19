---
type: reference
date: 2026-04-12
status: published
related:
  - upgrade.md
  - ../getting-started/installation.md
angela_mode: polish
---
# lore check-update

Check if a newer version of Lore is available.

## Synopsis

```
lore check-update
```

## What Does This Do?

The read-only counterpart to `lore upgrade`. It checks GitHub for newer releases and shows what's available — without downloading or installing anything.

**In plain terms:** "Am I behind? What versions are out there?"

> Lore never checks for updates automatically. This command is the only way to find out — fully opt-in, zero implicit network calls.

## Real World Scenario

> Before starting a big refactor, you want to make sure you're on the latest version:
>
> ```bash
> lore check-update
> # Current: v1.0.0 — Latest: v1.2.0
> # Run: lore upgrade
> ```

## Flags

This command takes no flags and no arguments. It checks against the latest GitHub Release.

## How It Works

1. Fetches recent releases from GitHub (including pre-releases)
2. Compares your current version against what's available
3. Lists all newer versions, with `(pre-release)` labels where applicable

## Examples

### Newer Versions Available

```bash
lore check-update
# → Checking for updates...
# → Update available: v1.0.0 → v1.3.0-beta.1
# →
# →   v1.3.0-beta.1        (pre-release)
# →   v1.2.1
# →   v1.2.0
# →   v1.1.0
# →
# → Run: lore upgrade
```

### Already Up to Date

```bash
lore check-update
# → Checking for updates...
# → Up to date (v1.3.0).
```

### Dev Build

```bash
lore check-update
# → Checking for updates...
# → Update available: dev → v1.2.0
# →
# →   v1.2.0
# →   v1.1.0
# →   v1.0.0
# →
# → Run: lore upgrade
```

> On dev builds, `check-update` still works — `dev` is always treated as older than any published release, so all releases are shown.

## Common Questions

### "Does this make network calls?"

Yes — one GET request to the GitHub Releases API. This happens only when **you** run the command. Lore never checks for updates in the background or during other commands.

### "How do I install a specific version from the list?"

Use the `--version` flag on `lore upgrade`:

```bash
lore check-update        # See what's available
lore upgrade --version v1.2.0  # Install a specific one
```

### "Why doesn't `lore status` show this?"

By design. `lore status` is fully offline and never makes network calls. Update checks are explicitly opt-in via this command.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (whether up to date or updates available) |
| `1` | Error (network failure, no releases found) |

## Tips & Tricks

- **Before big refactors:** Confirm you're on the latest version to get the newest features and fixes.
- **Pair with upgrade:** Run `lore check-update` to see what's available, then `lore upgrade` when ready.
- **No automatic checks:** Lore never phones home. This is the only way to know if you're behind — fully opt-in.
- **Dev builds:** If you built from source without version tags, `check-update` still works — it compares against published releases.

## See Also

- [lore upgrade](upgrade.md) — Download and install a newer version
- [Installation](../getting-started/installation.md) — First-time installation methods
