---
type: feature
date: 2026-02-15T14:30:00Z
status: active
commit: b2c3d4e5f6a7
generated_by: lore-v1.0.0
---
# Add user authentication middleware

## Why

The API was completely open — any request could access any endpoint without verification. Before adding any user-facing features, we need a solid auth foundation. JWT was chosen because our API is stateless and we don't want to manage server-side sessions.

## Alternatives Considered

- **Session-based auth with Redis**: Works well but adds infrastructure dependency (Redis). We're trying to keep the stack minimal for now.
- **OAuth2 only**: Too complex for the initial implementation. We'll add it later as a provider on top of JWT.
- **API keys**: Too simple — no user context, no expiry management, hard to revoke granularly.

## Impact

All API endpoints now require a Bearer token in the Authorization header. The middleware validates JWT signatures, checks expiry, and injects user context into the request. Unauthenticated requests receive 401.
