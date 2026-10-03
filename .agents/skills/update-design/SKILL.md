---
name: update-design
description: Update the owning current-state canonical documents when bounded contexts, global structure, technology, runtime composition, or core design rules change.
---

# Syncing the current design

`SPECIFICATION_FORMAT.md` defines the canonical document kinds. Record each current fact in the smallest file
whose name owns that kind of content.

1. Update the cross-context boundary map in `docs/design/architecture/logical.md`, the top-down index in
   `docs/README.md`, and directory structure, dependency direction, and layers in `docs/domain/structure.md`.
2. Update runtime units in `docs/design/architecture/runtime.md`, deployment topology in
   `docs/design/architecture/deployment.md`, and trust boundaries in `docs/design/security/threat-model.md`; use the
   other matching whole-system file when it owns the changed concern.
3. Update a context boundary and group index in `docs/domain/<context>/README.md`. Put the component table,
   runtime flows, data, cross-cutting concepts, and the design-viewpoint coverage table in
   `docs/domain/<context>/design/`, and a decision that weighed alternatives in `design/decisions.md`; give
   a smaller reason to the rule it justifies as its `**判断**` field. When the change adds, renames, or
   removes a feature slice `backend/<context>/<feature>/`, do the same to the feature node
   `docs/domain/<context>/<group>/<feature>/` and its entry in the group index; `check-spec` rejects a slice
   without a node beyond `tools/check/feature-node-debt.json`, which only shrinks.
4. Revisit `docs/design/security/threat-model.md` when the change adds a trust boundary, a principal kind, an external
   integration, or a new kind of secret, personal data, or record that must later be proven. Move a row to
   `covered` only when a normative id names the control; a row citing a work item is not yet covered.
5. Keep change-specific alternatives, plans, and history in the work item. Keep API contracts in TypeSpec and
   observable behavior in the owning feature specification.
6. Pass `mise run check-spec` and `mise run check-boundaries`.

Canonical documents carry current design and rationale, so no separate ADR or architecture ledger is needed.
