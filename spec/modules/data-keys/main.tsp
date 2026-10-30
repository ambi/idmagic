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

model RotateTenantDataKeyHttpResponse {
  tenant_id: string;
  @doc("ローテーションで `active` になったバージョン。")
  active_version: int64;
  @doc("ローテーションで `retiring` になったバージョン。")
  previous_version: int64;
}

model RotateTenantDataKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: RotateTenantDataKeyHttpResponse;
}

model DataKeyLifecycleError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.InvalidRequestError;
}

union DataKeyLifecycleError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DataKeyLifecycleError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DataKeyLifecycleError403Body;
}

model RotateTenantDataKeyError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: IdMagic.Contract.TenantNotFoundError;
}

@doc("制御面の呼び出し元が、危殆化への対応として対象のテナントの DEK をローテーションする。`tenant_id` は ListTenantDataKeyHealth が返す値である。要求のたびにバージョンが増える。鍵素材は返さない（REQ-DATAKEYS-008）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("RotateTenantDataKey")
@route("/api/admin/v1/tenants/{tenant_id}/data-key/rotate")
@post
op RotateTenantDataKey(
  @path tenant_id: string,
): RotateTenantDataKeySuccess_200 | DataKeyLifecycleError400 | IdMagic.Contract.AuthenticationRequiredResponse | DataKeyLifecycleError403 | RotateTenantDataKeyError404;

model DisableTenantDataKeySuccess_204 {
  @statusCode statusCode: 204;
}

union DisableTenantDataKeyError404Body {
  IdMagic.Contract.TenantNotFoundError,
  IdMagic.Contract.DataKeyVersionNotFoundError,
}

model DisableTenantDataKeyError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DisableTenantDataKeyError404Body;
}

@doc("制御面の呼び出し元が、対象のテナントの `retiring` のバージョンを即時に無効化する。そのバージョンで暗号化した値は読めなくなり、元に戻せない（REQ-DATAKEYS-009）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DisableTenantDataKey")
@route("/api/admin/v1/tenants/{tenant_id}/data-key/versions/{version}/disable")
@post
op DisableTenantDataKey(
  @path tenant_id: string,
  @path version: int64,
): DisableTenantDataKeySuccess_204 | DataKeyLifecycleError400 | IdMagic.Contract.AuthenticationRequiredResponse | DataKeyLifecycleError403 | DisableTenantDataKeyError404;
}

}
