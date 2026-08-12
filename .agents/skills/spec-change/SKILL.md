---
name: spec-change
description: Specification-first workflow for feature and behavior changes. Update TypeSpec for models, APIs, and authentication, and the owning SPECIFICATION.md for normative scenarios, glossary, standards, state transitions, authorization boundaries, and design before implementation.
---

# Changing the specification first

Update the smallest owning specification before the implementation.

1. Put models, APIs, HTTP bindings, status and error shapes, and authentication schemes in
   `spec/contexts/<context>/{models,main}.tsp`.
2. Put normative behavior, scenarios, glossary, standards, state transitions, and authorization
   boundaries in the section of the owning `SPECIFICATION.md` that owns that meaning.
3. Express state transitions as a language-independent `From | Event | Guard | To | Effects` table.
4. Give each new observable normative behavior an unused `REQ-<CONTEXT>-NNN`. Never reuse or reorder
   an existing id. Retire a behavior with `(superseded by REQ-<CONTEXT>-NNN)` in its heading rather
   than deleting it.
5. Keep behavior that holds only when several contexts cooperate in the document that owns the
   cross-context view — in this repository `spec/contexts/system/SPECIFICATION.md`, as
   `REQ-SYSTEM-NNN` — naming the participating contexts.
6. Do not add a fine-grained authorization policy DSL. TypeSpec records the authentication scheme;
   authorization behavior stays in code and tests.
7. Sync the work item's `affected_spec` with the normative scenario or standard id, or the TypeSpec
   symbol.
8. Pass `just check-spec` and `just check-api-compat`. Do not commit generated OpenAPI or HTML.

Use the `update-design` Skill as well, but only when structure, technology, or directory conventions
change too.
