# Specification-first Development Workflow

## 1. Purpose

This workflow keeps product behavior, current design, implementation, and verification aligned with a
small set of established formats. It favors direct ownership, generated views, and focused checks over
custom specification languages, exhaustive registries, and separate decision archives.

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

## 3. Specification-first workflow

Before changing behavior, update the smallest owning specification:

- Change a model, API, HTTP contract, or authentication scheme in TypeSpec.
- Change a scenario, term, standard, state transition, or durable design in the owning
  `SPECIFICATION.md`.
- Give each normative behavior an immutable `REQ-<CONTEXT>-NNN` ID.
- Express state machines as language-independent transition tables.

Requirements remain a product concept even though they do not require a catch-all document section.
Place each normative statement in the section that owns its meaning. Never discard one based only on a
matching TypeSpec operation name or duplicated summary.

Then implement from inner behavior to outer adapters, add tests at the affected boundaries, regenerate
derived views, and run the repository verification suite.

## 4. Current-state documents

The root document owns cross-context structure and policy. Each context document owns that context's
overview, vocabulary, adopted standards, state transitions, technical design, and acceptance
scenarios. Keep one overview and one durable rationale in the owning document; do not create a second copy
beside one implementation language.

TypeSpec and `SPECIFICATION.md` are colocated under `spec/` so backend, frontend, workers, and external
implementations see the same language-independent source. A generated HTML view provides cross-document
navigation without becoming an authored format.

## 5. Work items and decisions

A work item is the design and execution record for one meaningful change. It holds motivation, scope,
alternatives, plan, tasks, risks, and completion evidence. When the work lands, copy only the conclusion
that remains true into TypeSpec or the owning `SPECIFICATION.md`.

Do not create new ADRs. An existing decision archive may remain read-only for historical links, but current
design must be understandable without opening it.

## 6. Context economy

Read the work item, the owning `SPECIFICATION.md`, adjacent TypeSpec, and the affected implementation slice.
Search by requirement ID or TypeSpec symbol. Do not preload generated artifacts, historical decisions,
unrelated contexts, or repository-wide method documents for an ordinary feature change.
