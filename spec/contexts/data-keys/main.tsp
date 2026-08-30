import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Data Keys")
namespace IdMagic.DataKeys {

namespace Operations {
model ListTenantDataKeyHealthHttpResponse {
  tenants: IdMagic.Contract.TenantDataKeyHealth[];
}

model ListTenantDataKeyHealthSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListTenantDataKeyHealthHttpResponse;
}

union ListTenantDataKeyHealthError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantDataKeyHealthError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantDataKeyHealthError403Body;
}

@doc("A system_admin lists DEK health across tenants: active version, status, and provider reachability. No key material is returned. Used to judge fail-closed conditions and to catch tenants a rotation missed.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListTenantDataKeyHealth")
@route("/api/admin/v1/data-keys/health")
@get
op ListTenantDataKeyHealth(

): ListTenantDataKeyHealthSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListTenantDataKeyHealthError403;
}

}
