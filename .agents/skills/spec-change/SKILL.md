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
2. Put context boundaries in `docs/contexts/<context>/README.md`, observable behavior in
   `docs/contexts/<context>/scenarios.md`, vocabulary in `glossary.md`, adopted protocol rules in
   `standards.md`, state machines in `states.md`, durable rationale in `decisions.md`, and durable mechanism
   that cannot be recovered from code in `internals.md`. Use the matching file directly under `docs/` for a
   whole-system fact.
3. Give each new observable normative behavior an unused `REQ-<CONTEXT>-NNN`. Retire a referenced behavior
   with `(superseded by REQ-<CONTEXT>-NNN)` in its heading rather than deleting or reusing its id.
4. Keep behavior that only several contexts can satisfy in `docs/scenarios.md`, name the participating
   contexts, and keep context-local fragments out of their individual scenario files.
5. Keep fine-grained authorization behavior in code and tests unless the project adopts a policy language.
   TypeSpec records authentication and enforced operation scopes; `docs/authorization.md` owns the shared
   principal, scope, tenant-boundary, and fail-closed rules.
6. Sync the work item's `affected_spec` with the normative scenario or standard id, or the TypeSpec symbol.
7. Pass `mise run check-spec` and `mise run check-api-compat`. Regenerate derived views with `spec-render`
   when the specification changed; generated OpenAPI and HTML remain untracked views.

Use `update-design` as well only when bounded contexts, global structure, technology, runtime composition, or
core design rules change.
