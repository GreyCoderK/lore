# lore list

List all documents in your corpus with metadata.

## Synopsis

```
lore list [flags]
```

## What Does This Do?

Shows a table of **every document** in your `.lore/docs/` folder, sorted by date (newest first). Think of it as the table of contents for your project's decision journal.

> **Analogy:** `lore list` is like looking at the index page of a book — you see all chapters at a glance with their dates and types.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type <type>` | string | — | Show only documents of this type |
| `--quiet` | bool | `false` | Tab-separated output for scripting |

## Output

```bash
lore list
```

```
TYPE       SLUG                                  DATE        TAGS
decision   database-selection-2026-02-10          2026-02-10  2 tags
feature    add-jwt-auth-2026-02-15                2026-02-15  3 tags
feature    add-rate-limiting-2026-03-16           2026-03-16  1 tag
refactor   extract-auth-middleware-2026-03-01     2026-03-01  0 tags
```

## Examples

```bash
# All documents
lore list

# Only decisions
lore list --type decision

# Count total documents
lore list --quiet | wc -l

# Extract just filenames (for scripting)
lore list --quiet | cut -f2
```

## Tips & Tricks

- **Before a code review:** `lore list --type decision` shows all architectural choices — great context for reviewers.
- **Before a release:** `lore list --after 2026-03` shows everything documented since the last release.
- **Quick count:** `lore list --quiet | wc -l` tells you how many documents you have.

## See Also

- [lore show](show.md) — Search and read a specific document
- [lore status](status.md) — Health dashboard with statistics
