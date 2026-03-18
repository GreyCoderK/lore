---
type: decision
date: "2026-03-16"
commit: abc1234
status: demo
tags:
    - authentication
    - jwt
    - middleware
generated_by: lore-demo
---
# Add JWT auth middleware for API routes

## What
Add JWT auth middleware for API routes

## Why
Stateless authentication for microservices — JWT tokens avoid session storage

## Alternatives Considered
- Session-based auth with Redis
- OAuth2 with external provider

## Impact
- API routes now require Bearer token
- Auth middleware added to router chain
