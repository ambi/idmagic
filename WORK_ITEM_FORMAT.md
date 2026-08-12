# Work Item Format

A work item is one unit of work that can describe, design, implement, and verify one semantic change.
Pending items live in `work-items/`; completed or cancelled items live in `work-items/done/`. File names use
`wi-<sequence>-<kebab-title>.md`.

A work item is also the task list, change-specific design document, and implementation history for that
change. When the item is completed, conclusions that remain current must be reflected in TypeSpec or the
owning context's `SPECIFICATION.md`.

```markdown
---
status: pending
authors: [name]
risk: low
created_at: 2026-01-01
depends_on: []
change_kind: feature
initial_context: # written when the item starts, not when it is filed
  specification: [spec/contexts/system/SPECIFICATION.md#REQ-SYSTEM-001]
  typespec: [Product.System.Operations.StartTask]
  source: [backend/system]
  tests: [backend/system]
  stop_before_reading: [frontend]
affected_spec:
  - { path: spec/contexts/system/SPECIFICATION.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/system/main.tsp, symbol: Product.System.Operations.StartTask }
---

# One-sentence semantic change

## Motivation
Why the change is needed.

## Scope
- Specifications and implementation included in the change.

## Out of Scope
- Work explicitly excluded from the change.

## Design
The selected design, considerations, and rejected alternatives.

## Plan
Implementation order, migration, and unresolved questions.

## Tasks
- [ ] T001 [Spec] Update the specification.
- [ ] T002 [App] Confirm RED and implement the behavior.
- [ ] T003 [Verify] Verify the change.

## Verification
- `just verify`

## Risk Notes
Risks and mitigations.
```

`affected_spec` is required for `feature`, `bugfix`, and `operations` items. It directly references a
normative scenario/standard ID or a TypeSpec symbol. Changes with no specification impact (`refactor`,
`docs`, `tooling`, or `maintenance`) may use:

```yaml
spec_impact: { kind: none, reason: "A concrete reason." }
```

`initial_context` is the reading list one agent starts from. Write it when the item moves to
`in_progress`, not when it is filed: a list written for a backlog item rots before the work begins, and a
reading list that points at moved or deleted files is worse than none. A pending item needs only
Motivation, Scope, and Out of Scope to be useful.

Once the item is `in_progress`, `just check-work-items` resolves that list: every path must exist, and a
`spec/**/SPECIFICATION.md#REQ-<CONTEXT>-NNN` entry must name a scenario the document declares.

For medium and larger changes, make `Design` and `Plan` concrete. Domain, Use Cases, and Adapters tasks
must retain the corresponding test and normative scenario ID as self-evidence.

When the work is complete, set `status` to `completed`, append the following section, and move the file
to `work-items/done/`:

```markdown
## Completion
- **Completed At**: 2026-01-01
- **Summary**:
  The semantic difference introduced by the work.
- **Verification Results**:
  - `just verify` - passed
```
