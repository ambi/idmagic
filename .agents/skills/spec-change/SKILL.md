---
name: spec-change
description: Specification-first workflow for feature and behavior changes. Update TypeSpec for models, APIs, and authentication, and the owning canonical document under docs/ before implementation.
---

# Changing the specification first

Update the smallest owning specification before the implementation. `SPECIFICATION_FORMAT.md` defines the
current document kinds and grammar.

1. Put models, API operations, HTTP bindings, request and response shapes, status codes, error unions,
   deprecation metadata, and authentication mechanisms in
   `spec/contexts/<context>/{models,main}.tsp`.
2. Put context boundaries in `docs/domain/<context>/README.md`, vocabulary in `glossary.md`, and adopted
   protocol rules in `standards.md`. Put one feature's model, state machines, rules, errors, and security
   considerations in its feature specification `docs/domain/<context>/<group>/<feature>/README.md` (or a
   chapter beside it), and its examples in `examples.feature.md` in the same directory. Give each operation
   its own H3 under `## 操作`, open it with the summary table (行為者, 入力, 成功時の作用, 拒否, 冪等性), and
   order its rules normal path, state-dependent branches, then refusals. Put a context's allocated quality
   requirements in `docs/domain/<context>/quality.md` and a feature's share under its `## 品質` section. The feature node
   matches the code slice `backend/<context>/<feature>/` by name (drop the hyphens to compare names). Put a
   mechanism shared by features in `docs/domain/<context>/design/`, and a decision that weighed alternatives
   in `design/decisions.md`. Use the matching file under `docs/requirements/`, `docs/architecture/`,
   `docs/design/`, `docs/verification/`, or `docs/operations/` for a whole-system fact. A context still listed
   in `tools/check/legacy-spec-layout.json` keeps its per-kind files (`scenarios.feature.md`, `states.md`,
   `decisions.md`, `internals.md`) until it moves.
3. Give each new observable normative behavior an unused `REQ-<CONTEXT>-NNN`, declared as a
   `#### REQ-<CONTEXT>-NNN <title>` heading under the operation it governs. Retire a referenced behavior
   with `(superseded by REQ-<CONTEXT>-NNN)` in its heading rather than deleting or reusing its id. Write its
   body in the rule format: one obligation per bullet, a table when conditions combine, the `**担保手段**`
   field naming the code symbol, and the `**判断**` field for the reason. Add at least one example under a
   `## Rule:` heading with the same id and title in `examples.feature.md`. Put a value the product keeps (a
   limit, a period, a formula) in the rule, never only in the design. A rule that departs from a higher rule
   states only the departure and links it from `**上位の規則**`; do not copy the higher rule down.
4. Keep behavior that only several contexts can satisfy in `docs/scenarios.feature.md`, name the participating
   contexts, and keep context-local fragments out of their individual scenario files.
5. Keep fine-grained authorization behavior in code and tests unless the project adopts a policy language.
   TypeSpec records authentication and enforced operation scopes; `docs/design/security/authorization.md` owns the shared
   principal, scope, tenant-boundary, and fail-closed rules.
6. Sync the work item's `affected_spec` with the normative scenario or standard id, or the TypeSpec symbol.
7. Pass `mise run check-spec` and `mise run check-api-compat`. Regenerate derived views with `spec-render`
   when the specification changed; generated OpenAPI and HTML remain untracked views.

Use `update-design` as well only when bounded contexts, global structure, technology, runtime composition, or
core design rules change.
