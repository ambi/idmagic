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

@doc("Dynamic Client Registration（RFC 7591）")
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

@doc("認可コードができる前に、サインインのポリシーがリクエストを拒否した。認可エンドポイントが `redirect_uri` へリダイレクトせずにブラウザーへ直接報告する唯一の拒否なので、OAuth のエラーボディではなく、ブラウザー向けのエンドポイントと同じく Problem Details で返す。")
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
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: AuthorizeError429Body;
}

model AuthorizeSuccess_303 {
  @statusCode statusCode: 303;
}

@doc("認可エンドポイント（RFC 6749 の 4.1.1 節）。`client_id` は、登録済みの Client としても、Client ID Metadata Document の URL としても受け付け、リクエストのたびに解決する。")
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

@doc("Authentication の Context の受信フェデレーションのコールバックが認証済みのセッションを確立すると、保存した認可のトランザクションを再開し、通常のアプリケーションのポリシー、同意、必須のアクションのゲートを適用する。")
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

@doc("SPA が、HttpOnly のトランザクションの Cookie に対応するサインインと同意のコンテキストを読む。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("SPA が、認可リクエストに対する同意または拒否を JSON API で確定する。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserDeviceError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserDeviceError403Body;
}

@doc("SPA が、デバイスの `user_code` を JSON API で承認または拒否する。")
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

@doc("SPA が、デバイスの `user_code` と CSRF トークンを読む。")
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

@doc("RP-Initiated Logout のエンドポイント（OpenID Connect RP-Initiated Logout 1.0）。リダイレクトの検証は OAuth 2.0 / OpenID Connect の境界づけられたコンテキストが行い、セッションの失効は Authentication の境界づけられたコンテキストの SessionManager に委ねる。`id_token_hint` を与えると、その署名、`iss`、`aud`、`sub`、`sid` を検証し、対象のセッションとクライアントをどちらもヒントから解決する。`client_id` と矛盾するヒントは拒否する。`id_token_hint` がないときは、従来の `client_id` とブラウザーの Cookie による解決を代わりの手段として残す。ローカルの失効が確定すると、その `sid` に結び付く ClientSession から FrontChannelLogout の対象を集めてレスポンスで返し、BackChannelLogout の配送を非同期に開始する。")
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

@doc("RP-Initiated Logout のエンドポイント（OpenID Connect RP-Initiated Logout 1.0）。リダイレクトの検証は OAuth 2.0 / OpenID Connect の境界づけられたコンテキストが行い、セッションの失効は Authentication の境界づけられたコンテキストの SessionManager に委ねる。`id_token_hint` を与えると、その署名、`iss`、`aud`、`sub`、`sid` を検証し、対象のセッションとクライアントをどちらもヒントから解決する。`client_id` と矛盾するヒントは拒否する。`id_token_hint` がないときは、従来の `client_id` とブラウザーの Cookie による解決を代わりの手段として残す。ローカルの失効が確定すると、その `sid` に結び付く ClientSession から FrontChannelLogout の対象を集めてレスポンスで返し、BackChannelLogout の配送を非同期に開始する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("PostEndSession")
@route("/end_session")
@post
op EndSession2(
  @header contentType: "application/x-www-form-urlencoded",
  @body body: IdMagic.Contract.EndSessionParameters
): EndSession2Success_200 | EndSession2Success_302 | EndSession2Success_303 | EndSession2Error400;

model CheckSessionIframeSuccess_200 {
  @statusCode statusCode: 200;
}

@doc("RP が OP のセッションの変化を問い合わせる非表示の iframe（OpenID Connect Session Management 1.0。仕様がまだドラフトで Final に達していないので、採用は任意である）。静的な HTML と JavaScript を返し、RP からの `postMessage` には、現在のブラウザーの Cookie が有効な LoginSession へ解決するかどうかだけを答える。`session_state` のソルト付きハッシュの比較を含む、仕様の詳細な照合のアルゴリズムは実装していない。")
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
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: PushAuthorizationRequestError429Body;
}

model PushAuthorizationRequestSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.ParResponse;
}

@doc("Pushed Authorization Request（RFC 9126）。`client_id` は、登録済みの Client としても、Client ID Metadata Document の URL としても受け付け、リクエストのたびに解決する。")
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
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: TokenError429Body;
}

@doc("トークンエンドポイント（RFC 6749 §3.2 + 9068 + 9449）")
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

@doc("通常の OAuth のトークンと、管理用に発行した JWT アクセストークンの両方を対象とする Token Introspection（RFC 7662）。主体が Agent のトークンでは、`issued_at` を SharedSignals の失効のエポック（キルスイッチ、所有者のオフボーディング、CAEP/SSF で届いた失効）と比べ、エポックより前に発行したトークンはフェイルクローズで `active=false` と報告する。")
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

@doc("通常の OAuth のトークンと、管理用に発行した JWT アクセストークンの両方を対象とする Token Revocation（RFC 7009）。")
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
  @header("WWW-Authenticate") wwwAuthenticate: string;
  @body body: UserInfoError401Body;
}

union UserInfoError403Body {
  IdMagic.Contract.OAuthInsufficientScopeError,
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

@doc("UserInfo（OpenID Connect Core の 5.3 節）。`additional_claims` は、スコープから決まる ClaimsForScopes と、クライアントごとの ClaimMapping のクレームのポリシーを合わせたものである。")
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
  @header("WWW-Authenticate") wwwAuthenticate: string;
  @body body: PostUserInfoError401Body;
}

union PostUserInfoError403Body {
  IdMagic.Contract.OAuthInsufficientScopeError,
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

@doc("UserInfo の POST バインディング（OpenID Connect Core の 5.3 節）。")
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
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: DeviceAuthorizationError429Body;
}

@doc("デバイス認可リクエスト（RFC 8628 §3.1）")
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
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: BackchannelAuthenticateError429Body;
}

@doc("Backchannel Authentication Request（OpenID CIBA Core の 7.1 節）。クライアントを認証し、人が対応する承認のリクエストを起こして、`auth_req_id` を返す。承認が確定するまで、トークンエンドポイントの CIBA のグラントは `authorization_pending` を返し、確定した後はちょうど一度だけトークンと交換する。")
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

@doc("OAuth 2.0 Authorization Server Metadata（RFC 8414）")
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

@doc("Protected Resource Metadata（RFC 9728）。`resource` がなければ、レルムの IdMagic API とその ApiTokenScopes を返す。あれば、登録済みで Active の McpResourceServer のメタデータを返す。登録されていないリソースと無効にしたリソースは `invalid_target` として拒否する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetProtectedResourceMetadata")
@route("/.well-known/oauth-protected-resource")
@get
op GetProtectedResourceMetadata(
  @query resource?: string
): GetProtectedResourceMetadataSuccess_200 | GetProtectedResourceMetadataError400;

model HealthHttpResponse {
  @doc("常に `ok` である。")
  status: string;
  @doc("プロセスが起動したときの永続化のバックエンド。")
  persistence: string;
  @doc("プロセスが起動したときのオブザーバビリティのバックエンド。")
  observability: string;
  @doc("プロセスが起動したときの AuthZEN のバックエンド。")
  authzen: string;
  @doc("このプロセスで有効になっている、実行時に選択できる機能の、バージョン付きのメタデータ。")
  features: IdMagic.Contract.FeatureRuntimeMetadata;
}

model HealthSuccess_200 {
  @statusCode statusCode: 200;
  @body body: HealthHttpResponse;
}

@doc("ヘルスチェック")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAuthorizationDetailTypesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAuthorizationDetailTypesError403Body;
}

@doc("管理者が、自身のテナントの `authorization_details` の種類を一覧する（RFC 9396）。")
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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、種類を一つ読む。他のテナントの種類は存在しないものとして扱う。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、自身のテナントに `authorization_details` の種類を登録する。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、種類のスキーマ、表示のテンプレート、説明、または状態を更新する。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAuthorizationDetailTypeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAuthorizationDetailTypeError403Body;
}

@doc("管理者が、種類を削除する。発行済みのトークンは、期限が切れるまで有効なままである。")
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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、自身のテナントのクライアントを、`client_id` の昇順に、双方向のキーセットページングで一覧する。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminOAuth2ClientError403Body;
}

union GetAdminOAuth2ClientError404Body {
  IdMagic.Contract.OAuth2ClientNotFoundError,
}

model GetAdminOAuth2ClientError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: GetAdminOAuth2ClientError404Body;
}

@doc("管理者が、自身のテナントのクライアントを読む。他のテナントのクライアントは存在しないものとして扱う。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:read"])
@TypeSpec.OpenAPI.operationId("GetAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@get
op GetAdminOAuth2Client(
  @path client_id: string
): GetAdminOAuth2ClientSuccess_200 | GetAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | GetAdminOAuth2ClientError403 | GetAdminOAuth2ClientError404;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、自身のテナントにクライアントを作成する。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminOAuth2ClientError403Body;
}

union UpdateAdminOAuth2ClientError404Body {
  IdMagic.Contract.OAuth2ClientNotFoundError,
}

model UpdateAdminOAuth2ClientError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateAdminOAuth2ClientError404Body;
}

@doc("管理者が、自身のテナントのクライアントのメタデータを更新する。他のテナントのクライアントは存在しないものとして扱い、変更しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:write"])
@TypeSpec.OpenAPI.operationId("UpdateAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@patch
op UpdateAdminOAuth2Client(
  @path client_id: string,
  @body body: IdMagic.Contract.AdminOAuth2ClientUpdateRequest
): UpdateAdminOAuth2ClientSuccess_200 | UpdateAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateAdminOAuth2ClientError403 | UpdateAdminOAuth2ClientError404;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminOAuth2ClientError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminOAuth2ClientError403Body;
}

union DeleteAdminOAuth2ClientError404Body {
  IdMagic.Contract.OAuth2ClientNotFoundError,
}

model DeleteAdminOAuth2ClientError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminOAuth2ClientError404Body;
}

union DeleteAdminOAuth2ClientError409Body {
  IdMagic.Contract.ApplicationOwnedProtocolError,
}

model DeleteAdminOAuth2ClientError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminOAuth2ClientError409Body;
}

@doc("管理者が、自身のテナントの、カタログの外にあるクライアントを削除する。`application_id` が null でないクライアントは ApplicationOwnedProtocolError（HTTP 409）で拒否し、代わりに Application を削除するよう呼び出し元へ案内する。他のテナントのクライアントは存在しないものとして扱い、変更しない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["oauth-clients:write"])
@TypeSpec.OpenAPI.operationId("DeleteAdminOAuth2Client")
@route("/api/admin/v1/clients/{client_id}")
@delete
op DeleteAdminOAuth2Client(
  @path client_id: string
): DeleteAdminOAuth2ClientSuccess_204 | DeleteAdminOAuth2ClientError400 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteAdminOAuth2ClientError403 | DeleteAdminOAuth2ClientError404 | DeleteAdminOAuth2ClientError409;

model ListAdminMcpResourceServersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.McpResourceServerListResponse;
}

union ListAdminMcpResourceServersError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminMcpResourceServersError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminMcpResourceServersError403Body;
}

@doc("管理者が、自身のテナントの McpResourceServer を一覧する（AdminMcpResourcesManage）。")
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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、自身のテナントの McpResourceServer を読む。他のテナントの登録は存在しないものとして扱う。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、自身のテナントに McpResourceServer を登録する。`resource` はテナントの中で一意でなければならない。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("管理者が、McpResourceServer の名前、スコープ、または状態を更新する。正規の URI である `resource` は変えられない。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteAdminMcpResourceServerError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteAdminMcpResourceServerError403Body;
}

@doc("管理者が McpResourceServer を削除する。以降、そのリソースを指定する新たな発行は `invalid_target` で拒否する。発行済みのトークンは、期限が切れるまで有効なままである。")
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
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、監査のために、自身のテナントの Consent を `user_id` と `client_id` の昇順に、双方向のキーセットページングで一覧する。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAdminConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAdminConsentError403Body;
}

@doc("管理者が、自身のテナントの Consent を、サブジェクトと `client_id` で読む。他のテナントの Consent は存在しないものとして扱う。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeAdminConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeAdminConsentError403Body;
}

@doc("管理者が、自身のテナントの Consent を論理的に失効させる。管理者には、Consent の作成やスコープの拡大は許されない。")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListAdminRolePoliciesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAdminRolePoliciesError403Body;
}

@doc("管理者が、自身に見えるロール、権限、HTTP のインターフェースの対応を読む。")
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

@doc("認証済みのユーザーが、自身が接続しているアプリケーションの一覧、すなわち有効な Consent を読む（セルフサービス）。対象は `actor.id` に固定され、他人の Consent を返すことはない。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyConsentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyConsentError403Body;
}

@doc("認証済みのユーザーが、指定した `client_id` に対する自身の Consent を論理的に失効させる（セルフサービス）。`actor.id` は `target.id` と等しく固定される。そのクライアントの次の認可では、改めて同意を求める。")
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

@doc("認証済みのユーザーが、自身宛ての保留中の承認のリクエストを読む（セルフサービス）。対象は `actor.id` に固定されるので、他人宛てのリクエストを返すことはなく、期限切れのリクエストも返さない。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.StepUpRequiredError,
  IdMagic.Contract.AccessDeniedError,
}

model DecideMyApprovalRequestError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DecideMyApprovalRequestError403Body;
}

@doc("認証済みのユーザーが、自身宛ての承認のリクエストを一つ承認または拒否する（セルフサービス）。ステップアップの再認証と CSRF の検証がどちらも必要で、`actor.id` は承認の対象の User に固定される。他人宛てのリクエスト、終端の状態のリクエスト、期限切れのリクエストを決定しようとすると拒否する。")
@TypeSpec.OpenAPI.operationId("DecideMyApprovalRequest")
@route("/api/account/v1/approval-requests/{id}/decision")
@post
op DecideMyApprovalRequest(
  @path id: string,
  @body body: IdMagic.Contract.AccountApprovalDecisionRequest
): DecideMyApprovalRequestSuccess_204 | DecideMyApprovalRequestError400 | IdMagic.Contract.AuthenticationRequiredResponse | DecideMyApprovalRequestError403;
}

}
