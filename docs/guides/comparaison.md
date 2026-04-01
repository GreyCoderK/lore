# How Lore Compares

## Quick Comparison

| | **Lore** | Swimm | Confluence | GitBook | Nothing |
|---|---|---|---|---|---|
| **When** | Commit-time | After the fact | After the fact | After the fact | Never |
| **Where** | Local (`.lore/`) | SaaS | SaaS | SaaS | — |
| **Friction** | 90 seconds | 30 minutes | 30 minutes | 15 minutes | 0 |
| **AI** | Angela (opt-in) | Generic | Generic | Generic | — |
| **Lock-in** | Markdown | Proprietary | Proprietary | Mixed | — |
| **Price** | Free (AGPL) | $28/seat | $5.75/user | $8/user | Free |

## Why commit-time matters

Documentation written *after the fact* fights against human nature. The longer you wait, the more context evaporates. Lore captures decisions at the moment they're made — when the reasoning is fresh.

## Lore vs ADRs

Lore is not a replacement for Architecture Decision Records. ADRs document big, rare decisions. Lore captures the daily *why* behind every commit. They're complementary:

- **ADRs** = "We chose PostgreSQL over MongoDB" (once a quarter)
- **Lore** = "Why we added this index" (every commit)

## Lore vs Conventional Commits

Conventional Commits standardize the *what* (`feat:`, `fix:`). Lore captures the *why*. They work together — Lore pre-fills the "What" field from your commit message.

## What about doing nothing?

Most teams do nothing. Six months later, nobody remembers why. The cost of that lost context is real: wrong refactors, repeated mistakes, onboarding friction.

Lore's bet: 90 seconds per commit is worth it.
