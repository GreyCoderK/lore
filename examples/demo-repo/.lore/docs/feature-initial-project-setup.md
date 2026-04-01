---
type: feature
date: 2026-02-01T09:00:00Z
status: active
commit: a1b2c3d4e5f6
generated_by: lore-v1.0.0
---
# Initial project setup

## Why

Every project needs a solid foundation. We chose Go with Cobra for the CLI framework because it compiles to a single binary with zero runtime dependencies — ideal for a developer tool that needs to install cleanly on any machine.

## Alternatives Considered

- **Python + Click**: Faster to prototype, but requires a Python runtime. Deployment friction.
- **Rust + Clap**: Great performance, but the team has more Go experience and the compilation times are faster for iteration.

## Impact

The project is bootstrapped with `cmd/` for Cobra commands, `internal/` for private packages, and `.lore/` for the documentation corpus. This structure follows Go conventions and supports clean separation of concerns.
