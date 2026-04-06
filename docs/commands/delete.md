# lore delete

Delete a document from the corpus.

## Synopsis

```
lore delete <filename> [flags]
```

## What Does This Do?

Removes a documentation file from `.lore/docs/`. Lore asks for confirmation before deleting (safety first).

> **Analogy:** Tearing a page out of your project journal. Lore makes sure you really want to, because once it's gone, the "why" is lost.

## Real World Scenario

> You refactored the auth system completely. The old document "session-based-auth-2026-01.md" is now misleading — it describes an approach you abandoned. Time to clean up:
>
> ```bash
> lore delete session-based-auth-2026-01.md
> ```
>
> Lore warns you that another document references it, asks for confirmation. You delete and run `lore doctor` to clean up.

![lore delete](../assets/vhs/delete.gif)
<!-- Generate: vhs assets/vhs/delete.tape -->

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

## Examples

```bash
# Find the filename first
lore list
# → decision  database-selection-2026-02-10.md  2026-02-10

# Delete with confirmation
lore delete database-selection-2026-02-10.md
# → decision — Database Selection (2026-02-10)
# → Delete? [y/N] y
# → Deleted

# Force delete in scripts
lore delete old-doc-2025-01-01.md --force
```

## Common Questions

### "Should I delete or archive?"

**Prefer archiving.** Edit the document and change `status: active` to `status: archived`. This preserves the historical record. Delete only when the document is truly wrong or harmful.

### "I deleted a document but another doc references it"

Run `lore doctor` — it will flag the broken reference. You can then edit the referencing document to remove or update the link.

### "Can I undo a delete?"

If you have not committed yet: `git checkout -- .lore/docs/filename.md`. If you have committed: `git show HEAD~1:.lore/docs/filename.md > .lore/docs/filename.md`. The document is just a file — Git is your undo button.

## See Also

- [lore list](list.md) — Find the filename to delete
- [lore doctor](doctor.md) — Check for broken references after deletion
