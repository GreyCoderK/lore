---
type: reference
date: 2026-04-12
status: published
related:
  - init.md
  - new.md
  - show.md
  - list.md
  - status.md
  - delete.md
  - pending.md
  - doctor.md
  - hook.md
  - config.md
  - release.md
  - demo.md
  - decision.md
  - angela-draft.md
  - angela-polish.md
  - angela-review.md
  - check-update.md
  - upgrade.md
  - completion.md
  - ../guides/document-types.md
  - ../guides/contextual-detection.md
  - ../guides/configuration.md
---
# Commands Overview

All Lore CLI commands at a glance.

## Core Commands

| Command | Description |
|---------|-------------|
| [`lore init`](init.md) | Initialize Lore in the current repository |
| [`lore new`](new.md) | Create documentation on demand |
| [`lore show`](show.md) | Search and display documents |
| [`lore list`](list.md) | List all documents in the corpus |
| [`lore status`](status.md) | Repository health dashboard |
| [`lore delete`](delete.md) | Delete a document |
| [`lore pending`](pending.md) | Manage undocumented commits |

## Maintenance

| Command | Description |
|---------|-------------|
| [`lore doctor`](doctor.md) | Diagnose and repair corpus issues |
| [`lore hook`](hook.md) | Manage the Git post-commit hook |
| [`lore config`](config.md) | View and manage configuration |
| [`lore release`](release.md) | Generate release notes |
| [`lore demo`](demo.md) | Interactive demo |
| [`lore decision`](decision.md) | Decision engine status |

## Angela (AI-Assisted)

| Command | Description |
|---------|-------------|
| [`lore angela draft`](angela-draft.md) | Zero-API structural analysis |
| [`lore angela polish`](angela-polish.md) | AI-assisted rewrite with diff review |
| [`lore angela review`](angela-review.md) | Corpus-wide coherence analysis |

## Updates

| Command | Description |
|---------|-------------|
| [`lore check-update`](check-update.md) | Check if a newer version is available |
| [`lore upgrade`](upgrade.md) | Upgrade lore to the latest version |

## Utilities

| Command | Description |
|---------|-------------|
| [`lore completion`](completion.md) | Generate shell completion scripts |

## Related Guides

- [Document Types & Metadata](../guides/document-types.md) — Types, statuses, and front matter reference
- [Contextual Detection](../guides/contextual-detection.md) — How the hook decides what to do
- [Configuration](../guides/configuration.md) — Full config reference
