---
name: spec-change
description: Specification-first workflow for feature and behavior changes. Update TypeSpec for models, APIs, and authentication, and the owning SPECIFICATION.md for normative scenarios, glossary, standards, state transitions, and design before implementation.
---

# Changing the specification first

Update the smallest owning specification before the implementation.

1. Put models, APIs, HTTP bindings, status and error shapes, and authentication schemes in
   `spec/contexts/<context>/{models,main}.tsp`.
2. Put normative behavior, scenarios, glossary, standards, and state transitions in the section of the
   owning `SPECIFICATION.md` that owns that meaning. Authorization boundaries have no section of their
   own: they belong to `Design`, under an `Authorization boundary` subsection named the same way in
   every document.
3. Before writing an `Overview`, read `spec/contexts/data-keys/SPECIFICATION.md` and
   `spec/contexts/sourcing/SPECIFICATION.md`. The first shows ownership stated together with what is
   delegated and to whom; the second shows the criterion that decides membership, with the adjacent
   cases it excludes named. Match them rather than the average of the other documents.
4. Express state transitions as a language-independent `From | Event | Guard | To | Effects` table.
5. Give each new observable normative behavior an unused `REQ-<CONTEXT>-NNN`. Never reuse or reorder
   an existing id. Retire a behavior with `(superseded by REQ-<CONTEXT>-NNN)` in its heading rather
   than deleting it.
6. Keep behavior that holds only when several contexts cooperate in the document that owns the
   cross-context view — in this repository `spec/contexts/system/SPECIFICATION.md`, as
   `REQ-SYSTEM-NNN` — naming the participating contexts.
7. Do not add a fine-grained authorization policy DSL. TypeSpec records the authentication scheme and
   each admin operation's `x-api-token-scopes`; authorization behavior stays in code and tests. Never
   restate the operation-to-scope mapping in prose — `just check-admin-scopes` verifies the annotation,
   nothing verifies the prose.
8. Sync the work item's `affected_spec` with the normative scenario or standard id, or the TypeSpec
   symbol.
9. Pass `just check-spec` and `just check-api-compat`. Do not commit generated OpenAPI or HTML.

Use the `update-design` Skill as well, but only when structure, technology, or directory conventions
change too.
