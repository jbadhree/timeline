---
type: decision
timeline_at: 2026-10-13T05:30:00Z
created_by: claude
decision: Link meanings belong to plugins, not the core
decided_by: Badhree Babu
status: current
---

## Context

The core should hold facts that are true in every domain; what a link means differs between goals, OKRs, incidents, and decisions.

## Consequences

The core resolves links and checks that they lead somewhere. Child relations, cycles, and inherited why move into a goals plugin when plugins exist.
