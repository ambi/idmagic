import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Signing Keys")
namespace IdMagic.SigningKeys {

namespace Operations {
model GetJwksHttpResponse {
  jwks: IdMagic.Contract.JWKS;
}

model GetJwksSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetJwksHttpResponse;
}

@doc("解決済みテナント (未 prefix リクエストは default) の JWKS (RFC 7517) を返す公開 interface。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetJwks")
@route("/jwks")
@get
op GetJwks(

): GetJwksSuccess_200;

model ListTenantJwksHttpResponse {
  jwks: IdMagic.Contract.JWKS;
}

model ListTenantJwksSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListTenantJwksHttpResponse;
}

@doc("テナント指定の JWKS (RFC 7517)。当該テナントの active + verifying\n(retired-not-expired) 鍵のみを返す。テナント A の JWKS にテナント B の kid は\n出ない。RP はこの URL で自テナント発行トークンの署名を検証する。認証を要求しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ListTenantJwks")
@route("/realms/{tenant_id}/jwks")
@get
op ListTenantJwks(
  @path tenant_id: string
): ListTenantJwksSuccess_200;

model ListAdminKeysHttpResponse {
  response: IdMagic.Contract.AdminKeyListResponse;
}

model ListAdminKeysSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminKeysHttpResponse;
}

union ListAdminKeysError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminKeysError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminKeysError403Body;
}

@doc("テナント内 admin が自テナントの JWKS に乗っている署名鍵 (active + verifying) の\nメタデータを取得する。対象は context.tenant_id に固定するため cross-tenant 読み出しは\n構造的に発生しない。")
@TypeSpec.OpenAPI.operationId("ListAdminKeys")
@route("/api/admin/v1/keys")
@get
op ListAdminKeys(

): ListAdminKeysSuccess_200 | ListAdminKeysError403;

model GetAdminKeyHttpResponse {
  key: IdMagic.Contract.AdminKeyResponse;
}

model GetAdminKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminKeyHttpResponse;
}

union GetAdminKeyError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminKeyError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminKeyError400Body;
}

union GetAdminKeyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminKeyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminKeyError403Body;
}

@doc("テナント内 admin が kid を指定して自テナントの署名鍵 1 件を取得する。他テナントの kid または存在しない kid は InvalidRequestError。")
@TypeSpec.OpenAPI.operationId("GetAdminKey")
@route("/api/admin/v1/keys/{kid}")
@get
op GetAdminKey(
  @path kid: string
): GetAdminKeySuccess_200 | GetAdminKeyError400 | GetAdminKeyError403;

model RotateTenantSigningKeyHttpResponse {
  response: IdMagic.Contract.AdminRotateKeyResponse;
}

model RotateTenantSigningKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: RotateTenantSigningKeyHttpResponse;
}

union RotateTenantSigningKeyError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RotateTenantSigningKeyError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RotateTenantSigningKeyError403Body;
}

@doc("テナント内 admin が自テナントに新しい署名鍵を生成し active を切り替える。旧 active 鍵は\nJWKS に verifying として残存し、SigningKeyMinJwksOverlap 期間後に Retire される\n(Retire 処理は別の運用ジョブ)。テナント単位のため回転の影響範囲は当該テナントに\n閉じる。")
@TypeSpec.OpenAPI.operationId("RotateTenantSigningKey")
@route("/api/admin/v1/keys/rotate")
@post
op RotateTenantSigningKey(

): RotateTenantSigningKeySuccess_200 | RotateTenantSigningKeyError403;

model DisableTenantKeyHttpRequest {
  kid: string;
}

model DisableTenantKeyHttpResponse {
  key: IdMagic.Contract.AdminKeyResponse;
}

model DisableTenantKeySuccess_200 {
  @statusCode statusCode: 200;
  @body body: DisableTenantKeyHttpResponse;
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

@doc("テナント内 admin が、回転後の移行期間にある非 active 署名鍵 1 件を kid 指定で\n即時無効化 (Retire) する。対象は JWKS から除去され、その鍵で署名された既発行\nトークンは検証できなくなる。現在の署名鍵はこの操作で無効化できない。")
@TypeSpec.OpenAPI.operationId("DisableTenantKey")
@route("/api/admin/v1/keys/{kid}/disable")
@post
op DisableTenantKey(
  @path kid: string,
  @body body: DisableTenantKeyHttpRequest
): DisableTenantKeySuccess_200 | DisableTenantKeyError400 | DisableTenantKeyError403;

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

@doc("system_admin がテナント横断で署名鍵ヘルス (provider / active kid /\n鍵数 / provider 到達性) を一覧する。秘密鍵は返さない。fail-closed 判定や\n回転漏れの検知に使う。")
@TypeSpec.OpenAPI.operationId("ListTenantKeyHealth")
@route("/api/admin/v1/keys/health")
@get
op ListTenantKeyHealth(

): ListTenantKeyHealthSuccess_200 | ListTenantKeyHealthError403;
}

}
