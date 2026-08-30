import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Identity Governance")
namespace IdMagic.IdGovernance {

namespace Operations {
model ListLifecycleWorkflowsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.LifecycleWorkflowListResponse;
}

union ListLifecycleWorkflowsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListLifecycleWorkflowsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListLifecycleWorkflowsError403Body;
}

@doc("An administrator lists the LifecycleWorkflows of their tenant that have not been deleted. Deleted definitions are not returned.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("ListLifecycleWorkflows")
@route("/api/admin/v1/lifecycle_workflows")
@get
op ListLifecycleWorkflows(

): ListLifecycleWorkflowsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListLifecycleWorkflowsError403;

model CreateLifecycleWorkflowSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.LifecycleWorkflowResponse;
}

union CreateLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateLifecycleWorkflowError400Body;
}

union CreateLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateLifecycleWorkflowError403Body;
}

@doc("An administrator creates a LifecycleWorkflow. The workflow itself and the draft of revision 1 are both persisted, and `name` is unique within the tenant among workflows that have not been deleted. The name of a deleted workflow may be used again.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("CreateLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows")
@post
op CreateLifecycleWorkflow(
  @body body: IdMagic.Contract.LifecycleWorkflowCreateRequest
): CreateLifecycleWorkflowSuccess_201 | CreateLifecycleWorkflowError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateLifecycleWorkflowError403;

model GetLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.LifecycleWorkflowResponse;
}

union GetLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetLifecycleWorkflowError400Body;
}

union GetLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetLifecycleWorkflowError403Body;
}

@doc("An administrator reads one LifecycleWorkflow. A workflow of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("GetLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}")
@get
op GetLifecycleWorkflow(
  @path workflow_id: string
): GetLifecycleWorkflowSuccess_200 | GetLifecycleWorkflowError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetLifecycleWorkflowError403;

model UpdateLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.LifecycleWorkflowResponse;
}

union UpdateLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateLifecycleWorkflowError400Body;
}

union UpdateLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateLifecycleWorkflowError403Body;
}

union UpdateLifecycleWorkflowError409Body {
  IdMagic.Contract.WorkflowRevisionConflictError,
}

model UpdateLifecycleWorkflowError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: UpdateLifecycleWorkflowError409Body;
}

@doc("An administrator updates the trigger, the actions, or the display fields of a LifecycleWorkflow. When `expected_revision`\ndoes not match `current_revision` the update is rejected with WorkflowRevisionConflictError, so a lost update cannot happen.\nA change to the trigger or the actions raises `current_revision` by one but leaves `status` and `enabled_revision` alone,\nso the old revision keeps being evaluated until EnableLifecycleWorkflow is called again. An archived workflow is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("UpdateLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}")
@put
op UpdateLifecycleWorkflow(
  @path workflow_id: string,
  @body body: IdMagic.Contract.LifecycleWorkflowUpdateRequest
): UpdateLifecycleWorkflowSuccess_200 | UpdateLifecycleWorkflowError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateLifecycleWorkflowError403 | UpdateLifecycleWorkflowError409;

model EnableLifecycleWorkflowHttpRequest {
  expected_revision: int64;
}

model EnableLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.LifecycleWorkflowResponse;
}

union EnableLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableLifecycleWorkflowError400Body;
}

union EnableLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableLifecycleWorkflowError403Body;
}

@doc("An administrator validates `current_revision` and, if it passes, sets it as `enabled_revision` and moves `status` to `enabled`.\nFrom then on it is that revision the trigger evaluates. A failed validation, and an archived workflow, are both refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("EnableLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/enable")
@post
op EnableLifecycleWorkflow(
  @path workflow_id: string,
  @body body: EnableLifecycleWorkflowHttpRequest
): EnableLifecycleWorkflowSuccess_200 | EnableLifecycleWorkflowError400 | EnableLifecycleWorkflowError403;

model DisableLifecycleWorkflowHttpRequest {
  expected_revision: int64;
}

model DisableLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.LifecycleWorkflowResponse;
}

union DisableLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableLifecycleWorkflowError400Body;
}

union DisableLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableLifecycleWorkflowError403Body;
}

@doc("An administrator disables a LifecycleWorkflow. No new run is triggered, and a queued run that has not started is canceled.\nA running run stops after the checkpoint of its current step and before the next one.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("DisableLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/disable")
@post
op DisableLifecycleWorkflow(
  @path workflow_id: string,
  @body body: DisableLifecycleWorkflowHttpRequest
): DisableLifecycleWorkflowSuccess_200 | DisableLifecycleWorkflowError400 | DisableLifecycleWorkflowError403;

model DeleteLifecycleWorkflowSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteLifecycleWorkflowError400Body;
}

union DeleteLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteLifecycleWorkflowError403Body;
}

@doc("An administrator deletes a LifecycleWorkflow whatever its state. Afterwards it is excluded from listing, reading, updating, and the name uniqueness check. If it was enabled, no new run is triggered and queued runs are canceled. Existing runs and audit events are kept.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("DeleteLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}")
@delete
op DeleteLifecycleWorkflow(
  @path workflow_id: string,
  @query expected_revision: int64
): DeleteLifecycleWorkflowSuccess_204 | DeleteLifecycleWorkflowError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteLifecycleWorkflowError403;

model DryRunLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.LifecycleWorkflowDryRunResponse;
}

union DryRunLifecycleWorkflowError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DryRunLifecycleWorkflowError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DryRunLifecycleWorkflowError400Body;
}

union DryRunLifecycleWorkflowError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DryRunLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DryRunLifecycleWorkflowError403Body;
}

@doc("An administrator dry-runs `enabled_revision`, or `current_revision` when none is named, against a target user. It uses the same\nvalidator, trigger evaluator, and action planner as a real run, and reads the target User's current group memberships,\napplication assignments, required actions, status, and email verification state to decide per action whether it would_change,\nis a no_op, or is blocked. When the trigger filter does not match the target User, every action comes back blocked with the\nreason `trigger_not_matched`. A target user that does not exist, or belongs to another tenant, gives InvalidRequestError.\nNothing is created or changed: no WorkflowRun, no Job, no membership, assignment, required action, status, or email.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("DryRunLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/dry_run")
@post
op DryRunLifecycleWorkflow(
  @path workflow_id: string,
  @body body: IdMagic.Contract.LifecycleWorkflowDryRunRequest
): DryRunLifecycleWorkflowSuccess_200 | DryRunLifecycleWorkflowError400 | IdMagic.Contract.AuthenticationRequiredResponse | DryRunLifecycleWorkflowError403;

model ListLifecycleWorkflowRunsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkflowRunListResponse;
}

union ListLifecycleWorkflowRunsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListLifecycleWorkflowRunsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListLifecycleWorkflowRunsError400Body;
}

union ListLifecycleWorkflowRunsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListLifecycleWorkflowRunsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListLifecycleWorkflowRunsError403Body;
}

@doc("An administrator lists the run history of one LifecycleWorkflow, newest first.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("ListLifecycleWorkflowRuns")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/runs")
@get
op ListLifecycleWorkflowRuns(
  @path workflow_id: string
): ListLifecycleWorkflowRunsSuccess_200 | ListLifecycleWorkflowRunsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListLifecycleWorkflowRunsError403;

model GetLifecycleWorkflowRunSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkflowRunResponse;
}

union GetLifecycleWorkflowRunError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetLifecycleWorkflowRunError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetLifecycleWorkflowRunError400Body;
}

union GetLifecycleWorkflowRunError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetLifecycleWorkflowRunError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetLifecycleWorkflowRunError403Body;
}

@doc("An administrator reads one WorkflowRun in detail: its trigger, revision, target user, Job attempts, and each step. A run of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("GetLifecycleWorkflowRun")
@route("/api/admin/v1/lifecycle_workflow_runs/{run_id}")
@get
op GetLifecycleWorkflowRun(
  @path run_id: string
): GetLifecycleWorkflowRunSuccess_200 | GetLifecycleWorkflowRunError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetLifecycleWorkflowRunError403;

model RetryLifecycleWorkflowRunSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkflowRunResponse;
}

union RetryLifecycleWorkflowRunError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RetryLifecycleWorkflowRunError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RetryLifecycleWorkflowRunError400Body;
}

union RetryLifecycleWorkflowRunError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RetryLifecycleWorkflowRunError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RetryLifecycleWorkflowRunError403Body;
}

@doc("An administrator retries a WorkflowRun that is `partially_failed` or `failed`. A new Job is attached to the failed steps only;\nsteps already `changed` or `no_op` are left alone. When it races a disable, no new step is started.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("RetryLifecycleWorkflowRun")
@route("/api/admin/v1/lifecycle_workflow_runs/{run_id}/retry")
@post
op RetryLifecycleWorkflowRun(
  @path run_id: string,): RetryLifecycleWorkflowRunSuccess_200 | RetryLifecycleWorkflowRunError400 | IdMagic.Contract.AuthenticationRequiredResponse | RetryLifecycleWorkflowRunError403;
}

}
