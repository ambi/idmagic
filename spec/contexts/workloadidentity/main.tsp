import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Workload Identity")
namespace IdMagic.WorkloadIdentity {

namespace Operations {
model ListWorkloadTrustBundlesHttpResponse {
  response: IdMagic.Contract.WorkloadTrustBundleListResponse;
}

model ListWorkloadTrustBundlesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListWorkloadTrustBundlesHttpResponse;
}

union ListWorkloadTrustBundlesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListWorkloadTrustBundlesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListWorkloadTrustBundlesError403Body;
}

@doc("管理者が所属テナントの WorkloadTrustBundle を一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:read"])
@TypeSpec.OpenAPI.operationId("ListWorkloadTrustBundles")
@route("/api/admin/v1/workload-identity/trust-bundles")
@get
op ListWorkloadTrustBundles(

): ListWorkloadTrustBundlesSuccess_200 | ListWorkloadTrustBundlesError403;

model GetWorkloadTrustBundleHttpResponse {
  response: IdMagic.Contract.WorkloadTrustBundleResponse;
}

model GetWorkloadTrustBundleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetWorkloadTrustBundleHttpResponse;
}

union GetWorkloadTrustBundleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetWorkloadTrustBundleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetWorkloadTrustBundleError400Body;
}

union GetWorkloadTrustBundleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetWorkloadTrustBundleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetWorkloadTrustBundleError403Body;
}

@doc("管理者が単一 WorkloadTrustBundle を取得する。別テナントの bundle は未存在として扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:read"])
@TypeSpec.OpenAPI.operationId("GetWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}")
@get
op GetWorkloadTrustBundle(
  @path trust_bundle_id: string
): GetWorkloadTrustBundleSuccess_200 | GetWorkloadTrustBundleError400 | GetWorkloadTrustBundleError403;

model RegisterWorkloadTrustBundleHttpRequest {
  request: IdMagic.Contract.WorkloadTrustBundleRegisterRequest;
}

model RegisterWorkloadTrustBundleHttpResponse {
  response: IdMagic.Contract.WorkloadTrustBundleResponse;
}

model RegisterWorkloadTrustBundleSuccess_201 {
  @statusCode statusCode: 201;
  @body body: RegisterWorkloadTrustBundleHttpResponse;
}

union RegisterWorkloadTrustBundleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RegisterWorkloadTrustBundleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RegisterWorkloadTrustBundleError400Body;
}

union RegisterWorkloadTrustBundleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RegisterWorkloadTrustBundleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RegisterWorkloadTrustBundleError403Body;
}

union RegisterWorkloadTrustBundleError422Body {
  IdMagic.Contract.WorkloadTrustBundleJwksRequiredError,
  IdMagic.Contract.WorkloadTrustBundleNameRequiredError,
  IdMagic.Contract.WorkloadTrustBundleIssuerRequiredError,
  IdMagic.Contract.WorkloadTrustBundleAudiencesRequiredError,
  IdMagic.Contract.WorkloadTrustBundleInvalidTtlError,
}

model RegisterWorkloadTrustBundleError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RegisterWorkloadTrustBundleError422Body;
}

@doc("管理者が所属テナントに WorkloadTrustBundle を登録する。name はテナント内で一意、issuer もテナント内で一意でなければならない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("RegisterWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles")
@post
op RegisterWorkloadTrustBundle(
  @body body: RegisterWorkloadTrustBundleHttpRequest
): RegisterWorkloadTrustBundleSuccess_201 | RegisterWorkloadTrustBundleError400 | RegisterWorkloadTrustBundleError403 | RegisterWorkloadTrustBundleError422;

model UpdateWorkloadTrustBundleHttpRequest {
  trust_bundle_id: string;
  request: IdMagic.Contract.WorkloadTrustBundleUpdateRequest;
}

model UpdateWorkloadTrustBundleHttpResponse {
  response: IdMagic.Contract.WorkloadTrustBundleResponse;
}

model UpdateWorkloadTrustBundleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateWorkloadTrustBundleHttpResponse;
}

union UpdateWorkloadTrustBundleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateWorkloadTrustBundleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateWorkloadTrustBundleError400Body;
}

union UpdateWorkloadTrustBundleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateWorkloadTrustBundleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateWorkloadTrustBundleError403Body;
}

union UpdateWorkloadTrustBundleError422Body {
  IdMagic.Contract.WorkloadTrustBundleNameRequiredError,
  IdMagic.Contract.WorkloadTrustBundleAudiencesRequiredError,
  IdMagic.Contract.WorkloadTrustBundleInvalidTtlError,
}

model UpdateWorkloadTrustBundleError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateWorkloadTrustBundleError422Body;
}

@doc("管理者が WorkloadTrustBundle の name / jwks_uri / jwks / accepted_audiences / max_subject_token_ttl_seconds を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("UpdateWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}")
@patch
op UpdateWorkloadTrustBundle(
  @path trust_bundle_id: string,
  @body body: UpdateWorkloadTrustBundleHttpRequest
): UpdateWorkloadTrustBundleSuccess_200 | UpdateWorkloadTrustBundleError400 | UpdateWorkloadTrustBundleError403 | UpdateWorkloadTrustBundleError422;

model DisableWorkloadTrustBundleHttpRequest {
  trust_bundle_id: string;
}

model DisableWorkloadTrustBundleSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableWorkloadTrustBundleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableWorkloadTrustBundleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableWorkloadTrustBundleError400Body;
}

union DisableWorkloadTrustBundleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableWorkloadTrustBundleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableWorkloadTrustBundleError403Body;
}

@doc("管理者が WorkloadTrustBundle を無効化する。以後配下の全 binding が交換に使えなくなる (fail-closed)。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DisableWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/disable")
@post
op DisableWorkloadTrustBundle(
  @path trust_bundle_id: string,
  @body body: DisableWorkloadTrustBundleHttpRequest
): DisableWorkloadTrustBundleSuccess_204 | DisableWorkloadTrustBundleError400 | DisableWorkloadTrustBundleError403;

model EnableWorkloadTrustBundleHttpRequest {
  trust_bundle_id: string;
}

model EnableWorkloadTrustBundleSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableWorkloadTrustBundleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableWorkloadTrustBundleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableWorkloadTrustBundleError400Body;
}

union EnableWorkloadTrustBundleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableWorkloadTrustBundleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableWorkloadTrustBundleError403Body;
}

@doc("管理者が無効化済みの WorkloadTrustBundle を再有効化する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("EnableWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/enable")
@post
op EnableWorkloadTrustBundle(
  @path trust_bundle_id: string,
  @body body: EnableWorkloadTrustBundleHttpRequest
): EnableWorkloadTrustBundleSuccess_204 | EnableWorkloadTrustBundleError400 | EnableWorkloadTrustBundleError403;

model DeleteWorkloadTrustBundleSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteWorkloadTrustBundleError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteWorkloadTrustBundleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteWorkloadTrustBundleError400Body;
}

union DeleteWorkloadTrustBundleError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteWorkloadTrustBundleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteWorkloadTrustBundleError403Body;
}

@doc("管理者が WorkloadTrustBundle を削除する。配下の AgentWorkloadBinding は cascade で削除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DeleteWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}")
@delete
op DeleteWorkloadTrustBundle(
  @path trust_bundle_id: string
): DeleteWorkloadTrustBundleSuccess_204 | DeleteWorkloadTrustBundleError400 | DeleteWorkloadTrustBundleError403;

model RefreshWorkloadTrustBundleJWKSHttpRequest {
  trust_bundle_id: string;
}

model RefreshWorkloadTrustBundleJWKSHttpResponse {
  response: IdMagic.Contract.WorkloadTrustBundleRefreshResponse;
}

model RefreshWorkloadTrustBundleJWKSSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RefreshWorkloadTrustBundleJWKSHttpResponse;
}

union RefreshWorkloadTrustBundleJWKSError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RefreshWorkloadTrustBundleJWKSError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RefreshWorkloadTrustBundleJWKSError400Body;
}

union RefreshWorkloadTrustBundleJWKSError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RefreshWorkloadTrustBundleJWKSError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RefreshWorkloadTrustBundleJWKSError403Body;
}

@doc("管理者が WorkloadTrustBundle の JWKS 到達性を即時確認する (設定ミスの事前検知)。成功時は jwks_cached_at を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("RefreshWorkloadTrustBundleJWKS")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/refresh")
@post
op RefreshWorkloadTrustBundleJWKS(
  @path trust_bundle_id: string,
  @body body: RefreshWorkloadTrustBundleJWKSHttpRequest
): RefreshWorkloadTrustBundleJWKSSuccess_200 | RefreshWorkloadTrustBundleJWKSError400 | RefreshWorkloadTrustBundleJWKSError403;

model ListAgentWorkloadBindingsHttpResponse {
  response: IdMagic.Contract.AgentWorkloadBindingListResponse;
}

model ListAgentWorkloadBindingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAgentWorkloadBindingsHttpResponse;
}

union ListAgentWorkloadBindingsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAgentWorkloadBindingsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAgentWorkloadBindingsError400Body;
}

union ListAgentWorkloadBindingsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAgentWorkloadBindingsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAgentWorkloadBindingsError403Body;
}

@doc("管理者が WorkloadTrustBundle 配下の AgentWorkloadBinding を一覧する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:read"])
@TypeSpec.OpenAPI.operationId("ListAgentWorkloadBindings")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/bindings")
@get
op ListAgentWorkloadBindings(
  @path trust_bundle_id: string
): ListAgentWorkloadBindingsSuccess_200 | ListAgentWorkloadBindingsError400 | ListAgentWorkloadBindingsError403;

model CreateAgentWorkloadBindingHttpRequest {
  trust_bundle_id: string;
  request: IdMagic.Contract.AgentWorkloadBindingCreateRequest;
}

model CreateAgentWorkloadBindingHttpResponse {
  response: IdMagic.Contract.AgentWorkloadBindingResponse;
}

model CreateAgentWorkloadBindingSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateAgentWorkloadBindingHttpResponse;
}

union CreateAgentWorkloadBindingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateAgentWorkloadBindingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateAgentWorkloadBindingError400Body;
}

union CreateAgentWorkloadBindingError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateAgentWorkloadBindingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAgentWorkloadBindingError403Body;
}

union CreateAgentWorkloadBindingError422Body {
  IdMagic.Contract.AgentWorkloadBindingAgentNotFoundError,
  IdMagic.Contract.AgentWorkloadBindingPatternRequiredError,
}

model CreateAgentWorkloadBindingError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateAgentWorkloadBindingError422Body;
}

@doc("管理者が WorkloadTrustBundle 配下に AgentWorkloadBinding を作成する。agent_id は同一テナントの既存 Agent でなければならない (WorkloadIdentityReferencesStayTenantLocal)。同一 trust_bundle_id 内で subject_pattern の完全重複は拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("CreateAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/bindings")
@post
op CreateAgentWorkloadBinding(
  @path trust_bundle_id: string,
  @body body: CreateAgentWorkloadBindingHttpRequest
): CreateAgentWorkloadBindingSuccess_201 | CreateAgentWorkloadBindingError400 | CreateAgentWorkloadBindingError403 | CreateAgentWorkloadBindingError422;

model DisableAgentWorkloadBindingHttpRequest {
  binding_id: string;
}

model DisableAgentWorkloadBindingSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableAgentWorkloadBindingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableAgentWorkloadBindingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableAgentWorkloadBindingError400Body;
}

union DisableAgentWorkloadBindingError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableAgentWorkloadBindingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableAgentWorkloadBindingError403Body;
}

@doc("管理者が AgentWorkloadBinding を無効化する。以後この binding は交換に使えなくなる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DisableAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/bindings/{binding_id}/disable")
@post
op DisableAgentWorkloadBinding(
  @path binding_id: string,
  @body body: DisableAgentWorkloadBindingHttpRequest
): DisableAgentWorkloadBindingSuccess_204 | DisableAgentWorkloadBindingError400 | DisableAgentWorkloadBindingError403;

model EnableAgentWorkloadBindingHttpRequest {
  binding_id: string;
}

model EnableAgentWorkloadBindingSuccess_204 {
  @statusCode statusCode: 204;
}

union EnableAgentWorkloadBindingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EnableAgentWorkloadBindingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: EnableAgentWorkloadBindingError400Body;
}

union EnableAgentWorkloadBindingError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model EnableAgentWorkloadBindingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: EnableAgentWorkloadBindingError403Body;
}

@doc("管理者が無効化済みの AgentWorkloadBinding を再有効化する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("EnableAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/bindings/{binding_id}/enable")
@post
op EnableAgentWorkloadBinding(
  @path binding_id: string,
  @body body: EnableAgentWorkloadBindingHttpRequest
): EnableAgentWorkloadBindingSuccess_204 | EnableAgentWorkloadBindingError400 | EnableAgentWorkloadBindingError403;

model DeleteAgentWorkloadBindingSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAgentWorkloadBindingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAgentWorkloadBindingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentWorkloadBindingError400Body;
}

union DeleteAgentWorkloadBindingError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAgentWorkloadBindingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAgentWorkloadBindingError403Body;
}

@doc("管理者が AgentWorkloadBinding を削除する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DeleteAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/bindings/{binding_id}")
@delete
op DeleteAgentWorkloadBinding(
  @path binding_id: string
): DeleteAgentWorkloadBindingSuccess_204 | DeleteAgentWorkloadBindingError400 | DeleteAgentWorkloadBindingError403;
}

}
