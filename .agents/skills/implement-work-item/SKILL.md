---
name: implement-work-item
description: "Implement a chosen work item end to end: specification first, separate acceptance and unit evidence, typed effect boundaries, refactoring, verification, completion, move to done, and commit."
---

# Implementing a work item

1. Begin from a working tree carrying no other work item's changes; one record is one commit, and a tree
   holding two records cannot be split into two without reading the diff back. Then run
   `mise run brief -- <work-item>` and make a **readiness pass** before editing frontmatter. Read what the brief
   names: the work item, its direct normative-scenario and standard references, its TypeSpec symbols, the
   canonical documents those references resolve to, and the smallest code and test slice involved. For a
   standards-coverage item, resolve every scoped standard id to its declaration, debt entry, named tests,
   smallest production entry point, and any active work item that owns missing behavior. If completion needs
   out-of-scope implementation, record the prerequisite or defect and stop before broad checks or implementation.
   Read nothing else until something you have read sends you there. Run `mise run lint-go` once while you read:
   the cold run is the expensive one, and paying it here leaves every run in step 8 at a few seconds.
2. When the work changes a specification, change it first with `spec-change` and pass `mise run check-spec`.
   When `spec_impact: none` is still correct after the readiness pass, do not invent a specification edit or run
   specification-only baseline gates; use the first check that can actually go RED for the work.
3. Resolve every open question that would change product behavior, the public contract, the selected design
   boundary, or the task breakdown. Move genuinely deferred choices to Out of Scope.
4. Rewrite `initial_context` to the smallest slice actually read during readiness — the brief's draft is a
   starting point, not the answer, and `stop_before_reading` is always yours to decide. It is an audit trail,
   not a reason to read more: leave a category empty instead of opening files only to populate it. Set
   `evidence_policy: risk-based-v3`, and apply the risk contract in
   `docs/development/specification-first-workflow.md`. For an applicable feature, bugfix, or standards change,
   name the primary use cases, Unit RED checks, E2E RED checks, and distinct fault models; otherwise name the
   intended Acceptance RED and Unit RED checks before you start.
5. Set the status to `in_progress` and pass `mise run check-work-items`. A later
   normative change returns to step 2; never weaken a scenario to pass code.
6. For changed core logic, make the work item's Design name the principal domain data types and operation
   signatures. Place time, randomness, identifier generation, configuration, persistence, notification, and
   other effects at explicit input, output, or port boundaries.
7. Run the named observable-boundary check and confirm Acceptance RED or the applicable E2E RED. Then implement
   Domain → Use Cases → Adapters → Infrastructure / UI one behavior at a time: confirm Unit RED, reach GREEN with the simplest
   complete behavior, refactor while GREEN, and widen through the adapters until the acceptance check passes.
   Retain both failing checks, test names, and applicable normative scenario ids in the task. For tooling,
   documentation, or pure refactoring without one of those boundaries, record `N/A: <reason>` and the alternate
   check that actually failed instead of inventing a product requirement or test boundary.
   Where the change parses, decodes, splits, normalizes, or compares untrusted input by hand, add a fuzz target
   beside the examples and give it an oracle stronger than "does not panic"; see Properties and fuzzing in
   `docs/development/specification-first-workflow.md`.
8. When bounded contexts, structure, technology, runtime composition, or core design rules change, use
   `update-design`. Keep the feedback loop tight: use `mise run test-go-test -- <package> <test>` or
   `mise run test-ui-unit-file -- <file>` for each RED, GREEN, and fault injection; run the containing package
   once after a coherent behavior is GREEN; use `mise run test-go-changed` after the change crosses package
   boundaries. Run the check that owns the layer you just touched while that layer is still what you are looking
   at, rather than saving it for final verification:

   | Just touched | Run next |
   | --- | --- |
   | An admin API or a DTO, after the specification is regenerated | `mise run check-contract-drift`, `mise run check-api-compat` |
   | Go, once one behavior is GREEN | `mise run lint-go` |
   | This record's frontmatter or Completion | `mise run check-work-items` |

   Each of those costs seconds once step 1 has paid the cold run. Run `lint-go` when a behavior reaches GREEN,
   not after every edit, so a lint fix never lands in the middle of a behavior that is still RED. The aggregate
   gates stay where they are, run once at step 10.
   Write the selected recipes into the task so the choice is made once rather than on every red-green turn.
9. Collect the risk-selected change-resistance evidence.
10. After every scoped behavior and its evidence can be completed, pass `mise run verify` once, and
    `mise run test-ui-e2e` as well when the change can reach the browser: the standard suite no longer starts
    the stack, so a browser regression is otherwise left to CI. Do not run an aggregate gate merely as a
    status check while a prerequisite still prevents completion. Complete
    every evidence field required by `WORK_ITEM_FORMAT.md`, reading the completion summary out of
    `mise run spec-diff`. Set the status to `completed`, pass
    `mise run check-work-items`, and move the file to `work-items/done/`.
11. Create a Conventional Commit with `commit`. Its body is the Completion Summary said in English, not a
    description written back out of the diff. Do not push until explicitly told to.

State the Out of Scope items and anything left undone in the final report.

When the user asks for timing analysis, measure decision intervals separately from command `real` time, and
record cold versus cached runs, environmental retries, and approval waits separately; otherwise the numbers
cannot distinguish workflow cost from tool or sandbox cost.
