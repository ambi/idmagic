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
| Normative scenarios, glossary, standards, states, decisions, mechanism | The Markdown under `spec/` |
| One change's motivation, alternatives, plan, and history | `work-items/wi-*.md` |
| Executable behavior | Application code and tests |
| Released HTTP compatibility | The tracked OpenAPI release baseline |

Generated OpenAPI and HTML documentation are views, not additional sources of truth. Fine-grained
authorization remains executable application behavior unless a later work item adopts a policy language.

## 3. The loop

| Stage | Skill | Gate |
|---|---|---|
| Frame one change | `new-work-item` | `mise run check-work-items`, `mise run check-ids` |
| Change the specification first | `spec-change` | `mise run check-spec` |
| Implement inner behavior to outer adapters | `implement-work-item` | layer-local tests |
| Sync current design when structure changes | `update-design` | `mise run check-boundaries` |
| Regenerate derived views | `spec-render` | `mise run check-api-compat` |
| Record completion and commit | `commit` | `mise run verify` |

Before changing behavior, update the smallest owning specification: models, APIs, HTTP contracts, and
authentication schemes in TypeSpec; scenarios, terms, standards, state transitions, decisions, and
mechanism in the file that holds that kind of content. The file name says which one — a new behavior goes
in `scenarios.md`, a new reason in `decisions.md` — so the smallest owning file is usually one file, not
a section inside a large one. Give each normative behavior an immutable `REQ-<CONTEXT>-NNN` ID, and
express state machines as a state table and a transition table.

## 4. Verification ladder

Run the cheapest gate that can still fail on what you just changed, and widen only at the end.

1. While changing the specification: `mise run check-spec`.
2. While implementing one layer: the narrowest per-package or per-file test recipe that covers what you
   touched — `mise run test-go-package <package>` or `mise run test-ui-unit-file <file>` here, whatever
   `mise tasks` offers elsewhere.
3. Before completing the work item: `mise run verify`.

Running the full suite after every edit is the most common way to lose time in this repository.

### Testing a refusal

A control that refuses — authorization, the tenant boundary, CSRF and origin, scope, a fail-closed
branch — is tested by two assertions, not one.

1. **What the caller observes.** The status and the kind of error.
2. **What the refusal left untouched.** Read the state back and assert the operation had no effect: the
   row was not created, the value still holds what it held, the event was not emitted.

The second is the one that matters and the one that gets left out. A test that checks only the first
passes just as happily against a control that writes "denied" and then performs the operation anyway.
That is not hypothetical: it is what shipped, survived review, and held full line coverage until a
refusal was tested for its effect rather than its wording.

Name the scenario the refusal comes from in the test, so the specification and the test can be read
against each other. `mise run check` requires this of a refusal declared from now on.

A helper that a caller guards with — anything called as `if err := guard(...); err != nil` — reports its
refusal through the return value. Write the response, then return an error; never hand back what writing
the response returned, because that is nil and the caller will carry on. `mise run check` rejects the
shape, and follows it through whatever helpers stand between the guard and the response: a wrapper that
returns what a writer returned is the same defect one call further away.

## 5. Current-state documents

`spec/` holds the cross-context structure and policy, one file per kind; `spec/contexts/<context>/` holds
that context's vocabulary, adopted standards, state transitions, decisions, mechanism, and acceptance
scenarios, again one file per kind. Each directory's `README.md` declares what it owns and indexes its
siblings.

Sections are not what divides the specification — files are. A context that grows does not grow one file;
it grows the file whose kind the new content belongs to, and the rest stay the size they were. Keep one
boundary declaration and one home for each fact; do not create a second copy beside one implementation
language.

TypeSpec and the canonical Markdown are colocated under `spec/` so backend, frontend, workers, and
external implementations see the same language-independent source. A generated HTML view provides
cross-document navigation without becoming an authored format.

## 6. Work items

A work item is the design and execution record for one meaningful change. It holds motivation, scope,
alternatives, plan, tasks, risks, and completion evidence. When the work lands, copy only the conclusion
that remains true into TypeSpec or the canonical file that owns that kind of content.

`mise run spec-diff [ref]` derives what the working tree changed in the specification — scenarios added,
removed, or changed, transition rows moved, declarations gained or lost. Read it before review and when
writing the completion summary, so the recorded semantic difference is observed rather than recalled.

Current design must be understandable from the canonical documents and the work item alone.

## 7. Context economy

Start from the work item's `initial_context`, which is written when the item starts and names the
specification, code, and tests to read — and what to leave unread. Naming a file is enough to say what to
read, because the file names carry the kinds: `scenarios.md` for what a context must do,
`decisions.md` for why it does it that way, `internals.md` for how a mechanism works. Reach anything else
with `mise run spec-where <requirement-id-or-term>`, which returns locations rather than whole files. Do not
preload generated artifacts, unrelated contexts, or repository-wide method documents for an ordinary
feature change.

Naming a requirement ID in a test or in the code that implements it is what makes that link findable
later, both from `mise run spec-where` and from the generated Traceability page.
