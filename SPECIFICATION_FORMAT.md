# 仕様書式

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
  README.md                     # system-document entry point and reading order
  product-overview.md           # purpose, users, situations, and system scope
  glossary.md                   # published language
  standards.md                  # external norms the whole system follows
  structure.md                  # repository and implementation layout
  scenarios.feature.md          # behavior no single context can satisfy alone
  requirements/                 # functional requirements, quality requirements, and constraints
  architecture/                 # system context, logical, runtime, and deployment views
  design/
    application/                # functional, API, UI, and software design
    data/                       # data and data-lifecycle design
    infrastructure/             # platform and network design
    security/                   # threat, authorization, and secret design
    reliability/                # availability, redundancy, and recovery design
    performance/                # performance, capacity, and scaling design
    observability/              # monitoring, logging, and tracing design
  contexts/<context>/
    README.md          # boundary declaration and index
    glossary.md
    standards.md
    states.md
    decisions.md
    internals.md       # only when a mechanism cannot be read out of the code
    scenarios.feature.md
  verification/                 # system verification and acceptance design
  development/                  # development workflow and procedures
  operations/                   # service management and maintenance
  runbooks/<event>.md           # what on-call reads mid-incident
  releases/                     # user-facing change and migration notices

spec/
  main.tsp
  tspconfig.yaml
  <product>.openapi.baseline.json
  contexts/<context>/{models.tsp,main.tsp}
```

The system tree is read from purpose through requirements, architecture, detailed design, verification, and
operation. This is an ownership and navigation order, not a one-pass lifecycle: feasibility and verification
findings return to their parent requirements and designs. A quality requirement is declared once under
`requirements/`; architecture allocates it, and the applicable design documents explain its realization.

Every fixed design directory has a `README.md` that declares its scope, exclusions, children, and adjacent
designs. Files below it are accepted only when the layout defines their responsibility. `docs/development/`,
`docs/runbooks/`, and `docs/releases/` remain open sets because procedures and change notices are not canonical
system-design kinds. `docs/operations/` and `docs/verification/` are fixed design sets even though they link to
procedures and evidence elsewhere.

`README.md` is the file a reader lands on when they open a directory, so it holds the boundary declaration
and child index. Create no file that has no content to hold: record an inapplicable concern and its reason in
the parent index. File names are *(checked)* for every fixed directory in the tree. A near miss reports the
intended name; another disallowed name reports the set that directory accepts. A small bounded context may
still need only `README.md` and `scenarios.feature.md`.

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

The root `README.md` additionally indexes the contexts themselves, and that index says of each one whether
it is `Core`, `Supporting`, or `Generic` *(checked)*. Every context directory appears exactly once, and a
directory the index does not list is rejected — a new context otherwise arrives unclassified and stays that
way, because nothing else in the layout ever asks. Which of the three a context is, and what the answer
governs, are decisions the checker cannot make; it only refuses to let the question go unanswered. The
reason a context sits where it does belongs in that context's `decisions.md`, not in the table, because
the reason differs per context and does not fit a cell. A workspace with no context directories needs no
such table.

A context owns only behavior it can satisfy and verify on its own. Behavior that holds only when several
contexts cooperate belongs to `docs/scenarios.feature.md`, and the scenario names the participating contexts.
Splitting such a flow into per-context fragments leaves no place where the real guarantee is stated.

### design-rules.md — how design choices are evaluated

The root `design-rules.md` owns the system-wide criteria for module interfaces, seams, adapters, type
ownership, effects, and errors. It states the current rule and the shape of a violation, so a reviewer can
apply it to a concrete change. It does not own directories or dependency direction (`structure.md`), the
rationale for one bounded decision (`decisions.md`), or mechanism that cannot be recovered from code
(`internals.md`). Context directories do not carry their own copy of this file.

### decisions.md — what was decided and why

One item per decision: what was decided, and why, each in a sentence. An item with no reason is a restated
rule, not a decision. The test is whether the code could be read to recover it; if it could, leave it out.
Include what was decided against, with the condition that would reopen it.

Make the heading the decision, never the aspect. `Invariants`, `Concurrency`, and `Failure handling` are
aspect names: a writer reads them as boxes to fill, and either invents prose for an aspect that does not
apply or splits one decision across several. Do not enumerate invariants at all — uniqueness and
referential integrity belong to the schema, observable properties to `scenarios.feature.md`, and construction and
postconditions to the type or operation as directed by `docs/design/application/design-rules.md`; the rest is unbounded. An
invariant worth writing down is usually a decision with a reason, and written as one it keeps the reason.

A decision large enough to need rejected alternatives, the conditions under which it holds, and the
condition that would reopen it gets a heading of its own.

### internals.md — how a mechanism works

Write this only when the working of a mechanism cannot be recovered from the code. The test is whether
someone could read the code alone and know how to fix the mechanism when it breaks. If they could, leave it
out. Write what is guaranteed, not the steps the implementation takes.

**How many contexts need one is a property of the domain, not a quota.** A context shaped like CRUD over a
table rarely has a mechanism worth explaining. One built on fail-closed refusals, key lifetimes, leases,
epochs, or same-transaction capture usually does, and a product made mostly of those will have one almost
everywhere. Never delete a file to reach an expected count: the test is the rule, and the count is whatever
the test leaves behind. The failure this guards against is the opposite of the obvious one — not a directory
of thin files, but the deletion of the few paragraphs that would have told the next person how to repair
something they cannot read out of the code.

Decisions and mechanism live in separate files because they have different lifetimes. A decision is
revisited when circumstances change and is audited as a list; a mechanism holds as long as the
implementation does and is read as prose. Together in one file, every audit of the first means skimming
past the second.

Neither file carries directory listings, package inventories, change history, comparisons of alternatives,
plans, summaries of external standards, states and transitions, acceptance examples, request and response
shapes, columns and indexes, permission assignments, or rules every context follows. Each of those has an
owner: the code, the work item, `standards.md`, `states.md`, `scenarios.feature.md`, TypeSpec, the schema file,
`docs/design/security/authorization.md`, or the matching file in the fixed system-document tree.

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

The same two columns read one step over when the product consumes a standard rather than provides it — a
client that sends, a receiver that follows someone else's contract. The capability is then what the product
exercises: `required` is what it always sends or always does, `optional` what it does only when a configured
or discovered condition holds, `partial` the part of the facility it uses, and `excluded` what it never
sends. An `optional` row written from that side states the condition and what goes out when the condition
does not hold, because a row that names neither cannot be told apart from a row that always sends.

`Statement` declares what the product does or refuses to do. It is not a summary of the standard's own
text — a row written from the standard's point of view reads as an obligation the product has accepted even
when `Adoption` says the opposite. IDs are stable and unique within the document *(checked)*; the value sets
for both columns are *(checked)*.

A row that nothing exercises is a claim, not a standard the product holds, so every id is named by a test
*(checked)*. The mention may sit anywhere in a test file rather than in the test's own name: an id pushed
into the name buys no more than an id written beside the assertion, and it costs the name the sentence that
says what the test does. Only tests count — an id mentioned in implementation code would let a comment carry
the whole obligation. Rows that predate the check are carried in a coverage debt list, one entry per id with
a reason for it, and that list only shrinks: an id it holds that has grown a test has to come off, and an id
added from here on is not admitted to it *(checked)*.

## 6. Scenarios and normative IDs

`scenarios.feature.md` is the sole source of truth for observable, non-negotiable behavior. It uses
[Markdown with Gherkin](https://github.com/cucumber/gherkin/blob/main/MARKDOWN_WITH_GHERKIN.md): one
`Feature` per file, one normative behavior per `Rule`, and one branch-free path per `Example`. The official
JavaScript Gherkin parser accepts every file *(checked)*. The `REQ-*` marker carries normative force; do not
add a second boilerplate sentence using `SHALL` or `MUST`.

IDs are immutable once referenced. Models and external interfaces belong in TypeSpec; adopted protocol
rules belong in `standards.md`; lifecycle invariants belong in `states.md`; decisions belong in
`decisions.md` and mechanism in `internals.md`. Do not duplicate those concerns as prose requirements. Use
`SHOULD` or `MAY` only in explanatory or standards policy text when an exception or option is genuinely
intended.

Retire a behavior instead of deleting it. Mark the rule, drop its examples, and state the successor:

```markdown
## Rule: REQ-ACCOUNT-002 a valid session opens the account (superseded by REQ-ACCOUNT-042)

Replaced by session-scoped account access.
```

Every live rule has at least one example *(checked)*. A normal `Example` starts its name with
`EX-<CONTEXT>-<REQ-NNN>-<sequence>`. A `Scenario Outline` puts that identifier in the `example_id` column of
each `Examples` row. Example IDs are immutable, globally unique, and belong to the `REQ-*` named by their
parent rule *(checked)*. A rule is covered only when every child example is named by a test or appears in the
reasoned example-coverage debt list *(checked)*. A test that names only the parent `REQ-*` does not cover any
child example. A retired rule is exempt because it has no executable example.

The successor must exist *(checked)*. A retired ID is never reused. Deleting the heading outright leaves
nothing saying the behavior existed, and the work item alone cannot be searched by ID. Before retiring,
map every precondition, postcondition, failure case, and invariant to its new owner — TypeSpec, scenarios,
`standards.md`, `states.md`, or `decisions.md`. A title matching a TypeSpec operation is not by itself
evidence that a scenario became redundant.

Use the English Markdown with Gherkin keywords and Japanese prose. Put the actor in the `When` subject;
`ACTOR` is not a step. Use `Given` for state, `When` for the trigger, and `Then` for observable results.
`And` and `But` continue the preceding step kind. Each example has at least one `When` and one `Then`
*(checked)*:

```markdown
# Feature: Account

## Rule: REQ-ACCOUNT-002 a valid session opens the account

### Example: EX-ACCOUNT-002-01 an available account is returned

- Given the end user has a valid session
- When the end user requests the account summary
- Then the account summary is returned
- And the activity timestamp is updated

### Example: EX-ACCOUNT-002-02 an unavailable account is not returned

- Given the end user has a valid session
- And the account is unavailable
- When the end user requests the account summary
- Then an error is returned
- And no account summary is returned
```

Do not encode a branch inside an example. An alternate success or refusal is another `Example`; multiple
observations are separate `Then` or `And` steps. Multi-operation flows may repeat `When` and `Then`.

Use `Scenario Outline` when paths have the same step structure and differ only by values. Every executable
row has a nonempty, unique `example_id`. A set of independent conditions that determines an outcome may be
named `Examples: Decision table (Unique)`. `Unique` is the only supported hit policy: each executable input
matches exactly one row. `any` in a condition cell means that condition does not affect the row's outcome.
Do not use `-` for this purpose: the official Markdown matcher treats any row containing a hyphen-only cell
as a GFM table separator and omits it from the AST. A decision row has at least one nonempty outcome cell
*(checked)*. Ordinary representative or boundary data remains a plain `Examples` table. Do not collapse
ordering, history, retry, or elapsed-time behavior into a decision table.

A refusal a security control is responsible for — an unauthorized caller, another tenant's resource, a
request that cannot prove it came from the product's own UI, a token without the scope, a decision that
cannot be made — is observable behavior, and belongs in the scenario on the same footing as the path that
succeeds. Write it as its own `Example` under the owning `Rule`. A control whose refusal is written down nowhere has nothing to be checked
against: an implementation that stops refusing then contradicts no statement, and the specification cannot
say the product regressed. Give each refusal its own example.

State what the caller observes and what the refusal leaves untouched. "Rejected with an error" is only
half of it; the half that matters to a reader deciding whether the control works is that the operation had
no effect.

## 7. Authorization

Authorization is not a section of each context. It is `docs/design/security/authorization.md`, because someone checking
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

## 8. Threat model

The other documents state what the product does, so an implemented control can be checked against them. A
control that was never built contradicts nothing: no scenario declares it, no test names it, and the refusal
coverage check has no declaration to look for. `docs/design/security/threat-model.md` is where that gap becomes visible. It
holds the trust boundaries and what is not trusted at each, the assets, and one row per identified threat
naming the control that answers it.

Give every threat a stable id, and a status from a closed set that separates a threat with a control from
one without. Do not encode the boundary or the category into the id: both are reclassified as the system
changes, and an id that carries them becomes a lie the moment it is. Carry them in columns instead.

The rows with no control are the document's main output, so keep them in the same table rather than in a
separate debt file. A reader who finishes the list must not be able to finish it without seeing them. State
for each whether it will be fixed or is accepted, and an accepted threat carries the condition that would
reopen it. An acceptance with no such condition is neglect with a label on it.

Separate a threat nothing answers from one a control answers without a norm behind it — a procedure, a
deployment requirement, a property of the toolchain. Both are unfinished, and they are not the same kind of
unfinished, so a status alone cannot carry the difference. Let the control column carry it. Do not put the
work item that will fix a threat into that column: a current-state document holds what exists, and a forward
reference rots the moment the work item is completed and moved. Point from the work item to the threat id
instead, so the link is findable and the direction stays one way.

Say in the document that the list is not exhaustive, and name what obliges a revisit. A threat model read as
a complete guarantee is worse than none, because a threat that was never considered becomes indistinguishable
from one considered and dismissed.

Reference existing control identifiers — normative scenarios, adopted standards, the rules in another
canonical document. Do not mint a second identifier for a control that already has one. Write what could
happen, never how: reproduction steps, concrete parameters, and the details of an unfixed path do not belong
in a specification.

## 9. Generated views and validation

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
