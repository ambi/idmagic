# Work Item Format

A work item is one unit of work that can describe, design, implement, and verify one semantic change.
Pending items live in `work-items/`; completed or cancelled items live in `work-items/done/`. File names use
`wi-<sequence>-<kebab-title>.md`.

A work item is also the task list, change-specific design document, and implementation history for that
change. When the item is completed, conclusions that remain current must be reflected in TypeSpec or the
canonical file that owns that kind of content.

```markdown
---
status: pending
authors: [name]
risk: low
created_at: 2026-01-01
priority: p1
depends_on: []
change_kind: feature
initial_context: # written when the item starts, not when it is filed
  specification: [docs/contexts/system/scenarios.md#REQ-SYSTEM-001]
  typespec: [Product.System.Operations.StartTask]
  source: [backend/system]
  tests: [backend/system]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
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
- `mise run verify`

## Risk Notes
Risks and mitigations.
```

`priority` (`p0`–`p3`) and `depends_on` answer different questions. `depends_on` states what must be
completed first; it is machine-checked and it constrains order. `priority` states what deserves attention
first among the items nothing blocks; it is advisory, and an item may be left unset to mean unranked.

When an item enters `in_progress`, add `evidence_policy: risk-based-v1`. For `medium`, `high`, and
`critical` risk, record the human approval before implementation begins:

```yaml
evidence_policy: risk-based-v1
approval:
  by: name
  at: 2026-01-01
  scope: "The behavior and design boundary that may be implemented."
  baseline: "The Git commit identifying the approved state."
```

`approval` is an implementation boundary, not permission to push, merge, operate production, or modify an
external system. If a normative change is discovered after approval, return to specification work, obtain
approval for the corrected scope, update the baseline, and retain an explanation in the completion record.
The risk-to-evidence rules live in [DEVELOPMENT.md](DEVELOPMENT.md#4-evidence-contract-and-approval).

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

Once the item is `in_progress`, `mise run check-work-items` resolves that list: every path must exist, and a
`docs/contexts/<context>/scenarios.md#REQ-<CONTEXT>-NNN` entry must name a scenario the document declares.

`affected_spec` is resolved for every record, completed ones included, because it indexes the normative
element the change touched rather than what someone read at the time. When a normative element moves to a
different file, repoint those references; when it is retired, the retired heading keeps them resolving.

For medium and larger changes, make `Design` and `Plan` concrete. Domain, Use Cases, and Adapters tasks
must retain the corresponding test and normative scenario ID as self-evidence. The independent verifier must
not be the person or agent that implemented the change; a fresh-context agent is acceptable.

When the work is complete, set `status` to `completed`, append the following section, and move the file
to `work-items/done/`:

```markdown
## Completion
- **Completed At**: 2026-01-01
- **Summary**:
  The semantic difference introduced by the work.
- **RED Evidence**:
  - **Test**: The failing test or check observed before implementation.
  - **Requirement**: `REQ-CONTEXT-NNN`, or `N/A: <reason>` for work with no normative product requirement.
  - **Observed Failure**: The expected failure that was actually observed.
  - **Detection Reason**: Why the assertion distinguishes a plausible wrong implementation from the required
    behavior. For work where RED is inapplicable, identify the alternate check that could have failed.
- **Post-Approval Changes**:
  For medium risk and above, the result of `mise run spec-diff <baseline>`, the reason for any normative
  change, and its reapproval. Write `none` when there was no change.
- **Independent Verification**:
  For medium risk and above, identify the independent verifier and summarize the standards and specification
  findings.
- **Change-Resistance Results**:
  For medium risk and above, record the representative incorrect implementation, diff mutation, or explicit
  fault injection and whether the tests detected it. For high and critical pure-logic changes, use a
  diff-scoped `mise run test-go-mutation-package -- <package> <git-ref>` check or explicit fault injection.
- **Verification Results**:
  - `mise run verify` - passed
```
