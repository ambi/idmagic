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
reversibility: reversible # optional; decide it, do not inherit it from this template
created_at: 2026-01-01
priority: p1
depends_on: []
change_kind: feature
evidence_policy: risk-based-v3 # required after the item starts
initial_context: # written when the item starts, not when it is filed
  specification: [docs/contexts/system/scenarios.md#REQ-SYSTEM-001]
  typespec: [Product.System.Operations.StartTask]
  source: [backend/system]
  tests: [backend/system]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
  - { path: spec/contexts/system/main.tsp, symbol: Product.System.Operations.StartTask }
primary_use_cases: # required for feature, bugfix, and standards.md work after it starts
  - id: start-task
    requirement: REQ-SYSTEM-001
    observable_result: The caller observes the task running.
    unit_test: { path: backend/system/usecases/start_task_test.go, name: TestStartTask_REQ_SYSTEM_001, task: test-go-race }
    e2e_test: { path: backend/system/e2e_test.go, name: TestE2E_StartTask_REQ_SYSTEM_001, task: test-go-race }
    unit_fault_model: The use case does not emit the start command.
    e2e_fault_model: The configured route does not connect the handler to the use case.
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
Implementation order, migration, and open questions. Resolve every question that would change what gets built
before implementation.

## Tasks
- [ ] T001 [Spec] Update the specification.
- [ ] T002 [Acceptance] Confirm Acceptance RED at an observable boundary.
- [ ] T003 [App] Confirm Unit RED, reach GREEN, and refactor the behavior.
- [ ] T004 [Verify] Verify the change.

## Verification
- `mise run verify`

## Risk Notes
Risks and mitigations.
```

`priority` (`p0`–`p3`) and `depends_on` answer different questions. `depends_on` states what must be
completed first; it is machine-checked and it constrains order. `priority` states what deserves attention
first among the items nothing blocks; it is advisory, and an item may be left unset to mean unranked.

`risk` and `reversibility` answer different questions too. `risk` states how much damage the change does when
it is wrong; `reversibility` states whether the decision it makes can be taken back afterwards. The two vary
independently: a replica topology, a cache policy, or a screen layout can be severe and still reversible,
while a wire format, the meaning of an identifier, a published schema, a destroyed key, or an assigned `REQ`
number is irreversible however small it looked. Write `irreversible` when undoing the decision would require
someone outside this repository to change what they already store, send, or trust.

`reversibility` selects no evidence of its own; it records which decisions cannot be withdrawn, so that a
later reader can tell a choice that is still open from one the repository now has to live with. It never
relaxes anything either: declaring an item `reversible` leaves its `risk` contract exactly as it was, because
a relaxation path would turn the declaration into a way around the evidence. The field is optional so that
records written before it existed stay valid, and an unstated value means the axis was not assessed rather
than that the change is reversible.

When an item enters `in_progress`, add `evidence_policy: risk-based-v3`. The risk selects the evidence the
item must produce before it can be completed; it grants no permission to push, merge, operate production, or
modify an external system. Filing the item is what authorizes the work, so the item keeps no separate
approval record. Resolve every open question that would change product behavior, the public contract, the
selected design boundary, or the task breakdown before implementation begins. If implementation discovers a
normative change, return to specification work; never weaken a scenario to let an implementation pass. The
risk-to-evidence rules live in [specification-first-workflow.md](docs/development/specification-first-workflow.md#4-evidence-contract).

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

For medium and larger changes, make `Design` and `Plan` concrete. For changed core logic, name the principal
domain data types and operation signatures, and identify time, randomness, identifier generation,
configuration, persistence, notification, and other effects at the boundary where they enter or leave the
calculation. Domain, Use Cases, and Adapters tasks must retain the corresponding tests and normative scenario
ID as self-evidence.

For `feature`, `bugfix`, and any item whose `affected_spec` references a requirement in a `standards.md`, add
`primary_use_cases` before implementation. Each entry declares one central successful route with a stable
kebab-case `id`, its exact `REQ-*` or standards requirement, the final `observable_result`, Unit and E2E test
references, and the distinct plausible fault each test must detect. A test reference contains the repository-
relative `path`, stable `name`, and required `mise` or CI `task`. The test may be absent while the item is
`in_progress`; at completion the checker requires the file and identifier, the requirement in the test source,
and reachability through the declared standard task. Do not use input acceptance, enum validation, line
coverage, or a directly constructed lower-level component as an E2E result when production uses a wider entry
and composition path.

`risk-based-v3` applies this contract to newly started work. Completed `risk-based-v1` and `risk-based-v2`
records remain valid history. An applicable item that was already `in_progress` at adoption moves to v3 and
adds the plan; no completed record is rewritten.

When the work is complete, set `status` to `completed`, append the following section, and move the file
to `work-items/done/`:

```markdown
## Completion
- **Completed At**: 2026-01-01
- **Summary**:
  The semantic difference introduced by the work, read from `mise run spec-diff` rather than recalled.
- **Acceptance RED Evidence**:
  - **Test**: The failing test or check observed before implementation.
  - **Requirement**: `REQ-CONTEXT-NNN`, or `N/A: <reason>` for work with no normative product requirement.
  - **Observed Failure**: The expected failure that was actually observed.
  - **Detection Reason**: Why the assertion distinguishes a plausible wrong implementation from the required
    behavior. When an acceptance boundary is inapplicable, identify the alternate check that actually failed.
- **Unit RED Evidence**:
  - **Test**: The failing test or check observed before implementation.
  - **Requirement**: `REQ-CONTEXT-NNN`, or `N/A: <reason>` for work with no normative product requirement.
  - **Observed Failure**: The expected failure that was actually observed.
  - **Detection Reason**: Why the assertion distinguishes a plausible wrong implementation from the required
    inner behavior. When a unit boundary is inapplicable, identify the alternate check that actually failed.
- **Change-Resistance Results**:
  For medium risk and above, record the representative incorrect implementation, diff mutation, or explicit
  fault injection and whether the tests detected it. For high and critical pure-logic changes, one
  representative is not enough: mutate the changed logic systematically or inject explicit faults across it,
  and record the equivalent mutations and the limits of the method rather than hiding them.
- **Verification Results**:
  - `mise run verify` - passed
```

The Acceptance and Unit RED fields above are the completion shape for work without a primary-use-case
requirement. Applicable `risk-based-v3` items use the following field instead; they may retain additional
Acceptance or Unit evidence for non-primary behavior, but it does not replace this evidence:

```markdown
- **Primary Use Case Evidence**:
  - id: start-task
    unit_red: TestStartTask_REQ_SYSTEM_001 failed because no start command was emitted.
    e2e_red: TestE2E_StartTask_REQ_SYSTEM_001 failed because the configured route produced no running task.
    unit_fault_injection: Removing command emission made TestStartTask_REQ_SYSTEM_001 fail.
    e2e_fault_injection: Disconnecting the route made TestE2E_StartTask_REQ_SYSTEM_001 fail.
```

The `id` must match a `primary_use_cases` plan entry. Every planned entry needs exactly one completion entry;
each RED and fault-injection result is a non-empty observation, not a future instruction.
