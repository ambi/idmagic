---
name: implement-work-item
description: "Implement a chosen work item end to end: specification first, inner behavior to outer adapters, tests per layer, verification, completion, move to done, and commit."
---

# Implementing a work item

1. Read only the work item, the normative scenarios and standard ids it references, the TypeSpec
   symbols, the owning `SPECIFICATION.md`, and the smallest slice of code involved.
2. Set the status to `in_progress` and rewrite `initial_context` to what you actually read. What was
   written when the item was filed usually rests on stale assumptions.
3. Change the specification first with the `spec-change` Skill, and pass `just check-spec`.
4. Implement in order: Domain → Use Cases → Adapters → Infrastructure / UI. For behavior layers,
   confirm RED first, and record the test name and the normative scenario id in the task.
5. When structure, technology, or conventions change, sync the Design section of the owning
   `SPECIFICATION.md` with the `update-design` Skill. Do not create an ADR or another architecture
   ledger.
6. Update each task as it completes. Run the narrowest test recipe that covers what you touched, then
   pass `just verify`.
7. Record the semantic difference and the verification results in `Completion` — `just spec-diff`
   shows what the specification actually gained, lost, or changed. Set the status to completed and
   move the file to `work-items/done/`.
8. Create a Conventional Commit with the `commit` Skill. Do not push until explicitly told to.

State the Out of Scope items and anything left undone in the final report.
