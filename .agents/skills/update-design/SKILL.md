---
name: update-design
description: Update the owning current-state canonical documents when modules, their public packages, composition points, table ownership, global structure, technology, runtime composition, or core design rules change.
---

# Syncing the current design

`SPECIFICATION_FORMAT.md` defines the canonical document kinds. Record each current fact in the smallest file
whose name owns that kind of content.

1. Update the module responsibility table (modules, publication mode, public packages, composition points) in
   `docs/design/architecture/logical.md`, the top-down index in `docs/README.md`, and directory structure,
   dependency rules, and layers in `docs/domain/structure.md`. Table ownership lives in the table list of
   `docs/design/data/database.md`. Do not write module dependency edges into any document; the code is their
   source. A change to a module, its public packages, a composition point, or a table owner is a boundary
   change: apply the boundary selection procedure in `docs/design/application/design-guidelines.md` and record
   the comparison in the work item.
2. Update runtime units in `docs/design/architecture/runtime.md`, deployment topology in
   `docs/design/architecture/deployment.md`, and trust boundaries in `docs/design/security/threat-model.md`; use the
   other matching whole-system file when it owns the changed concern. Constraints, the solution strategy, and
   known risks and technical debt live in `docs/design/architecture/{constraints,strategy,risks}.md`.
3. Update a module boundary, public contracts, and feature index in `docs/domain/<context>/README.md`,
   and its allocated quality requirements in `quality.md`. Put the design in `docs/domain/<context>/design/`
   under the same topic vocabulary as `docs/design/`: `architecture.md` (context, strategy, components,
   runtime flows), `data.md`, `security.md`, `reliability.md`, `performance.md`, `risks.md`, a
   cross-cutting concept as its own file, and a decision that weighed alternatives in `design/decisions.md`;
   give a smaller reason to the rule it justifies as its `**判断**` field. Keep the topic index in
   `design/README.md` (and in a feature's `design.md`) listing every topic with a link or `該当なし：<reason>`;
   `check-spec` rejects an index that leaves a topic out. When the change adds, renames, or
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
