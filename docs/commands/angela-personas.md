---
type: reference
date: "2026-04-17"
status: published
related:
    - angela-draft.md
    - angela-polish.md
    - angela-review.md
    - angela-consult.md
angela_mode: reference
---
# Angela Personas

Angela evaluates your documentation through **7 distinct expert lenses** — each with its own priorities, blind spots, and signature questions. Together they cover the axes along which technical docs fail: unclear prose, missing user mental models, incomplete contracts, untested assumptions, narrative drift, business misalignment, and weak structure.

Personas are available in three activation modes: `angela consult` for a single-persona deep dive, `angela polish --persona` to steer a rewrite, and `angela review --persona` to inject lenses into corpus-wide coherence analysis.

## The Seven

| Icon | ID | Name | Focus |
|---|---|---|---|
| ✏️ | `tech-writer` | Sialou | Technical writing precision and clarity |
| 🎨 | `ux-designer` | Gougou | User empathy, mental models, and accessibility |
| 🔌 | `api-designer` | Ouattara | API contracts, synthesizer-ready docs, HTTP semantics |
| 🔍 | `qa-reviewer` | Kouame | Quality assurance and validation criteria |
| 🏗️ | `architect` | Doumbia | System design, trade-offs, and scalability |
| 📊 | `business-analyst` | Beda | Requirements traceability and business value |
| 📖 | `storyteller` | Affoue | Narrative clarity and authentic storytelling |

The display names (Sialou, Gougou, Ouattara, Kouame, Doumbia, Beda, Affoue) are common Ivorian given names. Lore is built in Côte d'Ivoire and the project embraces its cultural roots rather than defaulting to generic tech-industry placeholders. The emoji keeps persona identity scannable in terminal output where names may truncate.

## When to Use Which

- **Writing a feature doc?** Start with `tech-writer` (Sialou) for prose quality, then `ux-designer` (Gougou) for the reader's mental model.
- **Documenting an API endpoint?** `api-designer` (Ouattara) catches missing methods, inconsistent naming, and body/header gaps that break Postman imports.
- **Shipping a decision?** Run `architect` (Doumbia) for trade-off clarity, and `qa-reviewer` (Kouame) to force the "what could go wrong" column.
- **A long-form guide or onboarding piece?** `storyteller` (Affoue) checks the narrative doesn't drift mid-paragraph.
- **A product or feature spec?** `business-analyst` (Beda) verifies requirements trace back to a named business outcome.

For single-document work, pick 1 persona. For corpus-wide review, 3–4 complementary personas give cross-lens signal (when two personas independently flag the same issue, that convergence is a high-confidence marker — Angela surfaces it via the `Flagged by:` attribution).

## Activation modes

### `angela consult <persona> <file>` — single-lens offline check

```bash
lore angela consult api-designer docs/features/login.md
lore angela consult tech-writer docs/guides/quickstart.md
```

No AI call, no write. Runs the persona's draft-check lens and prints suggestions. Useful after a polish or manual edit when you want one expert's take without the full draft pipeline.

Run `lore angela consult` (no arguments) to list all personas.

### `angela polish --persona <id>` — steer the AI rewrite

```bash
lore angela polish docs/features/login.md --persona ux-designer
```

Biases the AI polish toward the persona's priorities — e.g. `ux-designer` emphasizes user flows and error-path clarity; `api-designer` emphasizes request/response shapes. See [angela-polish.md](angela-polish.md) for the full polish flow.

### `angela review --persona <id>` — multi-persona corpus coherence

```bash
lore angela review --persona tech-writer --persona ux-designer --persona api-designer --persona qa-reviewer
```

Each persona lens is injected into the review prompt. The AI is instructed to attribute each finding to the persona(s) whose expertise flagged it. When multiple personas concur, they're listed together under `Flagged by:` — a strong signal that the issue matters across lenses. See [angela-review.md](angela-review.md).

Alternatively, configure personas in `.lorerc` and activate them via `--use-configured-personas` (skips the interactive confirmation prompt).

## Example session

```bash
# 1. See available personas
$ lore angela consult

Available personas:
  🔌 api-designer         Ouattara
                          API contracts, synthesizer-ready docs, HTTP semantics
  # ... (7 personas)

# 2. Spot-check an API doc
$ lore angela consult api-designer docs/features/invoices.md
  warning  persona  [🔌 Ouattara] Endpoints listed without an HTTP request example — add a ```http block with method, URL, headers, and body

# 3. Review the whole corpus through multiple lenses
$ lore angela review --persona tech-writer --persona qa-reviewer

  + gap   Missing Angela persona documentation
          [abc123] commands/angela-consult.md vs commands/angela-consult.fr.md
          Flagged by: Kouame, Gougou
```

## Persona selection config

You can pin a default set of personas in `.lorerc`:

```yaml
angela:
  review:
    personas:
      selection: "manual"
      manual_list:
        - tech-writer
        - api-designer
        - qa-reviewer
```

Then run `lore angela review --use-configured-personas` to skip the interactive prompt. See [angela-review.md](angela-review.md) and [config.md](config.md) for the full cascade.

## I4 — Zero hallucination

Persona injection does **not** relax the evidence rule. Every persona-attributed finding must carry a verifiable quote from the corpus. Findings without evidence are rejected by the post-processing validator, regardless of which persona flagged them.

## See also

- [angela consult](angela-consult.md) — single-persona on-demand check
- [angela polish](angela-polish.md) — AI-assisted rewrite with `--persona` support
- [angela review](angela-review.md) — corpus-wide coherence with multi-persona lenses
- [config](config.md) — how to configure default personas in `.lorerc`
