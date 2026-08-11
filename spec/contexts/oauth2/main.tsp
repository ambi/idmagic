import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("OAuth 2.0 and OpenID Connect")
namespace IdMagic.OAuth2 {

namespace Operations {
model RegisterClientHttpRequest {
  request: IdMagic.Contract.ClientRegistrationRequest;
}

model RegisterClientHttpResponse {
  client: IdMagic.Contract.OAuth2Client;
}

model RegisterClientSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RegisterClientHttpResponse;
}

union RegisterClientError400Body {
  IdMagic.Contract.InvalidRequestError,
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
  @header contentType: "application/json";
  @body body: RegisterClientError422Body;
}

@doc("Dynamic Client Registration (RFC 7591)")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("RegisterClient")
@route("/register")
@post
op RegisterClient(
  @body body: RegisterClientHttpRequest
): RegisterClientSuccess_200 | RegisterClientError400 | RegisterClientError422;

model AuthorizeSuccess_302 {
  @statusCode statusCode: 302;
}

union AuthorizeError400Body {
  IdMagic.Contract.InvalidRequestError,
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

model AuthorizeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/json";
  @body body: AuthorizeError403Body;
}

union AuthorizeError429Body {
  IdMagic.Contract.RateLimitedError,
}

model AuthorizeError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @body body: AuthorizeError429Body;
}

@doc("Authorization endpoint (RFC 6749 §4.1.1)。client_id は登録済み Client、または Client ID Metadata Document の URL を都度解決したものを受理する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("Authorize")
@route("/authorize")
@get
op Authorize(
  @query params: IdMagic.Contract.AuthorizeParameters
): AuthorizeSuccess_302 | AuthorizeError400 | AuthorizeError401 | AuthorizeError403 | AuthorizeError429;

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

@doc("Authentication context の inbound federation callback が認証済み session を確立した後、保存済み authorization transaction を再開して通常の application policy・consent・required-action gate を適用する。")
@TypeSpec.OpenAPI.operationId("ResumeFederatedAuthorization")
@route("/authorize/resume")
@get
op ResumeFederatedAuthorization(

): ResumeFederatedAuthorizationSuccess_302 | ResumeFederatedAuthorizationError400 | ResumeFederatedAuthorizationError403;

model GetBrowserTransactionHttpResponse {
  response: IdMagic.Contract.BrowserTransactionResponse;
}

model GetBrowserTransactionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetBrowserTransactionHttpResponse;
}

union GetBrowserTransactionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetBrowserTransactionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetBrowserTransactionError400Body;
}

@doc("SPA が HttpOnly transaction cookie に対応するログイン・同意コンテキストを取得する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetBrowserTransaction")
@route("/api/auth/transaction")
@get
op GetBrowserTransaction(

): GetBrowserTransactionSuccess_200 | GetBrowserTransactionError400;

model SubmitBrowserConsentHttpRequest {
  request: IdMagic.Contract.BrowserConsentRequest;
}

model SubmitBrowserConsentHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model SubmitBrowserConsentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SubmitBrowserConsentHttpResponse;
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

@doc("SPA JSON API から authorization request の同意・拒否を確定する。")
@TypeSpec.OpenAPI.operationId("SubmitBrowserConsent")
@route("/api/auth/consent")
@post
op SubmitBrowserConsent(
  @body body: SubmitBrowserConsentHttpRequest
): SubmitBrowserConsentSuccess_200 | SubmitBrowserConsentError400 | SubmitBrowserConsentError403 | SubmitBrowserConsentError422;

model SubmitBrowserDeviceHttpRequest {
  request: IdMagic.Contract.BrowserDeviceRequest;
}

model SubmitBrowserDeviceHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model SubmitBrowserDeviceSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SubmitBrowserDeviceHttpResponse;
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

@doc("SPA JSON API から device user_code を承認または拒否する。")
@TypeSpec.OpenAPI.operationId("SubmitBrowserDevice")
@route("/api/auth/device")
@post
op SubmitBrowserDevice(
  @body body: SubmitBrowserDeviceHttpRequest
): SubmitBrowserDeviceSuccess_200 | SubmitBrowserDeviceError400 | SubmitBrowserDeviceError403;

model GetBrowserDeviceContextHttpResponse {
  response: IdMagic.Contract.BrowserDeviceContextResponse;
}

model GetBrowserDeviceContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetBrowserDeviceContextHttpResponse;
}

@doc("SPA が device user_code と CSRF token を取得する。")
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
  IdMagic.Contract.InvalidRequestError,
}

model EndSession1Error400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: EndSession1Error400Body;
}

@doc("RP-Initiated Logout endpoint (OIDC RP-Initiated Logout 1.0)。redirect 検証は\nOAuth2/OIDC bounded context が行い、session revocation は Authentication bounded\ncontext の SessionManager に委譲する。id_token_hint が与えられれば署名・iss・aud・sub・\nsid を検証して対象 session と client を hint から解決し、client_id と\n矛盾する hint は拒否する。id_token_hint が無ければ client_id + browser cookie による\n既存の解決方法を fallback として維持する。ローカル revoke 確定後、対象 sid に紐づく\nClientSession から前段の FrontChannelLogout target 一覧を応答へ含め、\nBackChannelLogout 配送を非同期に起動する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("EndSession")
@route("/end_session")
@get
op EndSession1(
  @query params: IdMagic.Contract.EndSessionParameters
): EndSession1Success_200 | EndSession1Success_302 | EndSession1Error400;

model EndSession2HttpRequest {
  params: IdMagic.Contract.EndSessionParameters;
}

model EndSession2Success_200 {
  @statusCode statusCode: 200;
}

model EndSession2Success_302 {
  @statusCode statusCode: 302;
}

union EndSession2Error400Body {
  IdMagic.Contract.InvalidRequestError,
}

model EndSession2Error400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: EndSession2Error400Body;
}

@doc("RP-Initiated Logout endpoint (OIDC RP-Initiated Logout 1.0)。redirect 検証は\nOAuth2/OIDC bounded context が行い、session revocation は Authentication bounded\ncontext の SessionManager に委譲する。id_token_hint が与えられれば署名・iss・aud・sub・\nsid を検証して対象 session と client を hint から解決し、client_id と\n矛盾する hint は拒否する。id_token_hint が無ければ client_id + browser cookie による\n既存の解決方法を fallback として維持する。ローカル revoke 確定後、対象 sid に紐づく\nClientSession から前段の FrontChannelLogout target 一覧を応答へ含め、\nBackChannelLogout 配送を非同期に起動する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("EndSession")
@route("/end_session")
@post
op EndSession2(
  @body body: EndSession2HttpRequest
): EndSession2Success_200 | EndSession2Success_302 | EndSession2Error400;

model CheckSessionIframeSuccess_200 {
  @statusCode statusCode: 200;
}

@doc("OP session の変化をポーリングする RP 用 hidden iframe (OIDC Session Management\n1.0、Draft のまま Final に達していない仕様のため adoption は optional)。\n静的な HTML/JS を返し、RP からの postMessage に対し現在の browser cookie が有効な\nLoginSession に解決できるかどうかだけを応答する。session_state の salted hash 比較\nなど仕様の詳細な相関アルゴリズムは実装しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CheckSessionIframe")
@route("/session/check")
@get
op CheckSessionIframe(

): CheckSessionIframeSuccess_200;

model PushAuthorizationRequestHttpRequest {
  request: IdMagic.Contract.ParRequest;
}

model PushAuthorizationRequestHttpResponse {
  response: IdMagic.Contract.ParResponse;
}

model PushAuthorizationRequestSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PushAuthorizationRequestHttpResponse;
}

union PushAuthorizationRequestError400Body {
  IdMagic.Contract.InvalidRequestError,
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
  @body body: PushAuthorizationRequestError429Body;
}

@doc("Pushed Authorization Request (RFC 9126)。client_id は登録済み Client、または Client ID Metadata Document の URL を都度解決したものを受理する。")
@TypeSpec.OpenAPI.operationId("PushAuthorizationRequest")
@route("/par")
@post
op PushAuthorizationRequest(
  @body body: PushAuthorizationRequestHttpRequest
): PushAuthorizationRequestSuccess_200 | PushAuthorizationRequestError400 | PushAuthorizationRequestError401 | PushAuthorizationRequestError429;

model TokenHttpRequest {
  request: IdMagic.Contract.TokenRequest;
}

model TokenHttpResponse {
  response: IdMagic.Contract.TokenResponse;
}

model TokenSuccess_200 {
  @statusCode statusCode: 200;
  @body body: TokenHttpResponse;
}

union TokenError400Body {
  IdMagic.Contract.InvalidRequestError,
  IdMagic.Contract.InvalidGrantError,
  IdMagic.Contract.UnsupportedGrantTypeError,
  IdMagic.Contract.InvalidScopeError,
  IdMagic.Contract.AuthorizationPendingError,
  IdMagic.Contract.SlowDownError,
  IdMagic.Contract.ExpiredTokenError,
  IdMagic.Contract.InvalidTargetError,
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

union TokenError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model TokenError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/json";
  @body body: TokenError403Body;
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
  @body body: TokenError429Body;
}

@doc("Token endpoint (RFC 6749 §3.2 + 9068 + 9449)")
@TypeSpec.OpenAPI.operationId("Token")
@route("/token")
@post
op Token(
  @body body: TokenHttpRequest
): TokenSuccess_200 | TokenError400 | TokenError401 | TokenError403 | TokenError422 | TokenError429;

model IntrospectHttpRequest {
  request: IdMagic.Contract.IntrospectionRequest;
}

model IntrospectHttpResponse {
  response: IdMagic.Contract.IntrospectionResponse;
}

model IntrospectSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IntrospectHttpResponse;
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

@doc("通常 OAuth token と管理発行 JWT access token の Token Introspection (RFC 7662)。Agent 主体の token は SharedSignals の revocation epoch (kill-switch・所有者オフボード・CAEP/SSF 経由の失効) と issued_at を比較し、epoch 以前に発行された token は fail-closed で active=false とする。")
@TypeSpec.OpenAPI.operationId("Introspect")
@route("/introspect")
@post
op Introspect(
  @body body: IntrospectHttpRequest
): IntrospectSuccess_200 | IntrospectError401;

model RevokeHttpRequest {
  request: IdMagic.Contract.RevokeRequest;
}

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

@doc("通常 OAuth token と管理発行 JWT access token の Token Revocation (RFC 7009)")
@TypeSpec.OpenAPI.operationId("Revoke")
@route("/revoke")
@post
op Revoke(
  @body body: RevokeHttpRequest
): RevokeSuccess_200 | RevokeError401;

model UserInfoHttpResponse {
  response: IdMagic.Contract.UserInfoResponse;
}

model UserInfoSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UserInfoHttpResponse;
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

@doc("UserInfo (OIDC Core §5.3)。additional_claims は scope 主導の ClaimsForScopes と\nclient 単位の ClaimMapping claim_policy をマージする。")
@TypeSpec.OpenAPI.operationId("UserInfo")
@route("/userinfo")
@get
op UserInfo(

): UserInfoSuccess_200 | UserInfoError401 | UserInfoError403 | UserInfoError422;

model PostUserInfoHttpResponse {
  response: IdMagic.Contract.UserInfoResponse;
}

model PostUserInfoSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PostUserInfoHttpResponse;
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

@doc("UserInfo の POST binding (OIDC Core §5.3)")
@TypeSpec.OpenAPI.operationId("PostUserInfo")
@route("/userinfo")
@post
op PostUserInfo(

): PostUserInfoSuccess_200 | PostUserInfoError401 | PostUserInfoError403 | PostUserInfoError422;

model DeviceAuthorizationHttpRequest {
  request: IdMagic.Contract.DeviceAuthorizationRequest;
}

model DeviceAuthorizationHttpResponse {
  response: IdMagic.Contract.DeviceAuthorizationResponse;
}

model DeviceAuthorizationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DeviceAuthorizationHttpResponse;
}

union DeviceAuthorizationError400Body {
  IdMagic.Contract.InvalidRequestError,
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
  @body body: DeviceAuthorizationError429Body;
}

@doc("Device Authorization Request (RFC 8628 §3.1)")
@TypeSpec.OpenAPI.operationId("DeviceAuthorization")
@route("/device_authorization")
@post
op DeviceAuthorization(
  @body body: DeviceAuthorizationHttpRequest
): DeviceAuthorizationSuccess_200 | DeviceAuthorizationError400 | DeviceAuthorizationError401 | DeviceAuthorizationError429;

model GetOpenidConfigurationHttpResponse {
  discovery: IdMagic.Contract.DiscoveryDocument;
}

model GetOpenidConfigurationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetOpenidConfigurationHttpResponse;
}

@doc("OIDC Discovery")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetOpenidConfiguration")
@route("/.well-known/openid-configuration")
@get
op GetOpenidConfiguration(

): GetOpenidConfigurationSuccess_200;

model GetOauthAuthorizationServerHttpResponse {
  discovery: IdMagic.Contract.DiscoveryDocument;
}

model GetOauthAuthorizationServerSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetOauthAuthorizationServerHttpResponse;
}

@doc("OAuth 2.0 Authorization Server Metadata (RFC 8414)")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetOauthAuthorizationServer")
@route("/.well-known/oauth-authorization-server")
@get
op GetOauthAuthorizationServer(

): GetOauthAuthorizationServerSuccess_200;

model GetProtectedResourceMetadataHttpResponse {
  metadata: IdMagic.Contract.ProtectedResourceMetadata;
}

model GetProtectedResourceMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetProtectedResourceMetadataHttpResponse;
}

union GetProtectedResourceMetadataError400Body {
  IdMagic.Contract.InvalidRequestError,
  IdMagic.Contract.InvalidTargetError,
}

model GetProtectedResourceMetadataError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/json";
  @body body: GetProtectedResourceMetadataError400Body;
}

@doc("Protected Resource Metadata (RFC 9728)。resource 未指定時は realm の IdMagic API と\nApiTokenScope を、resource 指定時は登録済み Active な McpResourceServer の metadata を返す。\n未登録・Disabled は invalid_target として拒否する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetProtectedResourceMetadata")
@route("/.well-known/oauth-protected-resource")
@get
op GetProtectedResourceMetadata(
  @query resource?: string
): GetProtectedResourceMetadataSuccess_200 | GetProtectedResourceMetadataError400;

model HealthSuccess_200 {
  @statusCode statusCode: 200;
}

@doc("Health check")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("Health")
@route("/health")
@get
op Health(

): HealthSuccess_200;

model ListAuthorizationDetailTypesHttpResponse {
  response: IdMagic.Contract.AuthorizationDetailTypeListResponse;
}

model ListAuthorizationDetailTypesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAuthorizationDetailTypesHttpResponse;
}

union ListAuthorizationDetailTypesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAuthorizationDetailTypesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAuthorizationDetailTypesError403Body;
}

@doc("管理者が所属テナントの authorization_details type を一覧する (RFC 9396)。")
@TypeSpec.OpenAPI.operationId("ListAuthorizationDetailTypes")
@route("/api/admin/v1/authorization-detail-types")
@get
op ListAuthorizationDetailTypes(

): ListAuthorizationDetailTypesSuccess_200 | ListAuthorizationDetailTypesError403;

model GetAuthorizationDetailTypeHttpResponse {
  response: IdMagic.Contract.AuthorizationDetailTypeResponse;
}

model GetAuthorizationDetailTypeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAuthorizationDetailTypeHttpResponse;
}

union GetAuthorizationDetailTypeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAuthorizationDetailTypeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAuthorizationDetailTypeError400Body;
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

@doc("管理者が単一 type を取得する。別テナントの type は未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types/{type}")
@get
op GetAuthorizationDetailType(
  @path type: string
): GetAuthorizationDetailTypeSuccess_200 | GetAuthorizationDetailTypeError400 | GetAuthorizationDetailTypeError403 | GetAuthorizationDetailTypeError422;

model CreateAuthorizationDetailTypeHttpRequest {
  request: IdMagic.Contract.AuthorizationDetailTypeRequest;
}

model CreateAuthorizationDetailTypeHttpResponse {
  response: IdMagic.Contract.AuthorizationDetailTypeResponse;
}

model CreateAuthorizationDetailTypeSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateAuthorizationDetailTypeHttpResponse;
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

@doc("管理者が所属テナントに authorization_details type を登録する。")
@TypeSpec.OpenAPI.operationId("CreateAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types")
@post
op CreateAuthorizationDetailType(
  @body body: CreateAuthorizationDetailTypeHttpRequest
): CreateAuthorizationDetailTypeSuccess_201 | CreateAuthorizationDetailTypeError400 | CreateAuthorizationDetailTypeError403 | CreateAuthorizationDetailTypeError422;

model UpdateAuthorizationDetailTypeHttpRequest {
  type: string;
  request: IdMagic.Contract.AuthorizationDetailTypeRequest;
}

model UpdateAuthorizationDetailTypeHttpResponse {
  response: IdMagic.Contract.AuthorizationDetailTypeResponse;
}

model UpdateAuthorizationDetailTypeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAuthorizationDetailTypeHttpResponse;
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

@doc("管理者が type の schema / display_template / description / state を更新する。")
@TypeSpec.OpenAPI.operationId("UpdateAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types/{type}")
@patch
op UpdateAuthorizationDetailType(
  @path type: string,
  @body body: UpdateAuthorizationDetailTypeHttpRequest
): UpdateAuthorizationDetailTypeSuccess_200 | UpdateAuthorizationDetailTypeError400 | UpdateAuthorizationDetailTypeError403 | UpdateAuthorizationDetailTypeError422;

model DeleteAuthorizationDetailTypeSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAuthorizationDetailTypeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAuthorizationDetailTypeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAuthorizationDetailTypeError400Body;
}

union DeleteAuthorizationDetailTypeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAuthorizationDetailTypeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAuthorizationDetailTypeError403Body;
}

@doc("管理者が type を削除する。発行済みトークンは失効まで有効。")
@TypeSpec.OpenAPI.operationId("DeleteAuthorizationDetailType")
@route("/api/admin/v1/authorization-detail-types/{type}")
@delete
op DeleteAuthorizationDetailType(
  @path type: string
): DeleteAuthorizationDetailTypeSuccess_204 | DeleteAuthorizationDetailTypeError400 | DeleteAuthorizationDetailTypeError403;

model ListAdminOAuth2ClientsHttpResponse {
  response: IdMagic.Contract.AdminOAuth2ClientListResponse;
}

model ListAdminOAuth2ClientsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminOAuth2ClientsHttpResponse;
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

@doc("管理者が所属テナントの client を client_id 昇順の双方向 keyset pagination で一覧する。\ncursor は応答の Link response header (rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.operationId("ListAdminOAuth2Clients")
@route("/api/admin/v1/clients")
@get
op ListAdminOAuth2Clients(
  @query cursor?: string,
  @query limit?: int64
): ListAdminOAuth2ClientsSuccess_200 | ListAdminOAuth2ClientsError403 | ListAdminOAuth2ClientsError400;

model GetAdminOAuth2ClientHttpResponse {
  client: IdMagic.Contract.AdminOAuth2ClientResponse;
}

model GetAdminOAuth2ClientSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminOAuth2ClientHttpResponse;
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

@doc("管理者が所属テナントの client を取得する。別テナントの client は未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@get
op GetAdminOAuth2Client(
  @path client_id: string
): GetAdminOAuth2ClientSuccess_200 | GetAdminOAuth2ClientError400 | GetAdminOAuth2ClientError403;

model CreateAdminOAuth2ClientHttpRequest {
  request: IdMagic.Contract.AdminOAuth2ClientCreateRequest;
}

model CreateAdminOAuth2ClientHttpResponse {
  response: IdMagic.Contract.AdminOAuth2ClientCreateResponse;
}

model CreateAdminOAuth2ClientSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateAdminOAuth2ClientHttpResponse;
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

@doc("管理者が所属テナントに client を作成する。")
@TypeSpec.OpenAPI.operationId("CreateAdminOAuth2Client")
@route("/api/admin/v1/clients")
@post
op CreateAdminOAuth2Client(
  @body body: CreateAdminOAuth2ClientHttpRequest
): CreateAdminOAuth2ClientSuccess_201 | CreateAdminOAuth2ClientError400 | CreateAdminOAuth2ClientError403 | CreateAdminOAuth2ClientError422;

model UpdateAdminOAuth2ClientHttpRequest {
  client_id: string;
  request: IdMagic.Contract.AdminOAuth2ClientUpdateRequest;
}

model UpdateAdminOAuth2ClientHttpResponse {
  client: IdMagic.Contract.AdminOAuth2ClientResponse;
}

model UpdateAdminOAuth2ClientSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAdminOAuth2ClientHttpResponse;
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

@doc("管理者が所属テナントの client metadata を更新する。")
@TypeSpec.OpenAPI.operationId("UpdateAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@patch
op UpdateAdminOAuth2Client(
  @path client_id: string,
  @body body: UpdateAdminOAuth2ClientHttpRequest
): UpdateAdminOAuth2ClientSuccess_200 | UpdateAdminOAuth2ClientError400 | UpdateAdminOAuth2ClientError403;

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

@doc("管理者が所属テナントの catalog 外 client を削除する。application_id が非 NULL の client は ApplicationOwnedProtocolError (HTTP 409) で拒否し、Application 削除へ誘導する。")
@TypeSpec.OpenAPI.operationId("DeleteAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@delete
op DeleteAdminOAuth2Client(
  @path client_id: string
): DeleteAdminOAuth2ClientSuccess_204 | DeleteAdminOAuth2ClientError400 | DeleteAdminOAuth2ClientError403 | DeleteAdminOAuth2ClientError409;

model ListAdminMcpResourceServersHttpResponse {
  response: IdMagic.Contract.McpResourceServerListResponse;
}

model ListAdminMcpResourceServersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminMcpResourceServersHttpResponse;
}

union ListAdminMcpResourceServersError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminMcpResourceServersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminMcpResourceServersError403Body;
}

@doc("管理者が所属テナントの McpResourceServer を一覧する (AdminMcpResourcesManage)。")
@TypeSpec.OpenAPI.operationId("ListAdminMcpResourceServers")
@route("/api/admin/v1/mcp-resource-servers")
@get
op ListAdminMcpResourceServers(

): ListAdminMcpResourceServersSuccess_200 | ListAdminMcpResourceServersError403;

model GetAdminMcpResourceServerHttpResponse {
  response: IdMagic.Contract.McpResourceServerResponse;
}

model GetAdminMcpResourceServerSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminMcpResourceServerHttpResponse;
}

union GetAdminMcpResourceServerError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GetAdminMcpResourceServerError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GetAdminMcpResourceServerError400Body;
}

union GetAdminMcpResourceServerError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminMcpResourceServerError403Body;
}

@doc("管理者が所属テナントの McpResourceServer を取得する。別テナントの登録は未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers/{id}")
@get
op GetAdminMcpResourceServer(
  @path id: string
): GetAdminMcpResourceServerSuccess_200 | GetAdminMcpResourceServerError400 | GetAdminMcpResourceServerError403;

model CreateAdminMcpResourceServerHttpRequest {
  request: IdMagic.Contract.McpResourceServerRequest;
}

model CreateAdminMcpResourceServerHttpResponse {
  response: IdMagic.Contract.McpResourceServerResponse;
}

model CreateAdminMcpResourceServerSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateAdminMcpResourceServerHttpResponse;
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

@doc("管理者が所属テナントに McpResourceServer を登録する。resource は tenant 内で一意でなければならない。")
@TypeSpec.OpenAPI.operationId("CreateAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers")
@post
op CreateAdminMcpResourceServer(
  @body body: CreateAdminMcpResourceServerHttpRequest
): CreateAdminMcpResourceServerSuccess_201 | CreateAdminMcpResourceServerError400 | CreateAdminMcpResourceServerError403;

model UpdateAdminMcpResourceServerHttpRequest {
  id: string;
  request: IdMagic.Contract.McpResourceServerRequest;
}

model UpdateAdminMcpResourceServerHttpResponse {
  response: IdMagic.Contract.McpResourceServerResponse;
}

model UpdateAdminMcpResourceServerSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateAdminMcpResourceServerHttpResponse;
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

@doc("管理者が McpResourceServer の name / scopes / state を更新する。resource (canonical URI) は不変。")
@TypeSpec.OpenAPI.operationId("UpdateAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers/{id}")
@patch
op UpdateAdminMcpResourceServer(
  @path id: string,
  @body body: UpdateAdminMcpResourceServerHttpRequest
): UpdateAdminMcpResourceServerSuccess_200 | UpdateAdminMcpResourceServerError400 | UpdateAdminMcpResourceServerError403;

model DeleteAdminMcpResourceServerSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteAdminMcpResourceServerError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteAdminMcpResourceServerError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminMcpResourceServerError400Body;
}

union DeleteAdminMcpResourceServerError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminMcpResourceServerError403Body;
}

@doc("管理者が McpResourceServer を削除する。以後この resource を指定する新規発行は invalid_target で拒否する (発行済みトークンは失効まで有効)。")
@TypeSpec.OpenAPI.operationId("DeleteAdminMcpResourceServer")
@route("/api/admin/v1/mcp-resource-servers/{id}")
@delete
op DeleteAdminMcpResourceServer(
  @path id: string
): DeleteAdminMcpResourceServerSuccess_204 | DeleteAdminMcpResourceServerError400 | DeleteAdminMcpResourceServerError403;

model ListAdminConsentsHttpResponse {
  response: IdMagic.Contract.AdminConsentListResponse;
}

model ListAdminConsentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAdminConsentsHttpResponse;
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

@doc("管理者が所属テナントの Consent を監査目的で user_id, client_id 昇順の双方向 keyset\npagination で一覧する。cursor は応答の Link response header (rel=\"prev\" / rel=\"next\") から取得する。")
@TypeSpec.OpenAPI.operationId("ListAdminConsents")
@route("/api/admin/v1/consents")
@get
op ListAdminConsents(
  @query cursor?: string,
  @query limit?: int64
): ListAdminConsentsSuccess_200 | ListAdminConsentsError403 | ListAdminConsentsError400;

model GetAdminConsentHttpResponse {
  consent: IdMagic.Contract.AdminConsentResponse;
}

model GetAdminConsentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAdminConsentHttpResponse;
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

@doc("管理者が所属テナントの Consent を subject と client_id で取得する。別テナントの Consent は未存在として扱う。")
@TypeSpec.OpenAPI.operationId("GetAdminConsent")
@route("/api/admin/v1/consents/{user_id}/{client_id}")
@get
op GetAdminConsent(
  @path user_id: string,
  @path client_id: string
): GetAdminConsentSuccess_200 | GetAdminConsentError400 | GetAdminConsentError403;

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

@doc("管理者が所属テナントの Consent を論理撤回する。管理者による Consent の作成・scope 拡張は許可しない。")
@TypeSpec.OpenAPI.operationId("RevokeAdminConsent")
@route("/api/admin/v1/consents/{user_id}/{client_id}")
@delete
op RevokeAdminConsent(
  @path user_id: string,
  @path client_id: string
): RevokeAdminConsentSuccess_204 | RevokeAdminConsentError400 | RevokeAdminConsentError403;

model ListAdminRolePoliciesHttpResponse {
  roles: unknown;
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

@doc("管理者が SCL authorization から導出された、自身に可視な role・permission・HTTP interface の対応を取得する。")
@TypeSpec.OpenAPI.operationId("ListAdminRolePolicies")
@route("/api/admin/v1/policy/roles")
@get
op ListAdminRolePolicies(

): ListAdminRolePoliciesSuccess_200 | ListAdminRolePoliciesError403;

model ListMyConsentsHttpResponse {
  response: IdMagic.Contract.AccountConsentListResponse;
}

model ListMyConsentsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListMyConsentsHttpResponse;
}

union ListMyConsentsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMyConsentsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMyConsentsError403Body;
}

@doc("認証済みユーザーが自身の接続済みアプリ (active な Consent) 一覧を取得する\n(self-service)。対象は actor.id に固定し、他人の Consent は返さない。")
@TypeSpec.OpenAPI.operationId("ListMyConsents")
@route("/api/account/v1/consents")
@get
op ListMyConsents(

): ListMyConsentsSuccess_200 | ListMyConsentsError403;

model RevokeMyConsentHttpRequest {
  client_id: string;
}

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

@doc("認証済みユーザーが自身の Consent を client_id 指定で論理撤回する (self-service)。\nactor.id == target.id に固定。次回その client の認可では consent が再要求される。")
@TypeSpec.OpenAPI.operationId("RevokeMyConsent")
@route("/api/account/v1/consents/{client_id}/revoke")
@post
op RevokeMyConsent(
  @path client_id: string,
  @body body: RevokeMyConsentHttpRequest
): RevokeMyConsentSuccess_204 | RevokeMyConsentError400 | RevokeMyConsentError403;
}

}
