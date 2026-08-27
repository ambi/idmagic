---
name: implement-work-item
description: "Implement a chosen work item end to end: specification first, separate acceptance and unit evidence, typed effect boundaries, refactoring, verification, completion, move to done, and commit."
---

# Implementing a work item

1. Read only the work item, its direct normative-scenario and standard references, its TypeSpec symbols, the
   canonical documents those references resolve to, and the smallest code and test slice involved.
2. Change the specification first with `spec-change`, and pass `mise run check-spec`.
3. Resolve every open question that would change product behavior, the public contract, the selected design
   boundary, or the task breakdown. Move genuinely deferred choices to Out of Scope.
4. Rewrite `initial_context` to what you actually read, set `evidence_policy: risk-based-v2`, and apply the
   risk contract in `DEVELOPMENT.md`, naming the intended Acceptance RED and Unit RED checks before you start.
5. Set the status to `in_progress` and pass `mise run check-work-items` and `mise run check-ids`. A later
   normative change returns to step 2; never weaken a scenario to pass code.
6. For changed core logic, make the work item's Design name the principal domain data types and operation
   signatures. Place time, randomness, identifier generation, configuration, persistence, notification, and
   other effects at explicit input, output, or port boundaries.
7. Run the named observable-boundary check and confirm Acceptance RED. Then implement Domain → Use Cases →
   Adapters → Infrastructure / UI one behavior at a time: confirm Unit RED, reach GREEN with the simplest
   complete behavior, refactor while GREEN, and widen through the adapters until the acceptance check passes.
   Retain both failing checks, test names, and applicable normative scenario ids in the task. For tooling,
   documentation, or pure refactoring without one of those boundaries, record `N/A: <reason>` and the alternate
   check that actually failed instead of inventing a product requirement or test boundary.
   Where the change parses, decodes, splits, normalizes, or compares untrusted input by hand, add a fuzz target
   beside the examples and give it an oracle stronger than "does not panic"; see Properties and fuzzing in
   `DEVELOPMENT.md`.
8. When bounded contexts, structure, technology, runtime composition, or core design rules change, use
   `update-design`. Run the narrowest test recipe after each behavior and update its task as it completes.
9. Collect the risk-selected change-resistance evidence and have a person or fresh-context agent that did not
   implement the change run an independent specification and standards review.
10. Pass `mise run verify`. Complete every evidence field required by `WORK_ITEM_FORMAT.md`, reading the
    completion summary out of `mise run spec-diff`. Set the status to `completed`, pass
    `mise run check-work-items` and `mise run check-ids`, and move the file to `work-items/done/`.
11. Create a Conventional Commit with `commit`. Do not push until explicitly told to.

State the Out of Scope items and anything left undone in the final report.
