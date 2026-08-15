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

@doc("system_admin がテナント横断で DEK 健全性 (active version / status / provider 到達性) を一覧する。鍵材料は返さない。fail-closed 判定や rotation 漏れの検知に使う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListTenantDataKeyHealth")
@route("/api/admin/v1/data-keys/health")
@get
op ListTenantDataKeyHealth(

): ListTenantDataKeyHealthSuccess_200 | ListTenantDataKeyHealthError403;
}

}
