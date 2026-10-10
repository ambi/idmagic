import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Provisioning")
namespace IdMagic.Provisioning {

namespace Operations {
model RegisterProvisioningConnectionHttpRequest {
  base_url: url;
  credential: IdMagic.Contract.ProvisioningCredentialInput;
}

union RegisterProvisioningConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterProvisioningConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterProvisioningConnectionError400Body;
}

union RegisterProvisioningConnectionError409Body {
  IdMagic.Contract.ProvisioningConnectionAlreadyExistsError,
}

model RegisterProvisioningConnectionError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: RegisterProvisioningConnectionError409Body;
}

union RegisterProvisioningConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RegisterProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterProvisioningConnectionError403Body;
}

model RegisterProvisioningConnectionSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.ProvisioningConnection;
}

@doc("管理者が、Application に新しい ProvisioningConnection を登録する。属性の対応付けには、SCIM のコアの User のデフォルトを自動で入れる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("RegisterProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@post
op RegisterProvisioningConnection(
  @path id: string,
  @body body: RegisterProvisioningConnectionHttpRequest
): RegisterProvisioningConnectionSuccess_201 | RegisterProvisioningConnectionError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterProvisioningConnectionError409 | RegisterProvisioningConnectionError403;

model GetProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningConnection;
}

union GetProvisioningConnectionError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model GetProvisioningConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningConnectionError404Body;
}

union GetProvisioningConnectionError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningConnectionError403Body;
}

@doc("管理者が、Application の ProvisioningConnection を読む。他のテナントのものは存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("GetProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@get
op GetProvisioningConnection(
  @path id: string
): GetProvisioningConnectionSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetProvisioningConnectionError404 | GetProvisioningConnectionError403;

model UpdateProvisioningConnectionHttpRequest {
  base_url?: url;
  status?: IdMagic.Contract.ProvisioningConnectionStatus;
  credential?: IdMagic.Contract.ProvisioningCredentialInput;
  feature_flags?: IdMagic.Contract.ProvisioningFeatureFlags;
  scope?: IdMagic.Contract.ProvisioningScope;
  group_push?: IdMagic.Contract.GroupPushConfig;
  attribute_mappings?: IdMagic.Contract.AttributeMappingRule[];
  matching?: IdMagic.Contract.MatchingRule;
  deprovision_policy?: IdMagic.Contract.DeprovisionPolicy;
  rate_limit_per_minute?: int64;
  max_attempts?: int64;
  notification_email?: string;
  quarantine_after_consecutive_failures?: int64;
}

model UpdateProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningConnection;
}

union UpdateProvisioningConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateProvisioningConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateProvisioningConnectionError400Body;
}

union UpdateProvisioningConnectionError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model UpdateProvisioningConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateProvisioningConnectionError404Body;
}

union UpdateProvisioningConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateProvisioningConnectionError403Body;
}

@doc("管理者が、ProvisioningConnection の設定を更新する。資格情報を含む更新は資格情報のローテーションとみなし、ProvisioningCredentialRotated も発行する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("UpdateProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@patch
op UpdateProvisioningConnection(
  @path id: string,
  @body body: UpdateProvisioningConnectionHttpRequest
): UpdateProvisioningConnectionSuccess_200 | UpdateProvisioningConnectionError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateProvisioningConnectionError404 | UpdateProvisioningConnectionError403;

union DeleteProvisioningConnectionError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model DeleteProvisioningConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteProvisioningConnectionError404Body;
}

union DeleteProvisioningConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteProvisioningConnectionError403Body;
}

model DeleteProvisioningConnectionSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("管理者が、ProvisioningConnection を完全に削除する。記録済みの RemoteResourceLink と ProvisioningTask は履歴として残す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("DeleteProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@delete
op DeleteProvisioningConnection(
  @path id: string
): DeleteProvisioningConnectionSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteProvisioningConnectionError404 | DeleteProvisioningConnectionError403;

model TestProvisioningConnectionHttpResponse {
  reachable: boolean;
  capabilities?: IdMagic.Contract.ProvisioningCapabilities;
  error?: string;
}

model TestProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: TestProvisioningConnectionHttpResponse;
}

union TestProvisioningConnectionError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model TestProvisioningConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: TestProvisioningConnectionError404Body;
}

union TestProvisioningConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model TestProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: TestProvisioningConnectionError403Body;
}

@doc("管理者が、下流の /ServiceProviderConfig に到達できることを確かめ、その対応機能を読み直してキャッシュを更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("TestProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning/test")
@post
op TestProvisioningConnection(
  @path id: string,): TestProvisioningConnectionSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | TestProvisioningConnectionError404 | TestProvisioningConnectionError403;

model ProvisionOnDemandHttpRequest {
  subject_type: IdMagic.Contract.ProvisioningSourceType;
  subject_id: string;
}

union ProvisionOnDemandError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model ProvisionOnDemandError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: ProvisionOnDemandError404Body;
}

union ProvisionOnDemandError422Body {
  IdMagic.Contract.ProvisioningSubjectNotInScopeError,
}

model ProvisionOnDemandError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: ProvisionOnDemandError422Body;
}

union ProvisionOnDemandError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ProvisionOnDemandError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ProvisionOnDemandError403Body;
}

model ProvisionOnDemandSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.ProvisioningTask;
}

union ProvisionOnDemandError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ProvisionOnDemandError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ProvisionOnDemandError400Body;
}

@doc("管理者が、指定した一つのサブジェクトを試しに即座にプロビジョニングする ProvisioningTask を作成する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("ProvisionOnDemand")
@route("/api/admin/v1/applications/{id}/provisioning/on-demand")
@post
op ProvisionOnDemand(
  @path id: string,
  @body body: ProvisionOnDemandHttpRequest
): ProvisionOnDemandSuccess_201 | ProvisionOnDemandError400 | IdMagic.Contract.AuthenticationRequiredResponse | ProvisionOnDemandError404 | ProvisionOnDemandError422 | ProvisionOnDemandError403;

model StartFullResyncHttpResponse {
  enqueued_count: int64;
}

model StartFullResyncSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartFullResyncHttpResponse;
}

union StartFullResyncError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model StartFullResyncError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: StartFullResyncError404Body;
}

union StartFullResyncError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartFullResyncError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartFullResyncError403Body;
}

@doc("管理者がフル同期を開始する。接続の範囲内のすべてのサブジェクトを、収束するまでたどり直す。完了は非同期で、FullResyncCompleted を発行する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("StartFullResync")
@route("/api/admin/v1/applications/{id}/provisioning/full-resync")
@post
op StartFullResync(
  @path id: string,): StartFullResyncSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | StartFullResyncError404 | StartFullResyncError403;

model ListProvisioningTasksHttpResponse {
  tasks: IdMagic.Contract.ProvisioningTask[];
}

model ListProvisioningTasksSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListProvisioningTasksHttpResponse;
}

union ListProvisioningTasksError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListProvisioningTasksError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListProvisioningTasksError403Body;
}

union ListProvisioningTasksError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListProvisioningTasksError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListProvisioningTasksError400Body;
}

@doc("管理者が、接続のタスクの状態を、`created_at` の新しい順（同じなら id の順）に、双方向のキーセットページングで一覧する。`status` と `source_type` で絞り込める。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("ListProvisioningTasks")
@route("/api/admin/v1/applications/{id}/provisioning/tasks")
@get
op ListProvisioningTasks(
  @path id: string,
  @query status?: IdMagic.Contract.ProvisioningTaskStatus,
  @query source_type?: IdMagic.Contract.ProvisioningSourceType,
  @query cursor?: string,
  @query limit?: int64
): ListProvisioningTasksSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListProvisioningTasksError403 | ListProvisioningTasksError400;

model GetProvisioningTaskSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningTask;
}

union GetProvisioningTaskError404Body {
  IdMagic.Contract.ProvisioningTaskNotFoundError,
}

model GetProvisioningTaskError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningTaskError404Body;
}

union GetProvisioningTaskError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetProvisioningTaskError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningTaskError403Body;
}

@doc("管理者が、一つのタスクの詳細を読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("GetProvisioningTask")
@route("/api/admin/v1/applications/{id}/provisioning/tasks/{task_id}")
@get
op GetProvisioningTask(
  @path id: string,
  @path task_id: string
): GetProvisioningTaskSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetProvisioningTaskError404 | GetProvisioningTaskError403;

model RetryProvisioningTaskSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningTask;
}

union RetryProvisioningTaskError404Body {
  IdMagic.Contract.ProvisioningTaskNotFoundError,
}

model RetryProvisioningTaskError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: RetryProvisioningTaskError404Body;
}

union RetryProvisioningTaskError409Body {
  IdMagic.Contract.ProvisioningTaskNotRetryableError,
}

model RetryProvisioningTaskError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: RetryProvisioningTaskError409Body;
}

union RetryProvisioningTaskError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RetryProvisioningTaskError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RetryProvisioningTaskError403Body;
}

@doc("管理者が、デッドレターになったタスクを手動で再試行する。タスクは pending へ戻り、ディスパッチャーが再び拾う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("RetryProvisioningTask")
@route("/api/admin/v1/applications/{id}/provisioning/tasks/{task_id}/retry")
@post
op RetryProvisioningTask(
  @path id: string,
  @path task_id: string,): RetryProvisioningTaskSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | RetryProvisioningTaskError404 | RetryProvisioningTaskError409 | RetryProvisioningTaskError403;

model ResumeProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningConnection;
}

union ResumeProvisioningConnectionError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model ResumeProvisioningConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: ResumeProvisioningConnectionError404Body;
}

union ResumeProvisioningConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ResumeProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResumeProvisioningConnectionError403Body;
}

@doc("管理者が、ProvisioningConnection の隔離を解除し、プロビジョニングを再開させる。運用手順書に隔離から抜ける方法が記されていなかったので、このインターフェースを設けた。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("ResumeProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning/resume")
@post
op ResumeProvisioningConnection(
  @path id: string,): ResumeProvisioningConnectionSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ResumeProvisioningConnectionError404 | ResumeProvisioningConnectionError403;

model ListTenantProvisioningConnectionsHttpResponse {
  connections: IdMagic.Contract.ProvisioningConnection[];
}

model ListTenantProvisioningConnectionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListTenantProvisioningConnectionsHttpResponse;
}

union ListTenantProvisioningConnectionsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantProvisioningConnectionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantProvisioningConnectionsError403Body;
}

@doc("管理者が、自身のテナントの有効な ProvisioningConnection を、テナント設定の読み取り専用の集約ビューとして一覧する。書き込みは Application ごとのインターフェースで行う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("ListTenantProvisioningConnections")
@route("/api/admin/v1/provisioning/connections")
@get
op ListTenantProvisioningConnections(

): ListTenantProvisioningConnectionsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListTenantProvisioningConnectionsError403;
}

}
