# lore angela draft

Zero-API structural analysis of your documents — no internet required.

## Synopsis

```
lore angela draft [filename] [flags]
```

## What Does This Do?

`lore angela draft` is like having a writing coach review your document — except this coach works **entirely offline**. It checks structure, style, and consistency without making any network calls or needing an API key.

> **Analogy:** Imagine a spell-checker, but instead of checking spelling, it checks: "Did you explain *why*? Did you mention alternatives? Is this consistent with your other documents?" All locally, all free.

## Real World Scenario

> Before pushing your PR, you want to check that the 3 new documents you created are well-structured — without spending API credits:
>
> ```bash
> lore angela draft --all
> # 2 docs need attention, 1 is great
> ```
>
> Free, offline, instant. Fix the issues, THEN polish with AI.

![lore angela draft](../assets/vhs/angela-draft-polish.gif)
<!-- Generate: vhs assets/vhs/angela-draft-polish.tape -->

## Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `filename` | No | Specific document to analyze (default: most recent) |

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | Analyze every document in the corpus |
| `--verbose`, `-v` | bool | `false` | With `--all`: print every suggestion inline (default shows warnings only) |
| `--path` | string | `.lore/docs` | Path to a markdown directory (standalone mode — no `lore init` required) |

## Standalone Mode

Angela can analyze **any directory of Markdown files**, even without `lore init`:

```bash
# Analyze docs in an external project
lore angela draft --all --path ./my-project/docs

# Single file in a custom directory
lore angela draft --path ./wiki api-guide.md
```

In standalone mode:
- Files **with** YAML front matter get full analysis (type, tags, scope)
- Files **without** front matter get synthetic metadata (type=note, tags from filename)
- No `.lorerc` needed — sensible defaults apply
- VHS tape/doc cross-checks run if an `assets/vhs/` directory is found

This makes Angela usable as a **CI quality gate** on any Markdown documentation. See the [Angela in CI](../guides/angela-ci.md) guide.

## What It Checks

| Category | What it looks for | Example finding |
|----------|-------------------|-----------------|
| **Structure** | Missing sections (Why, Alternatives, Impact) | "Missing 'Alternatives Considered' section" |
| **Style** | Passive voice, vague language, tone issues | "Passive voice overused in Why section" |
| **Coherence** | Contradictions or connections with other docs | "Related: feature-add-auth-2026-02-15.md" |
| **Completeness** | Empty or too-short sections | "Why section is only 5 words — consider expanding" |

## Output (Single Document)

```bash
lore angela draft decision-database-2026-02-10.md
```

```
lore angela draft — decision-database-2026-02-10.md
  Reviewed by: Sialou + Doumbia  (relevance: 7)

  error    structure       Missing "Impact" section — decisions should describe consequences
  warning  tone            "We just picked PostgreSQL" — avoid "just", it undermines the decision
  info     coherence       Related: feature-user-model-2026-02-12.md (mentions same schema)

3 suggestions
```

### Understanding Severity

| Severity | Meaning | Action |
|----------|---------|--------|
| **error** | Something important is missing | Fix before considering the doc "done" |
| **warning** | Could be better | Improve when you have time |
| **info** | Informational — connections and context | Good to know, no action needed |

## Output (Corpus-wide `--all`)

```bash
lore angela draft --all
```

By default, Angela prints the summary line for every document and the inline
detail of every **warning** (blockers you should act on):

```
lore angela draft --all — 12 documents

  B    review   decision-database-2026-02-10.md      3 suggestions (2 warnings)
         warning  structure      Missing "Impact" section
         warning  completeness   "Why" section is only 5 words
  C    review   feature-rate-limit-2026-03-16.md      1 suggestions
  A    ok       refactor-extract-auth-2026-03-01.md
  A    ok       feature-add-jwt-2026-02-15.md
  ...

2/12 documents need attention. 4 total suggestions.
Run with --verbose (-v) to see every suggestion.
```

### `--verbose` / `-v`

To see every suggestion (info, warning, error), pass `-v`:

```bash
lore angela draft --all -v
```

```
  B    review   decision-database-2026-02-10.md      3 suggestions (2 warnings)
         warning  structure      Missing "Impact" section
         warning  completeness   "Why" section is only 5 words
         info     coherence      Related: feature-user-model-2026-02-12.md
```

## Process Flow

```mermaid
graph TD
    A[lore angela draft] --> B{--all?}
    B -->|No| C[Load single document]
    B -->|Yes| D[Load entire corpus]
    C --> E[Parse front matter + content]
    D --> E
    E --> F[Check structure: sections present?]
    F --> G[Check style: tone, clarity, length]
    G --> H[Check coherence: cross-references]
    H --> I[Score with personas]
    I --> J[Generate suggestions]
    J --> K[Display report]
```

## Common Questions

### "Do I need an API key for this?"

**No.** `angela draft` is 100% local. It uses built-in rules and heuristics, not AI. Think of it as a sophisticated linter for documentation.

### "What's the difference between `draft` and `polish`?"

| | `angela draft` | `angela polish` |
|---|---|---|
| **Network** | None (offline) | 1 API call |
| **Cost** | Free | Uses API credits |
| **What it does** | Points out problems | Rewrites the document |
| **Output** | Suggestions list | Interactive diff |

> **Best practice:** Always run `draft` first (free), fix the easy issues, then `polish` (costs credits) for the finishing touch.

### "What are 'personas'?"

Angela uses 6 virtual reviewers with different perspectives. The top 3 are activated based on document type and content:

| Persona | Icon | Focus |
|---------|------|-------|
| **Affoue** (Storyteller) | 📖 | Narrative clarity, "Why" sections |
| **Sialou** (Tech Writer) | ✏️ | Technical precision, structure |
| **Kouame** (QA Reviewer) | 🔍 | Validation criteria, edge cases |
| **Doumbia** (Architect) | 🏗️ | Trade-offs, system design |
| **Gougou** (UX Designer) | 🎨 | User empathy, accessibility |
| **Beda** (Business Analyst) | 📊 | Business value, requirements |

Each persona runs local checks and produces typed suggestions. For example, Affoue checks that the "Why" section tells a story rather than just listing bullets. Kouame checks that claims have verification criteria.

## Tips & Tricks

- **Before every PR:** Run `lore angela draft --all` to catch quality issues.
- **Run `draft` before `polish`:** Fix free issues first, then spend API credits on polish.
- **The score is relative:** 7/10 is good, 9/10 is excellent. Don't aim for 10/10 on every doc.
- **Customize style rules** in `.lorerc` under `angela.style_guide` for team conventions.

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (even if suggestions found) |
| `1` | Error (`.lore/` not found, file not found) |

## See Also

- [lore angela polish](angela-polish.md) — AI-assisted rewrite (next step)
- [lore angela review](angela-review.md) — Corpus-wide coherence via AI
- [Document Types](../guides/document-types.md) — What sections each type expects
