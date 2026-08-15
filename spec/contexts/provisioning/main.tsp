import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Provisioning")
namespace IdMagic.Provisioning {

namespace Operations {
model RegisterProvisioningConnectionHttpRequest {
  id: string;
  base_url: url;
  credential: IdMagic.Contract.ProvisioningCredentialInput;
}

model RegisterProvisioningConnectionHttpResponse {
  connection: IdMagic.Contract.ProvisioningConnection;
}

model RegisterProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RegisterProvisioningConnectionHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model RegisterProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterProvisioningConnectionError403Body;
}

@doc("管理者が Application に ProvisioningConnection を新規登録する。属性マッピングは SCIM core User の既定を自動 seed する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("RegisterProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@post
op RegisterProvisioningConnection(
  @path id: string,
  @body body: RegisterProvisioningConnectionHttpRequest
): RegisterProvisioningConnectionSuccess_200 | RegisterProvisioningConnectionError400 | RegisterProvisioningConnectionError409 | RegisterProvisioningConnectionError403;

model GetProvisioningConnectionHttpResponse {
  connection: IdMagic.Contract.ProvisioningConnection;
}

model GetProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetProvisioningConnectionHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model GetProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningConnectionError403Body;
}

@doc("管理者が Application の ProvisioningConnection を取得する。別テナントは未存在として扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("GetProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@get
op GetProvisioningConnection(
  @path id: string
): GetProvisioningConnectionSuccess_200 | GetProvisioningConnectionError404 | GetProvisioningConnectionError403;

model UpdateProvisioningConnectionHttpRequest {
  id: string;
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

model UpdateProvisioningConnectionHttpResponse {
  connection: IdMagic.Contract.ProvisioningConnection;
}

model UpdateProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateProvisioningConnectionHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateProvisioningConnectionError403Body;
}

@doc("管理者が ProvisioningConnection の設定を更新する。credential を含む更新は\ncredential rotation として扱い ProvisioningCredentialRotated も発行する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("UpdateProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@patch
op UpdateProvisioningConnection(
  @path id: string,
  @body body: UpdateProvisioningConnectionHttpRequest
): UpdateProvisioningConnectionSuccess_200 | UpdateProvisioningConnectionError400 | UpdateProvisioningConnectionError404 | UpdateProvisioningConnectionError403;

model DeleteProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
}

union DeleteProvisioningConnectionError404Body {
  IdMagic.Contract.ProvisioningConnectionNotFoundError,
}

model DeleteProvisioningConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteProvisioningConnectionError404Body;
}

union DeleteProvisioningConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteProvisioningConnectionError403Body;
}

@doc("管理者が ProvisioningConnection を削除する (hard delete)。既存の RemoteResourceLink・ProvisioningDelivery は履歴として残す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("DeleteProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@delete
op DeleteProvisioningConnection(
  @path id: string
): DeleteProvisioningConnectionSuccess_200 | DeleteProvisioningConnectionError404 | DeleteProvisioningConnectionError403;

model TestProvisioningConnectionHttpRequest {
  id: string;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model TestProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: TestProvisioningConnectionError403Body;
}

@doc("管理者が下流 /ServiceProviderConfig への到達性を確認し capabilities を再取得してキャッシュを更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("TestProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning/test")
@post
op TestProvisioningConnection(
  @path id: string,
  @body body: TestProvisioningConnectionHttpRequest
): TestProvisioningConnectionSuccess_200 | TestProvisioningConnectionError404 | TestProvisioningConnectionError403;

model ProvisionOnDemandHttpRequest {
  id: string;
  subject_type: IdMagic.Contract.ProvisioningSourceType;
  subject_id: string;
}

model ProvisionOnDemandHttpResponse {
  delivery: IdMagic.Contract.ProvisioningDelivery;
}

model ProvisionOnDemandSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ProvisionOnDemandHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model ProvisionOnDemandError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ProvisionOnDemandError403Body;
}

@doc("管理者が単一 subject を指定して即時に試験配送する ProvisioningDelivery を作成する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("ProvisionOnDemand")
@route("/api/admin/v1/applications/{id}/provisioning/on-demand")
@post
op ProvisionOnDemand(
  @path id: string,
  @body body: ProvisionOnDemandHttpRequest
): ProvisionOnDemandSuccess_200 | ProvisionOnDemandError404 | ProvisionOnDemandError422 | ProvisionOnDemandError403;

model StartFullResyncHttpRequest {
  id: string;
}

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
  IdMagic.Contract.AccessDeniedError,
}

model StartFullResyncError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartFullResyncError403Body;
}

@doc("管理者が connection の scope 内 subject 全件を再走査し収束させる full resync を開始する。完了は非同期で FullResyncCompleted が発行される。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("StartFullResync")
@route("/api/admin/v1/applications/{id}/provisioning/full-resync")
@post
op StartFullResync(
  @path id: string,
  @body body: StartFullResyncHttpRequest
): StartFullResyncSuccess_200 | StartFullResyncError404 | StartFullResyncError403;

model ListProvisioningDeliveriesHttpResponse {
  deliveries: IdMagic.Contract.ProvisioningDelivery[];
}

model ListProvisioningDeliveriesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListProvisioningDeliveriesHttpResponse;
}

union ListProvisioningDeliveriesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListProvisioningDeliveriesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListProvisioningDeliveriesError403Body;
}

union ListProvisioningDeliveriesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListProvisioningDeliveriesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListProvisioningDeliveriesError400Body;
}

@doc("管理者が connection の配送状況を created_at 降順 (同値は id で tie-break) の双方向 keyset\npagination で一覧する。status/source_type で絞り込める。cursor は応答の Link response header\n(rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("ListProvisioningDeliveries")
@route("/api/admin/v1/applications/{id}/provisioning/deliveries")
@get
op ListProvisioningDeliveries(
  @path id: string,
  @query status?: IdMagic.Contract.ProvisioningDeliveryStatus,
  @query source_type?: IdMagic.Contract.ProvisioningSourceType,
  @query cursor?: string,
  @query limit?: int64
): ListProvisioningDeliveriesSuccess_200 | ListProvisioningDeliveriesError403 | ListProvisioningDeliveriesError400;

model GetProvisioningDeliveryHttpResponse {
  delivery: IdMagic.Contract.ProvisioningDelivery;
}

model GetProvisioningDeliverySuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetProvisioningDeliveryHttpResponse;
}

union GetProvisioningDeliveryError404Body {
  IdMagic.Contract.ProvisioningDeliveryNotFoundError,
}

model GetProvisioningDeliveryError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningDeliveryError404Body;
}

union GetProvisioningDeliveryError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetProvisioningDeliveryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningDeliveryError403Body;
}

@doc("管理者が配送 1 件の詳細を取得する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("GetProvisioningDelivery")
@route("/api/admin/v1/applications/{id}/provisioning/deliveries/{delivery_id}")
@get
op GetProvisioningDelivery(
  @path id: string,
  @path delivery_id: string
): GetProvisioningDeliverySuccess_200 | GetProvisioningDeliveryError404 | GetProvisioningDeliveryError403;

model RetryProvisioningDeliveryHttpRequest {
  id: string;
  delivery_id: string;
}

model RetryProvisioningDeliveryHttpResponse {
  delivery: IdMagic.Contract.ProvisioningDelivery;
}

model RetryProvisioningDeliverySuccess_200 {
  @statusCode statusCode: 200;
  @body body: RetryProvisioningDeliveryHttpResponse;
}

union RetryProvisioningDeliveryError404Body {
  IdMagic.Contract.ProvisioningDeliveryNotFoundError,
}

model RetryProvisioningDeliveryError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: RetryProvisioningDeliveryError404Body;
}

union RetryProvisioningDeliveryError409Body {
  IdMagic.Contract.ProvisioningDeliveryNotRetryableError,
}

model RetryProvisioningDeliveryError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: RetryProvisioningDeliveryError409Body;
}

union RetryProvisioningDeliveryError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RetryProvisioningDeliveryError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RetryProvisioningDeliveryError403Body;
}

@doc("管理者が dead_letter の配送を手動で再試行する。pending へ戻し dispatcher の再走査対象にする。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("RetryProvisioningDelivery")
@route("/api/admin/v1/applications/{id}/provisioning/deliveries/{delivery_id}/retry")
@post
op RetryProvisioningDelivery(
  @path id: string,
  @path delivery_id: string,
  @body body: RetryProvisioningDeliveryHttpRequest
): RetryProvisioningDeliverySuccess_200 | RetryProvisioningDeliveryError404 | RetryProvisioningDeliveryError409 | RetryProvisioningDeliveryError403;

model ResumeProvisioningConnectionHttpRequest {
  id: string;
}

model ResumeProvisioningConnectionHttpResponse {
  connection: IdMagic.Contract.ProvisioningConnection;
}

model ResumeProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ResumeProvisioningConnectionHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model ResumeProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResumeProvisioningConnectionError403Body;
}

@doc("管理者が quarantine された ProvisioningConnection を解除し配送を再開する。運用手順に quarantine 解除の経路が明記されていなかったため、本 interface を追加した。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("ResumeProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning/resume")
@post
op ResumeProvisioningConnection(
  @path id: string,
  @body body: ResumeProvisioningConnectionHttpRequest
): ResumeProvisioningConnectionSuccess_200 | ResumeProvisioningConnectionError404 | ResumeProvisioningConnectionError403;

model ListTenantProvisioningConnectionsHttpResponse {
  connections: IdMagic.Contract.ProvisioningConnection[];
}

model ListTenantProvisioningConnectionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListTenantProvisioningConnectionsHttpResponse;
}

union ListTenantProvisioningConnectionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantProvisioningConnectionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantProvisioningConnectionsError403Body;
}

@doc("管理者がテナント内で有効な ProvisioningConnection を横断的に一覧する (テナント設定の read-only 集約ビュー)。書き込みは各 Application 側の interface へ誘導する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("ListTenantProvisioningConnections")
@route("/api/admin/v1/provisioning/connections")
@get
op ListTenantProvisioningConnections(

): ListTenantProvisioningConnectionsSuccess_200 | ListTenantProvisioningConnectionsError403;
}

}
