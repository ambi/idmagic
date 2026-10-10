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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantDataKeyHealthError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantDataKeyHealthError403Body;
}

@doc("制御面の呼び出し元が、テナントを横断して DEK の健全性（有効なバージョン、状態、プロバイダーへの到達性）を一覧する。呼び出し元は、制御面テナントに所属し、有効ロールに `system_admin` を含み、制御面テナントの経路から呼び出す有効なユーザーでなければならない。他のテナントの `system_admin` は AccessDeniedError で拒否され、他のテナントの識別子も状態も観測しない。暗号鍵は返さない。フェイルクローズの条件を判断し、ローテーションから漏れたテナントを見つけるために使う（REQ-DATAKEYS-006）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListTenantDataKeyHealth")
@route("/api/admin/v1/data-keys/health")
@get
op ListTenantDataKeyHealth(

): ListTenantDataKeyHealthSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListTenantDataKeyHealthError403;
}

}
