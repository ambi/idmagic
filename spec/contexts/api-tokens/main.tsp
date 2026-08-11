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
  IdMagic.Contract.AccessDeniedError,
}

model ListApiTokensError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListApiTokensError403Body;
}

@doc("テナントの API アクセストークンを一覧する。JWT 本文は返さない。")
@TypeSpec.OpenAPI.operationId("ListApiTokens")
@route("/api/admin/v1/api-tokens")
@get
op ListApiTokens(
  @query tenant_id: string
): ListApiTokensSuccess_200 | ListApiTokensError403;

model IssueApiTokenHttpRequest {
  tenant_id: string;
  client_id: string;
  description: string;
  scopes: IdMagic.Contract.ApiTokenScope[];
  expiry_days: int64;
  dpop_jkt?: string;
}

model IssueApiTokenHttpResponse {
  token: string;
  meta: IdMagic.Contract.ApiTokenMetadata;
}

model IssueApiTokenSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IssueApiTokenHttpResponse;
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
  IdMagic.Contract.AccessDeniedError,
}

model IssueApiTokenError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IssueApiTokenError403Body;
}

@doc("テナントの API アクセストークンを新規発行する。付与する scope 集合と有効日数を 指定する。RFC 9068 JWT access token を含む応答を一度だけ返す。")
@TypeSpec.OpenAPI.operationId("IssueApiToken")
@route("/api/admin/v1/api-tokens")
@post
op IssueApiToken(
  @body body: IssueApiTokenHttpRequest
): IssueApiTokenSuccess_200 | IssueApiTokenError400 | IssueApiTokenError403;

model RevokeApiTokenSuccess_200 {
  @statusCode statusCode: 200;
}

union RevokeApiTokenError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeApiTokenError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeApiTokenError400Body;
}

union RevokeApiTokenError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeApiTokenError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeApiTokenError403Body;
}

@doc("テナントの API アクセストークンを失効する。存在しない id への呼び出しも冪等に成功する。")
@TypeSpec.OpenAPI.operationId("RevokeApiToken")
@route("/api/admin/v1/api-tokens/{id}")
@delete
op RevokeApiToken(
  @query tenant_id: string,
  @path id: string
): RevokeApiTokenSuccess_200 | RevokeApiTokenError400 | RevokeApiTokenError403;
}

}
