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
  IdMagic.Contract.AccessDeniedError,
}

model ListJobsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListJobsError403Body;
}

@doc("An administrator lists the Jobs of their tenant, newest-created first, narrowed by status, kind, and lane. Another tenant's Jobs are absent from the page and from any count derived from it. `all_tenants` widens the listing to every tenant, and is honoured only for a `system_admin` calling through the control-plane tenant. `params`, `result`, and `dedup_key` are not returned (REQ-JOBS-012, REQ-JOBS-014).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:read"])
@TypeSpec.OpenAPI.operationId("ListJobs")
@route("/api/admin/v1/jobs")
@get
op ListJobs(
  @query(#{ explode: true }) status?: IdMagic.Contract.JobStatus[],
  @query(#{ explode: true }) kind?: IdMagic.Contract.JobKind[],
  @query lane?: IdMagic.Contract.ExecutionLane,
  @query limit?: int64,
  @query cursor?: string,
  @query all_tenants?: boolean
): ListJobsSuccess_200 | ListJobsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListJobsError403;

model GetJobSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminJobResponse;
}

union GetJobError403Body {
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

@doc("An administrator reads one Job. A Job of another tenant is treated as nonexistent, so an id alone reveals nothing about a tenant the caller cannot see. `params`, `result`, and `dedup_key` are not returned (REQ-JOBS-014).")
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

@doc("An administrator cancels a Job that has not reached a terminal state, moving it straight to Canceled. A Running Job's handler is not interrupted where it stands: cancelling drops its lease, so its next heartbeat or completion report fails and the handler aborts itself. Side effects already committed are not undone, and the Job is never retried afterwards. A Job already in a terminal state is refused with JobNotCancelableError rather than answered as a no-op success (REQ-JOBS-013).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:cancel"])
@TypeSpec.OpenAPI.operationId("CancelJob")
@route("/api/admin/v1/jobs/{job_id}/cancel")
@post
op CancelJob(
  @path job_id: string
): CancelJobSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | CancelJobError403 | CancelJobError404 | CancelJobError409;
}

}
