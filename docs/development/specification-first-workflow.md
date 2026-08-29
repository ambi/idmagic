# Specification-first Development Workflow

## 1. Purpose

This workflow keeps product behavior, current design, implementation, and verification aligned with a
small set of established formats. It favors direct ownership, generated views, and focused checks over
custom specification languages, exhaustive registries, and separate decision archives.

Three documents carry the formats: this one for the loop, [SPECIFICATION_FORMAT.md](../../SPECIFICATION_FORMAT.md)
for specification documents, and [WORK_ITEM_FORMAT.md](../../WORK_ITEM_FORMAT.md) for work items. Read the
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
| Resolve material questions and fix the evidence contract | `implement-work-item` | `mise run check-work-items` |
| Confirm Acceptance RED | `implement-work-item` | the narrowest test at an observable boundary |
| Confirm Unit RED, reach GREEN, and refactor inner behavior to outer adapters | `implement-work-item` | layer-local tests |
| Sync current design when structure changes | `update-design` | `mise run check-boundaries` |
| Regenerate derived views | `spec-render` | `mise run check-api-compat` |
| Verify change resistance | `code-review` | evidence selected by risk, `mise run verify` |
| Record completion and commit | `commit` | `mise run check-work-items`, `mise run check-ids` |

Before changing behavior, update the smallest owning specification: models, APIs, HTTP contracts, and
authentication schemes in TypeSpec; scenarios, terms, standards, state transitions, decisions, and
mechanism in the file that holds that kind of content. The file name says which one — a new behavior goes
in `scenarios.md`, a new reason in `decisions.md` — so the smallest owning file is usually one file, not
a section inside a large one. Give each normative behavior an immutable `REQ-<CONTEXT>-NNN` ID, and
express state machines as a state table and a transition table. A normative change discovered during
implementation returns to the specification stage; do not relax a scenario merely to make an implementation
pass.

## 4. Evidence contract

Every work item that enters `in_progress` declares `evidence_policy: risk-based-v2`. The risk selects the
minimum evidence the work must produce; it does not grant permission to push, merge, write to an external
system, or operate production. Filing the work item is what authorizes the work, so nothing here records a
separate approval: an approval field would be signed without thought, while the checks below can only be
satisfied by an observation.

| Risk | Before implementation | Before completion |
|---|---|---|
| `low` | The implementer fixes `initial_context`, resolves questions that would change what gets built, and names the intended Acceptance RED and Unit RED checks. | Record both RED results. If either is not applicable, record why and the cheapest alternate check that was actually observed failing. |
| `medium` | Apply the `low` requirements. | Read `mise run spec-diff` into the completion summary and show that a representative incorrect implementation is detected. |
| `high` / `critical` | Apply the `medium` requirements and make security, compatibility, migration, and rollback assumptions explicit. | Apply the `medium` requirements and use `mise run test-go-mutation-package -- <package> <git-ref>` or explicit fault injection for changed pure logic. Record equivalent mutations and tool limits rather than hiding them. |

Authentication, authorization, tenant boundaries, cryptography, protocol compatibility, and persistent-data
migrations reach the stronger rows quickly. Raise the risk when the table's stronger contract describes the
actual consequence, rather than leaving the initial classification in place.

The risk column reads one axis only: how much damage a wrong change does. It says nothing about whether the
decision can be taken back, and those two come apart constantly. A work item that declares
`reversibility: irreversible` — the field and its examples are defined in
[WORK_ITEM_FORMAT.md](../../WORK_ITEM_FORMAT.md) — records that the decision cannot be withdrawn later, so a
reader can see which choices are load-bearing. It adds no evidence of its own, and `reversible` never lowers
what the risk row already asks for.

A second reader belongs to review, not to this contract. Extreme Programming gets one onto every line as the
line is written; a repository worked by one person and by agents spends that attention when the change is
read as a whole instead. What the evidence contract can ask for is different in kind: an observation the
implementer cannot satisfy by intending something. That is what the RED results and the change-resistance
check are, and asking a work item to also record that somebody read it would only restate the review that
already happened.

### Acceptance and unit evidence

Acceptance RED and Unit RED have different responsibilities. Acceptance RED fails at the narrowest boundary
where a caller can observe the normative behavior and names the applicable `REQ-<CONTEXT>-NNN`. Unit RED fails
on the changed domain or use-case logic without depending on the acceptance test as its only proof. A product
behavior change records both before implementation. Tooling, documentation, and pure refactoring may mark one
or both as not applicable only when they record the reason and an alternate check that was actually observed
failing.

After both boundaries are fixed, implement one behavior at a time: make the narrow unit test GREEN with the
simplest complete behavior, refactor while it remains GREEN, then widen through adapters until the acceptance
test passes. Do not treat a generated or broad acceptance test as the unit test for the inner calculation.

### Refactoring

Refactoring is changing structure without changing behavior, and the test is what makes that claim checkable:
if a change is a refactoring, the tests do not move. Editing a test in the same step is the signal that
behavior changed too — separate the two steps and let the behavior change go through its own RED.

Refactor in the moment the test just went GREEN, while what the code is supposed to do is still in front of
you. Deferring it turns it into a separate piece of work that has to re-establish that context, and separate
work is what gets dropped. Stop when the next behavior can be added without fighting the current shape. That
is the whole condition: not a metric, not a pass over everything the change touched, and not a general
tidying of code the change did not need. A refactoring that outruns the behavior it was clearing the way for
is a second change riding on the first one's evidence.

Refactoring done this way carries no evidence of its own — it happens inside a behavior's GREEN step and that
behavior's RED results already cover it. A work item that is *only* refactoring is the case the previous
section's not-applicable path exists for, and what it records as the alternate check is the check that made the
refactoring necessary: a boundary or structural gate that was failing, an import rule, a duplicated rule the
`check` suite now rejects. If nothing was failing and no gate was asking for the change, that is worth
noticing before starting rather than after.

### Type and effect design

Before implementation, resolve every open question whose answer would change product behavior,
the public contract, the chosen design boundary, or the task breakdown. Record genuinely deferred choices in
Out of Scope instead of turning an unstated assumption into implementation.

For changed core logic, use the work item's Design to sketch the domain data types and principal operation
signatures before filling in behavior. Identify time, randomness, identifier generation, configuration,
persistence, notification, and other effects as explicit inputs, outputs, or ports. Keep deterministic
decisions as calculations over data where that separation makes the rule easier to test or read, and let
use cases orchestrate the actions. This is a design test for the changed logic, not a repository-wide purity
or wrapper-type quota.

## 5. Verification ladder

Run the cheapest gate that can still fail on what you just changed, and widen only at the end.

1. While changing the specification: `mise run check-spec`.
2. Before implementing behavior: run the named acceptance check and observe the required behavior fail.
3. While implementing one layer: confirm Unit RED, reach GREEN, refactor while GREEN, and run the narrowest
   per-package or per-file test recipe that covers what you touched — `mise run test-go-package <package>` or
   `mise run test-ui-unit-file <file>` here, whatever `mise tasks` offers elsewhere.
4. For `medium` risk and above: perform the selected change-resistance check.
5. Before completing the work item: `mise run verify`.

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

### Properties and fuzzing

An example-based test is written by whoever wrote the branches, out of the same reading of the problem, so it
inherits that reading's blind spots. This weighs more when an agent writes both: implementation and table of
examples come out of one pass over one understanding, and the table then agrees with the code about what could
go wrong. A property is stated from the specification instead, and a generator supplies the cases nobody
thought of. It is the cheapest check available that is not downstream of the author's own assumptions.

Reach for a property or a fuzz target when the input crosses a trust boundary and is decoded, parsed, split,
normalized, or compared before a decision is made about it; when the parsing is hand-written rather than a call
into a library that already has this treatment; when there is a round trip to state, such as encode and decode,
derive and verify, or parse and serialize; or when one rule is restated at several call sites, because writing
the target forces the rule into one place. Do not reach for it for orchestration, for a workflow whose
correctness is a policy choice rather than a property, or for anything that needs a database to say what
correct means. There is no coverage quota: the count follows the boundaries, and a boundary already covered
upstream does not get a second target.

**The oracle is the whole exercise.** "Does not panic" is not an oracle — it passes just as happily against a
parser that accepts everything. State one of these instead.

- **Round trip.** What was correctly built is accepted, and any mutation of it is rejected.
- **Strictness.** Acceptance implies exact equality with something registered. On its own this also passes
  against an implementation that rejects everything, so pair it with the assertion that a legitimate input is
  accepted. Strictness without that pairing is a vacuous test that looks like a strong one.
- **Structural bound.** A declared limit — size after expansion, nesting depth, segment count — always refuses.
- **Idempotence.** Normalizing twice changes nothing.

Never make time the oracle. "Returns within N milliseconds" varies by several multiples under load, on a shared
runner and on a laptop alike, so it becomes a permanently flaky assertion. State denial-of-service resistance as
a structural bound and leave hangs to the fuzzer's own detection. Never sign per input either: build keys and
certificates once in the seed phase, because signing inside the loop costs two or three orders of magnitude of
throughput and the search stops moving. A property that does not vary with the input — that an entity reference
is refused, say — is a table, not a target.

Put the target on the function that parses, not on the HTTP handler, or a failure will not say which of
routing, middleware, and parsing produced it. Effects enter as arguments: a fixed clock, a key set built in the
seed phase, a stub that always reports "first seen" for a replay store whose behavior is not what is under test.

**How much.** Seeds and the retained corpus run inside the ordinary test suite, cost milliseconds, and are the
part that earns its keep every day. Exploration is a separate, deliberate act: run it locally against what you
changed, on a time budget — `mise run test-go-fuzz -- <package> <target> <time>` for one target, and
`mise run test-go-fuzz-all -- <time>` for a sweep. Do not put exploration in the pull-request gate: it tries
different inputs each run, so it will eventually fail on a change that has nothing to do with the finding, and
a gate that fails for reasons unrelated to the change stops being read. What a crash costs forever is its
corpus entry, so minimize it, promote it to a named regression test, and fold inputs of one class into one
entry instead of accumulating raw findings.

Finally, check the oracle the way section 4 asks you to check any test: write the plausible wrong
implementation and watch the target catch it. Guards that stand separately in the code must be separated in the
evidence too. When one guard's cases are also caught by another, removing the first changes nothing and the
table says nothing about it. That is not hypothetical: a table meant to exercise a cost check here was entirely
shadowed by a length check standing in front of it, and the mutation run is what exposed the table as empty.

## 6. Current-state documents

`docs/` holds the cross-context structure and policy, one file per kind; `docs/contexts/<context>/` holds
that context's vocabulary, adopted standards, state transitions, decisions, mechanism, and acceptance
scenarios, again one file per kind. Each directory's `README.md` declares what it owns and indexes its
siblings.

Sections are not what divides the specification — files are. A context that grows does not grow one file;
it grows the file whose kind the new content belongs to, and the rest stay the size they were. Keep one
boundary declaration and one home for each fact; do not create a second copy beside one implementation
language.

A change that adds a trust boundary, a principal kind, an external integration, or a new kind of secret,
personal data, or record that must later be proven, revisits the threat model in the same pass. Those are
exactly the changes that introduce a threat with no control, and the threat model is the only document where
a missing control contradicts something. A threat recorded as accepted is revisited when the condition it
names is met.

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
writing the completion summary, so the recorded semantic difference is observed rather than recalled.

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
sources of truth or complete conformance. IdMagic's evidence contract is a repository-specific adaptation.

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
- **Property-Based Testing:** Koen Claessen and John Hughes's
  [QuickCheck: A Lightweight Tool for Random Testing of Haskell Programs](https://doi.org/10.1145/351240.351266)
  informs stating an invariant and letting generated input search for the counterexample, rather than
  enumerating the examples the author already had in mind.
- **Behavior-Driven Development:** Dan North's
  [Introducing BDD](https://dannorth.net/introducing-bdd/)
  informs behavior-oriented normative scenarios without introducing a second Gherkin source of truth.
- **Acceptance Test-Driven Development:** Robert C. Martin and Grigori Melnik's
  [Tests and Requirements, Requirements and Tests: A Möbius Strip](https://doi.org/10.1109/MS.2008.24)
  informs defining acceptance evidence before implementation.
- **Rational Reconstruction:** David L. Parnas and Paul C. Clements's
  [A Rational Design Process: How and Why to Fake It](https://doi.org/10.1109/TSE.1986.6312940)
  informs why a work item presents its evidence in the order the format asks for rather than the order the
  work happened in: real work is exploratory, and a record reproducing every detour would not be readable by
  the next person.
