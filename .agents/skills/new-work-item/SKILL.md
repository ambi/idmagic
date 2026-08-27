---
name: new-work-item
description: Create a specification-first work item under work-items using the canonical format, requirement/TypeSpec references, tasks, design alternatives, and verification plan.
---

# Creating a work item

1. Read `WORK_ITEM_FORMAT.md` as the authority for the format.
2. Find the highest number across `work-items/` and `done/`, then create an unused
   `wi-NNN-kebab-title.md`.
3. Write Motivation, Scope, Out of Scope, Design, Plan, Tasks, Verification, and Risk Notes. Surface open
   questions in Design or Plan; resolve every question that would change what gets built before
   implementation, and move genuinely deferred choices to Out of Scope.
4. For `feature`, `bugfix`, and `operations` items, make `affected_spec` a direct reference to a
   normative scenario or standard id, or to a TypeSpec symbol. Do not write `initial_context` when
   filing the item; it is written when the work starts.
5. Keep rejected options and change-specific judgment in Design. Do not create an ADR.
6. Pass `mise run check-work-items` and `mise run check-ids`.

On completion, append `Completion`, set the status to completed, and move the file to
`work-items/done/`.
