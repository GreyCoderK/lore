# Quickstart (5 minutes)

Get from zero to your first captured "why" in 5 minutes.

## 1. Initialize Lore

```bash
cd your-project
lore init
```

This creates the `.lore/` directory and installs a post-commit hook.

## 2. Make a commit

```bash
git add .
git commit -m "Add rate limiting to API"
```

Lore's hook triggers automatically:

```
[1/3] Type [feature]:
[2/3] What [Add rate limiting to API]:
[3/3] Why? API was getting 10K req/min from one client, causing latency for everyone
✓ Captured: feature-add-rate-limiting-2026-03-16.md
```

Three questions. Ninety seconds. Done.

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
