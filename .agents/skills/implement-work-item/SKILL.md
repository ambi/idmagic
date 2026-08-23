---
name: implement-work-item
description: "Implement a chosen work item end to end: specification first, inner behavior to outer adapters, tests per layer, verification, completion, move to done, and commit."
---

# Implementing a work item

1. Read only the work item, its direct normative-scenario and standard references, its TypeSpec symbols,
   the canonical documents those references resolve to, and the smallest code and test slice involved.
2. Change the specification first with the `spec-change` Skill, and pass `mise run check-spec`.
3. Rewrite `initial_context` to what you actually read, set `evidence_policy: risk-based-v1`, and apply
   the risk contract in `DEVELOPMENT.md`. For `medium` risk and above, record explicit human approval of
   the specified scope and its Git commit before changing the status; if it is absent, stop before implementation.
4. Set the status to `in_progress` and pass `mise run check-work-items` and `mise run check-ids`. A later
   normative change returns to step 2 and requires reapproval; never weaken a scenario to pass code.
5. Implement in order: Domain → Use Cases → Adapters → Infrastructure / UI. For behavior layers,
   confirm RED first, and retain the failing check, test name, and normative scenario id in the task.
6. When bounded contexts, structure, technology, or core design rules change, update the owning canonical
   design documents with the `update-design` Skill. Do not create an ADR or another architecture ledger.
7. Update each task as it completes and run the narrowest test recipe that covers what you touched. Before
   completion, collect the risk-selected change-resistance evidence and have a person or fresh-context agent
   that did not implement the change run an independent specification and standards review.
8. Pass `mise run verify`. Complete every evidence field required by `WORK_ITEM_FORMAT.md`; use
   `mise run spec-diff <approval.baseline>` for approved work. Set the status to `completed`, pass
   `mise run check-work-items` and `mise run check-ids`, and move the file to `work-items/done/`.
9. Create a Conventional Commit with the `commit` Skill. Do not push until explicitly told to.

State the Out of Scope items and anything left undone in the final report.
