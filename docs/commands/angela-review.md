# lore angela review

Corpus-wide coherence analysis via AI.

## Synopsis

## What Does This Do?

`lore angela review` is the "big picture" analysis. While `angela draft` checks one document, `review` checks your **entire corpus** for coherence — contradictions between documents, isolated docs with no connections, stale content, and coverage gaps.

> **Analogy:** If `angela draft` is a teacher grading one essay, `angela review` is the dean reviewing the entire curriculum for consistency.


```
lore angela review [flags]
```

## Description

Analyzes the entire documentation corpus for coherence: contradictions between documents, isolated documents, stale content, coverage gaps. Combines local pre-analysis (signals) with a single AI API call.

**Requires** an AI provider configured.

## Real World Scenario

> The team has been documenting for 2 weeks. 15 documents in the corpus. Before the sprint review, you want to check consistency:
>
> ```bash
> lore angela review
> # 1 contradiction found: auth-jwt.md vs auth-session.md
> # 2 isolated documents with no cross-references
> ```
>
> You catch the contradiction before it confuses a new team member.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--local` | bool | `false` | Local signals only (no AI call) |
| `--quiet` | bool | `false` | Suppress header and summary on stderr |

## Output

```
Corpus Review — 12 documents analyzed

SEVERITY  TITLE                            DOCUMENTS                    DESCRIPTION
error     Contradictory auth approach       auth-jwt.md, auth-session.md  JWT chosen in one, sessions in another
warning   Isolated document                 note-meeting-2026-03-01.md    No references to/from other docs
info      Coverage gap                      —                            No decisions documented for database layer

3 findings (1 error, 1 warning, 1 info)
```

## Process Flow

```mermaid
graph TD
    A[lore angela review] --> B[Load all documents]
    B --> C[Local pre-analysis: signals]
    C --> D{--local?}
    D -->|Yes| E[Display local findings]
    D -->|No| F[Prepare summaries for AI]
    F --> G[Single API call with corpus context]
    G --> H[Parse AI findings]
    H --> I[Merge local + AI findings]
    I --> J[Display report]
    J --> K[Cache results for lore status]
```

## Local Signals (always computed)

Pre-analysis without API calls:
- **Contradictions** — Documents about the same topic with conflicting content
- **Isolated docs** — No cross-references to/from other documents
- **Stale content** — Documents older than N days without updates

## Examples

```bash
# Full review (local + AI)
lore angela review

# Local signals only (free, no API)
lore angela review --local

# Quiet (for integration with lore status)
lore angela review --quiet
```

## Tips & Tricks

- Run before every release: `lore angela review` catches contradictions that would confuse readers.
- `--local` is free and fast — use it as a daily check.
- Results are cached: `lore status` shows review findings without re-running.
- Large corpus (> 50 docs): Lore warns about token usage before the API call.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (no provider configured, corpus too small) |

## Common Questions

### "How is this different from angela draft?"

| | `angela draft` | `angela review` |
|---|---|---|
| **Scope** | One document | Entire corpus |
| **Cost** | Free (zero-API) | 1 API call (or free with --local) |
| **Finds** | Missing sections, style issues | Contradictions, isolated docs, coverage gaps |

### "How often should I run this?"

Before every release, or every 1-2 weeks during active development. Results are cached — `lore status` shows the latest findings without re-running.

### "My corpus has 200+ documents. Will this be expensive?"

One API call regardless of corpus size. Lore compresses document summaries before sending. For very large corpora (50+ docs), Lore warns you about token usage before proceeding.

## See Also

- [lore angela draft](angela-draft.md) — Single document analysis
- [lore status](status.md) — Shows cached review findings
