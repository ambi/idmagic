# Specification-first Development Workflow

## 1. Purpose

This workflow keeps product behavior, current design, implementation, and verification aligned with a
small set of established formats. It favors direct ownership, generated views, and focused checks over
custom specification languages, exhaustive registries, and separate decision archives.

Three documents carry the formats: this one for the loop, [SPECIFICATION_FORMAT.md](SPECIFICATION_FORMAT.md)
for specification documents, and [WORK_ITEM_FORMAT.md](WORK_ITEM_FORMAT.md) for work items. Read the
section you need; none of them is required reading.

## 2. Sources of truth

| Concern | Source of truth |
|---|---|
| Models, API interfaces, HTTP bindings, authentication | TypeSpec (`spec/**/*.tsp`) |
| Normative scenarios, glossary, standards, states, current design | Root or context `SPECIFICATION.md` |
| One change's motivation, alternatives, plan, and history | `work-items/wi-*.md` |
| Executable behavior | Application code and tests |
| Released HTTP compatibility | The tracked OpenAPI release baseline |

Generated OpenAPI and HTML documentation are views, not additional sources of truth. Fine-grained
authorization remains executable application behavior unless a later work item adopts a policy language.

## 3. The loop

| Stage | Skill | Gate |
|---|---|---|
| Frame one change | `new-work-item` | `just check-work-items`, `just check-ids` |
| Change the specification first | `spec-change` | `just check-spec` |
| Implement inner behavior to outer adapters | `implement-work-item` | layer-local tests |
| Sync current design when structure changes | `update-design` | `just check-boundaries` |
| Regenerate derived views | `spec-render` | `just check-api-compat` |
| Record completion and commit | `commit` | `just verify` |

Before changing behavior, update the smallest owning specification: models, APIs, HTTP contracts, and
authentication schemes in TypeSpec; scenarios, terms, standards, state transitions, and durable design in
the owning `SPECIFICATION.md`. Give each normative behavior an immutable `REQ-<CONTEXT>-NNN` ID, and
express state machines as language-independent transition tables.

## 4. Verification ladder

Run the cheapest gate that can still fail on what you just changed, and widen only at the end.

1. While changing the specification: `just check-spec`.
2. While implementing one layer: the narrowest per-package or per-file test recipe that covers what you
   touched — `just test-go-package <package>` or `just test-ui-unit-file <file>` here, whatever
   `just --list` offers elsewhere.
3. Before completing the work item: `just verify`.

Running the full suite after every edit is the most common way to lose time in this repository.

## 5. Current-state documents

The root document owns cross-context structure and policy. Each context document owns that context's
overview, vocabulary, adopted standards, state transitions, technical design, and acceptance scenarios.
Keep one overview and one durable rationale in the owning document; do not create a second copy beside one
implementation language.

TypeSpec and `SPECIFICATION.md` are colocated under `spec/` so backend, frontend, workers, and external
implementations see the same language-independent source. A generated HTML view provides cross-document
navigation without becoming an authored format.

## 6. Work items

A work item is the design and execution record for one meaningful change. It holds motivation, scope,
alternatives, plan, tasks, risks, and completion evidence. When the work lands, copy only the conclusion
that remains true into TypeSpec or the owning `SPECIFICATION.md`.

`just spec-diff [ref]` derives what the working tree changed in the specification — scenarios added,
removed, or changed, transition rows moved, declarations gained or lost. Read it before review and when
writing the completion summary, so the recorded semantic difference is observed rather than recalled.

Current design must be understandable from `SPECIFICATION.md` and the work item alone.

## 7. Context economy

Start from the work item's `initial_context`, which is written when the item starts and names the
specification, code, and tests to read — and what to leave unread. Reach the rest with
`just spec-where <requirement-id-or-term>`, which returns locations across the specification, the code,
and the work items rather than a whole context document; those run to hundreds or thousands of lines once
a context matures. Do not preload generated artifacts, historical decisions, unrelated contexts, or
repository-wide method documents for an ordinary feature change.

Naming a requirement ID in a test or in the code that implements it is what makes that link findable
later, both from `just spec-where` and from the generated Traceability page.
