# Specification Format

The exact, current grammar is whatever `mise run check-spec` accepts; its diagnostics are the precise rule.
This document states intent, examples, and the decisions a checker cannot make for you. Rules marked
*(checked)* fail the build; the rest are review judgment.

## 1. Layout

Sections do not divide the specification; files do. A file's name says what kind of content it holds, and
that name is what the checker validates the body against.

Prose lives under `docs/`; the TypeSpec a compiler consumes lives under `spec/`. The two trees mirror each
other at `contexts/<context>/`, so one context's specification is one name looked up in two places — the
prose in `docs/contexts/oauth2/`, the contract in `spec/contexts/oauth2/`.

```text
docs/
  README.md            # boundary declaration, context map, index
  product-overview.md  # problem, users, non-goals
  structure.md         # directories, dependency direction, layers, architecture style
  glossary.md          # published language
  standards.md         # external norms the whole system follows
  api-rules.md         # rules for externally visible contracts
  observability.md     # correlation, logs, metrics
  deployment.md        # runtime units, trust boundaries, availability
  capacity.md          # assumed scale, how limits are set, degradation
  persistence.md       # database design policy
  authorization.md     # principals, scopes, authorization boundaries
  scenarios.md         # behavior no single context can satisfy alone
  contexts/<context>/
    README.md          # boundary declaration and index
    glossary.md
    standards.md
    states.md
    decisions.md
    internals.md       # rare; only when a mechanism needs explaining
    scenarios.md
  development/         # procedures: environment, build, generation, CI, testing
  operations/          # SLO, release and rollback, backup, <event>.md

spec/
  main.tsp
  tspconfig.yaml
  <product>.openapi.baseline.json
  contexts/<context>/{models.tsp,main.tsp}
```

`README.md` is the file a reader lands on when they open the directory, so it holds the boundary
declaration and the index of its siblings. Create no file that has no content to hold: a small context
needs only `README.md` and `scenarios.md`. The file set and the file names are *(checked)* for `docs/` and
`docs/contexts/<context>/`; anything else at those two levels is not a canonical document and is rejected.
`docs/development/` and `docs/operations/` sit below that closed set and name their files freely, because
procedures are not a fixed set of kinds.

`main.tsp` composes the TypeSpec program. `models.tsp` owns model declarations and the context `main.tsp`
owns operations. Generated OpenAPI and documentation live below ignored `spec/generated/`, which keeps
every generated artifact out of `docs/`: everything under `docs/` is written by a person.

## 2. TypeSpec scope

Use TypeSpec for models, constraints, API operations, HTTP routes, request and response shapes, status codes,
error unions, deprecation metadata, and authentication mechanisms. Prefer standard libraries and emitters.
Each operation must inherit an OpenAPI tag from its owning context; do not leave operations in `default`.

Keep stable wire names when source ownership moves. Do not recreate TypeSpec constructs in Markdown or a
project-specific YAML dialect.

## 3. The canonical documents

Every canonical document has exactly one H1 *(checked)*. There is no frontmatter and no fixed section set:
the file name has already said what the file holds, so what would have been a section is now a file, and
its H2s are free to name what the content actually is.

### README.md — the boundary declaration

`README.md` states what the directory owns, what it does not own and which owner takes it instead, and —
when membership is easy to get wrong — the criterion that decides. It is a boundary declaration, not a
guide to the document. A reading order and a plan for later both fail that test: the first describes the
file rather than the system, and the second describes a system that does not exist yet. Keep planned work
in the work item; a deliberate non-adoption belongs in `decisions.md` with the condition that would reopen
it.

```markdown
# Directory
<!-- good: ownership, delegation, and the criterion that settles the hard cases -->
Owns the lifecycle of X and the metadata around it.
Does not own the cryptography itself; that is a shared adapter. Signing keys belong to <other context>.
Membership follows whether a persistent external authority exists, not the direction of traffic, so
an administrator-driven import belongs to <other context> instead.

<!-- bad: a guide to the document, and a plan -->
This document covers A, then B, then C, in that order.
A fourth source kind will be added to this context later.
```

Below the declaration, index the sibling files as Markdown links. That index is what makes them reachable
from the generated site, and it is the only place a reader is told which of them exist.

A context owns only behavior it can satisfy and verify on its own. Behavior that holds only when several
contexts cooperate belongs to `docs/scenarios.md`, and the scenario names the participating contexts.
Splitting such a flow into per-context fragments leaves no place where the real guarantee is stated.

### decisions.md — what was decided and why

One item per decision: what was decided, and why, each in a sentence. An item with no reason is a restated
rule, not a decision. The test is whether the code could be read to recover it; if it could, leave it out.
Include what was decided against, with the condition that would reopen it.

Make the heading the decision, never the aspect. `Invariants`, `Concurrency`, and `Failure handling` are
aspect names: a writer reads them as boxes to fill, and either invents prose for an aspect that does not
apply or splits one decision across several. Do not enumerate invariants at all — uniqueness and
referential integrity belong to the schema, observable properties to `scenarios.md`, and the rest is
unbounded. An invariant worth writing down is usually a decision with a reason, and written as one it
keeps the reason.

A decision large enough to need rejected alternatives, the conditions under which it holds, and the
condition that would reopen it gets a heading of its own.

### internals.md — how a mechanism works

Write this only when the working of a mechanism cannot be recovered from the code. **Most contexts do not
need it.** The test is whether someone could read the code alone and know how to fix the mechanism when it
breaks. If they could, leave it out. Write what is guaranteed, not the steps the implementation takes.

Decisions and mechanism live in separate files because they have different lifetimes. A decision is
revisited when circumstances change and is audited as a list; a mechanism holds as long as the
implementation does and is read as prose. Together in one file, every audit of the first means skimming
past the second.

Neither file carries directory listings, package inventories, change history, comparisons of alternatives,
plans, summaries of external standards, states and transitions, acceptance examples, request and response
shapes, columns and indexes, permission assignments, or rules every context follows. Each of those has an
owner: the code, the work item, `standards.md`, `states.md`, `scenarios.md`, TypeSpec, the schema file,
`docs/authorization.md`, or the matching file directly under `docs/`.

## 4. State transitions

`states.md` gives every state machine an H2 heading and two language-independent tables under it, the
states first and then the transitions:

```markdown
| State | Kind | Meaning |
|---|---|---|

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
```

`Kind` is `initial`, `terminal`, or `—`. Exactly one state is `initial`; any number may be `terminal`
*(checked)*. Every `From` and `To` must name a state the state table declares *(checked)*.

The state table exists because the set of states is otherwise never stated. Derived from the `From` and
`To` columns it silently loses any state nothing transitions into, and no state gets to say in one line
what it means — which is exactly what a reader needs to tell two similar-looking waiting states apart.

Use `—` in `Guard` when the transition is unconditional *(checked)*. Do not use an empty string literal
such as `""`; it looks like an executable condition in both the source table and the derived diagram. A
guard may contain an escaped pipe, which does not end the cell.

Application code and tests are executable evidence, not the only state-transition documentation.
The tables are the normative source. The generated specification site derives one Mermaid state diagram
from the transition rows and displays the tables with the diagram; do not maintain a second hand-written
state diagram for the same machine.

## 5. Standards

In `standards.md`, give every adopted standard an H2 named after it, a source URL on its own line, and one
table:

```markdown
| ID | Adoption | Strength | Statement |
|---|---|---|---|
```

`Adoption` says whether the product takes the standard's capability at all — `required` when it is always
provided, `optional` when it is provided but its use is the caller's choice, `partial` when only some of it
is, `excluded` when it is not. `Strength` says how firmly the product holds the rule once taken, in RFC 2119
keywords: `MUST`, `MUST NOT`, `SHOULD`, `MAY`. The two are independent axes, and `optional` with `MUST` is
the ordinary case rather than a contradiction: offering the capability is a choice, honoring the rule once
offered is not. An `excluded` row cannot carry `MUST` or `SHOULD`, because there is no obligation to state
about a capability the product does not provide *(checked)*.

`Statement` declares what the product does or refuses to do. It is not a summary of the standard's own
text — a row written from the standard's point of view reads as an obligation the product has accepted even
when `Adoption` says the opposite. IDs are stable and unique within the document *(checked)*; the value sets
for both columns are *(checked)*.

## 6. Scenarios and normative IDs

In `scenarios.md`, an H3 heading identifies one observable, non-negotiable behavior. Scenario headings
stay at H3 even though nothing sits above them: `### REQ-...` is the form every reference, every checker,
and every generated anchor already uses, and the identifier is what carries normative force, not the
heading level. The `REQ-*` marker already carries
normative force. Do not add a separate `Requirements` section or a second boilerplate sentence using
`SHALL` or `MUST`. Write one behavior per scenario so a tester can determine whether it holds.

IDs are immutable once referenced. Models and external interfaces belong in TypeSpec; adopted protocol
rules belong in `standards.md`; lifecycle invariants belong in `states.md`; decisions belong in
`decisions.md` and mechanism in `internals.md`. Do not duplicate those concerns as prose requirements. Use
`SHOULD` or `MAY` only in explanatory or standards policy text when an exception or option is genuinely
intended.

Retire a behavior instead of deleting it. Mark the heading, drop the steps, and state the successor:

```markdown
### REQ-ACCOUNT-002: a valid session opens the account (superseded by REQ-ACCOUNT-042)
Replaced by session-scoped account access.
```

The successor must exist *(checked)*. A retired ID is never reused. Deleting the heading outright leaves
nothing saying the behavior existed, and the work item alone cannot be searched by ID. Before retiring,
map every precondition, postcondition, failure case, and invariant to its new owner — TypeSpec, Scenarios,
`standards.md`, `states.md`, or `decisions.md`. A title matching a TypeSpec operation is not by itself
evidence that a scenario became redundant.

Use uppercase keywords without colons *(checked)*. Nest an alternative immediately below the `WHEN` or
`THEN` step that it replaces or interrupts. The Markdown structure carries the relationship; do not add
local step numbers:

```markdown
### REQ-ACCOUNT-002: a valid session opens the account
- ACTOR EndUser
- GIVEN a valid session
- WHEN the account summary is requested
- THEN the account summary is returned
  - ALT the account is unavailable → an error is returned without an account summary
- THEN the activity timestamp is updated
```

`GIVEN` describes only state or preconditions that already hold before the behavior starts. `WHEN`
describes an operation, input, or external event that triggers the behavior. `THEN` describes an
observable result after that trigger. Split a sentence if it mixes a trigger and its result. Every
scenario has one or more `WHEN` and one or more `THEN`; multi-operation flows may repeat them.

An `ALT` is a two-space-indented child list item of exactly one `WHEN` or `THEN`. It separates its
condition and result with `→`. An alternative to setup belongs in a separate scenario or under the
operation whose behavior changes, not under `GIVEN`.

A refusal a security control is responsible for — an unauthorized caller, another tenant's resource, a
request that cannot prove it came from the product's own UI, a token without the scope, a decision that
cannot be made — is observable behavior, and belongs in the scenario on the same footing as the path that
succeeds. Write it as an `ALT` under the operation it refuses, or as its own scenario when the refusal is
the behavior being specified. A control whose refusal is written down nowhere has nothing to be checked
against: an implementation that stops refusing then contradicts no statement, and the specification cannot
say the product regressed.

State what the caller observes and what the refusal leaves untouched. "Rejected with an error" is only
half of it; the half that matters to a reader deciding whether the control works is that the operation had
no effect.

## 7. Authorization

Authorization is not a section of each context. It is `docs/authorization.md`, because someone checking
authorization wants the product's authorization, not one context's share of it. That file holds the
principal kinds, the scope namespaces, the tenant boundary, and the rules that apply when a decision
cannot be made. What one context decides about its own operations stays in that context's
`decisions.md`.

TypeSpec records whether an API is authenticated or public, and carries whatever per-operation permission
or scope annotation the project enforces. Fine-grained authorization is executable application behavior
with tests unless the project explicitly adopts a standard policy language. Do not add an ad hoc
authorization DSL to the specification format.

Which operation requires which scope is therefore contract data, checkable against the vocabulary it draws
from. Do not restate that mapping as prose: two copies of one mapping cannot be diffed against each other,
and the unchecked copy is the one that goes stale. Prose records what the annotation cannot — the roles and
principal kinds a boundary admits, what a response may never carry, whether authority propagates to a
downstream call, what stays inside a tenant, and what happens when the decision cannot be made. Name the
scope vocabulary a context uses, and the conclusion and reason wherever an operation's assignment does not
follow from its name.

## 8. Generated views and validation

- Compile TypeSpec and validate canonical documents through the repository's specification check.
- Compare generated OpenAPI with the released baseline for compatibility.
- Generate OpenAPI and a multi-page, navigation-linked HTML site from TypeSpec and the canonical Markdown.
  The entry point is `spec/generated/docs/index.html`; Method, whole-system, context, API, and model
  content are separate pages, and each canonical file is its own page reached from its directory's
  `README.md`.
- Delegate API operation/schema presentation to an OpenAPI-native viewer over the generated OpenAPI.
  Generate the broader model catalog from repository-owned TypeSpec model, enum, union, and scalar
  declarations, including declarations not reachable from HTTP operations. Transport wrapper declarations
  in `Operations` namespaces belong to the OpenAPI reference and are not duplicated in the catalog.
- Render Mermaid fences from canonical documents and derive state diagrams from normative transition
  tables. Scenario keywords remain plain Markdown grammar in the source and receive semantic styling in
  the generated view.
- Canonical documents state design and rationale inline rather than linking out to a separate decision
  archive for it. Linking between canonical files is how the layout works; linking to `decisions/` is
  rejected *(checked)*.
- Never edit or treat generated HTML/OpenAPI as normative source.
