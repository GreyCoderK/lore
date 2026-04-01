# lore release

Generate release notes from documented commits.

## Synopsis

```
lore release [flags]
```

## Description

Collects all documents between two Git refs and generates release notes grouped by type. Updates `CHANGELOG.md` and saves a release document in `.lore/docs/`.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--from` | string | Latest tag | Start of commit range (tag or SHA) |
| `--to` | string | HEAD | End of commit range |
| `--version` | string | — | Version label for the release notes |
| `--quiet` | bool | `false` | Output only the file path |

## Output

Generates `release-<version>-<date>.md` in `.lore/docs/` with:

```markdown
# Release v1.2.0 (2026-03-16)

## Features
- Add rate limiting to API endpoints
- Add user authentication middleware

## Bug Fixes
- Fix token refresh race condition

## Decisions
- Switch to PostgreSQL for data persistence
```

Also updates:
- `CHANGELOG.md` header with new version
- `.lore/releases.json` metadata

## Examples

```bash
# Release notes since last tag
lore release --version v1.2.0

# Between two tags
lore release --version v1.2.0 --from v1.1.0

# Quiet: just output the file path
lore release --version v1.2.0 --quiet
# → .lore/docs/release-v1.2.0-2026-03-16.md
```

## Tips & Tricks

- Run before `git tag` to include the release notes in the tagged commit.
- The release document becomes part of the corpus — searchable with `lore show --type release`.
- Pair with GoReleaser: the generated `CHANGELOG.md` feeds into `goreleaser release`.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Release notes generated |
| `1` | Error (no documents in range, no tags found) |

## See Also

- [lore list](list.md) — See all documents
- [lore angela review](angela-review.md) — Corpus coherence check before release
