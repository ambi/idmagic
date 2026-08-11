import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

namespace IdMagic.IdGovernance {

namespace Operations {
model ListLifecycleWorkflowsHttpResponse {
  response: IdMagic.Contract.LifecycleWorkflowListResponse;
}

model ListLifecycleWorkflowsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListLifecycleWorkflowsHttpResponse;
}

union ListLifecycleWorkflowsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListLifecycleWorkflowsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListLifecycleWorkflowsError403Body;
}

@doc("管理者が所属テナントの未削除 LifecycleWorkflow を一覧する。削除済み定義は返さない。")
@TypeSpec.OpenAPI.operationId("ListLifecycleWorkflows")
@route("/api/admin/v1/lifecycle_workflows")
@get
op ListLifecycleWorkflows(

): ListLifecycleWorkflowsSuccess_200 | ListLifecycleWorkflowsError403;

model CreateLifecycleWorkflowHttpRequest {
  request: IdMagic.Contract.LifecycleWorkflowCreateRequest;
}

model CreateLifecycleWorkflowHttpResponse {
  workflow: IdMagic.Contract.LifecycleWorkflowResponse;
}

model CreateLifecycleWorkflowSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateLifecycleWorkflowHttpResponse;
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

@doc("管理者が LifecycleWorkflow を作成する。workflow 本体と revision 1 の draft をともに永続化し、name は未削除 workflow の間でテナント内一意とする。削除済み workflow の name は再利用できる。")
@TypeSpec.OpenAPI.operationId("CreateLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows")
@post
op CreateLifecycleWorkflow(
  @body body: CreateLifecycleWorkflowHttpRequest
): CreateLifecycleWorkflowSuccess_201 | CreateLifecycleWorkflowError400 | CreateLifecycleWorkflowError403;

model GetLifecycleWorkflowHttpResponse {
  workflow: IdMagic.Contract.LifecycleWorkflowResponse;
}

model GetLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetLifecycleWorkflowHttpResponse;
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

@doc("管理者が LifecycleWorkflow 1 件を取得する。別テナントの workflow は未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}")
@get
op GetLifecycleWorkflow(
  @path workflow_id: string
): GetLifecycleWorkflowSuccess_200 | GetLifecycleWorkflowError400 | GetLifecycleWorkflowError403;

model UpdateLifecycleWorkflowHttpRequest {
  workflow_id: string;
  request: IdMagic.Contract.LifecycleWorkflowUpdateRequest;
}

model UpdateLifecycleWorkflowHttpResponse {
  workflow: IdMagic.Contract.LifecycleWorkflowResponse;
}

model UpdateLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateLifecycleWorkflowHttpResponse;
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

@doc("管理者が LifecycleWorkflow の trigger/actions/表示情報を更新する。expected_revision が\ncurrent_revision と一致しない場合は WorkflowRevisionConflictError で拒否する (lost update 防止)。\ntrigger または actions が変わる場合は current_revision を 1 増やすが status/enabled_revision は\n変えない (再度 EnableLifecycleWorkflow を呼ぶまで旧 revision の trigger 評価が続く)。archived\nworkflow は拒否する。")
@TypeSpec.OpenAPI.operationId("UpdateLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}")
@put
op UpdateLifecycleWorkflow(
  @path workflow_id: string,
  @body body: UpdateLifecycleWorkflowHttpRequest
): UpdateLifecycleWorkflowSuccess_200 | UpdateLifecycleWorkflowError400 | UpdateLifecycleWorkflowError403 | UpdateLifecycleWorkflowError409;

model EnableLifecycleWorkflowHttpRequest {
  workflow_id: string;
  expected_revision: int64;
}

model EnableLifecycleWorkflowHttpResponse {
  workflow: IdMagic.Contract.LifecycleWorkflowResponse;
}

model EnableLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: EnableLifecycleWorkflowHttpResponse;
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

@doc("管理者が current_revision を validation し、成功すれば enabled_revision に設定して\nstatus=enabled にする。以後この revision が trigger 評価対象になる。validation 失敗または\narchived workflow は拒否する。")
@TypeSpec.OpenAPI.operationId("EnableLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/enable")
@post
op EnableLifecycleWorkflow(
  @path workflow_id: string,
  @body body: EnableLifecycleWorkflowHttpRequest
): EnableLifecycleWorkflowSuccess_200 | EnableLifecycleWorkflowError400 | EnableLifecycleWorkflowError403;

model DisableLifecycleWorkflowHttpRequest {
  workflow_id: string;
  expected_revision: int64;
}

model DisableLifecycleWorkflowHttpResponse {
  workflow: IdMagic.Contract.LifecycleWorkflowResponse;
}

model DisableLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DisableLifecycleWorkflowHttpResponse;
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

@doc("管理者が LifecycleWorkflow を disable する。新規 trigger を止め、未開始の queued run を\ncanceled にする。running run は現在 step の checkpoint 後、次 step の前で停止する。")
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

@doc("管理者が LifecycleWorkflow を状態に関係なく削除する。削除後は一覧・取得・更新・名前重複判定から除外する。enabled の場合は新規 trigger を止め、queued run を canceled にする。既存 run と監査イベントは保持する。")
@TypeSpec.OpenAPI.operationId("DeleteLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}")
@delete
op DeleteLifecycleWorkflow(
  @path workflow_id: string,
  @query expected_revision: int64
): DeleteLifecycleWorkflowSuccess_204 | DeleteLifecycleWorkflowError400 | DeleteLifecycleWorkflowError403;

model DryRunLifecycleWorkflowHttpRequest {
  workflow_id: string;
  request: IdMagic.Contract.LifecycleWorkflowDryRunRequest;
}

model DryRunLifecycleWorkflowHttpResponse {
  response: IdMagic.Contract.LifecycleWorkflowDryRunResponse;
}

model DryRunLifecycleWorkflowSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DryRunLifecycleWorkflowHttpResponse;
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

@doc("管理者が enabled_revision (未指定なら current_revision) を対象 user へ試算する。同じ\nvalidator / trigger evaluator / action planner を使い、対象 User の現在の group membership /\napplication assignment / required action / status / email 検証状態を実際に読んで action ごとの\nwould_change / no_op / blocked を判定する。trigger の filter が対象 User に一致しない場合は\n全 action を blocked (reason: trigger_not_matched) として返す。対象 user が存在しないか別\ntenant の場合は InvalidRequestError を返す。WorkflowRun / Job / membership / assignment /\nrequired action / status / email のいずれも作成・変更しない。")
@TypeSpec.OpenAPI.operationId("DryRunLifecycleWorkflow")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/dry_run")
@post
op DryRunLifecycleWorkflow(
  @path workflow_id: string,
  @body body: DryRunLifecycleWorkflowHttpRequest
): DryRunLifecycleWorkflowSuccess_200 | DryRunLifecycleWorkflowError400 | DryRunLifecycleWorkflowError403;

model ListLifecycleWorkflowRunsHttpResponse {
  response: IdMagic.Contract.WorkflowRunListResponse;
}

model ListLifecycleWorkflowRunsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListLifecycleWorkflowRunsHttpResponse;
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

@doc("管理者が特定 LifecycleWorkflow の run 履歴を新しい順に一覧する。")
@TypeSpec.OpenAPI.operationId("ListLifecycleWorkflowRuns")
@route("/api/admin/v1/lifecycle_workflows/{workflow_id}/runs")
@get
op ListLifecycleWorkflowRuns(
  @path workflow_id: string
): ListLifecycleWorkflowRunsSuccess_200 | ListLifecycleWorkflowRunsError400 | ListLifecycleWorkflowRunsError403;

model GetLifecycleWorkflowRunHttpResponse {
  response: IdMagic.Contract.WorkflowRunResponse;
}

model GetLifecycleWorkflowRunSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetLifecycleWorkflowRunHttpResponse;
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

@doc("管理者が WorkflowRun 1 件の詳細 (trigger、revision、target user、Job attempt、各 step) を取得する。別テナントの run は未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetLifecycleWorkflowRun")
@route("/api/admin/v1/lifecycle_workflow_runs/{run_id}")
@get
op GetLifecycleWorkflowRun(
  @path run_id: string
): GetLifecycleWorkflowRunSuccess_200 | GetLifecycleWorkflowRunError400 | GetLifecycleWorkflowRunError403;

model RetryLifecycleWorkflowRunHttpRequest {
  run_id: string;
}

model RetryLifecycleWorkflowRunHttpResponse {
  response: IdMagic.Contract.WorkflowRunResponse;
}

model RetryLifecycleWorkflowRunSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RetryLifecycleWorkflowRunHttpResponse;
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

@doc("管理者が partially_failed または failed の WorkflowRun を再試行する。未完了 (failed) step\nだけを対象に新しい Job を関連付ける。changed / no_op 済み step は変更しない。disable と競合する\n場合は新しい step を開始しない。")
@TypeSpec.OpenAPI.operationId("RetryLifecycleWorkflowRun")
@route("/api/admin/v1/lifecycle_workflow_runs/{run_id}/retry")
@post
op RetryLifecycleWorkflowRun(
  @path run_id: string,
  @body body: RetryLifecycleWorkflowRunHttpRequest
): RetryLifecycleWorkflowRunSuccess_200 | RetryLifecycleWorkflowRunError400 | RetryLifecycleWorkflowRunError403;
}

}
