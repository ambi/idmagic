import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Signing Keys")
namespace IdMagic.SigningKeys {

namespace Operations {
model GetJwksSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.JWKS;
}

@doc("Public interface returning the JWKS (RFC 7517) of the resolved tenant; a request without a realm prefix resolves to the default tenant.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetJwks")
@route("/jwks")
@get
op GetJwks(

): GetJwksSuccess_200;

model ListTenantJwksSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.JWKS;
}

@doc("The JWKS (RFC 7517) of a named tenant. It carries only that tenant's active and verifying\n(retired but not expired) keys, so the kid of tenant B never appears in the JWKS of tenant A.\nA relying party verifies the signature of tokens issued by its own tenant against this URL. No authentication is required.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ListTenantJwks")
@route("/realms/{tenant_id}/jwks")
@get
op ListTenantJwks(
  @path tenant_id: string
): ListTenantJwksSuccess_200;

model ListAdminKeysSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminKeyListResponse;
}

union ListAdminKeysError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminKeysError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminKeysError403Body;
}

@doc("A tenant administrator reads the metadata of the signing keys on their tenant's JWKS (active and verifying).\nThe target is fixed to `context.tenant_id`, so a cross-tenant read cannot arise structurally.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:read"])
@TypeSpec.OpenAPI.operationId("ListAdminKeys")
@route("/api/admin/v1/keys")
@get
op ListAdminKeys(

): ListAdminKeysSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminKeysError403;

model GetAdminKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminKeyResponse;
}

union GetAdminKeyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminKeyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminKeyError403Body;
}

union GetAdminKeyError404Body {
  IdMagic.Contract.SigningKeyNotFoundError,
}

model GetAdminKeyError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminKeyError404Body;
}

@doc("A tenant administrator reads one signing key of their tenant by kid. A kid of another tenant, and a kid that does not exist, both give InvalidRequestError.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:read"])
@TypeSpec.OpenAPI.operationId("GetAdminKey")
@route("/api/admin/v1/keys/{kid}")
@get
op GetAdminKey(
  @path kid: string
): GetAdminKeySuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminKeyError403 | GetAdminKeyError404;

model RotateTenantSigningKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminRotateKeyResponse;
}

union RotateTenantSigningKeyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RotateTenantSigningKeyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RotateTenantSigningKeyError403Body;
}

union RotateTenantSigningKeyError503Body {
  IdMagic.Contract.KeyStoreUnavailableError,
}

model RotateTenantSigningKeyError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: RotateTenantSigningKeyError503Body;
}

@doc("A tenant administrator generates a new signing key for their tenant and switches the active key to it. The previous\nactive key stays on the JWKS as verifying and is retired after SigningKeyMinJwksOverlap has passed; retiring is a\nseparate operational job. Rotation is per tenant, so its blast radius is confined to that tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:write"])
@TypeSpec.OpenAPI.operationId("RotateTenantSigningKey")
@route("/api/admin/v1/keys/rotate")
@post
op RotateTenantSigningKey(

): RotateTenantSigningKeySuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | RotateTenantSigningKeyError403 | RotateTenantSigningKeyError503;

model DisableTenantKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminKeyResponse;
}

union DisableTenantKeyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableTenantKeyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableTenantKeyError400Body;
}

union DisableTenantKeyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableTenantKeyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableTenantKeyError403Body;
}

union DisableTenantKeyError404Body {
  IdMagic.Contract.SigningKeyNotFoundError,
}

model DisableTenantKeyError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DisableTenantKeyError404Body;
}

union DisableTenantKeyError503Body {
  IdMagic.Contract.KeyStoreUnavailableError,
}

model DisableTenantKeyError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: DisableTenantKeyError503Body;
}

@doc("A tenant administrator retires one non-active signing key by kid, immediately, while it is still inside the\npost-rotation overlap. The key leaves the JWKS, and tokens already signed with it can no longer be verified.\nThe current signing key cannot be retired this way.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:write"])
@TypeSpec.OpenAPI.operationId("DisableTenantKey")
@route("/api/admin/v1/keys/{kid}/disable")
@post
op DisableTenantKey(
  @path kid: string,): DisableTenantKeySuccess_200 | DisableTenantKeyError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableTenantKeyError403 | DisableTenantKeyError404 | DisableTenantKeyError503;

model ListTenantKeyHealthHttpResponse {
  tenants: IdMagic.Contract.TenantSigningKey[];
}

model ListTenantKeyHealthSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListTenantKeyHealthHttpResponse;
}

union ListTenantKeyHealthError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantKeyHealthError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantKeyHealthError403Body;
}

@doc("A system_admin lists signing key health across tenants: provider, active kid, key count, and provider reachability.\nNo private key is returned. Used to judge fail-closed conditions and to catch tenants a rotation missed.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:read"])
@TypeSpec.OpenAPI.operationId("ListTenantKeyHealth")
@route("/api/admin/v1/keys/health")
@get
op ListTenantKeyHealth(

): ListTenantKeyHealthSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListTenantKeyHealthError403;
}

}
