namespace IdMagic.Contract {

@doc("The operational state of a User. Published-language stub for IdManagement.UserStatus, used only to evaluate the `from_status` and `to_status` of a trigger. IdManagement owns the single source of truth for the values.")
enum UserStatus {
  Active: "active",
  Disabled: "disabled",
  PendingDeletion: "pending_deletion",
  Deleted: "deleted",
  Locked: "locked",
  Staged: "staged",
  Suspended: "suspended",
}


@doc("The action forced on a user at their next sign-in. Published-language stub for IdManagement.RequiredAction, used only to name the action of a `set_required_action` or `clear_required_action`.")
enum RequiredAction {
  UpdatePassword: "update_password",
  VerifyEmail: "verify_email",
  ConfigureTotp: "configure_totp",
  UpdateProfile: "update_profile",
  TermsAndConditions: "terms_and_conditions",
}


@doc("The visibility of an assignment. Published-language stub for a definition the Application context owns, used only for the `visibility` of an `assign_application` action.")
enum AssignmentVisibility {
  visible: "visible",
  hidden: "hidden",
}


@doc("The lifecycle of a LifecycleWorkflow. `archived` is an implementation state that keeps a deleted definition internally so the run history stays referentially intact; it is exposed neither in the administration console nor in the ordinary API.")
enum LifecycleWorkflowStatus {
  draft: "draft",
  enabled: "enabled",
  disabled: "disabled",
  archived: "archived",
}


@doc("The kinds of condition a WorkflowTrigger fires on. The first version has only the three that start from a User mutation; date, cron, and wait steps are out of scope.")
enum WorkflowTriggerKind {
  user_created: "user_created",
  user_attributes_changed: "user_attributes_changed",
  user_status_changed: "user_status_changed",
}


@doc("The comparison operators a WorkflowFilter may use. `eq` and `not_eq` are exact match and mismatch, `in` matches any member of a value set, and `exists` tests whether a value is present at all.")
enum WorkflowFilterOperator {
  eq: "eq",
  not_eq: "not_eq",
  in: "in",
  exists: "exists",
}


@doc("The kinds of WorkflowAction. Every one of them states a desired state, so each is idempotent and becomes a no_op when the state already holds.")
enum WorkflowActionKind {
  add_group_member: "add_group_member",
  remove_group_member: "remove_group_member",
  assign_application: "assign_application",
  unassign_application: "unassign_application",
  set_required_action: "set_required_action",
  clear_required_action: "clear_required_action",
  enable_user: "enable_user",
  disable_user: "disable_user",
  send_email: "send_email",
}


@doc("The states of WorkflowRunLifecycle. A no-op counts as a success.")
enum WorkflowRunStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  partially_failed: "partially_failed",
  failed: "failed",
  canceled: "canceled",
}


@doc("The outcome of one WorkflowStep. `pending` is the initial value, before anything has been attempted.")
enum WorkflowStepOutcome {
  pending: "pending",
  changed: "changed",
  no_op: "no_op",
  failed: "failed",
  canceled: "canceled",
}


@doc("UpdateLifecycleWorkflow found that `expected_revision` does not match the current `LifecycleWorkflow.current_revision`, and rejected the lost update. The caller re-reads the latest revision and retries. RFC 9457 Problem Details with `type` = `urn:idmagic:error:workflow_revision_conflict`.")
model WorkflowRevisionConflictError is ProblemDetails;


@doc("One term of `WorkflowTrigger.filters`. Only a field that exists in the TenantUserAttributeSchema and whose type matches can be\nstored; a free-text regex over PII and an unknown field are both refused at save time. Several filters are ANDed.")
model WorkflowFilter {
  @doc("A `TenantUserAttributeSchema.key`, or a built-in attribute key of User.")
  @minLength(1)
  field: string;
  operator: WorkflowFilterOperator;
  @doc("Any JSON value, because the type follows the AttributeType of `field`: a string, a number, a boolean, a date string, or an array of strings. The `in` operator takes an array of that type, and `exists` takes no value at all. One TypeSpec type cannot express that dependence, so the contract records the value as unconstrained and the server checks it against the tenant attribute schema.")
  value?: unknown;
}


@doc("The trigger definition a LifecycleWorkflowRevision fixes. Depending on the kind it uses `watched_attributes`\nor `from_status` and `to_status`. `filters` holds 0 to 20 terms, ANDed.")
model WorkflowTriggerDef {
  kind: WorkflowTriggerKind;
  @doc("Required when `kind` is `user_attributes_changed`. It fires only when one of the listed fields actually changed; a field\nmerely present in the request with an unchanged value does not fire it.")
  watched_attributes?: string[];
  @doc("Required when `kind` is `user_status_changed`: the `from` of a permitted UserLifecycle transition.")
  from_status?: UserStatus;
  @doc("Required when `kind` is `user_status_changed`: the `to` of a permitted UserLifecycle transition.")
  to_status?: UserStatus;
  filters: WorkflowFilter[];
}


@doc("One action definition a LifecycleWorkflowRevision fixes. Which fields it uses depends on the kind, and the existence of\nwhat it refers to is checked both on create/update and on enable.")
model WorkflowActionDef {
  kind: WorkflowActionKind;
  @doc("Required when `kind` is `add_group_member` or `remove_group_member`. Naming a dynamic group is a validation error at save time.")
  group_id?: string;
  @doc("Required when `kind` is `assign_application` or `unassign_application`.")
  application_id?: string;
  @doc("Optional when `kind` is `assign_application`; `visible` when left out. Published-language stub for a definition the Application context owns.")
  visibility?: AssignmentVisibility;
  @doc("Required when `kind` is `set_required_action` or `clear_required_action`.")
  required_action?: RequiredAction;
  @doc("An optional reason recorded when `kind` is `enable_user` or `disable_user`.")
  reason?: string;
  @doc("Required when `kind` is `send_email`. A label the workflow author gives the notification. The wording is resolved from LifecycleWorkflowNotification in the Tenancy notification template catalogue, and this value is passed into the body as a substitution variable. Assigning a different template per workflow is outside this context.")
  template_key?: string;
}


@doc("The immutable snapshot of a LifecycleWorkflow definition. Every update that changes its meaning raises the revision by one,\nand past revisions stay immutable. Only the enabled revision creates new WorkflowRuns.")
model LifecycleWorkflowRevision {
  workflow_id: string;
  tenant_id: string;
  revision: int64;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
  created_at: utcDateTime;
}


@doc("The workflow definition aggregate, whose `name` is unique within the tenant. Creating one makes revision 1 as a draft.\nOnly an enabled revision is evaluated by the trigger, and an update that changes the meaning raises `current_revision`\nwhile leaving `status` and `enabled_revision` alone. An administrator may delete it whatever its state, and a deleted\ndefinition is excluded from listing, reading, and the name uniqueness check. Existing runs and audit events are kept\nindependently of the definition.")
model LifecycleWorkflow {
  @doc("The generated identifier (UUID v4). Immutable.")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("The tenant it belongs to. Cross-tenant references are forbidden.")
  tenant_id: string = "default";
  @doc("The display name, unique within the tenant.")
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  status: LifecycleWorkflowStatus = LifecycleWorkflowStatus.draft;
  @doc("The number of the most recently created revision.")
  current_revision: int64;
  @doc("The revision the trigger currently evaluates. It may be unset while the workflow is draft, disabled, or archived.")
  enabled_revision?: int64;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("The summary of a trigger firing, fixed when the WorkflowRun is created. The history keeps only the names of the changed\nfields and whether each predicate held; the attribute values themselves are not kept, to protect PII.")
model WorkflowRunTriggerSnapshot {
  kind: WorkflowTriggerKind;
  @doc("The id of the User mutation domain event that fired it. Redelivery of the same occurrence converges on the same run, so it is deduplicated.")
  source_occurrence_id: string;
  @doc("When `kind` is `user_attributes_changed`, the names of the fields that actually changed. The values are not included.")
  changed_fields: string[];
  from_status?: UserStatus;
  to_status?: UserStatus;
}


@doc("The unit of work produced by one firing of a WorkflowTrigger. The workflow_id, the revision, the trigger snapshot, the\ntarget_user_id, and the expanded action list are all fixed at creation, so changing the definition afterwards does not\nchange what an existing run means. The dispatcher attaches `job_id` after enqueueing. Immediately before an action runs,\nthe target user and resource are re-checked inside the tenant, and runs against the same target_user_id are serialized in\n`triggered_at` order.\n`(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` is the uniqueness constraint.")
model WorkflowRun {
  @doc("The generated identifier (UUID v4).")
  id: string;
  tenant_id: string;
  workflow_id: string;
  revision: int64;
  trigger: WorkflowRunTriggerSnapshot;
  target_user_id: string;
  @doc("The action list, expanded from the revision and fixed here. A later change to the definition does not change what this run means.")
  actions: WorkflowActionDef[];
  status: WorkflowRunStatus;
  @doc("The `Jobs.JobRef.id` the dispatcher attaches after enqueueing. Unset until then.")
  job_id?: string;
  triggered_at: utcDateTime;
  started_at?: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("The execution record of one action inside a WorkflowRun. It holds the index in definition order and checkpoints the\noutcome and a sanitized error code. It keeps no attribute value, email body, or secret.")
model WorkflowStep {
  run_id: string;
  tenant_id: string;
  @doc("The zero-based position within `WorkflowRun.actions`.")
  step_index: int64;
  action_kind: WorkflowActionKind;
  outcome: WorkflowStepOutcome = WorkflowStepOutcome.pending;
  @doc("A sanitized error code. It carries no attribute value and no email body.")
  error_code?: string;
  attempted_at?: utcDateTime;
}


@doc("The delivery idempotency key of a `send_email` action. It records that exactly one message was sent for a given `(run_id, step_index)`.")
model LifecycleNotificationDelivery {
  run_id: string;
  step_index: int64;
  tenant_id: string;
  template_key: string;
  delivered_at: utcDateTime;
}


@doc("The input naming the user a dry run targets. The revision used is `enabled_revision`, or `current_revision` when none is named.")
model LifecycleWorkflowDryRunRequest {
  target_user_id: string;
}


@doc("The verdict for one action of a dry run. It reads the target User's current state -- group memberships, application\nassignments, required actions, status, and email verification state -- and returns the same verdict the real action\nexecutor would reach (would_change, no_op, or blocked), without any side effect. It carries no attribute value and no\nemail address.")
model LifecycleWorkflowDryRunStepResult {
  action_kind: WorkflowActionKind;
  @doc("One of would_change, no_op, or blocked.")
  would_change: string;
  @doc("Why it is blocked: resource_not_found, dependency_unavailable, notification_unavailable, trigger_not_matched, and the\nlike. Left out for would_change and no_op.")
  reason?: string;
}


@doc("The result of a dry run. Nothing is created or changed: no WorkflowRun, Job, membership, assignment, required action,\nstatus, or email. It carries `evaluated_at` and the revision, and is not presented as a guarantee about the real run.")
model LifecycleWorkflowDryRunResponse {
  workflow_id: string;
  revision: int64;
  target_user_id: string;
  evaluated_at: utcDateTime;
  steps: LifecycleWorkflowDryRunStepResult[];
}


@doc("The request to create a workflow. Creating one makes revision 1 as a draft.")
model LifecycleWorkflowCreateRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
}


@doc("The request to update a workflow. When `expected_revision` does not match the current `current_revision` it is refused\nwith WorkflowRevisionConflictError, so a lost update cannot happen.")
model LifecycleWorkflowUpdateRequest {
  expected_revision: int64;
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  trigger?: WorkflowTriggerDef;
  actions?: WorkflowActionDef[];
}


@doc("How a workflow definition is represented, including the contents of `current_revision`.")
model LifecycleWorkflowResponse {
  id: string;
  name: string;
  description?: string;
  status: LifecycleWorkflowStatus;
  current_revision: int64;
  enabled_revision?: int64;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("The workflow list response.")
model LifecycleWorkflowListResponse {
  workflows: LifecycleWorkflowResponse[];
}


@doc("How a step is represented inside a run detail. It carries no secret, email body, or attribute value.")
model WorkflowStepResponse {
  step_index: int64;
  action_kind: WorkflowActionKind;
  outcome: WorkflowStepOutcome;
  error_code?: string;
  attempted_at?: utcDateTime;
}


@doc("The run detail response. It shows the trigger, the fixed revision, the target user, the Job attempts, and the outcome,\nerror code, and timestamps of each step. It carries no secret, email body, or attribute value.")
model WorkflowRunResponse {
  id: string;
  workflow_id: string;
  revision: int64;
  trigger: WorkflowRunTriggerSnapshot;
  target_user_id: string;
  status: WorkflowRunStatus;
  job_id?: string;
  steps: WorkflowStepResponse[];
  triggered_at: utcDateTime;
  started_at?: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("The run list response for a workflow.")
model WorkflowRunListResponse {
  runs: WorkflowRunResponse[];
  @doc("The cursor that continues pagination.")
  next_after?: utcDateTime;
}


@doc("An administrator created a LifecycleWorkflow, as revision 1 in draft.")
model LifecycleWorkflowCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("An administrator updated a LifecycleWorkflow. When the trigger or the actions changed and the revision rose, `newRevision` is included.")
model LifecycleWorkflowUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
  newRevision?: int64;
}


@doc("An administrator got `current_revision` through validation and enabled it as `enabled_revision`.")
model LifecycleWorkflowEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
  revision: int64;
}


@doc("An administrator disabled a LifecycleWorkflow. No new run is triggered, and a queued run that has not started is canceled.")
model LifecycleWorkflowDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("An administrator deleted a LifecycleWorkflow. It is emitted whatever the state was beforehand, and kept as an audit trail.")
model LifecycleWorkflowDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("The dispatcher handed a WorkflowRun to a worker and the first step began.")
model LifecycleWorkflowRunStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("Every step of a WorkflowRun ended as changed or no_op.")
model LifecycleWorkflowRunSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("A WorkflowRun reached its attempt ceiling and ended with successes and failures mixed.")
model LifecycleWorkflowRunPartiallyFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("A WorkflowRun reached its attempt ceiling and ended without a single successful step.")
model LifecycleWorkflowRunFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("A disable stopped a queued run that had not started, or stopped a running one after the checkpoint of its current step.")
model LifecycleWorkflowRunCanceled {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("One WorkflowStep settled with a failed outcome. The workflow_id, run_id, step_id, and error_code are kept so an audit search can find it.")
model LifecycleWorkflowStepFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  stepIndex: int64;
  actionKind: WorkflowActionKind;
  errorCode: string;
}

}
