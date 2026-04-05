# lore list

List all documents in your corpus with metadata.

## Synopsis

```
lore list [flags]
```

## What Does This Do?

Shows a table of **every document** in your `.lore/docs/` folder, sorted by date (newest first). Think of it as the table of contents for your project's decision journal.

> **Analogy:** `lore list` is like looking at the index page of a book — you see all chapters at a glance with their dates and types. `lore show` is reading a specific chapter.

## Real World Scenario

> Release day. You need to see everything documented since the last tag. A quick glance at the full corpus:
>
> ```bash
> lore list --type decision
> ```
>
> 12 decisions, sorted by date. You know exactly what changed and why.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type <type>` | string | — | Show only documents of this type (`decision`, `feature`, `bugfix`, `refactor`, `note`, `release`) |
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
note       meeting-api-versioning-2026-03-20      2026-03-20  1 tag
```

## Examples

### Browse by type

```bash
# All decisions — great before architecture reviews
lore list --type decision

# All bugfixes — useful for post-mortems
lore list --type bugfix

# All notes — find that meeting summary
lore list --type note
```

### Scripting

```bash
# Count total documents
lore list --quiet | wc -l

# Extract just filenames
lore list --quiet | cut -f2

# Find documents from March
lore list --quiet | grep "2026-03"

# Feed into a loop
lore list --quiet | while IFS=$'\t' read -r type slug date tags; do
  echo "Processing: $slug"
done
```

### Combined with other commands

```bash
# List → Pick → Read (two-step workflow)
lore list --type decision
# → See: database-selection-2026-02-10
lore show "database"
# → Full document displayed
```

## Tips & Tricks

- **Before a code review:** `lore list --type decision` shows all architectural choices — great context for reviewers.
- **Before a release:** `lore list` shows everything since the project started. Combine with `lore release` to generate notes.
- **Quick count:** `lore list --quiet | wc -l` tells you how many documents you have — useful for the "coverage" mental model.
- **Empty corpus?** `lore list` shows a helpful message with suggestions: "No documents yet. Try `lore new` or make a commit."

## How It Differs from `lore show`

| | `lore list` | `lore show` |
|---|---|---|
| **Purpose** | See ALL documents at a glance | SEARCH for specific documents |
| **Output** | Table with metadata (type, date, tags) | Full document content |
| **Input** | No arguments needed | Requires a keyword |
| **Analogy** | Table of contents | Reading a chapter |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (even if empty — shows helpful message) |
| `1` | Error (`.lore/` not found) |

## See Also

- [lore show](show.md) — Search and read a specific document
- [lore status](status.md) — Health dashboard with statistics
- [lore release](release.md) — Generate release notes from corpus
