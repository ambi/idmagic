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

@doc("管理者が、リクエストのテナントの Job を作成の新しい順に、状態、種類、レーンで絞り込んで一覧する。他のテナントの Job は、ページにも、そこから導く件数にも含まれない。範囲は経路が決め、どの入力でも広がらない。制御面の資格情報を持つ呼び出し元であっても、ここではテナントを越えない。テナントを横断する一覧は ListSystemJobs だけである。呼び出し元には、リクエストのテナントでの `admin` ロールが必要である。`params`、`result`、`dedup_key` は返さない（REQ-JOBS-012、REQ-JOBS-014）。")
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

@doc("管理者が、リクエストのテナントの Job を一つ読む。他のテナントの Job は存在しないものとして扱うので、id だけでは呼び出し元が見られないテナントについて何も明かさず、ここでは制御面の資格情報でもそれは変わらない。他のテナントに届く読み取りは GetSystemJob だけである。呼び出し元には、リクエストのテナントでの `admin` ロールが必要である。`params`、`result`、`dedup_key` は返さない（REQ-JOBS-012、REQ-JOBS-014）。")
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

@doc("管理者が、終端の状態に達していない Job を取り消し、直ちに Canceled へ移す。Running の Job のハンドラーをその場で中断はしない。取り消しはリースを外すので、次のハートビートまたは完了の報告が失敗し、ハンドラーが自ら中止する。すでに確定した副作用は元に戻さず、その後 Job を再試行することもない。すでに終端の状態にある Job は、何もせず成功を返すのではなく、JobNotCancelableError で拒否する。Job はリクエストのテナントに属していなければならない。他のテナントの Job は、制御面の資格情報であっても、存在しないものとして扱い、変更しない。他のテナントに届く取り消しは CancelSystemJob だけである。呼び出し元には、リクエストのテナントでの `admin` ロールが必要である（REQ-JOBS-013）。")
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

@doc("制御面の運用者が、すべてのテナントの Job を作成の新しい順に、状態、種類、レーンで絞り込んで一覧する。範囲は入力ではなく経路が決めるすべてのテナントなので、ここでの一覧をリクエストのテナント自身の一覧と取り違えることはない。呼び出し元は、制御面テナントに所属し、有効ロールに `system_admin` を含み、制御面テナントの経路から呼び出す有効なユーザーでなければならない。そうした呼び出し元に自身の `admin` ロールは不要で、三つの条件のどれかを満たさない呼び出し元は AccessDeniedError で拒否する。ListJobs が発行したカーソルは、ここでは続きとして扱わず拒否する。`params`、`result`、`dedup_key` は返さない（REQ-JOBS-015、REQ-JOBS-014）。")
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

@doc("制御面の運用者が、任意のテナントの Job を一つ読む。呼び出し元は、制御面テナントに所属し、有効ロールに `system_admin` を含み、制御面テナントの経路から呼び出す有効なユーザーでなければならない。三つの条件のどれかを満たさない呼び出し元には応答せず、AccessDeniedError で拒否する。`params`、`result`、`dedup_key` は返さない（REQ-JOBS-015、REQ-JOBS-014）。")
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

@doc("制御面の運用者が、任意のテナントの、終端の状態に達していない Job を一つ取り消す。意味は CancelJob と同じで、Job は直ちに Canceled へ移り、Running のハンドラーはその場で中断されずにリースを失って次の報告で自ら中止し、確定した副作用は元に戻さず、Job は再試行しない。すでに終端の状態にある Job は JobNotCancelableError で拒否し、変更しない。呼び出し元は、制御面テナントに所属し、有効ロールに `system_admin` を含み、制御面テナントの経路から呼び出す有効なユーザーでなければならず、そうした呼び出し元に自身の `admin` ロールは不要である。状態を変えるブラウザーの経路なので、リクエストが製品自身の UI から来たことを CSRF と Origin で示す必要もあり、示せないリクエストはどの Job も変更しない（REQ-JOBS-015）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["jobs:cancel"])
@TypeSpec.OpenAPI.operationId("CancelSystemJob")
@route("/api/admin/v1/system/jobs/{job_id}/cancel")
@post
op CancelSystemJob(
  @path job_id: string
): CancelSystemJobSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | CancelSystemJobError403 | CancelSystemJobError404 | CancelSystemJobError409;
}

}
