import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Workload Identity")
namespace IdMagic.WorkloadIdentity {

namespace Operations {
model ListWorkloadTrustBundlesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkloadTrustBundleListResponse;
}

union ListWorkloadTrustBundlesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListWorkloadTrustBundlesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListWorkloadTrustBundlesError403Body;
}

@doc("An administrator lists the WorkloadTrustBundles of their tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:read"])
@TypeSpec.OpenAPI.operationId("ListWorkloadTrustBundles")
@route("/api/admin/v1/workload-identity/trust-bundles")
@get
op ListWorkloadTrustBundles(

): ListWorkloadTrustBundlesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListWorkloadTrustBundlesError403;

model GetWorkloadTrustBundleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkloadTrustBundleResponse;
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

@doc("An administrator reads one WorkloadTrustBundle. A bundle of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:read"])
@TypeSpec.OpenAPI.operationId("GetWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}")
@get
op GetWorkloadTrustBundle(
  @path trust_bundle_id: string
): GetWorkloadTrustBundleSuccess_200 | GetWorkloadTrustBundleError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetWorkloadTrustBundleError403;

model RegisterWorkloadTrustBundleSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.WorkloadTrustBundleResponse;
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

@doc("An administrator registers a WorkloadTrustBundle in their tenant. Both `name` and `issuer` must be unique within the tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("RegisterWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles")
@post
op RegisterWorkloadTrustBundle(
  @body body: IdMagic.Contract.WorkloadTrustBundleRegisterRequest
): RegisterWorkloadTrustBundleSuccess_201 | RegisterWorkloadTrustBundleError400 | IdMagic.Contract.AuthenticationRequiredResponse | RegisterWorkloadTrustBundleError403 | RegisterWorkloadTrustBundleError422;

model UpdateWorkloadTrustBundleSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkloadTrustBundleResponse;
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

@doc("An administrator updates the `name`, `jwks_uri`, `jwks`, `accepted_audiences`, or `max_subject_token_ttl_seconds` of a WorkloadTrustBundle.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("UpdateWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}")
@patch
op UpdateWorkloadTrustBundle(
  @path trust_bundle_id: string,
  @body body: IdMagic.Contract.WorkloadTrustBundleUpdateRequest
): UpdateWorkloadTrustBundleSuccess_200 | UpdateWorkloadTrustBundleError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateWorkloadTrustBundleError403 | UpdateWorkloadTrustBundleError422;

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

@doc("An administrator disables a WorkloadTrustBundle. Every binding under it stops working for token exchange, fail-closed.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DisableWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/disable")
@post
op DisableWorkloadTrustBundle(
  @path trust_bundle_id: string,): DisableWorkloadTrustBundleSuccess_204 | DisableWorkloadTrustBundleError400 | DisableWorkloadTrustBundleError403;

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

@doc("An administrator re-enables a disabled WorkloadTrustBundle.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("EnableWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/enable")
@post
op EnableWorkloadTrustBundle(
  @path trust_bundle_id: string,): EnableWorkloadTrustBundleSuccess_204 | EnableWorkloadTrustBundleError400 | EnableWorkloadTrustBundleError403;

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

@doc("An administrator deletes a WorkloadTrustBundle. The AgentWorkloadBindings under it are deleted with it.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DeleteWorkloadTrustBundle")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}")
@delete
op DeleteWorkloadTrustBundle(
  @path trust_bundle_id: string
): DeleteWorkloadTrustBundleSuccess_204 | DeleteWorkloadTrustBundleError400 | DeleteWorkloadTrustBundleError403;

model RefreshWorkloadTrustBundleJWKSSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WorkloadTrustBundleRefreshResponse;
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

@doc("An administrator checks right away that the JWKS of a WorkloadTrustBundle is reachable, so a misconfiguration surfaces before it matters. On success `jwks_cached_at` is refreshed.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("RefreshWorkloadTrustBundleJWKS")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/refresh")
@post
op RefreshWorkloadTrustBundleJWKS(
  @path trust_bundle_id: string,): RefreshWorkloadTrustBundleJWKSSuccess_200 | RefreshWorkloadTrustBundleJWKSError400 | IdMagic.Contract.AuthenticationRequiredResponse | RefreshWorkloadTrustBundleJWKSError403;

model ListAgentWorkloadBindingsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AgentWorkloadBindingListResponse;
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

@doc("An administrator lists the AgentWorkloadBindings under a WorkloadTrustBundle.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:read"])
@TypeSpec.OpenAPI.operationId("ListAgentWorkloadBindings")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/bindings")
@get
op ListAgentWorkloadBindings(
  @path trust_bundle_id: string
): ListAgentWorkloadBindingsSuccess_200 | ListAgentWorkloadBindingsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ListAgentWorkloadBindingsError403;

model CreateAgentWorkloadBindingSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.AgentWorkloadBindingResponse;
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

@doc("An administrator creates an AgentWorkloadBinding under a WorkloadTrustBundle. `agent_id` must name an Agent that exists in the same tenant (WorkloadIdentityReferencesStayTenantLocal). Within one `trust_bundle_id`, an exact duplicate of `subject_pattern` is refused.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("CreateAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/trust-bundles/{trust_bundle_id}/bindings")
@post
op CreateAgentWorkloadBinding(
  @path trust_bundle_id: string,
  @body body: IdMagic.Contract.AgentWorkloadBindingCreateRequest
): CreateAgentWorkloadBindingSuccess_201 | CreateAgentWorkloadBindingError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAgentWorkloadBindingError403 | CreateAgentWorkloadBindingError422;

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

@doc("An administrator disables an AgentWorkloadBinding. It stops working for token exchange from then on.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DisableAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/bindings/{binding_id}/disable")
@post
op DisableAgentWorkloadBinding(
  @path binding_id: string,): DisableAgentWorkloadBindingSuccess_204 | DisableAgentWorkloadBindingError400 | DisableAgentWorkloadBindingError403;

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

@doc("An administrator re-enables a disabled AgentWorkloadBinding.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("EnableAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/bindings/{binding_id}/enable")
@post
op EnableAgentWorkloadBinding(
  @path binding_id: string,): EnableAgentWorkloadBindingSuccess_204 | EnableAgentWorkloadBindingError400 | EnableAgentWorkloadBindingError403;

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

@doc("An administrator deletes an AgentWorkloadBinding.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["workload-identity:write"])
@TypeSpec.OpenAPI.operationId("DeleteAgentWorkloadBinding")
@route("/api/admin/v1/workload-identity/bindings/{binding_id}")
@delete
op DeleteAgentWorkloadBinding(
  @path binding_id: string
): DeleteAgentWorkloadBindingSuccess_204 | DeleteAgentWorkloadBindingError400 | DeleteAgentWorkloadBindingError403;
}

}
