# Specification Format

The exact, current grammar is whatever `just check-spec` accepts; its diagnostics are the precise rule.
This document states intent, examples, and the decisions a checker cannot make for you. Rules marked
*(checked)* fail the build; the rest are review judgment.

## 1. Layout

```text
spec/
  main.tsp
  tspconfig.yaml
  SPECIFICATION.md
  <product>.openapi.baseline.json
  contexts/<context>/
    models.tsp
    main.tsp
    SPECIFICATION.md
```

`main.tsp` composes the TypeSpec program. `models.tsp` owns model declarations and the context `main.tsp`
owns operations. Generated OpenAPI and documentation live below ignored `spec/generated/`.

## 2. TypeSpec scope

Use TypeSpec for models, constraints, API operations, HTTP routes, request and response shapes, status codes,
error unions, deprecation metadata, and authentication mechanisms. Prefer standard libraries and emitters.
Each operation must inherit an OpenAPI tag from its owning context; do not leave operations in `default`.

Keep stable wire names when source ownership moves. Do not recreate TypeSpec constructs in Markdown or a
project-specific YAML dialect.

## 3. Canonical document

Every root or context `SPECIFICATION.md` has frontmatter with a lowercase context slug and update date, one
H1 title, and these H2 sections in order when applicable:

1. `Overview`
2. `Glossary`
3. `Standards`
4. `State Transitions`
5. `Design`
6. `Scenarios`

`Overview` is required. Omit empty optional sections. Frontmatter, the single H1, the section set, and
their order are *(checked)*. Put the overview in this document once; do not repeat it in an
implementation-side design file.

`Overview` states what the context owns, what it does not own and which owner takes it instead, and — when
membership is easy to get wrong — the criterion that decides. It is a boundary declaration, not a guide to
the document. A reading order and a plan for later both fail that test: the first describes the file rather
than the system, and the second describes a system that does not exist yet. Keep planned work in the work
item; a deliberate non-adoption belongs in `Design` with the condition that would reopen it.

```markdown
## Overview
<!-- good: ownership, delegation, and the criterion that settles the hard cases -->
Owns the lifecycle of X and the metadata around it.
Does not own the cryptography itself; that is a shared adapter. Signing keys belong to <other context>.
Membership follows whether a persistent external authority exists, not the direction of traffic, so
an administrator-driven import belongs to <other context> instead.

<!-- bad: a guide to the document, and a plan -->
This document covers A, then B, then C, in that order.
A fourth source kind will be added to this context later.
```

A context document owns only behavior that context can satisfy and verify on its own. Behavior that holds
only when several contexts cooperate belongs to whichever document declares ownership of the
cross-context view — the root document, or a system-level context that says so in its `Overview` — and
names the participating contexts in the scenario. Splitting such a flow into per-context fragments leaves
no place where the real guarantee is stated.

`Design` records current structure, dependency direction, runtime composition, adopted technologies,
security boundaries, operational constraints, and concise durable rationale. Change-specific comparisons
and rejected alternatives stay in the work item.

Authorization boundaries are one of those security boundaries and have no section of their own. Give them
a subsection of `Design` and keep the name identical across documents, so a reviewer can still read every
boundary in one sweep. Where the boundary is a single sentence, state it beside the mechanism it
constrains rather than under a heading of its own.

## 4. State transitions

Use a language-independent table for every state machine:

```markdown
| From | Event | Guard | To | Effects |
|---|---|---|---|---|
```

Use `—` in `Guard` when the transition is unconditional *(checked)*. Do not use an empty string literal
such as `""`; it looks like an executable condition in both the source table and the derived diagram.

Application code and tests are executable evidence, not the only state-transition documentation.
The table is the normative source. The generated specification site derives one Mermaid state diagram
from the same rows and displays the table with the diagram; do not maintain a second hand-written state
diagram for the same machine.

## 5. Standards

Give every adopted standard a subsection named after it, a source URL on its own line, and one table:

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

A scenario heading identifies one observable, non-negotiable behavior. The `REQ-*` marker already carries
normative force. Do not add a separate `Requirements` section or a second boilerplate sentence using
`SHALL` or `MUST`. Write one behavior per scenario so a tester can determine whether it holds.

IDs are immutable once referenced. Models and external interfaces belong in TypeSpec; adopted protocol
rules belong in Standards; lifecycle invariants belong in State Transitions; internal interface and
structural constraints belong in Design. Do not duplicate those concerns as prose requirements. Use
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
Standards, State Transitions, or Design. A title matching a TypeSpec operation is not by itself evidence
that a scenario became redundant.

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

## 7. Authorization

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
- Generate OpenAPI and a multi-page, navigation-linked HTML site from TypeSpec and `SPECIFICATION.md`
  sources. The entry point is `spec/generated/docs/index.html`; Method, whole-system, context, API, and
  model content are separate pages.
- Delegate API operation/schema presentation to an OpenAPI-native viewer over the generated OpenAPI.
  Generate the broader model catalog from repository-owned TypeSpec model, enum, union, and scalar
  declarations, including declarations not reachable from HTTP operations. Transport wrapper declarations
  in `Operations` namespaces belong to the OpenAPI reference and are not duplicated in the catalog.
- Render Mermaid fences from canonical documents and derive state diagrams from normative transition
  tables. Scenario keywords remain plain Markdown grammar in the source and receive semantic styling in
  the generated view.
- Current `SPECIFICATION.md` files must be self-contained: state design and rationale inline rather than
  linking out to another document for it.
- Never edit or treat generated HTML/OpenAPI as normative source.
