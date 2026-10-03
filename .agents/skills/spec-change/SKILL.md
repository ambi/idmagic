---
name: spec-change
description: Specification-first workflow for feature and behavior changes. Update TypeSpec for models, APIs, and authentication, and the owning canonical document under docs/ before implementation.
---

# Changing the specification first

Update the smallest owning specification before the implementation. `SPECIFICATION_FORMAT.md` defines the
current document kinds and grammar; read it, not its rationale document.

1. Write the requirement delta in the work item's Design first: the requirements to add, change, or retire,
   each with its id and requirement sentences. Apply the delta to the canonical documents afterwards. Never
   rewrite a whole feature to make one change.
2. Put models, API operations, HTTP bindings, request and response shapes, status codes, error unions,
   deprecation metadata, and authentication mechanisms in
   `spec/contexts/<context>/{models,main}.tsp`.
3. Put context boundaries in `docs/domain/<context>/README.md`, together with the value objects several
   features share (normalization, comparison, uniqueness scope); vocabulary in `glossary.md`; adopted protocol
   rules in `standards.md`. Put one feature's model, state machines, requirements, and security
   considerations in its feature specification `docs/domain/<context>/<group>/<feature>/README.md` (or a
   chapter beside it). Give each operation its own H3 under `## 操作` holding only requirements, ordered normal
   path, state-dependent branches, then refusals. Do not write a per-operation summary table or an `## エラー`
   section. Each state machine carries the state table, the transition table, and the state transition matrix
   with every cell filled. Put a context's allocated quality requirements in `docs/domain/<context>/quality.md`
   and a feature's share under `## 品質`. The feature node matches the code slice `backend/<context>/<feature>/`
   by name (drop the hyphens to compare names). Put a mechanism shared by features in
   `docs/domain/<context>/design/`, a decision that weighed alternatives in `design/decisions.md`, and only
   what code cannot show (how a mechanism guarantees a result, how to repair it) in the optional feature
   `design.md`. Use the matching file under `docs/requirements/` or `docs/design/` for a whole-system fact. A
   context still listed in `tools/check/legacy-spec-layout.json` keeps its per-kind files until it moves.
4. Before writing requirements, answer every question in the observation table of
   `docs/development/specification-first-workflow.md` (仕様の漏れを探す観点) for each operation you touch.
   An unanswered question is a gap. Classify each gap as (a) a requirement, (b) left unspecified, or
   (c) an implementation to change, as that document's 仕様にない振る舞いの分類 defines, and record the
   classification in the work item. Ask the user when a classification changes what users observe.
5. Give each new externally observable behavior an unused `REQ-<CONTEXT>-NNN`, declared as a
   `#### REQ-<CONTEXT>-NNN <title>` heading under the operation it governs. Write one obligation per bullet
   in one of the five EARS forms, a table when conditions combine, and the `**判断**` field only when the
   reason is not evident. Do not write `**担保手段**` or `**要判断**`; tests trace requirements through
   `//spec:covers`, and an open question becomes a work item. Retire a referenced behavior with
   `(superseded by REQ-<CONTEXT>-NNN)` in its heading rather than deleting or reusing its id. Put a value the
   product keeps (a limit, a period, a formula) in the requirement, never only in the design. State a
   requirement shared by several operations once, under the operation it governs most directly; the others
   link it from `**上位の要件**` and state only their departure.
6. Add an example in `examples.feature.md` only when a boundary or a surprising behavior is hard to read from
   the requirement sentences. Prefer generating it from `testdata/*.examples.json`. Never add an example that
   restates a requirement.
7. Keep behavior that only several contexts can satisfy in `docs/domain/scenarios.feature.md`, name the
   participating contexts, and keep context-local fragments out of it.
8. Keep fine-grained authorization behavior in code and tests unless the project adopts a policy language.
   TypeSpec records authentication and enforced operation scopes; `docs/design/security/authorization.md`
   owns the shared principal, scope, tenant-boundary, and fail-closed rules.
9. Sync the work item's `affected_spec` with the requirement or standard id, or the TypeSpec symbol.
10. Pass `mise run check-spec` and `mise run check-api-compat`. Regenerate derived views with `spec-render`
    when the specification changed; generated OpenAPI and HTML remain untracked views.

Use `update-design` as well only when bounded contexts, global structure, technology, runtime composition, or
core design rules change.
