---
type: reference
date: 2026-04-12
status: published
related:
  - status.md
  - ../guides/configuration.md
angela_mode: polish
---
# lore doctor

Diagnose and repair your documentation corpus.

## Synopsis

```
lore doctor [flags]
```

## What Does This Do?

`lore doctor` runs a health checkup on your documentation corpus. It scans for problems — corrupted files, broken references, outdated caches — and fixes most of them automatically.

> **Analogy:** Just as a doctor checks your vitals and prescribes treatment, `lore doctor` assesses corpus health and prescribes `--fix`.

## Real World Scenario

> After merging 3 feature branches, something feels off — `lore show` returns stale results. Time for a checkup:
>
> ```bash
> lore doctor
> # ✗ stale-index (out of sync)
> lore doctor --fix
> # ✓ Fixed: rebuilt index
> ```
>
> Like running `npm audit` or `go vet` — a habit that prevents surprises.

![lore doctor](../assets/vhs/doctor-fix.gif)
<!-- Generate: vhs assets/vhs/doctor-fix.tape -->

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--fix` | bool | `false` | Automatically repair fixable issues |
| `--config` | bool | `false` | Only check `.lorerc` configuration (skip corpus) |
| `--rebuild-store` | bool | `false` | Reconstruct `store.db` from scratch |
| `--prune` | bool | `false` | Run garbage collection on generated artifacts (polish backups, polish.log, corrupt-state quarantine). Mutually exclusive with `--fix`, `--rebuild-store`, `--config`. |
| `--dry-run` | bool | `false` | With `--prune`: report what would be removed without touching the filesystem |
| `--quiet` | bool | `false` | Output only the issue count (or tab-separated `feature\tremoved\tkept\tbytes` with `--prune`) |

## Diagnostic Checks

| Check | What it detects | Auto-fixable? |
|-------|----------------|--------------|
| **orphan-tmp** | Leftover `.tmp` files from interrupted writes | ✅ Yes — deletes them |
| **stale-index** | Index file doesn't match actual documents | ✅ Yes — rebuilds index |
| **stale-cache** | Angela review cache is outdated | ✅ Yes — clears cache |
| **broken-ref** | A document references another that doesn't exist | ❌ No — manual fix |
| **invalid-frontmatter** | YAML metadata can't be parsed | ⚠ Depends on subkind — see below |
| **config** | Typos or invalid values in `.lorerc` | ❌ No — manual fix |

### Invalid frontmatter — subkinds

For `invalid-frontmatter` issues, doctor distinguishes two sub-cases and treats them differently:

| Subkind | What it detects | Auto-fixable? |
|---------|----------------|--------------|
| `missing` | No `---` delimiter at all | ✅ Yes — synthesizes a safe frontmatter block (`type` inferred from filename, `status: draft`, today's date) |
| `malformed` | `---` delimiter present but YAML is unparseable (unclosed quote, broken indent, duplicate keys, BOM + cassé, CRLF + cassé, etc.) | ❌ No — the authentic content may still be recoverable, auto-fix would destroy it |

On `malformed`, doctor emits a suggestion block so you know what to do:

```text
✗  invalid-frontmatter    decision-auth.md (malformed: YAML parse error: ...)
      Suggested actions:
        - Restore from a polish backup:
            lore angela polish --restore 'decision-auth.md'
        - Edit the file manually to repair the YAML block.
```

The `--restore` hint appears only when a polish-backup for the filename actually exists; filenames containing shell metacharacters (spaces, `;`, backticks) are single-quoted so copy-paste into a shell is injection-safe.

This protection is invariant **I31** — *doctor never rewrites a frontmatter block when a `---` delimiter is present*. It guards against a single manual YAML typo destroying authentic content.

## Output

```bash
lore doctor
```

```
Docs Check:
  ✓ orphan-tmp         (none found)
  ✗ stale-index        .lore/docs/index.md (last updated 2026-01-01)
  ✓ broken-ref         (none found)
  ✓ stale-cache        (none found)
  ✓ invalid-frontmatter (none found)

Config Check:
  ✓ .lorerc            (valid)
  ✓ .lorerc.local      (valid, mode 0600)

1 issue found. Run: lore doctor --fix
```

```bash
lore doctor --fix
```

```
  ✓ Fixed: stale-index (rebuilt from 12 documents)

All issues resolved.
```

## Config Validation (`--config`)

Catches common `.lorerc` mistakes:

```bash
lore doctor --config
```

```
Config Check:
  ✗ .lorerc line 3: unknown key "ai.providr"
    → Did you mean "ai.provider"? (Levenshtein distance: 1)
  ✗ .lorerc line 7: "hooks.post_commit" expects boolean, got "yes"
    → Use true/false (YAML boolean), not "yes"/"no"

2 issues found.
```

> **How it suggests corrections:** Lore uses [Levenshtein distance](https://en.wikipedia.org/wiki/Levenshtein_distance) — a measure of how similar two words are. If you type `providr`, it knows you probably meant `provider` (1 character away).

## Prune Generated Artifacts { #prune-generated-artifacts }

`lore doctor --prune` runs garbage collection across every family of growing artifacts Lore produces — bounded disk footprint in one command.

| Family | Pattern | Policy |
|--------|---------|--------|
| `polish-backups` | `polish-backups/*.bak` | Delete backups older than `angela.polish.backup.retention_days` (default 30) |
| `polish-log` | `polish.log` | Two-pass: drop entries older than `angela.polish.log.retention_days` (default 30), then trim oldest until under `angela.polish.log.max_size_mb` (default 10 MB) |
| `corrupt-quarantine` | `*.corrupt-<stamp>` | Delete quarantined state files older than `angela.gc.corrupt_quarantine.retention_days` (default 14). Symlinks and non-regular files are skipped. |

```bash
# Preview what would be removed — no files touched
lore doctor --prune --dry-run

Pruning generated artifacts:
  ✓  polish-backups         removed 12 / kept 3  (14.2 KB)
  ✓  polish-log             removed 87 / kept 412  (68.1 KB)
  ✓  corrupt-quarantine     removed 2 / kept 0  (2.4 KB)
                            84.7 KB total
  (dry-run: no files changed)

# Actually prune
lore doctor --prune
# Same layout, no (dry-run) footer

# Machine output for CI
lore doctor --prune --quiet
# polish-backups<TAB>12<TAB>3<TAB>14540
# polish-log<TAB>87<TAB>412<TAB>69734
# corrupt-quarantine<TAB>2<TAB>0<TAB>2457
```

Invariant **I32** ensures every file family Lore can produce has a registered `Pruner`. A future growing artifact added without a matching pruner fails the I32 regression test — so `--prune` is future-proof by construction.

### Concurrency-safe

The polish-log pruner acquires the same advisory `flock` as the writer (`AppendLogEntry`), and re-stats size + mtime just before the rewrite. If a writer bypasses the lock and appends during the prune, the drift is detected and the prune aborts without data loss.

## Rebuild Store (`--rebuild-store`)

The `store.db` file is a SQLite database that indexes your documents for fast search. It is **always reconstructible** from your Markdown files — they are the source of truth.

```bash
# If store.db gets corrupted or you want a fresh start
lore doctor --rebuild-store
# → Rebuilt store.db from 12 documents and 47 commits
```

> **Safe to run anytime.** The store is a cache, not a source of truth. Rebuilding it loses nothing.

## Process Flow

```mermaid
graph TD
    A[lore doctor] --> B{Mode?}
    B -->|--config| C[Validate .lorerc + .lorerc.local]
    B -->|--prune| P[Run all registered Pruners]
    B -->|default| D[Run all diagnostic checks]
    D --> E{Issues found?}
    E -->|No| F[✓ All good]
    E -->|Yes| G{--fix flag?}
    G -->|Yes| H[Auto-fix what's safe]
    H --> I[Report fixed + manual items]
    G -->|No| J[Report issues + suggest --fix]
    C --> K[Report config issues with suggestions]
    P --> Q{--dry-run?}
    Q -->|Yes| R[Report would-remove counts + bytes]
    Q -->|No| S[Delete + report actual counts]
```

## Examples

```bash
# Full checkup
lore doctor

# Fix everything fixable
lore doctor --fix

# Just check config
lore doctor --config

# Preview prune (no filesystem change)
lore doctor --prune --dry-run

# Actually prune growing artifacts
lore doctor --prune

# Nuclear option: rebuild everything
lore doctor --fix --rebuild-store

# CI gate: fail if unhealthy
[ $(lore doctor --quiet) -eq 0 ] || exit 1
```

## When to Run

| Situation | Run |
|-----------|-----|
| After pulling from remote | `lore doctor` — other people's changes may cause inconsistencies |
| After deleting documents | `lore doctor` — check for broken references |
| After editing `.lorerc` | `lore doctor --config` — catch typos |
| After migration/upgrade | `lore doctor --fix --rebuild-store` — full reset |
| Something feels wrong | `lore doctor --fix` — let Lore figure it out |
| `.lore/` disk footprint growing | `lore doctor --prune` — garbage-collect backups, logs, quarantine |
| Before committing a large polish batch | `lore doctor --prune --dry-run` — preview cleanup |

## Tips & Tricks

- **Make it a habit:** Run `lore doctor` weekly, like you'd run `npm audit` or `go vet`.
- **CI integration:** `lore doctor --quiet` returns the issue count — perfect for CI gates.
- **After team merges:** Pull → `lore doctor --fix` → done. Keeps everyone in sync.
- **Config typos:** The Levenshtein suggestions catch 90% of typos. Trust them.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | No issues (or all fixed with `--fix`) |
| `1` | Issues found (need `--fix` or manual intervention) |
| `4` | Configuration error |

## Common Questions

### "Is `--rebuild-store` safe?"

Yes. `store.db` is a cache reconstructed from your Markdown files. Rebuilding loses nothing — it re-indexes everything from the source of truth.

### "Doctor says 'manual fix required'"

Broken references and malformed frontmatter cannot be auto-fixed because Lore cannot infer the correct value without risking loss of authentic content (invariant I31). Open the flagged file, fix it manually — or run `lore angela polish --restore <file>` if a backup exists — then re-run `lore doctor`.

### "What does `--prune` actually delete?"

Only three families: `polish-backups/*.bak` (polish atomic backups), `polish.log` entries older than retention (or over the size cap), and `*.corrupt-<ts>` quarantined state files. Symlinks and non-regular files are skipped. Your source markdown, `.lorerc`, and `store.db` are never touched by `--prune`.

### "Should I run doctor after every merge?"

Good habit. `lore doctor --fix` takes under a second and catches stale indexes caused by teammates' changes.

## See Also

- [lore status](status.md) — Quick health overview
- [Configuration](../guides/configuration.md) — Fix config issues
