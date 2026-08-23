---
name: update-design
description: Update the owning current-state canonical documents when bounded contexts, global structure, technology, runtime composition, or core design rules change.
---

# Syncing the current design

`SPECIFICATION_FORMAT.md` defines the canonical document kinds. Record each current fact in the smallest file
whose name owns that kind of content.

1. Update the cross-context boundary map and index in `docs/README.md`, and directory structure, dependency
   direction, layers, and architecture style in `docs/structure.md`.
2. Update runtime units and trust boundaries in `docs/deployment.md`; use the other matching whole-system file
   when it owns the changed concern.
3. Update a context boundary and sibling index in `docs/contexts/<context>/README.md`. Put durable rationale in
   its `decisions.md`, and mechanism that cannot be recovered from code in `internals.md`.
4. Keep change-specific alternatives, plans, and history in the work item. Keep API contracts in TypeSpec and
   observable behavior in the owning `scenarios.md`.
5. Pass `mise run check-spec` and `mise run check-boundaries`.

Canonical documents carry current design and rationale, so no separate ADR or architecture ledger is needed.
