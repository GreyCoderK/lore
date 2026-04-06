# lore show

Search and display documents from your corpus.

## Synopsis

```
lore show [keyword] [flags]
```

## What Does This Do?

`lore show` is how you **read** your documentation. Give it a keyword, and it finds matching documents. Think of it like a search engine for your project's decision history.

> **Analogy:** If `.lore/docs/` is your project's diary, `lore show` is like flipping through it and finding the page where you wrote about "authentication".

## Real World Scenario

> Code review. The reviewer asks: "Why JWT instead of sessions?" Instead of digging through Slack, you search your corpus:
>
> ```bash
> lore show "JWT"
> ```
>
> 3 seconds later, the full reasoning is on screen — written the day the decision was made.

![lore show](../assets/vhs/show-search.gif)
<!-- Generate: vhs assets/vhs/show-search.tape -->

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `keyword` | Yes (unless `--all`) | Search term — matches titles and content |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--type <type>` | string | — | Show only documents of this type |
| `--after <date>` | string | — | Show documents after this date (`YYYY-MM` or `YYYY-MM-DD`) |
| `--all` | bool | `false` | Show all documents (prefer `lore list` for listing) |
| `--quiet` | bool | `false` | Machine-readable output (tab-separated) |
| `--feature` | bool | — | Shorthand for `--type feature` |
| `--decision` | bool | — | Shorthand for `--type decision` |
| `--bugfix` | bool | — | Shorthand for `--type bugfix` |
| `--refactor` | bool | — | Shorthand for `--type refactor` |
| `--note` | bool | — | Shorthand for `--type note` |

> **Note:** Type shorthands are mutually exclusive — you can't use `--feature` and `--decision` at the same time.

## How Search Works

Lore searches through **titles** and **content** of all documents in `.lore/docs/`. It's not a fuzzy search (yet) — it matches exact keywords.

### One Result → Shows It Directly

```bash
lore show "JWT auth"
```

```markdown
---
type: decision
date: 2026-02-15
commit: b2c3d4e
---
# Add JWT auth middleware

## Why
Stateless authentication scales better than sessions...
```

### Multiple Results → Interactive Selection (TTY)

```bash
lore show "auth"
```

```
  1  decision  Add JWT auth middleware        2026-02-15
  2  refactor  Extract auth middleware        2026-03-01
  3  feature   Add OAuth2 provider           2026-03-10

Select [1-3]:
```

### Multiple Results → List (Non-TTY / Quiet)

When piped or with `--quiet`, output is tab-separated (for scripting):

```bash
lore show "auth" --quiet
# decision	Add JWT auth middleware	2026-02-15
# refactor	Extract auth middleware	2026-03-01
```

## Examples

### Basic Search

```bash
# Find documents about "database"
lore show "database"

# Find all decisions
lore show "api" --decision

# Find recent documents
lore show "rate" --after 2026-03
```

### Combining with Other Commands

```bash
# Find and pipe to less
lore show "auth" --quiet | less

# Count decisions about a topic
lore show "auth" --decision --quiet | wc -l

# Export a document
lore show "JWT auth" > auth-decision.md
```

## Common Questions

### "No results — what am I doing wrong?"

- Check spelling — search is exact, not fuzzy
- Try broader terms: "auth" instead of "authentication middleware"
- Check if documents exist: `lore list`

### "How is this different from `lore list`?"

| Command | Purpose |
|---------|---------|
| `lore list` | Show ALL documents with metadata (type, date, tags) |
| `lore show` | **Search** for specific documents by keyword and display content |

Think of it as: `lore list` = table of contents, `lore show` = reading a specific chapter.

## Tips & Tricks

- **Pipe-friendly:** `lore show "auth" --quiet | less` for paging through results.
- **Export:** `lore show "JWT auth" > auth-decision.md` saves a document to a file.
- **Combine with grep:** `lore show "api" --quiet | grep decision` — filter results.
- **No results?** Try broader terms. Search is exact (not fuzzy yet — fuzzy coming in Circle 4).
- **Type shorthands:** `--decision` is faster to type than `--type decision`.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Match found |
| `2` | No matches (not an error — just nothing found) |
| `3` | No keyword provided |

## See Also

- [lore list](list.md) — Browse all documents
- [lore status](status.md) — Corpus statistics and health
