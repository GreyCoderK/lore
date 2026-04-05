# Philosophy

## The Problem Lore Solves

Code tells you **what** was built. Git tells you **when** it changed. But neither tells you **why**.

Six months from now, someone will stare at a piece of code and wonder: *"Why did we do it this way?"* The answer is already gone — buried in a Slack thread that got archived, a PR comment nobody will find, or the memory of a developer who moved on.

This isn't a tooling problem. It's a **knowledge preservation** problem. And it gets worse with every commit.

## Three Principles

### 1. Zero Friction — 90 Seconds or Nothing

If documentation takes more than 90 seconds, developers stop doing it. That's not a character flaw — it's human nature. Lore doesn't ask for an essay. It asks 3 questions:

- **Type** — What kind of change? (one selection)
- **What** — Pre-filled from your commit message (just press Enter)
- **Why** — The one that matters (one sentence is enough)

The post-commit hook makes it automatic. You don't decide to document — it's part of the commit flow. Like buckling your seatbelt: you don't think about it, you just do it.

> **The insight:** The best documentation system is the one developers actually use. A perfect wiki that nobody updates is worth less than a simple "why" captured at every commit.

### 2. Local-First, Offline-First

Your team's decisions shouldn't live on someone else's server.

- **Markdown files in your repo** — portable, versionable, grep-able
- **No SaaS** — no subscription, no "your trial expires in 14 days"
- **No network calls** — everything works on a plane, in a bunker, offline
- **No lock-in** — your data is Markdown. Take it anywhere.

AI features (Angela) are opt-in. When you use them, a single API call goes to your chosen provider. The rest of the time: zero network, zero tracking, zero dependency on external services.

> **The insight:** Developer tools should respect the developer. Your code is yours. Your decisions are yours. Your documentation should be yours too.

### 3. The "Why" Is a Treasure

The name **Lore** carries a double meaning:

- In English: *lore* — ancestral knowledge passed from generation to generation
- In French: *l'or* — gold, treasure, something precious

Every commit contains gold — the reasoning behind a choice, the alternatives considered, the context that made the decision obvious at the time. Most teams let that gold evaporate. Lore extracts it.

> **The insight:** A codebase with documented "whys" is fundamentally different from one without. New team members onboard faster. Code reviews have context. Refactors don't repeat past mistakes. The knowledge compounds.

## Design Decisions That Follow

These principles translate into non-negotiable architectural choices:

| Principle | Architectural Decision |
|-----------|----------------------|
| Zero friction | Post-commit hook, Decision Engine auto-skip, Express mode |
| Local-first | Markdown source of truth, `.lore/` in repo, SQLite reconstructible |
| Offline-first | Zero implicit network calls, AI is opt-in, everything works without internet |
| The "why" matters | 3 questions (not 10), "Why" is the mandatory field, corpus is searchable |

## What Lore Is Not

- **Not a wiki replacement** — Wikis are for long-form documentation. Lore is for commit-time decisions.
- **Not an ADR tool** — ADRs capture big, rare architectural decisions. Lore captures the daily "why." They're complementary.
- **Not a commit linter** — Conventional Commits standardize the "what." Lore captures the "why." They work together.
- **Not a surveillance tool** — Lore doesn't track who documents or who doesn't. It's a personal practice that benefits the team.

## About Angela

The AI companion inside Lore is named **Angela**.

Angela is the embedded reviewer who reads your documentation, knows your project's style, and checks consistency before you publish — like a colleague who has read every document your team ever wrote.

She can also step back and analyze your entire corpus at once — like a librarian looking at the full collection and telling you: "This book contradicts that one. There's a missing chapter on this subject."

She's opt-in. She respects resources. She never makes automatic decisions without consent.

**Angela is named after the creator's niece, who was lost to cancer.**

It's not just a name in a config file. It's a way to keep her present in what's being built. To honor her through something that helps people, that lasts, that travels far.

Every time Angela reviews a document, every time she catches a contradiction, every time she helps someone write a clearer "why" — a little piece of that legacy lives on.

## The Vision

Today, Lore captures the "why." Tomorrow, Lore understands it, connects it, and shares it.

The corpus you build today becomes more valuable with every future feature. Angela will grow. The "why" you capture now is the foundation for everything that comes next.

## See Also

- [Roadmap](roadmap.md) — Where Lore is heading
- [How Lore Compares](comparaison.md) — Lore vs alternatives
