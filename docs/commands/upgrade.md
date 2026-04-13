---
type: reference
date: 2026-04-12
status: published
related:
  - check-update.md
  - ../getting-started/installation.md
---
# lore upgrade

Upgrade lore to the latest version — or a specific one.

## Synopsis

```
lore upgrade [flags]
```

## What Does This Do?

`lore upgrade` handles the entire update in one command: it checks for a new release, downloads the correct binary for your OS and architecture, verifies its checksum, and replaces the current binary in place. No manual download or reinstall needed.

## Real World Scenario

> Version 1.2.0 just dropped with Angela improvements. You want to update without leaving your terminal:
>
> ```bash
> lore upgrade
> # v1.0.0 → v1.2.0 — downloaded, verified, installed. Done.
> ```

## How It Works

When you run `lore upgrade`:

1. **Detects the installation method** — Homebrew, `go install`, or direct binary
2. **Checks GitHub Releases** for the newest version
3. **Downloads** the correct archive for your OS and architecture
4. **Verifies the SHA256 checksum** to confirm file integrity
5. **Replaces** the current binary with the new one

> If lore was installed via **Homebrew** or **go install**, it will not self-update. Instead, it prints the correct command (`brew upgrade lore` or `go install ...`) to avoid conflicts with your package manager.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--version` | string | *(latest)* | Target a specific version (e.g. `v1.2.0` or `v1.3.0-beta.1`) |

## Global Flags

| Flag | Type | Description |
|------|------|-------------|
| `--language` | string | Override display language (`en`, `fr`) |
| `--quiet` | bool | Suppress non-essential output |
| `--no-color` | bool | Disable colored output |

## Examples

### Upgrade to Latest (Most Common)

```bash
lore upgrade
# → Checking for updates...
# → New version available: v1.0.0 → v1.2.0
# → Downloading v1.2.0...
# → Verifying checksum...
# → Installing...
# →   Upgraded  Upgraded to v1.2.0.
```

### Upgrade to a Specific Version

```bash
lore upgrade --version v1.1.0
# → Downloads and installs v1.1.0 specifically
```

### Install a Pre-Release

```bash
lore upgrade --version v1.3.0-beta.1
# → Useful for beta testers who need a specific build
```

### Already Up to Date

```bash
lore upgrade
# → Checking for updates...
# → Already up to date (v1.2.0).
```

### Homebrew Installation Detected

```bash
lore upgrade
# → Installed via Homebrew. Run:
# →   brew upgrade lore
```

### Dev Build

```bash
# If built from source without version tags:
lore upgrade
# → Running a dev build — upgrade is only available for release binaries.
```

## Common Questions

### "Can I downgrade to an older version?"

Yes. Use `--version` to target any published release:

```bash
lore upgrade --version v1.0.0
```

### "Is it safe? What if the download fails mid-way?"

The upgrade is atomic. The old binary is only replaced after the new one is fully downloaded and its checksum verified. If anything fails, the old binary stays in place.

### "Does this phone home or check automatically?"

No. `lore upgrade` only runs when you explicitly call it — zero implicit network calls. Use `lore check-update` to check for a new version without installing.

### "What about permissions?"

If lore is installed in a system directory (e.g. `/usr/local/bin`), you may need `sudo`:

```bash
sudo lore upgrade
```

## Tips & Tricks

- **Check first:** `lore check-update` before upgrading to see what changed.
- **Rollback:** `lore upgrade --version v1.0.0` to downgrade if the new version has issues.
- **Homebrew users:** Lore detects Homebrew and tells you to use `brew upgrade lore` instead.
- **Permissions:** If installed in `/usr/local/bin`, you may need `sudo lore upgrade`.
- **Regenerate completions:** After upgrading, re-run `eval "$(lore completion zsh)"` to get completions for new commands.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Upgrade successful, or already up to date |
| `1` | Error (network, checksum, permissions, no releases found) |

## See Also

- [lore check-update](check-update.md) — Check if a newer version is available without installing
- [Installation](../getting-started/installation.md) — First-time installation methods
