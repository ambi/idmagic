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

@doc("解決したテナントの JWKS（RFC 7517）を返す公開インターフェース。レルムの接頭辞のないリクエストは、デフォルトのテナントへ解決する。")
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

@doc("指定したテナントの JWKS（RFC 7517）。そのテナントの active と verifying（retired だが期限切れでない）の鍵だけを含むので、テナント B の kid がテナント A の JWKS に現れることはない。Relying Party は、自身のテナントが発行したトークンの署名をこの URL で検証する。認証は不要である。")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminKeysError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminKeysError403Body;
}

@doc("テナントの管理者が、自身のテナントの JWKS にある署名鍵（active と verifying）のメタデータを読む。対象は `context.tenant_id` に固定されるので、テナントを横断した読み取りは構造上起こらない。")
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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("テナントの管理者が、自身のテナントの署名鍵を kid で一つ読む。他のテナントの kid も、存在しない kid も InvalidRequestError になる。")
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

union RotateTenantSigningKeyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RotateTenantSigningKeyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RotateTenantSigningKeyError400Body;
}

union RotateTenantSigningKeyError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("テナントの管理者が、求めた用途の新しい鍵を自身のテナントに生成し、有効な鍵をそれへ切り替える。それまで有効だった鍵は verifying として公開し続け（Signing では JWKS、XmlFederationSigning では SAML と WS-Federation のメタデータ）、SigningKeyMinJwksOverlap を過ぎると外す。アーカイブは別の運用ジョブである。ローテーションはテナントと用途ごとなので、影響範囲はそのテナントのその用途の鍵に限られる。`usage` が KeyUsage でないボディは InvalidRequestError になり、何もローテーションしない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:write"])
@TypeSpec.OpenAPI.operationId("RotateTenantSigningKey")
@route("/api/admin/v1/keys/rotate")
@post
op RotateTenantSigningKey(
  @body body?: IdMagic.Contract.AdminRotateKeyRequest
): RotateTenantSigningKeySuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | RotateTenantSigningKeyError400 | RotateTenantSigningKeyError403 | RotateTenantSigningKeyError503;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("テナントの管理者が、ローテーション後の重複期間内にある有効でない署名鍵を、kid で一つ即座に退役させる。鍵は JWKS から外れ、その鍵で署名済みのトークンは検証できなくなる。現在の署名鍵はこの方法では退役させられない。")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListTenantKeyHealthError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListTenantKeyHealthError403Body;
}

@doc("制御面の呼び出し元が、テナントを横断して署名鍵の健全性（プロバイダー、有効な kid、鍵の数、プロバイダーへの到達性）を一覧する。呼び出し元は、制御面テナントに所属し、有効ロールに `system_admin` を含み、制御面テナントの経路から呼び出す有効なユーザーでなければならない。他のテナントの `system_admin` は AccessDeniedError で拒否され、他のテナントの識別子も状態も観測しない。秘密鍵は返さない。フェイルクローズの条件を判断し、ローテーションから漏れたテナントを見つけるために使う（REQ-SIGNINGKEYS-008、REQ-SIGNINGKEYS-009）。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["signing-keys:read"])
@TypeSpec.OpenAPI.operationId("ListTenantKeyHealth")
@route("/api/admin/v1/keys/health")
@get
op ListTenantKeyHealth(

): ListTenantKeyHealthSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListTenantKeyHealthError403;
}

}
