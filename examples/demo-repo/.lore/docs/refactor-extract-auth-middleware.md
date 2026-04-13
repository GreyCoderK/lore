---
type: refactor
date: 2026-03-01T16:45:00Z
status: active
commit: d4e5f6a7b8c9
tags: [authentication, middleware, refactor, testing]
generated_by: lore-v1.0.0
---
# Extract auth middleware into dedicated package

## What

Extracted JWT validation logic from 4 duplicate handler implementations into a single `internal/middleware/auth.go` package.

## Why

The authentication logic was scattered across 4 different handlers, each with its own token validation code. This made it impossible to test token refresh independently, and any auth change required updating 4 files. A single middleware package eliminates duplication and creates one place to maintain auth logic.

## Alternatives Considered

- **Keep inline**: Simpler short-term, but the duplication was already causing bugs (one handler had an outdated token validation).
- **Decorator pattern**: More flexible but overly complex for our needs. A simple middleware chain is enough.

## Impact

New `internal/middleware/auth.go` package handles all JWT validation. Handlers no longer deal with tokens directly — they receive a validated user context. Test coverage for auth went from 45% to 92%.
