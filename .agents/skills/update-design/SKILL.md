---
name: update-design
description: Update the current-state Design section in the owning root or context SPECIFICATION.md when bounded contexts, structure, technology, runtime composition, or core design rules change.
---

# Syncing the current design

`SPECIFICATION_FORMAT.md` is the authority. Write the current design and its short rationale in the
Design section of the owning `SPECIFICATION.md`.

- Cross-context design belongs in `spec/SPECIFICATION.md`; context-specific design belongs in
  `spec/contexts/<context>/SPECIFICATION.md`.
- Product requirements and current design belong in the matching sections of the same canonical
  document, API contracts belong in TypeSpec, and change-specific comparisons and history belong in
  the work item.
- Do not create an architecture ledger. Structure is derived from directories and imports, and only
  forbidden boundaries are checked, by `just check-boundaries`.
- Do not create a new ADR. Fold rationale that stays true into the design text, concisely.
- Pass `just check-spec` and `just check-boundaries` afterwards.
