# Specification Format

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
5. `Authorization Boundary`
6. `Design`
7. `Scenarios`

`Overview` is required. Omit empty optional sections. Put the overview in this document once; do not repeat
it in an implementation-side design file.

`Design` records current structure, dependency direction, runtime composition, adopted technologies,
security boundaries, operational constraints, and concise durable rationale. Change-specific comparisons
and rejected alternatives stay in the work item.

## 4. State transitions

Use a language-independent table for every state machine:

```markdown
| From | Event | Guard | To | Effects |
|---|---|---|---|---|
```

Use `—` in `Guard` when the transition is unconditional. Do not use an empty string literal such as `""`;
it looks like an executable condition in both the source table and the derived diagram.

Application code and tests are executable evidence, not the only state-transition documentation.
The table is the normative source. The generated specification site derives one Mermaid state diagram
from the same rows and displays the table with the diagram; do not maintain a second hand-written state
diagram for the same machine.

## 5. Scenarios and normative IDs

A scenario heading identifies one observable, non-negotiable behavior. The `REQ-*` marker already carries
normative force. Do not add a separate `Requirements` section or a second boilerplate sentence using
`SHALL` or `MUST`. Write one behavior per scenario so a tester can determine whether it holds.

IDs are immutable once referenced. Models and external interfaces belong in TypeSpec; adopted protocol
rules belong in Standards; lifecycle invariants belong in State Transitions; internal interface and
structural constraints belong in Design. Do not duplicate those concerns as prose requirements. Use
`SHOULD` or `MAY` only in explanatory or standards policy text when an exception or option is genuinely
intended.

Do not remove an existing requirement merely because its title matches a TypeSpec operation. First map
every normative precondition, postcondition, failure case, and invariant to TypeSpec, Scenarios, Standards,
State Transitions, Authorization Boundary, or Design. A standalone Requirements section is unnecessary
only when that semantic audit leaves no requirement owned solely by it.

Use uppercase keywords without colons. Nest an alternative immediately below the `WHEN` or `THEN` step
that it replaces or interrupts. The Markdown structure carries the relationship; do not add local step
numbers:

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

## 6. Authorization

TypeSpec records whether an API is authenticated or public. Fine-grained authorization is executable
application behavior with tests unless the project explicitly adopts a standard policy language. Do not
add an ad hoc authorization DSL to the specification format.

## 7. Generated views and validation

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
- Current `SPECIFICATION.md` files must be self-contained and must not link to the historical ADR archive.
  Move still-current conclusions and concise rationale into the owning Design section before removing a
  historical link.
- Never edit or treat generated HTML/OpenAPI as normative source.
