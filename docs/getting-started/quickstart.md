---
type: guide
date: 2026-04-12
status: published
related:
  - ../commands/index.md
  - ../guides/configuration.md
  - ../commands/angela-draft.md
  - ../guides/angela-ci.md
---
# Quickstart (5 minutes)

Go from zero to your first captured "why" in 5 minutes.

## 1. Initialize Lore

```bash
cd your-project
lore init
```

This creates the `.lore/` directory and installs a post-commit git hook.

## 2. Make a commit

```bash
git add .
git commit -m "Add rate limiting to API"
```

lore's hook triggers automatically:

```
[1/3] Type [feature]:
[2/3] What [Add rate limiting to API]:
[3/3] Why? API was getting 10K req/min from one client, causing latency for everyone
✓ Captured: feature-add-rate-limiting-2026-03-16.md
```

Three questions. Ninety seconds. Done.

![lore interactive](../assets/vhs/interactive.gif)
<!-- Generate: vhs assets/vhs/interactive.tape -->

> **What just happened?** lore's post-commit hook detected your commit, asked 3 questions, and saved a Markdown file in `.lore/docs/`. The file contains YAML front matter (type, date, commit hash) and your "why" — permanently linked to that commit.

## 3. View your document

```bash
lore show
```

```markdown
---
type: feature
date: 2026-03-16
commit: e4f5a6b
---
# Add rate limiting to API

## Why
API was getting 10K req/min from one client, causing latency for everyone.
```

## 4. Check your corpus health

```bash
lore status
```

```
Documents: 1 | Pending: 0 | Coverage: 100%
```

> **What just happened?** lore scanned your commits and documents. 1 commit, 1 document = 100% coverage. As you keep committing, this dashboard tracks documentation health over time.

## 5. Explore more

```bash
# Document a past commit retroactively
lore new --commit abc1234

# List all documents
lore list

# Run diagnostics
lore doctor
```

## What's next?

- [Commands Reference](../commands/index.md) — All commands in detail
- [Configuration](../guides/configuration.md) — Customize Lore
- [Angela AI](../commands/angela-draft.md) — AI-assisted documentation
- [Angela in CI](../guides/angela-ci.md) — Use Angela as a documentation quality gate in CI (no `lore init` required)
