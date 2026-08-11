# Regenerative Architecture

## 1. Purpose

Regenerative Architecture keeps the product specification, current design, implementation, and verification mutually recoverable without forcing every concern into one custom language. The repository prefers established formats and small checks over exhaustive registries.

## 2. Sources of truth

| Concern | Source of truth |
|---|---|
| Models, API interfaces, HTTP bindings, authentication | TypeSpec (`spec/**/*.tsp`) |
| Requirements, scenarios, glossary, standards, state transitions | Markdown (`spec/**/requirements.md`) |
| Current technical design and durable rationale | Root or context `ARCHITECTURE.md` |
| One change's motivation, alternatives, plan, and history | `work-items/wi-*.md` |
| Executable behavior | Application code and tests |
| Released HTTP compatibility | `spec/idmagic.openapi.baseline.json` |

Generated OpenAPI is untracked and reproducible with `just spec-render`. Authorization policy evaluation remains application behavior; adopting Cedar or another policy language requires a separate work item.

## 3. Specification-first development

Before changing behavior, update the smallest relevant specification source:

- Change a model, API, or authentication scheme in the owning context's TypeSpec.
- Change a requirement, scenario, term, standard, or state transition in its `requirements.md`.
- Give each normative requirement a stable `REQ-<CONTEXT>-NNN` ID.
- Express state machines as language-independent transition tables with `From`, `Event`, `Guard`, `To`, and `Effects` columns. Implementation code is executable evidence, not the only documentation.

Then implement from inner behavior to outer adapters, writing failing tests first for domain, use-case, and adapter behavior. Run `just check` after specification changes and `just verify` before completion.

## 4. Architecture

Architecture documentation records the current design and the reason it is appropriate now. Root design belongs in root `ARCHITECTURE.md`; bounded-context design belongs beside that context's code. Directory layout and imports are themselves the module map, so no exhaustive module or edge ledger is maintained.

The boundary check infers layers from paths and rejects outward dependencies. Adding a permitted edge requires no registry edit. Complexity and style limits belong in language tool configuration.

## 5. Work items and decision history

A work item is the design and implementation record for one meaningful change. It holds motivation, scope, rejected alternatives, plan, tasks, verification, and completion evidence. Durable current-state conclusions are copied into the relevant specification or Architecture document when the work lands.

Do not create new ADRs. The existing `decisions/` directory is retained as read-only history so old links remain useful. A future change that revisits an old decision records the new analysis in its work item and updates current documentation directly.

## 6. Context economy

Read only the owning context's `requirements.md`, TypeSpec files, `ARCHITECTURE.md`, work item, and implementation slice. Search by requirement ID or TypeSpec symbol. Do not preload repository-wide meta-documents, generated OpenAPI, historical ADRs, or unrelated contexts.
