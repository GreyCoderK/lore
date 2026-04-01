# lore delete

Delete a document from the corpus.

## Synopsis

```
lore delete <filename> [flags]
```

## What Does This Do?

Removes a documentation file from `.lore/docs/`. Lore asks for confirmation before deleting (safety first).

> **Analogy:** Tearing a page out of your project journal. Lore makes sure you really want to, because once it's gone, the "why" is lost.

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `filename` | Yes | The exact filename (e.g., `decision-auth-strategy-2026-03-07.md`) |

> **How to find the filename:** Run `lore list` to see all documents with their slugs/filenames.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--force` | bool | `false` | Delete without asking (for scripts) |

## What Happens

### Interactive (Default)

```bash
lore delete decision-auth-strategy-2026-03-07.md
```

```
  decision — Switch to PostgreSQL (2026-03-07)
  ⚠ Referenced by: feature-add-user-model-2026-03-08.md

  Delete this document? [y/N] y
  ✓ Deleted
```

Lore shows you:
1. **What you're deleting** — type, title, date
2. **References** — other docs that mention this one (so you know what might break)
3. **Confirmation** — you must type `y` to proceed

### Force Mode (Scripts/CI)

```bash
lore delete decision-auth-strategy-2026-03-07.md --force
# → Deleted immediately, no questions asked
```

### Safety Rules

| Scenario | Behavior |
|----------|----------|
| **Normal (TTY)** | Asks for confirmation |
| **Pipe/Non-TTY (without `--force`)** | Error — won't auto-delete in scripts |
| **Demo documents** | No confirmation needed (they're just demos) |
| **File not found** | Friendly error with suggestion |

## Tips & Tricks

- **Prefer archiving over deleting:** Edit the file and set `status: archived` instead. This preserves the historical record.
- **Bulk cleanup:** `lore list --type note --quiet | xargs -I{} lore delete {} --force` (careful!).
- **After deleting:** Run `lore doctor` to check for broken references.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Deleted successfully |
| `1` | Error (not found, non-TTY without `--force`) |

## See Also

- [lore list](list.md) — Find the filename to delete
- [lore doctor](doctor.md) — Check for broken references after deletion
