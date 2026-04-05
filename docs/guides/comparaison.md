# How Lore Compares

## The Landscape

There are many ways to document software decisions. Most of them share a fundamental flaw: they ask developers to stop what they're doing and write documentation *separately* from coding. That's like asking a surgeon to write their operation notes the next day from memory.

Lore takes a different approach: **capture at the moment of the decision**, not after.

## Quick Comparison

| | **Lore** | Swimm | Confluence | GitBook | ADRs | Nothing |
|---|---|---|---|---|---|---|
| **When** | Commit-time | After the fact | After the fact | After the fact | When remembered | Never |
| **Where** | Local (`.lore/`) | SaaS | SaaS | SaaS | Local (Markdown) | — |
| **Friction** | 90 seconds | 30 minutes | 30 minutes | 15 minutes | 15 minutes | 0 |
| **AI** | Angela (opt-in) | Generic | Generic AI | Generic AI | None | — |
| **Lock-in** | Markdown | Proprietary | Proprietary | Mixed | Markdown | — |
| **Offline** | Yes (everything) | No | No | No | Yes | — |
| **Automated** | Post-commit hook | Manual | Manual | Manual | Manual | — |
| **Bilingual** | EN/FR built-in | EN only | Multi-language | Multi-language | Manual | — |
| **Price** | Free (AGPL) | $28/seat | $5.75/user | $8/user | Free | Free |

## Why Commit-Time Matters

Knowledge has a half-life. At the moment of the commit, the developer knows exactly why they made this choice. One hour later, details start fading. One week later, it's vague. Six months later — gone.

```
Commit moment  ████████████████████ 100% context
1 hour later   ████████████████░░░░  80% context
1 day later    ████████████░░░░░░░░  60% context
1 week later   ████████░░░░░░░░░░░░  40% context
1 month later  ████░░░░░░░░░░░░░░░░  20% context
6 months later ░░░░░░░░░░░░░░░░░░░░   0% context
```

Lore captures at the peak. Everything else captures on the decline.

## Detailed Comparisons

### Lore vs Swimm

**Swimm** is a SaaS documentation platform that lives alongside your code. It's well-designed and has good IDE integrations.

| Aspect | Lore | Swimm |
|--------|------|-------|
| **Capture moment** | Automatically at commit | Manually, when you remember |
| **Data location** | Your repo (`.lore/docs/`) | Swimm's servers |
| **Offline** | Fully functional | Requires internet |
| **AI** | Angela: zero-API + optional AI | Generic AI assistant |
| **Price** | Free forever | $28/seat/month |
| **Vendor risk** | None (Markdown files) | Company could pivot, raise prices, or shut down |

**When Swimm is better:** Large teams that need collaborative editing, visual documentation, and IDE widgets. **When Lore is better:** Developers who want zero-friction, local-first, commit-time capture with no subscription.

### Lore vs Confluence

**Confluence** is Atlassian's wiki. It's the default enterprise choice.

The core problem: nobody updates Confluence. Pages rot. The "Authentication Architecture" page was written 18 months ago by someone who left. It describes a system that no longer exists. Everyone knows it's wrong, but nobody has time to fix it.

Lore doesn't have this problem because documents are created **at the moment of change**. They can't rot silently — `lore angela review` catches contradictions, and `lore doctor` flags stale content.

### Lore vs ADRs

**Architecture Decision Records** (ADRs) are Markdown files that document big architectural decisions. They're great.

Lore is **not** a replacement for ADRs. They're complementary:

| | ADRs | Lore |
|---|---|---|
| **Scope** | Big, rare decisions | Daily commit-level decisions |
| **Frequency** | Once a quarter | Every commit |
| **Trigger** | Manual ("someone should write an ADR") | Automatic (post-commit hook) |
| **Example** | "We chose PostgreSQL over MongoDB" | "Why we added this index to the users table" |

The best setup: ADRs for the big picture, Lore for the daily details. Over time, Lore documents naturally feed into ADR discussions.

### Lore vs Conventional Commits

**Conventional Commits** (`feat:`, `fix:`, `docs:`) standardize the **what**. Lore captures the **why**. They work beautifully together:

```bash
git commit -m "feat(auth): add JWT middleware"
# Conventional Commit tells you: it's a feature, in the auth scope
# Lore asks: WHY JWT? WHY now? What alternatives were considered?
```

Lore even pre-fills the "What" field from your commit message. If you use Conventional Commits, Lore's Decision Engine recognizes the type prefix and adjusts scoring accordingly.

### Lore vs Doing Nothing

Most teams do nothing. It works — until it doesn't.

The cost of lost context is real but invisible:

- **Wrong refactors** — Removing code that was there for a reason nobody remembers
- **Repeated mistakes** — Making the same decision (and same mistake) that was already made and undone
- **Onboarding friction** — New team members spend weeks asking "why is this like this?"
- **Review delays** — PRs stall because reviewers don't understand the reasoning

Lore's bet: **90 seconds per commit is worth it.** Over a year, that's ~6 hours of documentation for ~1500 commits. The return: a searchable knowledge base that saves hundreds of hours of "why did we do this?"

## See Also

- [Philosophy](philosophy.md) — The principles behind Lore
- [Quickstart](../getting-started/quickstart.md) — Try it in 5 minutes
- [FAQ](../faq.md) — Common questions
