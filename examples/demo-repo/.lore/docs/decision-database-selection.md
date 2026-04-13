---
type: decision
date: 2026-02-10T11:00:00Z
status: active
commit: c3d4e5f6a7b8
generated_by: lore-v1.0.0
---
# Database selection: PostgreSQL over MongoDB

## What

Chose PostgreSQL as the primary database for the API backend, replacing the original plan to evaluate MongoDB.

## Why

We need relational integrity for user accounts, permissions, and audit trails. PostgreSQL gives us ACID transactions, mature tooling, and excellent Go driver support (pgx). Our data is inherently relational — users have roles, roles have permissions, actions have audit logs.

## Alternatives Considered

- **MongoDB**: Flexible schema is nice for rapid prototyping, but our data model is well-defined and relational. We'd end up reimplementing foreign keys in application code.
- **SQLite**: Perfect for embedded use (and we use it for LKS!), but not suitable for a multi-user API backend.
- **CockroachDB**: Distributed PostgreSQL-compatible, but overkill for our current scale. We can migrate later if needed.

## Impact

All persistence goes through PostgreSQL via pgx. Migrations are managed with golang-migrate. The schema is versioned in `migrations/` and applied automatically on startup.
