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
| Normative scenarios, glossary, standards, states, decisions, mechanism | The Markdown under `docs/` |
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
| Fix the evidence contract and obtain any required approval | `implement-work-item` | `mise run check-work-items` |
| Confirm RED and implement inner behavior to outer adapters | `implement-work-item` | layer-local tests |
| Sync current design when structure changes | `update-design` | `mise run check-boundaries` |
| Regenerate derived views | `spec-render` | `mise run check-api-compat` |
| Verify change resistance and review independently when required | `code-review` | risk-selected evidence, `mise run verify` |
| Record completion and commit | `commit` | `mise run check-work-items`, `mise run check-ids` |

Before changing behavior, update the smallest owning specification: models, APIs, HTTP contracts, and
authentication schemes in TypeSpec; scenarios, terms, standards, state transitions, decisions, and
mechanism in the file that holds that kind of content. The file name says which one — a new behavior goes
in `scenarios.md`, a new reason in `decisions.md` — so the smallest owning file is usually one file, not
a section inside a large one. Give each normative behavior an immutable `REQ-<CONTEXT>-NNN` ID, and
express state machines as a state table and a transition table. After approval, a normative change returns
to the specification stage; do not relax a scenario merely to make an implementation pass.

## 4. Evidence contract and approval

Every work item that enters `in_progress` declares `evidence_policy: risk-based-v1`. The risk selects the
minimum evidence and approval contract; it does not grant permission to push, merge, write to an external
system, or operate production.

| Risk | Before implementation | Before completion |
|---|---|---|
| `low` | The implementer fixes `initial_context` and the intended RED check. | Record RED evidence. If RED is not applicable, record why and the cheapest alternate check that could have failed. |
| `medium` | A human approves the scope and baseline in `approval` before implementation. | Run `mise run spec-diff <baseline>`, obtain independent verification, and show that a representative incorrect implementation is detected. |
| `high` / `critical` | Apply the `medium` requirements and make security, compatibility, migration, and rollback assumptions explicit. | Apply the `medium` requirements and use `mise run test-go-mutation-package -- <package> <git-ref>` or explicit fault injection for changed pure logic. Record equivalent mutations and tool limits rather than hiding them. |

Authentication, authorization, tenant boundaries, cryptography, protocol compatibility, and persistent-data
migrations require independent verification even if their work item was initially classified lower. Raise the
risk when the table's stronger contract describes the actual consequence.

The `approval` object records who approved, when, the approved scope, and the Git commit containing the
specification under review. If implementation discovers a legitimate normative change, return to the
specification stage, update the
baseline after reapproval, and describe the change in `Post-Approval Changes`. `none` is an acceptable result,
not an omitted check.

Independent verification is performed by a person or a fresh-context agent that did not implement the
change. It compares the normative diff and implementation diff, checks repository standards, and asks whether
the tests reject plausible wrong behavior rather than merely repeat the implementation. It reports findings
back to the owning stage; it does not redesign the change while reviewing it.

## 5. Verification ladder

Run the cheapest gate that can still fail on what you just changed, and widen only at the end.

1. While changing the specification: `mise run check-spec`.
2. While implementing one layer: the narrowest per-package or per-file test recipe that covers what you
   touched — `mise run test-go-package <package>` or `mise run test-ui-unit-file <file>` here, whatever
   `mise tasks` offers elsewhere.
3. For `medium` risk and above: perform the selected change-resistance check and independent verification.
4. Before completing the work item: `mise run verify`.

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

## 6. Current-state documents

`docs/` holds the cross-context structure and policy, one file per kind; `docs/contexts/<context>/` holds
that context's vocabulary, adopted standards, state transitions, decisions, mechanism, and acceptance
scenarios, again one file per kind. Each directory's `README.md` declares what it owns and indexes its
siblings.

Sections are not what divides the specification — files are. A context that grows does not grow one file;
it grows the file whose kind the new content belongs to, and the rest stay the size they were. Keep one
boundary declaration and one home for each fact; do not create a second copy beside one implementation
language.

TypeSpec under `spec/` and the canonical Markdown under `docs/` mirror each other at `contexts/<context>/`,
so backend, frontend, workers, and external implementations see the same
language-independent source. A generated HTML view provides
cross-document navigation without becoming an authored format.

## 7. Work items

A work item is the design and execution record for one meaningful change. It holds motivation, scope,
alternatives, plan, tasks, risks, and completion evidence. When the work lands, copy only the conclusion
that remains true into TypeSpec or the canonical file that owns that kind of content.

`mise run spec-diff [ref]` derives what the working tree changed in the specification — scenarios added,
removed, or changed, transition rows moved, declarations gained or lost. Read it before review and when
writing the completion summary, so the recorded semantic difference is observed rather than recalled. For
approved work, run it against the recorded baseline and retain the result in the completion evidence.

Current design must be understandable from the canonical documents and the work item alone.

## 8. Context economy

Start from the work item's `initial_context`, which is written when the item starts and names the
specification, code, and tests to read — and what to leave unread. Naming a file is enough to say what to
read, because the file names carry the kinds: `scenarios.md` for what a context must do,
`decisions.md` for why it does it that way, `internals.md` for how a mechanism works. Reach anything else
with `mise run spec-where <requirement-id-or-term>`, which returns locations rather than whole files. Do not
preload generated artifacts, unrelated contexts, or repository-wide method documents for an ordinary
feature change.

Naming a requirement ID in a test or in the code that implements it is what makes that link findable
later, both from `mise run spec-where` and from the generated Traceability page.

## 9. Influences and references

Each entry names one representative source for one influence. The list explains provenance, not additional
sources of truth or complete conformance. IdMagic's approval and evidence contract is a repository-specific
adaptation.

- **Spec-Driven Development:** GitHub's
  [Specification-Driven Development](https://github.com/github/spec-kit/blob/27f50f7e6b618ea14d74dd4037f9e7c60218b16c/spec-driven.md)
  informs using specifications to drive planning, implementation, and verification.
- **OpenSpec:** Fission AI's
  [OpenSpec](https://github.com/Fission-AI/OpenSpec/blob/f1b521dffac38ed6638689cd28b0c204b1eef0f1/README.md)
  informs the change-oriented proposal → specifications → design → tasks → apply/archive loop. IdMagic keeps
  its own formats and does not adopt the OpenSpec CLI.
- **Agentic Discipline:** Robert C. Martin and Justin Martin's
  [Clean AI: Agentic Engineering](https://learning.oreilly.com/course/clean-ai-agentic/9780135968819/)
  provides the umbrella reference for disciplined agent-assisted engineering. The evidence boundary and public
  examples used for this adaptation are recorded in `wi-409`.
- **Domain-Driven Design:** Eric Evans's
  [Domain-Driven Design Reference](https://www.domainlanguage.com/wp-content/uploads/2016/05/DDD_Reference_2015-03.pdf)
  informs bounded contexts and consistent domain vocabulary.
- **Clean Architecture:** Robert C. Martin's
  [The Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
  informs inward-pointing dependencies.
- **Hexagonal Architecture (Ports and Adapters):** Alistair Cockburn's
  [Hexagonal Architecture](https://alistair.cockburn.us/hexagonal-architecture/)
  informs explicit application ports and technology-specific adapters.
- **Screaming Architecture:** Robert C. Martin's
  [Screaming Architecture](https://blog.cleancoder.com/uncle-bob/2011/09/30/Screaming-Architecture.html)
  informs the context- and capability-oriented repository layout.
- **Modular Monolith:** Simon Brown's
  [Modular monolith and package by component](https://simonbrown.je/modular-monolith/)
  informs the current single deployment unit with enforced context boundaries.
- **Microservice Architecture:** James Lewis and Martin Fowler's
  [Microservices](https://martinfowler.com/articles/microservices.html)
  informs business-capability boundaries and loose coupling; it does not imply that IdMagic currently deploys
  each context independently.
- **Functional Design:** Eric Normand's
  [Grokking Simplicity](https://www.manning.com/books/grokking-simplicity)
  informs the separation of immutable data, deterministic calculations, and effectful actions.
- **Type-First Development:** Tomas Petricek's
  [Why type-first development matters](https://tomasp.net/blog/type-first-development.aspx/)
  informs defining TypeSpec contracts and domain data types before dependent implementation.
- **Extreme Programming:** Kent Beck's
  [Extreme Programming Explained](https://www.informit.com/store/extreme-programming-explained-embrace-change-9780321278654)
  informs small changes, rapid feedback, simple design, continuous integration, and refactoring.
- **Test-Driven Development:** Kent Beck's
  [Test Driven Development: By Example](https://www.informit.com/store/test-driven-development-by-example-9780321146533)
  informs the RED-first implementation loop.
- **Behavior-Driven Development:** Dan North's
  [Introducing BDD](https://dannorth.net/introducing-bdd/)
  informs behavior-oriented normative scenarios without introducing a second Gherkin source of truth.
- **Acceptance Test-Driven Development:** Robert C. Martin and Grigori Melnik's
  [Tests and Requirements, Requirements and Tests: A Möbius Strip](https://doi.org/10.1109/MS.2008.24)
  informs defining acceptance evidence before implementation.
