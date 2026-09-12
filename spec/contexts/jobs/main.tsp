import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Jobs")
namespace IdMagic.Jobs {

namespace Operations {
model ListJobsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobListResponse;
}

union ListJobsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListJobsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListJobsError400Body;
}

union ListJobsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListJobsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListJobsError403Body;
}

@doc("An administrator lists the Jobs of the request tenant, newest-created first, narrowed by status, kind, and lane. Another tenant's Jobs are absent from the page and from any count derived from it. The scope is fixed by the route: no input widens it, and a caller holding control-plane credentials crosses no tenant here either -- ListSystemJobs is the only listing that spans tenants. The caller needs the `admin` role in the request tenant. `params`, `result`, and `dedup_key` are not returned (REQ-JOBS-012, REQ-JOBS-014).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:read"])
@TypeSpec.OpenAPI.operationId("ListJobs")
@route("/api/admin/v1/jobs")
@get
op ListJobs(
  @query(#{ explode: true }) status?: IdMagic.Contract.JobStatus[],
  @query(#{ explode: true }) kind?: IdMagic.Contract.JobKind[],
  @query lane?: IdMagic.Contract.ExecutionLane,
  @query limit?: int64,
  @query cursor?: string
): ListJobsSuccess_200 | ListJobsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListJobsError403;

model GetJobSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobResponse;
}

union GetJobError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetJobError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetJobError403Body;
}

union GetJobError404Body {
  IdMagic.Contract.JobNotFoundError,
}

model GetJobError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetJobError404Body;
}

@doc("An administrator reads one Job of the request tenant. A Job of another tenant is treated as nonexistent, so an id alone reveals nothing about a tenant the caller cannot see, and control-plane credentials do not change that here -- GetSystemJob is the only read that reaches another tenant. The caller needs the `admin` role in the request tenant. `params`, `result`, and `dedup_key` are not returned (REQ-JOBS-012, REQ-JOBS-014).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:read"])
@TypeSpec.OpenAPI.operationId("GetJob")
@route("/api/admin/v1/jobs/{job_id}")
@get
op GetJob(
  @path job_id: string
): GetJobSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetJobError403 | GetJobError404;

model CancelJobSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobResponse;
}

union CancelJobError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CancelJobError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelJobError403Body;
}

union CancelJobError404Body {
  IdMagic.Contract.JobNotFoundError,
}

model CancelJobError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: CancelJobError404Body;
}

union CancelJobError409Body {
  IdMagic.Contract.JobNotCancelableError,
}

model CancelJobError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CancelJobError409Body;
}

@doc("An administrator cancels a Job that has not reached a terminal state, moving it straight to Canceled. A Running Job's handler is not interrupted where it stands: cancelling drops its lease, so its next heartbeat or completion report fails and the handler aborts itself. Side effects already committed are not undone, and the Job is never retried afterwards. A Job already in a terminal state is refused with JobNotCancelableError rather than answered as a no-op success. The Job must belong to the request tenant; another tenant's Job is treated as nonexistent and is left untouched, control-plane credentials included -- CancelSystemJob is the only cancellation that reaches another tenant. The caller needs the `admin` role in the request tenant (REQ-JOBS-013).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:cancel"])
@TypeSpec.OpenAPI.operationId("CancelJob")
@route("/api/admin/v1/jobs/{job_id}/cancel")
@post
op CancelJob(
  @path job_id: string
): CancelJobSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | CancelJobError403 | CancelJobError404 | CancelJobError409;

model ListSystemJobsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobListResponse;
}

union ListSystemJobsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListSystemJobsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListSystemJobsError400Body;
}

union ListSystemJobsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSystemJobsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSystemJobsError403Body;
}

@doc("A control-plane operator lists every tenant's Jobs, newest-created first, narrowed by status, kind, and lane. The scope is every tenant, fixed by the route rather than by an input, so a listing here is never confused with the request tenant's own. The caller must be an active user who belongs to the control-plane tenant, carries `system_admin` among its effective roles, and calls through the control-plane tenant; such a caller needs no `admin` role of its own, and anyone who fails any of the three is refused with AccessDeniedError. A cursor issued by ListJobs is rejected rather than continued here. `params`, `result`, and `dedup_key` are not returned (REQ-JOBS-015, REQ-JOBS-014).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:read"])
@TypeSpec.OpenAPI.operationId("ListSystemJobs")
@route("/api/admin/v1/system/jobs")
@get
op ListSystemJobs(
  @query(#{ explode: true }) status?: IdMagic.Contract.JobStatus[],
  @query(#{ explode: true }) kind?: IdMagic.Contract.JobKind[],
  @query lane?: IdMagic.Contract.ExecutionLane,
  @query limit?: int64,
  @query cursor?: string
): ListSystemJobsSuccess_200 | ListSystemJobsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListSystemJobsError403;

model GetSystemJobSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobResponse;
}

union GetSystemJobError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetSystemJobError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetSystemJobError403Body;
}

union GetSystemJobError404Body {
  IdMagic.Contract.JobNotFoundError,
}

model GetSystemJobError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetSystemJobError404Body;
}

@doc("A control-plane operator reads one Job of any tenant. The caller must be an active user who belongs to the control-plane tenant, carries `system_admin` among its effective roles, and calls through the control-plane tenant; anyone who fails any of the three is refused with AccessDeniedError rather than answered. `params`, `result`, and `dedup_key` are not returned (REQ-JOBS-015, REQ-JOBS-014).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:read"])
@TypeSpec.OpenAPI.operationId("GetSystemJob")
@route("/api/admin/v1/system/jobs/{job_id}")
@get
op GetSystemJob(
  @path job_id: string
): GetSystemJobSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetSystemJobError403 | GetSystemJobError404;

model CancelSystemJobSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobResponse;
}

union CancelSystemJobError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CancelSystemJobError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CancelSystemJobError403Body;
}

union CancelSystemJobError404Body {
  IdMagic.Contract.JobNotFoundError,
}

model CancelSystemJobError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: CancelSystemJobError404Body;
}

union CancelSystemJobError409Body {
  IdMagic.Contract.JobNotCancelableError,
}

model CancelSystemJobError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CancelSystemJobError409Body;
}

@doc("A control-plane operator cancels one Job of any tenant that has not reached a terminal state, with the same semantics CancelJob describes: the Job moves straight to Canceled, a Running handler is not interrupted where it stands but loses its lease and aborts itself at its next report, committed side effects are not undone, and the Job is never retried. A Job already in a terminal state is refused with JobNotCancelableError and left unchanged. The caller must be an active user who belongs to the control-plane tenant, carries `system_admin` among its effective roles, and calls through the control-plane tenant; such a caller needs no `admin` role of its own. Being a browser route that changes state, it also proves the request came from the product's own UI through CSRF and Origin, and a request that cannot leaves every Job untouched (REQ-JOBS-015).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:cancel"])
@TypeSpec.OpenAPI.operationId("CancelSystemJob")
@route("/api/admin/v1/system/jobs/{job_id}/cancel")
@post
op CancelSystemJob(
  @path job_id: string
): CancelSystemJobSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | CancelSystemJobError403 | CancelSystemJobError404 | CancelSystemJobError409;
}

}
