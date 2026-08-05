---
context: idgovernance
updated_at: 2026-08-06
---

# Architecture: idgovernance

## Overview

The `IdGovernance` context owns `LifecycleWorkflow` policy and orchestration — trigger definitions,
action definitions, and run/step execution — while record contexts (`IdManagement`, `Application`)
keep owning the state those actions change. This context was split out of `IdManagement` once the
lifecycle-workflow slice had grown large enough (~20 backend files, ~166 SCL lines) to smear across
layers of a context whose primary job is being the identity-principal record of truth; the rejected
alternative of leaving it module-local inside `IdManagement` would not provide a home for the
broader IGA roadmap (access campaigns, entitlements, JIT elevation). Rationale in
[ADR-117](../../decisions/ADR-117-extract-identity-governance-context.md).

## Trigger capture via transactional outbox

`IdManagement` writes User lifecycle events (`UserCreated`, `UserAttributesChanged`,
`UserStatusChanged`) to the existing transactional outbox in the same transaction as the user
mutation itself. `IdGovernance` consumes these to create `WorkflowRun`/`WorkflowStep` rows. This
replaces the earlier same-context, same-transaction capture
([ADR-113](../../decisions/ADR-113-identity-lifecycle-workflow-execution-model.md) decision 2, from
when LifecycleWorkflow still lived inside `IdManagement`) with a cross-context contract that closes
the same failure window — a User update whose triggering run silently never gets created — without
requiring a single shared transaction across contexts. Because delivery is at-least-once, the
`(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` uniqueness constraint
from ADR-113 decision 4 still collapses duplicate deliveries of the same trigger occurrence into one
run.

## Action execution via published command surface

The nine executor actions (group membership add/remove, application assign/unassign, user
enable/disable, required-action set/clear, send email) never call record-context domain types
directly. `IdManagement` and `Application` instead expose idempotent command surfaces as published
interfaces that the executor calls across the context boundary; composition-root wiring injects the
concrete adapters (the same ports-and-adapters pattern ADR-113 established before the context
split). Promoting these to published interfaces — rather than leaving them as ADR-113's internal
interfaces — was possible once `IdGovernance` became the dependency source: `IdGovernance` depending
on `IdManagement`/`Application` cannot cycle back, unlike the earlier `IdManagement -> Application`
direction that ADR-113 had to route around.

## Partial failure and loop suppression

A single attempt runs every not-yet-completed step in definition order and does not stop at a
failed step, so an unrelated notification failure cannot block an access-revocation step; the run
terminates `succeeded`, `partially_failed`, or `failed` depending on the mix, with no cross-context
compensation/rollback. `WorkflowRunTriggerSnapshot` records the origin run/step of any User mutation
an action performs, and trigger evaluation excludes mutations carrying that origin metadata — this
closes the action-to-trigger loop structurally, at the evaluator's input, rather than with a runtime
guard. `WorkflowRun`/`WorkflowStep`/`LifecycleNotificationDelivery` follow the same 30-day retention
as `Job` records. Details in
[ADR-113](../../decisions/ADR-113-identity-lifecycle-workflow-execution-model.md) decisions 3-7
(revision pinning, dedup, partial failure, loop suppression, retention — all carried forward
unchanged by ADR-117).
