import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("OAuth 2.0 and OpenID Connect")
namespace IdMagic.OAuth2 {

namespace Operations {
union RegisterClientError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
}

model RegisterClientError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: RegisterClientError400Body;
}

union RegisterClientError422Body {
  IdMagic.Contract.QuotaExceededError,
}

model RegisterClientError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: RegisterClientError422Body;
}

model RegisterClientSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.ClientRegistrationResponse;
}

@doc("Dynamic Client Registration (RFC 7591)")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("RegisterClient")
@route("/register")
@post
op RegisterClient(
  @body body: IdMagic.Contract.ClientRegistrationRequest
): RegisterClientSuccess_201 | RegisterClientError400 | RegisterClientError422;

model AuthorizeSuccess_302 {
  @statusCode statusCode: 302;
}

union AuthorizeError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
  IdMagic.Contract.InvalidTargetError,
}

model AuthorizeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: AuthorizeError400Body;
}

union AuthorizeError401Body {
  IdMagic.Contract.InvalidClientError,
}

model AuthorizeError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: AuthorizeError401Body;
}

union AuthorizeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

@doc("The sign-in policy refuses the request before an authorization code exists. This is the one refusal the authorization endpoint reports to the browser directly rather than redirecting to `redirect_uri`, so it carries Problem Details like the browser-facing endpoints rather than the OAuth error body.")
model AuthorizeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: AuthorizeError403Body;
}

union AuthorizeError429Body {
  IdMagic.Contract.RateLimitedError,
}

model AuthorizeError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: AuthorizeError429Body;
}

model AuthorizeSuccess_303 {
  @statusCode statusCode: 303;
}

@doc("The authorization endpoint (RFC 6749 section 4.1.1). `client_id` is accepted either as a registered Client or as the URL of a Client ID Metadata Document, resolved on each request.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("Authorize")
@route("/authorize")
@get
op Authorize(
  @query params: IdMagic.Contract.AuthorizeParameters
): AuthorizeSuccess_302 | AuthorizeSuccess_303 | AuthorizeError400 | AuthorizeError401 | AuthorizeError403 | AuthorizeError429;

model ResumeFederatedAuthorizationSuccess_302 {
  @statusCode statusCode: 302;
}

union ResumeFederatedAuthorizationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ResumeFederatedAuthorizationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ResumeFederatedAuthorizationError400Body;
}

union ResumeFederatedAuthorizationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ResumeFederatedAuthorizationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResumeFederatedAuthorizationError403Body;
}

model ResumeFederatedAuthorizationSuccess_303 {
  @statusCode statusCode: 303;
}

@doc("Once the inbound federation callback of the Authentication context has established an authenticated session, the stored authorization transaction resumes and the ordinary application policy, consent, and required-action gates apply.")
@TypeSpec.OpenAPI.operationId("ResumeFederatedAuthorization")
@route("/authorize/resume")
@get
op ResumeFederatedAuthorization(

): ResumeFederatedAuthorizationSuccess_302 | ResumeFederatedAuthorizationSuccess_303 | ResumeFederatedAuthorizationError400 | IdMagic.Contract.AuthenticationRequiredResponse | ResumeFederatedAuthorizationError403;

model GetBrowserTransactionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserTransactionResponse;
}

union GetBrowserTransactionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetBrowserTransactionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetBrowserTransactionError400Body;
}

@doc("The SPA reads the sign-in and consent context that goes with its HttpOnly transaction cookie.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetBrowserTransaction")
@route("/api/auth/transaction")
@get
op GetBrowserTransaction(

): GetBrowserTransactionSuccess_200 | GetBrowserTransactionError400 | IdMagic.Contract.AuthenticationRequiredResponse;

model SubmitBrowserConsentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
}

union SubmitBrowserConsentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SubmitBrowserConsentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserConsentError400Body;
}

union SubmitBrowserConsentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserConsentError403Body;
}

union SubmitBrowserConsentError422Body {
  IdMagic.Contract.QuotaExceededError,
}

model SubmitBrowserConsentError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserConsentError422Body;
}

@doc("The SPA settles the consent or the refusal of an authorization request through the JSON API.")
@TypeSpec.OpenAPI.operationId("SubmitBrowserConsent")
@route("/api/auth/consent")
@post
op SubmitBrowserConsent(
  @body body: IdMagic.Contract.BrowserConsentRequest
): SubmitBrowserConsentSuccess_200 | SubmitBrowserConsentError400 | IdMagic.Contract.AuthenticationRequiredResponse | SubmitBrowserConsentError403 | SubmitBrowserConsentError422;

model SubmitBrowserDeviceSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
}

union SubmitBrowserDeviceError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SubmitBrowserDeviceError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserDeviceError400Body;
}

union SubmitBrowserDeviceError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserDeviceError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserDeviceError403Body;
}

@doc("The SPA approves or refuses a device `user_code` through the JSON API.")
@TypeSpec.OpenAPI.operationId("SubmitBrowserDevice")
@route("/api/auth/device")
@post
op SubmitBrowserDevice(
  @body body: IdMagic.Contract.BrowserDeviceRequest
): SubmitBrowserDeviceSuccess_200 | SubmitBrowserDeviceError400 | IdMagic.Contract.AuthenticationRequiredResponse | SubmitBrowserDeviceError403;

model GetBrowserDeviceContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserDeviceContextResponse;
}

@doc("The SPA reads the device `user_code` and the CSRF token.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetBrowserDeviceContext")
@route("/api/auth/device")
@get
op GetBrowserDeviceContext(

): GetBrowserDeviceContextSuccess_200;

model EndSession1Success_200 {
  @statusCode statusCode: 200;
}

model EndSession1Success_302 {
  @statusCode statusCode: 302;
}

union EndSession1Error400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
}

model EndSession1Error400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: EndSession1Error400Body;
}

model EndSession1Success_303 {
  @statusCode statusCode: 303;
}

@doc("The RP-Initiated Logout endpoint (OpenID Connect RP-Initiated Logout 1.0). Redirect validation is done by the\nOAuth 2.0 / OpenID Connect bounded context, and session revocation is delegated to the SessionManager of the\nAuthentication bounded context. When an `id_token_hint` is given, its signature, `iss`, `aud`, `sub`, and `sid` are\nverified and both the target session and the client are resolved from the hint; a hint that contradicts `client_id` is\nrefused. Without an `id_token_hint`, the existing resolution by `client_id` plus the browser cookie is kept as a\nfallback. Once the local revocation is settled, the FrontChannelLogout targets are collected from the ClientSessions\nbound to that `sid` and returned in the response, and BackChannelLogout delivery is started asynchronously.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("EndSession")
@route("/end_session")
@get
op EndSession1(
  @query params: IdMagic.Contract.EndSessionParameters
): EndSession1Success_200 | EndSession1Success_302 | EndSession1Success_303 | EndSession1Error400;

model EndSession2Success_200 {
  @statusCode statusCode: 200;
}

model EndSession2Success_302 {
  @statusCode statusCode: 302;
}

union EndSession2Error400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
}

model EndSession2Error400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: EndSession2Error400Body;
}

model EndSession2Success_303 {
  @statusCode statusCode: 303;
}

@doc("The RP-Initiated Logout endpoint (OpenID Connect RP-Initiated Logout 1.0). Redirect validation is done by the\nOAuth 2.0 / OpenID Connect bounded context, and session revocation is delegated to the SessionManager of the\nAuthentication bounded context. When an `id_token_hint` is given, its signature, `iss`, `aud`, `sub`, and `sid` are\nverified and both the target session and the client are resolved from the hint; a hint that contradicts `client_id` is\nrefused. Without an `id_token_hint`, the existing resolution by `client_id` plus the browser cookie is kept as a\nfallback. Once the local revocation is settled, the FrontChannelLogout targets are collected from the ClientSessions\nbound to that `sid` and returned in the response, and BackChannelLogout delivery is started asynchronously.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("EndSession")
@route("/end_session")
@post
op EndSession2(
  @header contentType: "application/x-www-form-urlencoded",
  @body body: IdMagic.Contract.EndSessionParameters
): EndSession2Success_200 | EndSession2Success_302 | EndSession2Success_303 | EndSession2Error400;

model CheckSessionIframeSuccess_200 {
  @statusCode statusCode: 200;
}

@doc("The hidden iframe an RP polls for changes to the OP session (OpenID Connect Session Management 1.0; adoption is\noptional, because the specification is still a draft and has never reached Final).\nIt returns static HTML and JavaScript, and answers a `postMessage` from the RP with nothing more than whether the\ncurrent browser cookie resolves to a valid LoginSession. The detailed correlation algorithm of the specification, the\nsalted hash comparison of `session_state` among it, is not implemented.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CheckSessionIframe")
@route("/session/check")
@get
op CheckSessionIframe(

): CheckSessionIframeSuccess_200;

union PushAuthorizationRequestError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
  IdMagic.Contract.InvalidTargetError,
}

model PushAuthorizationRequestError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: PushAuthorizationRequestError400Body;
}

union PushAuthorizationRequestError401Body {
  IdMagic.Contract.InvalidClientError,
}

model PushAuthorizationRequestError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: PushAuthorizationRequestError401Body;
}

union PushAuthorizationRequestError429Body {
  IdMagic.Contract.RateLimitedError,
}

model PushAuthorizationRequestError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: PushAuthorizationRequestError429Body;
}

model PushAuthorizationRequestSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.ParResponse;
}

@doc("Pushed Authorization Request (RFC 9126). `client_id` is accepted either as a registered Client or as the URL of a Client ID Metadata Document, resolved on each request.")
@TypeSpec.OpenAPI.operationId("PushAuthorizationRequest")
@route("/par")
@post
op PushAuthorizationRequest(
  @body body: IdMagic.Contract.ParRequest
): PushAuthorizationRequestSuccess_201 | PushAuthorizationRequestError400 | PushAuthorizationRequestError401 | PushAuthorizationRequestError429;

model TokenSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TokenResponse;
}

union TokenError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
  IdMagic.Contract.InvalidGrantError,
  IdMagic.Contract.UnauthorizedClientError,
  IdMagic.Contract.UnsupportedGrantTypeError,
  IdMagic.Contract.InvalidScopeError,
  IdMagic.Contract.AuthorizationPendingError,
  IdMagic.Contract.SlowDownError,
  IdMagic.Contract.ExpiredTokenError,
  IdMagic.Contract.InvalidTargetError,
  IdMagic.Contract.OAuthAccessDeniedError,
}

model TokenError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: TokenError400Body;
}

union TokenError401Body {
  IdMagic.Contract.InvalidClientError,
  IdMagic.Contract.InvalidDpopProofError,
}

model TokenError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: TokenError401Body;
}

union TokenError422Body {
  IdMagic.Contract.ClaimReleaseDeniedError,
}

model TokenError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/json";
  @body body: TokenError422Body;
}

union TokenError429Body {
  IdMagic.Contract.RateLimitedError,
}

model TokenError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: TokenError429Body;
}

@doc("Token endpoint (RFC 6749 §3.2 + 9068 + 9449)")
@TypeSpec.OpenAPI.operationId("Token")
@route("/token")
@post
op Token(
  @body body: IdMagic.Contract.TokenRequest
): TokenSuccess_200 | TokenError400 | TokenError401 | TokenError422 | TokenError429;

model IntrospectSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.IntrospectionResponse;
}

union IntrospectError401Body {
  IdMagic.Contract.InvalidClientError,
  IdMagic.Contract.InvalidTokenError,
}

model IntrospectError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: IntrospectError401Body;
}

union IntrospectError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
}

model IntrospectError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: IntrospectError400Body;
}

@doc("Token Introspection (RFC 7662) for both ordinary OAuth tokens and administratively issued JWT access tokens. For a token whose principal is an Agent, `issued_at` is compared against the revocation epoch of SharedSignals -- the kill switch, an owner offboarding, or a revocation arriving over CAEP/SSF -- and a token issued before the epoch is reported `active=false`, fail-closed.")
@TypeSpec.OpenAPI.operationId("Introspect")
@route("/introspect")
@post
op Introspect(
  @body body: IdMagic.Contract.IntrospectionRequest
): IntrospectSuccess_200 | IntrospectError400 | IntrospectError401;

model RevokeSuccess_200 {
  @statusCode statusCode: 200;
}

union RevokeError401Body {
  IdMagic.Contract.InvalidClientError,
}

model RevokeError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: RevokeError401Body;
}

union RevokeError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
}

model RevokeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: RevokeError400Body;
}

@doc("Token Revocation (RFC 7009) for both ordinary OAuth tokens and administratively issued JWT access tokens.")
@TypeSpec.OpenAPI.operationId("Revoke")
@route("/revoke")
@post
op Revoke(
  @body body: IdMagic.Contract.RevokeRequest
): RevokeSuccess_200 | RevokeError400 | RevokeError401;

model UserInfoSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserInfoResponse;
}

union UserInfoError401Body {
  IdMagic.Contract.InvalidTokenError,
}

model UserInfoError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: UserInfoError401Body;
}

union UserInfoError403Body {
  IdMagic.Contract.InsufficientScopeError,
}

model UserInfoError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/json";
  @body body: UserInfoError403Body;
}

union UserInfoError422Body {
  IdMagic.Contract.ClaimReleaseDeniedError,
}

model UserInfoError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/json";
  @body body: UserInfoError422Body;
}

@doc("UserInfo (OpenID Connect Core section 5.3). `additional_claims` merges the scope-driven ClaimsForScopes with the per-client ClaimMapping claim policy.")
@TypeSpec.OpenAPI.operationId("UserInfo")
@route("/userinfo")
@get
op UserInfo(

): UserInfoSuccess_200 | UserInfoError401 | UserInfoError403 | UserInfoError422;

model PostUserInfoSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserInfoResponse;
}

union PostUserInfoError401Body {
  IdMagic.Contract.InvalidTokenError,
}

model PostUserInfoError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: PostUserInfoError401Body;
}

union PostUserInfoError403Body {
  IdMagic.Contract.InsufficientScopeError,
}

model PostUserInfoError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/json";
  @body body: PostUserInfoError403Body;
}

union PostUserInfoError422Body {
  IdMagic.Contract.ClaimReleaseDeniedError,
}

model PostUserInfoError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/json";
  @body body: PostUserInfoError422Body;
}

@doc("The POST binding of UserInfo (OpenID Connect Core section 5.3).")
@TypeSpec.OpenAPI.operationId("PostUserInfo")
@route("/userinfo")
@post
op PostUserInfo(

): PostUserInfoSuccess_200 | PostUserInfoError401 | PostUserInfoError403 | PostUserInfoError422;

model DeviceAuthorizationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DeviceAuthorizationResponse;
}

union DeviceAuthorizationError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
}

model DeviceAuthorizationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: DeviceAuthorizationError400Body;
}

union DeviceAuthorizationError401Body {
  IdMagic.Contract.InvalidClientError,
}

model DeviceAuthorizationError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: DeviceAuthorizationError401Body;
}

union DeviceAuthorizationError429Body {
  IdMagic.Contract.RateLimitedError,
}

model DeviceAuthorizationError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: DeviceAuthorizationError429Body;
}

@doc("Device Authorization Request (RFC 8628 §3.1)")
@TypeSpec.OpenAPI.operationId("DeviceAuthorization")
@route("/device_authorization")
@post
op DeviceAuthorization(
  @body body: IdMagic.Contract.DeviceAuthorizationRequest
): DeviceAuthorizationSuccess_200 | DeviceAuthorizationError400 | DeviceAuthorizationError401 | DeviceAuthorizationError429;

model BackchannelAuthenticateSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BackchannelAuthenticationResponse;
}

union BackchannelAuthenticateError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
  IdMagic.Contract.InvalidScopeError,
  IdMagic.Contract.InvalidBindingMessageError,
  IdMagic.Contract.UnknownUserIdError,
  IdMagic.Contract.UnauthorizedClientError,
}

model BackchannelAuthenticateError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: BackchannelAuthenticateError400Body;
}

union BackchannelAuthenticateError401Body {
  IdMagic.Contract.InvalidClientError,
}

model BackchannelAuthenticateError401 {
  @statusCode statusCode: 401;
  @header contentType: "application/json";
  @body body: BackchannelAuthenticateError401Body;
}

union BackchannelAuthenticateError429Body {
  IdMagic.Contract.RateLimitedError,
}

model BackchannelAuthenticateError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: BackchannelAuthenticateError429Body;
}

@doc("Backchannel Authentication Request (OpenID CIBA Core section 7.1). It authenticates the client, raises an approval request for a human to act on, and returns an `auth_req_id`. Until the approval is settled the CIBA grant at the token endpoint returns `authorization_pending`, and once settled it is exchanged for tokens exactly once.")
@TypeSpec.OpenAPI.operationId("BackchannelAuthenticate")
@route("/bc-authorize")
@post
op BackchannelAuthenticate(
  @body body: IdMagic.Contract.BackchannelAuthenticationRequest
): BackchannelAuthenticateSuccess_200 | BackchannelAuthenticateError400 | BackchannelAuthenticateError401 | BackchannelAuthenticateError429;

model GetOpenidConfigurationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DiscoveryDocument;
}

@doc("OIDC Discovery")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetOpenidConfiguration")
@route("/.well-known/openid-configuration")
@get
op GetOpenidConfiguration(

): GetOpenidConfigurationSuccess_200;

model GetOauthAuthorizationServerSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.DiscoveryDocument;
}

@doc("OAuth 2.0 Authorization Server Metadata (RFC 8414)")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetOauthAuthorizationServer")
@route("/.well-known/oauth-authorization-server")
@get
op GetOauthAuthorizationServer(

): GetOauthAuthorizationServerSuccess_200;

model GetProtectedResourceMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.ProtectedResourceMetadata;
}

union GetProtectedResourceMetadataError400Body {
  IdMagic.Contract.OAuthInvalidRequestError,
  IdMagic.Contract.InvalidTargetError,
}

model GetProtectedResourceMetadataError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: GetProtectedResourceMetadataError400Body;
}

@doc("Protected Resource Metadata (RFC 9728). Without a `resource` it returns the realm's IdMagic API and its ApiTokenScopes; with one it returns the metadata of the registered, Active McpResourceServer. A resource that is unregistered or disabled is refused as `invalid_target`.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetProtectedResourceMetadata")
@route("/.well-known/oauth-protected-resource")
@get
op GetProtectedResourceMetadata(
  @query resource?: string
): GetProtectedResourceMetadataSuccess_200 | GetProtectedResourceMetadataError400;

model HealthHttpResponse {
  @doc("Always `ok`.")
  status: string;
  @doc("The persistence backend the process started with.")
  persistence: string;
  @doc("The observability backend the process started with.")
  observability: string;
  @doc("The AuthZEN backend the process started with.")
  authzen: string;
  @doc("Versioned metadata for the runtime-selectable features enabled for this process.")
  features: IdMagic.Contract.FeatureRuntimeMetadata;
}

model HealthSuccess_200 {
  @statusCode statusCode: 200;
  @body body: HealthHttpResponse;
}

@doc("Health check")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("Health")
@route("/health")
@get
op Health(

): HealthSuccess_200;

model ListAuthorizationDetailTypesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuthorizationDetailTypeListResponse;
}

union ListAuthorizationDetailTypesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAuthorizationDetailTypesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAuthorizationDetailTypesError403Body;
}

@doc("An administrator lists the `authorization_details` types of their tenant (RFC 9396).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["authorization-detail-types:read"])
@TypeSpec.OpenAPI.operationId("ListAuthorizationDetailTypes")
@route("/api/admin/v1/authorization-detail-types")
@get
op ListAuthorizationDetailTypes(

): ListAuthorizationDetailTypesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAuthorizationDetailTypesError403;

model GetAuthorizationDetailTypeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuthorizationDetailTypeResponse;
}

union GetAuthorizationDetailTypeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAuthorizationDetailTypeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAuthorizationDetailTypeError403Body;
}

union GetAuthorizationDetailTypeError422Body {
  IdMagic.Contract.InvalidAuthorizationDetailTypeError,
}

model GetAuthorizationDetailTypeError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: GetAuthorizationDetailTypeError422Body;
}

union GetAuthorizationDetailTypeError404Body {
  IdMagic.Contract.AuthorizationDetailTypeNotFoundError,
}

model GetAuthorizationDetailTypeError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAuthorizationDetailTypeError404Body;
}

@doc("An administrator reads one type. A type of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["authorization-detail-types:read"])
@TypeSpec.OpenAPI.operationId("GetAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types/{type}")
@get
op GetAuthorizationDetailType(
  @path type: string
): GetAuthorizationDetailTypeSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAuthorizationDetailTypeError403 | GetAuthorizationDetailTypeError404 | GetAuthorizationDetailTypeError422;

model CreateAuthorizationDetailTypeSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.AuthorizationDetailTypeResponse;
}

union CreateAuthorizationDetailTypeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateAuthorizationDetailTypeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateAuthorizationDetailTypeError400Body;
}

union CreateAuthorizationDetailTypeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateAuthorizationDetailTypeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAuthorizationDetailTypeError403Body;
}

union CreateAuthorizationDetailTypeError422Body {
  IdMagic.Contract.InvalidAuthorizationDetailTypeError,
}

model CreateAuthorizationDetailTypeError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateAuthorizationDetailTypeError422Body;
}

union CreateAuthorizationDetailTypeError409Body {
  IdMagic.Contract.AuthorizationDetailTypeExistsError,
}

model CreateAuthorizationDetailTypeError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CreateAuthorizationDetailTypeError409Body;
}

@doc("An administrator registers an `authorization_details` type in their tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["authorization-detail-types:write"])
@TypeSpec.OpenAPI.operationId("CreateAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types")
@post
op CreateAuthorizationDetailType(
  @body body: IdMagic.Contract.AuthorizationDetailTypeRequest
): CreateAuthorizationDetailTypeSuccess_201 | CreateAuthorizationDetailTypeError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAuthorizationDetailTypeError403 | CreateAuthorizationDetailTypeError409 | CreateAuthorizationDetailTypeError422;

model UpdateAuthorizationDetailTypeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuthorizationDetailTypeResponse;
}

union UpdateAuthorizationDetailTypeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAuthorizationDetailTypeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAuthorizationDetailTypeError400Body;
}

union UpdateAuthorizationDetailTypeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAuthorizationDetailTypeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAuthorizationDetailTypeError403Body;
}

union UpdateAuthorizationDetailTypeError422Body {
  IdMagic.Contract.InvalidAuthorizationDetailTypeError,
}

model UpdateAuthorizationDetailTypeError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: UpdateAuthorizationDetailTypeError422Body;
}

union UpdateAuthorizationDetailTypeError404Body {
  IdMagic.Contract.AuthorizationDetailTypeNotFoundError,
}

model UpdateAuthorizationDetailTypeError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateAuthorizationDetailTypeError404Body;
}

@doc("An administrator updates the schema, display template, description, or state of a type.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["authorization-detail-types:write"])
@TypeSpec.OpenAPI.operationId("UpdateAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types/{type}")
@patch
op UpdateAuthorizationDetailType(
  @path type: string,
  @body body: IdMagic.Contract.UpdateAuthorizationDetailTypeRequest
): UpdateAuthorizationDetailTypeSuccess_200 | UpdateAuthorizationDetailTypeError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAuthorizationDetailTypeError403 | UpdateAuthorizationDetailTypeError404 | UpdateAuthorizationDetailTypeError422;

model DeleteAuthorizationDetailTypeSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAuthorizationDetailTypeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAuthorizationDetailTypeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAuthorizationDetailTypeError403Body;
}

@doc("An administrator deletes a type. Tokens already issued stay valid until they expire.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["authorization-detail-types:write"])
@TypeSpec.OpenAPI.operationId("DeleteAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types/{type}")
@delete
op DeleteAuthorizationDetailType(
  @path type: string
): DeleteAuthorizationDetailTypeSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAuthorizationDetailTypeError403;

model ListAdminOAuth2ClientsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminOAuth2ClientListResponse;
}

union ListAdminOAuth2ClientsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminOAuth2ClientsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminOAuth2ClientsError403Body;
}

union ListAdminOAuth2ClientsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAdminOAuth2ClientsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAdminOAuth2ClientsError400Body;
}

@doc("An administrator lists the clients of their tenant by `client_id` ascending, through bidirectional keyset pagination.\nThe cursor comes from the Link response header of the response (`rel=\"prev\"` and `rel=\"next\"`).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:read"])
@TypeSpec.OpenAPI.operationId("ListAdminOAuth2Clients")
@route("/api/admin/v1/clients")
@get
op ListAdminOAuth2Clients(
  @query cursor?: string,
  @query limit?: int64
): ListAdminOAuth2ClientsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminOAuth2ClientsError403 | ListAdminOAuth2ClientsError400;

model GetAdminOAuth2ClientSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminOAuth2ClientResponse;
}

union GetAdminOAuth2ClientError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminOAuth2ClientError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminOAuth2ClientError400Body;
}

union GetAdminOAuth2ClientError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminOAuth2ClientError403Body;
}

@doc("An administrator reads a client of their tenant. A client of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:read"])
@TypeSpec.OpenAPI.operationId("GetAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@get
op GetAdminOAuth2Client(
  @path client_id: string
): GetAdminOAuth2ClientSuccess_200 | GetAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminOAuth2ClientError403;

model CreateAdminOAuth2ClientSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.AdminOAuth2ClientCreateResponse;
}

union CreateAdminOAuth2ClientError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateAdminOAuth2ClientError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateAdminOAuth2ClientError400Body;
}

union CreateAdminOAuth2ClientError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAdminOAuth2ClientError403Body;
}

union CreateAdminOAuth2ClientError422Body {
  IdMagic.Contract.QuotaExceededError,
}

model CreateAdminOAuth2ClientError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CreateAdminOAuth2ClientError422Body;
}

@doc("An administrator creates a client in their tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminOAuth2Client")
@route("/api/admin/v1/clients")
@post
op CreateAdminOAuth2Client(
  @body body: IdMagic.Contract.AdminOAuth2ClientCreateRequest
): CreateAdminOAuth2ClientSuccess_201 | CreateAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAdminOAuth2ClientError403 | CreateAdminOAuth2ClientError422;

model UpdateAdminOAuth2ClientSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminOAuth2ClientResponse;
}

union UpdateAdminOAuth2ClientError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAdminOAuth2ClientError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminOAuth2ClientError400Body;
}

union UpdateAdminOAuth2ClientError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminOAuth2ClientError403Body;
}

@doc("An administrator updates the client metadata of their tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@patch
op UpdateAdminOAuth2Client(
  @path client_id: string,
  @body body: IdMagic.Contract.AdminOAuth2ClientUpdateRequest
): UpdateAdminOAuth2ClientSuccess_200 | UpdateAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminOAuth2ClientError403;

model DeleteAdminOAuth2ClientSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAdminOAuth2ClientError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAdminOAuth2ClientError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminOAuth2ClientError400Body;
}

union DeleteAdminOAuth2ClientError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminOAuth2ClientError403Body;
}

union DeleteAdminOAuth2ClientError409Body {
  IdMagic.Contract.ApplicationOwnedProtocolError,
}

model DeleteAdminOAuth2ClientError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminOAuth2ClientError409Body;
}

@doc("An administrator deletes a client of their tenant that sits outside the catalogue. A client whose `application_id` is not null is refused with ApplicationOwnedProtocolError (HTTP 409) and the caller is directed to delete the Application instead.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@delete
op DeleteAdminOAuth2Client(
  @path client_id: string
): DeleteAdminOAuth2ClientSuccess_204 | DeleteAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAdminOAuth2ClientError403 | DeleteAdminOAuth2ClientError409;

model ListAdminMcpResourceServersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.McpResourceServerListResponse;
}

union ListAdminMcpResourceServersError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminMcpResourceServersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminMcpResourceServersError403Body;
}

@doc("An administrator lists the McpResourceServers of their tenant (AdminMcpResourcesManage).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["mcp-resource-servers:read"])
@TypeSpec.OpenAPI.operationId("ListAdminMcpResourceServers")
@route("/api/admin/v1/mcp-resource-servers")
@get
op ListAdminMcpResourceServers(

): ListAdminMcpResourceServersSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminMcpResourceServersError403;

model GetAdminMcpResourceServerSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.McpResourceServerResponse;
}

union GetAdminMcpResourceServerError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminMcpResourceServerError403Body;
}

union GetAdminMcpResourceServerError404Body {
  IdMagic.Contract.McpResourceServerNotFoundError,
}

model GetAdminMcpResourceServerError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminMcpResourceServerError404Body;
}

@doc("An administrator reads an McpResourceServer of their tenant. A registration of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["mcp-resource-servers:read"])
@TypeSpec.OpenAPI.operationId("GetAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers/{id}")
@get
op GetAdminMcpResourceServer(
  @path id: string
): GetAdminMcpResourceServerSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminMcpResourceServerError403 | GetAdminMcpResourceServerError404;

model CreateAdminMcpResourceServerSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.McpResourceServerResponse;
}

union CreateAdminMcpResourceServerError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateAdminMcpResourceServerError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateAdminMcpResourceServerError400Body;
}

union CreateAdminMcpResourceServerError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateAdminMcpResourceServerError403Body;
}

union CreateAdminMcpResourceServerError409Body {
  IdMagic.Contract.McpResourceServerExistsError,
}

model CreateAdminMcpResourceServerError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: CreateAdminMcpResourceServerError409Body;
}

@doc("An administrator registers an McpResourceServer in their tenant. The `resource` must be unique within the tenant.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["mcp-resource-servers:write"])
@TypeSpec.OpenAPI.operationId("CreateAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers")
@post
op CreateAdminMcpResourceServer(
  @body body: IdMagic.Contract.McpResourceServerRequest
): CreateAdminMcpResourceServerSuccess_201 | CreateAdminMcpResourceServerError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateAdminMcpResourceServerError403 | CreateAdminMcpResourceServerError409;

model UpdateAdminMcpResourceServerSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.McpResourceServerResponse;
}

union UpdateAdminMcpResourceServerError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateAdminMcpResourceServerError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminMcpResourceServerError400Body;
}

union UpdateAdminMcpResourceServerError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminMcpResourceServerError403Body;
}

union UpdateAdminMcpResourceServerError404Body {
  IdMagic.Contract.McpResourceServerNotFoundError,
}

model UpdateAdminMcpResourceServerError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminMcpResourceServerError404Body;
}

@doc("An administrator updates the name, scopes, or state of an McpResourceServer. The `resource`, its canonical URI, is immutable.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["mcp-resource-servers:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers/{id}")
@patch
op UpdateAdminMcpResourceServer(
  @path id: string,
  @body body: IdMagic.Contract.McpResourceServerRequest
): UpdateAdminMcpResourceServerSuccess_200 | UpdateAdminMcpResourceServerError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminMcpResourceServerError403 | UpdateAdminMcpResourceServerError404;

model DeleteAdminMcpResourceServerSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAdminMcpResourceServerError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminMcpResourceServerError403Body;
}

@doc("An administrator deletes an McpResourceServer. A new issuance naming that resource is refused with `invalid_target` afterwards; tokens already issued stay valid until they expire.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["mcp-resource-servers:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers/{id}")
@delete
op DeleteAdminMcpResourceServer(
  @path id: string
): DeleteAdminMcpResourceServerSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAdminMcpResourceServerError403;

model ListAdminConsentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminConsentListResponse;
}

union ListAdminConsentsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminConsentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminConsentsError403Body;
}

union ListAdminConsentsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAdminConsentsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAdminConsentsError400Body;
}

@doc("An administrator lists the Consents of their tenant for audit, by `user_id` and `client_id` ascending, through bidirectional\nkeyset pagination. The cursor comes from the Link response header of the response (`rel=\"prev\"` and `rel=\"next\"`).")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["consents:read"])
@TypeSpec.OpenAPI.operationId("ListAdminConsents")
@route("/api/admin/v1/consents")
@get
op ListAdminConsents(
  @query cursor?: string,
  @query limit?: int64
): ListAdminConsentsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminConsentsError403 | ListAdminConsentsError400;

model GetAdminConsentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AdminConsentResponse;
}

union GetAdminConsentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminConsentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminConsentError400Body;
}

union GetAdminConsentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminConsentError403Body;
}

@doc("An administrator reads a Consent of their tenant by subject and `client_id`. A Consent of another tenant is treated as nonexistent.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["consents:read"])
@TypeSpec.OpenAPI.operationId("GetAdminConsent")
@route("/api/admin/v1/consents/{user_id}/{client_id}")
@get
op GetAdminConsent(
  @path user_id: string,
  @path client_id: string
): GetAdminConsentSuccess_200 | GetAdminConsentError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminConsentError403;

model RevokeAdminConsentSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeAdminConsentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeAdminConsentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeAdminConsentError400Body;
}

union RevokeAdminConsentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeAdminConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeAdminConsentError403Body;
}

@doc("An administrator revokes a Consent of their tenant logically. Creating a Consent, or widening its scope, is not permitted to an administrator.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["consents:write"])
@TypeSpec.OpenAPI.operationId("RevokeAdminConsent")
@route("/api/admin/v1/consents/{user_id}/{client_id}")
@delete
op RevokeAdminConsent(
  @path user_id: string,
  @path client_id: string
): RevokeAdminConsentSuccess_204 | RevokeAdminConsentError400 | IdMagic.Contract.AuthenticationRequiredResponse | RevokeAdminConsentError403;

model ListAdminRolePoliciesHttpResponse {
  roles: IdMagic.Contract.AdminRolePolicy[];
}

model ListAdminRolePoliciesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminRolePoliciesHttpResponse;
}

union ListAdminRolePoliciesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminRolePoliciesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminRolePoliciesError403Body;
}

@doc("An administrator reads how the roles, permissions, and HTTP interfaces visible to them line up.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["settings:read"])
@TypeSpec.OpenAPI.operationId("ListAdminRolePolicies")
@route("/api/admin/v1/policy/roles")
@get
op ListAdminRolePolicies(

): ListAdminRolePoliciesSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAdminRolePoliciesError403;

model ListMyConsentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountConsentListResponse;
}

union ListMyConsentsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMyConsentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMyConsentsError403Body;
}

@doc("An authenticated user reads the list of applications they are connected to, that is, their active Consents (self-service).\nThe target is fixed to `actor.id`, and another person's Consents are never returned.")
@TypeSpec.OpenAPI.operationId("ListMyConsents")
@route("/api/account/v1/consents")
@get
op ListMyConsents(

): ListMyConsentsSuccess_200 | ListMyConsentsError403;

model RevokeMyConsentSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMyConsentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeMyConsentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeMyConsentError400Body;
}

union RevokeMyConsentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyConsentError403Body;
}

@doc("An authenticated user revokes their own Consent for a named `client_id`, logically (self-service).\n`actor.id` is fixed equal to `target.id`. The next authorization for that client asks for consent again.")
@TypeSpec.OpenAPI.operationId("RevokeMyConsent")
@route("/api/account/v1/consents/{client_id}/revoke")
@post
op RevokeMyConsent(
  @path client_id: string,): RevokeMyConsentSuccess_204 | RevokeMyConsentError400 | RevokeMyConsentError403;

model ListMyApprovalRequestsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountApprovalRequestListResponse;
}

union ListMyApprovalRequestsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMyApprovalRequestsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMyApprovalRequestsError403Body;
}

@doc("An authenticated user reads the approval requests pending for them (self-service). The target is fixed to `actor.id`, so\nrequests addressed to another person are never returned, and expired requests are not returned either.")
@TypeSpec.OpenAPI.operationId("ListMyApprovalRequests")
@route("/api/account/v1/approval-requests")
@get
op ListMyApprovalRequests(

): ListMyApprovalRequestsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListMyApprovalRequestsError403;

model DecideMyApprovalRequestSuccess_204 {
  @statusCode statusCode: 204;
}

union DecideMyApprovalRequestError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DecideMyApprovalRequestError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DecideMyApprovalRequestError400Body;
}

union DecideMyApprovalRequestError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DecideMyApprovalRequestError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DecideMyApprovalRequestError403Body;
}

@doc("An authenticated user approves or refuses one approval request addressed to them (self-service). Step-up reauthentication and\nCSRF verification are both required, and `actor.id` is fixed to the User the approval is about. Deciding a request addressed to\nanother person, one in a terminal state, or an expired one is refused.")
@TypeSpec.OpenAPI.operationId("DecideMyApprovalRequest")
@route("/api/account/v1/approval-requests/{id}/decision")
@post
op DecideMyApprovalRequest(
  @path id: string,
  @body body: IdMagic.Contract.AccountApprovalDecisionRequest
): DecideMyApprovalRequestSuccess_204 | DecideMyApprovalRequestError400 | IdMagic.Contract.AuthenticationRequiredResponse | DecideMyApprovalRequestError403;
}

}
