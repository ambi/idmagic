import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("API Tokens")
namespace IdMagic.ApiTokens {

namespace Operations {
model ListApiTokensHttpResponse {
  tokens: IdMagic.Contract.ApiTokenMetadata[];
}

model ListApiTokensSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListApiTokensHttpResponse;
}

union ListApiTokensError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListApiTokensError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApiTokensError403Body;
}

@doc("テナントの API アクセストークンを一覧する。JWT そのものは返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListApiTokens")
@route("/api/admin/v1/api-tokens")
@get
op ListApiTokens(
  @query tenant_id: string
): ListApiTokensSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListApiTokensError403;

@doc("API トークンを発行するリクエストのボディ。`client_id` は含まない。すべての API トークンは組み込みの `idmagic-api-token` クライアントで発行するので、呼び出し元が選ぶものはなく、以前宣言していたこのフィールドをサーバーは読んでいなかった。")
model IssueApiTokenHttpRequest {
  description: string;
  scopes: IdMagic.Contract.ApiTokenScope[];
  expiry_days: int64;
  dpop_jkt?: string;
}

model IssueApiTokenHttpResponse {
  token: string;
  meta: IdMagic.Contract.ApiTokenMetadata;
}

union IssueApiTokenError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model IssueApiTokenError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: IssueApiTokenError400Body;
}

union IssueApiTokenError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model IssueApiTokenError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IssueApiTokenError403Body;
}

model IssueApiTokenSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IssueApiTokenHttpResponse;
}

@doc("呼び出し元が求めたスコープの集合と日数単位の有効期間で、テナントの API アクセストークンを新たに発行する。レスポンスは RFC 9068 の JWT アクセストークンを含み、トークンを返すのはこのときだけである。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("IssueApiToken")
@route("/api/admin/v1/api-tokens")
@post
op IssueApiToken(
  @body body: IssueApiTokenHttpRequest
): IssueApiTokenSuccess_201 | IssueApiTokenError400 | IdMagic.Contract.AuthenticationRequiredResponse | IssueApiTokenError403;

union RevokeApiTokenError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeApiTokenError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeApiTokenError403Body;
}

model RevokeApiTokenSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("テナントの API アクセストークンを失効させる。存在しない id を指定しても成功するので、この操作は冪等である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("RevokeApiToken")
@route("/api/admin/v1/api-tokens/{id}")
@delete
op RevokeApiToken(
  @query tenant_id: string,
  @path id: string
): RevokeApiTokenSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | RevokeApiTokenError403;
}

}
