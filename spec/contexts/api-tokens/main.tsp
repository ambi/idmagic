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

@doc("Lists the API access tokens of the tenant. The JWT itself is never returned.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListApiTokens")
@route("/api/admin/v1/api-tokens")
@get
op ListApiTokens(
  @query tenant_id: string
): ListApiTokensSuccess_200 | ListApiTokensError403;

@doc("The body issuing an API token. It carries no `client_id`: every API token is issued under the built-in `idmagic-api-token` client, so the caller has nothing to choose and the server never read the field it used to declare.")
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

@doc("Issues a new API access token for the tenant, with the scope set and the lifetime in days the caller asks for. The response carries the RFC 9068 JWT access token, and it is the only time the token is returned.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
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

@doc("Revokes an API access token of the tenant. Calling it for an id that does not exist succeeds as well, so the operation is idempotent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("RevokeApiToken")
@route("/api/admin/v1/api-tokens/{id}")
@delete
op RevokeApiToken(
  @query tenant_id: string,
  @path id: string
): RevokeApiTokenSuccess_200 | RevokeApiTokenError400 | RevokeApiTokenError403;
}

}
