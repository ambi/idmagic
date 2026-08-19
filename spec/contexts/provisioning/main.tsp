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

model RegisterProvisioningConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningConnection;
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

@doc("An administrator registers a new ProvisioningConnection on an Application. The attribute mappings are seeded automatically with the SCIM core User defaults.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("RegisterProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@post
op RegisterProvisioningConnection(
  @path id: string,
  @body body: RegisterProvisioningConnectionHttpRequest
): RegisterProvisioningConnectionSuccess_200 | RegisterProvisioningConnectionError400 | RegisterProvisioningConnectionError409 | RegisterProvisioningConnectionError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model GetProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetProvisioningConnectionError403Body;
}

@doc("An administrator reads the ProvisioningConnection of an Application. Another tenant's is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("GetProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@get
op GetProvisioningConnection(
  @path id: string
): GetProvisioningConnectionSuccess_200 | GetProvisioningConnectionError404 | GetProvisioningConnectionError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateProvisioningConnectionError403Body;
}

@doc("An administrator updates the settings of a ProvisioningConnection. An update that carries a credential counts as a\ncredential rotation and also emits ProvisioningCredentialRotated.")
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

@doc("An administrator deletes a ProvisioningConnection outright. The RemoteResourceLinks and ProvisioningDeliveries already recorded are kept as history.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("DeleteProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning")
@delete
op DeleteProvisioningConnection(
  @path id: string
): DeleteProvisioningConnectionSuccess_200 | DeleteProvisioningConnectionError404 | DeleteProvisioningConnectionError403;

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

@doc("An administrator checks that the downstream /ServiceProviderConfig is reachable, re-reads its capabilities, and refreshes the cache.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("TestProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning/test")
@post
op TestProvisioningConnection(
  @path id: string,): TestProvisioningConnectionSuccess_200 | TestProvisioningConnectionError404 | TestProvisioningConnectionError403;

model ProvisionOnDemandHttpRequest {
  subject_type: IdMagic.Contract.ProvisioningSourceType;
  subject_id: string;
}

model ProvisionOnDemandSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningDelivery;
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

@doc("An administrator creates a ProvisioningDelivery that delivers one named subject immediately, as a trial.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("ProvisionOnDemand")
@route("/api/admin/v1/applications/{id}/provisioning/on-demand")
@post
op ProvisionOnDemand(
  @path id: string,
  @body body: ProvisionOnDemandHttpRequest
): ProvisionOnDemandSuccess_200 | ProvisionOnDemandError404 | ProvisionOnDemandError422 | ProvisionOnDemandError403;

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

@doc("An administrator starts a full resync, which re-walks every subject inside the connection's scope until they converge. Completion is asynchronous and emits FullResyncCompleted.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("StartFullResync")
@route("/api/admin/v1/applications/{id}/provisioning/full-resync")
@post
op StartFullResync(
  @path id: string,): StartFullResyncSuccess_200 | StartFullResyncError404 | StartFullResyncError403;

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

@doc("An administrator lists the delivery state of a connection, newest `created_at` first with the id breaking ties, through\nbidirectional keyset pagination. `status` and `source_type` narrow the list. The cursor comes from the Link response header\nof the response (`rel=\"prev\"` and `rel=\"next\"`).")
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

model GetProvisioningDeliverySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningDelivery;
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

@doc("An administrator reads one delivery in detail.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("GetProvisioningDelivery")
@route("/api/admin/v1/applications/{id}/provisioning/deliveries/{delivery_id}")
@get
op GetProvisioningDelivery(
  @path id: string,
  @path delivery_id: string
): GetProvisioningDeliverySuccess_200 | GetProvisioningDeliveryError404 | GetProvisioningDeliveryError403;

model RetryProvisioningDeliverySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProvisioningDelivery;
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

@doc("An administrator retries a dead-lettered delivery by hand. It returns to pending and the dispatcher picks it up again.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("RetryProvisioningDelivery")
@route("/api/admin/v1/applications/{id}/provisioning/deliveries/{delivery_id}/retry")
@post
op RetryProvisioningDelivery(
  @path id: string,
  @path delivery_id: string,): RetryProvisioningDeliverySuccess_200 | RetryProvisioningDeliveryError404 | RetryProvisioningDeliveryError409 | RetryProvisioningDeliveryError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model ResumeProvisioningConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResumeProvisioningConnectionError403Body;
}

@doc("An administrator lifts the quarantine on a ProvisioningConnection and lets delivery resume. The interface exists because the operational runbook had no documented way out of quarantine.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:write"])
@TypeSpec.OpenAPI.operationId("ResumeProvisioningConnection")
@route("/api/admin/v1/applications/{id}/provisioning/resume")
@post
op ResumeProvisioningConnection(
  @path id: string,): ResumeProvisioningConnectionSuccess_200 | ResumeProvisioningConnectionError404 | ResumeProvisioningConnectionError403;

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

@doc("An administrator lists the live ProvisioningConnections across their tenant, as a read-only aggregate view of the tenant settings. Writes are directed to the per-Application interfaces.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["provisioning:read"])
@TypeSpec.OpenAPI.operationId("ListTenantProvisioningConnections")
@route("/api/admin/v1/provisioning/connections")
@get
op ListTenantProvisioningConnections(

): ListTenantProvisioningConnectionsSuccess_200 | ListTenantProvisioningConnectionsError403;
}

}
