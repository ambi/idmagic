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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ListLifecycleWorkflowsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListLifecycleWorkflowsError403Body;
}

@doc("管理者が、自身のテナントの削除されていない LifecycleWorkflow を一覧する。削除済みの定義は返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("ListLifecycleWorkflows")
@route("/api/admin/v1/lifecycle-workflows")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateLifecycleWorkflowError403Body;
}

union CreateLifecycleWorkflowError409Body {
  IdMagic.Contract.WorkflowNameConflictError,
}

model CreateLifecycleWorkflowError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CreateLifecycleWorkflowError409Body;
}

@doc("管理者が LifecycleWorkflow を作成する。ワークフロー本体とリビジョン 1 の下書きをどちらも永続化し、`name` はテナントの削除されていないワークフローの中で一意である。削除済みのワークフローの名前は再び使える。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("CreateLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows")
@post
op CreateLifecycleWorkflow(
  @body body: IdMagic.Contract.LifecycleWorkflowCreateRequest
): CreateLifecycleWorkflowSuccess_201 | CreateLifecycleWorkflowError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateLifecycleWorkflowError403 | CreateLifecycleWorkflowError409;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model GetLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetLifecycleWorkflowError403Body;
}

@doc("管理者が LifecycleWorkflow を一つ読む。他のテナントのワークフローは存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("GetLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、LifecycleWorkflow のトリガー、アクション、または表示用のフィールドを更新する。`expected_revision` が `current_revision` と一致しないときは WorkflowRevisionConflictError で更新を拒否するので、更新の消失は起こらない。トリガーまたはアクションの変更は `current_revision` を 1 増やすが、`status` と `enabled_revision` は変えないので、EnableLifecycleWorkflow を再び呼ぶまでは古いリビジョンを評価し続ける。アーカイブ済みのワークフローは拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("UpdateLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}")
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

@doc("管理者が `current_revision` を検証し、通れば `enabled_revision` に設定して `status` を `enabled` へ移す。以降、トリガーが評価するのはそのリビジョンである。検証の失敗とアーカイブ済みのワークフローは、どちらも拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("EnableLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}/enable")
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

@doc("管理者が LifecycleWorkflow を無効にする。新しい実行は起動せず、開始していないキュー上の実行は取り消す。実行中の実行は、現在のステップのチェックポイントの後、次のステップの前で止まる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("DisableLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}/disable")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteLifecycleWorkflowError403Body;
}

@doc("管理者が、状態にかかわらず LifecycleWorkflow を削除する。以降は一覧、読み取り、更新、名前の一意性の検査から外れる。有効だった場合は、新しい実行を起動せず、キュー上の実行を取り消す。既存の実行と監査イベントは残す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("DeleteLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DryRunLifecycleWorkflowError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DryRunLifecycleWorkflowError403Body;
}

@doc("管理者が、`enabled_revision`、または指定がなければ `current_revision` を、対象のユーザーに対してドライランする。実際の実行と同じ検証器、トリガーの評価器、アクションの計画器を使い、対象の User の現在のグループのメンバーシップ、アプリケーションの割り当て、必須のアクション、状態、メールの確認状態を読んで、アクションごとに would_change、no_op、blocked のどれになるかを判断する。トリガーの絞り込み条件が対象の User に一致しないときは、すべてのアクションを理由 `trigger_not_matched` の blocked として返す。存在しない対象のユーザー、または他のテナントに属する対象のユーザーは InvalidRequestError になる。何も作成も変更もしない。WorkflowRun、Job、メンバーシップ、割り当て、必須のアクション、状態、メールのどれも変わらない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("DryRunLifecycleWorkflow")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}/dry-run")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ListLifecycleWorkflowRunsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListLifecycleWorkflowRunsError403Body;
}

@doc("管理者が、一つの LifecycleWorkflow の実行の履歴を新しい順に一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("ListLifecycleWorkflowRuns")
@route("/api/admin/v1/lifecycle-workflows/{workflow_id}/runs")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model GetLifecycleWorkflowRunError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetLifecycleWorkflowRunError403Body;
}

@doc("管理者が、一つの WorkflowRun の詳細（トリガー、リビジョン、対象のユーザー、Job の試行、各ステップ）を読む。他のテナントの実行は存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:read"])
@TypeSpec.OpenAPI.operationId("GetLifecycleWorkflowRun")
@route("/api/admin/v1/lifecycle-workflow-runs/{run_id}")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RetryLifecycleWorkflowRunError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RetryLifecycleWorkflowRunError403Body;
}

@doc("管理者が、`partially_failed` または `failed` の WorkflowRun を再試行する。新しい Job は失敗したステップだけに付け、すでに `changed` または `no_op` のステップは変えない。無効化と競合したときは、新しいステップを開始しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["lifecycle-workflows:write"])
@TypeSpec.OpenAPI.operationId("RetryLifecycleWorkflowRun")
@route("/api/admin/v1/lifecycle-workflow-runs/{run_id}/retry")
@post
op RetryLifecycleWorkflowRun(
  @path run_id: string,): RetryLifecycleWorkflowRunSuccess_200 | RetryLifecycleWorkflowRunError400 | IdMagic.Contract.AuthenticationRequiredResponse | RetryLifecycleWorkflowRunError403;
}

}
