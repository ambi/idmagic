import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Authentication")
namespace IdMagic.Authentication {

namespace Operations {
model GetAccountContextHttpResponse {
  context: IdMagic.Contract.AccountContextResponse;
}

model GetAccountContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAccountContextHttpResponse;
}

union GetAccountContextError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountContextError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountContextError403Body;
}

@doc("認証済み browser session の portal 共通 bootstrap 情報と CSRF token を取得する。 Bearer token では管理 portal の idmagic.admin、account portal の idmagic.account、 または自己管理 API client の account:read scope のいずれかを要求する。認証途中の session と、それ以外の scope だけを持つ Bearer token は拒否する。")
@TypeSpec.OpenAPI.operationId("GetAccountContext")
@route("/api/auth/account")
@get
op GetAccountContext(

): GetAccountContextSuccess_200 | GetAccountContextError403;

model GetPasswordResetContextHttpResponse {
  context: IdMagic.Contract.BrowserCsrfContext;
}

model GetPasswordResetContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetPasswordResetContextHttpResponse;
}

@doc("未認証のパスワードリセット画面が POST を保護する CSRF token を取得する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetPasswordResetContext")
@route("/api/auth/password_reset_context")
@get
op GetPasswordResetContext(

): GetPasswordResetContextSuccess_200;

model StartStepUpWebAuthnChallengeHttpResponse {
  options: IdMagic.Contract.WebAuthnAssertionOptions;
}

model StartStepUpWebAuthnChallengeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartStepUpWebAuthnChallengeHttpResponse;
}

union StartStepUpWebAuthnChallengeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartStepUpWebAuthnChallengeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartStepUpWebAuthnChallengeError400Body;
}

union StartStepUpWebAuthnChallengeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartStepUpWebAuthnChallengeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartStepUpWebAuthnChallengeError403Body;
}

@doc("認証済み session に束縛した step-up 用 WebAuthn assertion challenge を発行する。CSRF を検証し、WebAuthn 無効時は fail closed とする。")
@TypeSpec.OpenAPI.operationId("StartStepUpWebAuthnChallenge")
@route("/api/account/v1/step_up/webauthn/challenge")
@post
op StartStepUpWebAuthnChallenge(

): StartStepUpWebAuthnChallengeSuccess_200 | StartStepUpWebAuthnChallengeError400 | StartStepUpWebAuthnChallengeError403;

model SubmitBrowserLoginHttpRequest {
  body: {
    username: string;
    password: string;
    @doc("同一 realm の /admin 配下に限定したログイン後遷移先。")
    return_to?: string;
  };
}

model SubmitBrowserLoginHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model SubmitBrowserLoginSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SubmitBrowserLoginHttpResponse;
}

union SubmitBrowserLoginError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SubmitBrowserLoginError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserLoginError400Body;
}

union SubmitBrowserLoginError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserLoginError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserLoginError403Body;
}

union SubmitBrowserLoginError422Body {
  IdMagic.Contract.QuotaExceededError,
}

model SubmitBrowserLoginError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserLoginError422Body;
}

union SubmitBrowserLoginError429Body {
  IdMagic.Contract.RateLimitedError,
}

model SubmitBrowserLoginError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserLoginError429Body;
}

@doc("SPA JSON API から browser password step を処理する。MFA 必須かつ factor 未登録の場合、強制開始前は password session でログインを許可して事前登録を促し、開始後は有効な単発 enrollment bypass と猶予期限の両方を満たすときだけ同一 LoginSession を Enrollment pending にする。それ以外は fail closed でログインを完了しない。per-account (10 回 / 900 秒窓、900 秒ロックアウト) と per-IP (30 回 / 900 秒窓、900 秒ロックアウト) のスロットリングを cluster_wide カウンタで適用し、共有カウンタストア到達不能時も fail_closed とする。この失敗回数ベースの login throttle とは別に、成功/失敗を問わず全リクエストを消費する EndpointRateLimitPolicy (IP+tenant ベース) も適用する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserLogin")
@route("/api/auth/login")
@post
op SubmitBrowserLogin(
  @body body: SubmitBrowserLoginHttpRequest
): SubmitBrowserLoginSuccess_200 | SubmitBrowserLoginError400 | SubmitBrowserLoginError403 | SubmitBrowserLoginError422 | SubmitBrowserLoginError429;

model StartBrowserMfaEnrollmentHttpResponse {
  enrollment: IdMagic.Contract.TOTPEnrollmentStart;
}

model StartBrowserMfaEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartBrowserMfaEnrollmentHttpResponse;
}

union StartBrowserMfaEnrollmentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartBrowserMfaEnrollmentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartBrowserMfaEnrollmentError400Body;
}

union StartBrowserMfaEnrollmentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartBrowserMfaEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartBrowserMfaEnrollmentError403Body;
}

@doc("pending_purpose=Enrollment かつ期限内の LoginSession だけが TOTP 登録を開始する。通常の未認証 session、Challenge pending、期限切れ、既登録 user は拒否する。secret は確定まで永続化しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartBrowserMfaEnrollment")
@route("/api/auth/mfa/enrollment/totp/start")
@post
op StartBrowserMfaEnrollment(

): StartBrowserMfaEnrollmentSuccess_200 | StartBrowserMfaEnrollmentError400 | StartBrowserMfaEnrollmentError403;

model ConfirmBrowserMfaEnrollmentHttpRequest {
  request: IdMagic.Contract.BrowserMfaEnrollmentConfirmRequest;
}

model ConfirmBrowserMfaEnrollmentHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model ConfirmBrowserMfaEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ConfirmBrowserMfaEnrollmentHttpResponse;
}

union ConfirmBrowserMfaEnrollmentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfirmBrowserMfaEnrollmentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfirmBrowserMfaEnrollmentError400Body;
}

union ConfirmBrowserMfaEnrollmentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ConfirmBrowserMfaEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfirmBrowserMfaEnrollmentError403Body;
}

@doc("pending_purpose=Enrollment かつ期限内の LoginSession で TOTP 所持証明を検証して factor を保存し、同一 LoginSession に otp を加えて MFA 済みに昇格させ、元の authorization transaction を継続する。通常 self-service 登録 endpoint と権限を共有しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ConfirmBrowserMfaEnrollment")
@route("/api/auth/mfa/enrollment/totp/confirm")
@post
op ConfirmBrowserMfaEnrollment(
  @body body: ConfirmBrowserMfaEnrollmentHttpRequest
): ConfirmBrowserMfaEnrollmentSuccess_200 | ConfirmBrowserMfaEnrollmentError400 | ConfirmBrowserMfaEnrollmentError403;

model SubmitBrowserTotpHttpRequest {
  request: IdMagic.Contract.BrowserTotpRequest;
}

model SubmitBrowserTotpHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model SubmitBrowserTotpSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SubmitBrowserTotpHttpResponse;
}

union SubmitBrowserTotpError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SubmitBrowserTotpError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserTotpError400Body;
}

union SubmitBrowserTotpError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserTotpError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserTotpError403Body;
}

@doc("SPA JSON API から TOTP コードを検証する。成功すれば LoginSession を認証完了状態に昇格させ、後続フロー (consent / authorization code 発行) に進む。他 factor は別 endpoint として定義する。未認証の pending login session cookie が対象ユーザーを解決するため認証は要求しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserTotp")
@route("/api/auth/totp")
@post
op SubmitBrowserTotp(
  @body body: SubmitBrowserTotpHttpRequest
): SubmitBrowserTotpSuccess_200 | SubmitBrowserTotpError400 | SubmitBrowserTotpError403;

model ChangePasswordHttpRequest {
  request: IdMagic.Contract.BrowserChangePasswordRequest;
}

model ChangePasswordSuccess_200 {
  @statusCode statusCode: 200;
}

union ChangePasswordError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ChangePasswordError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ChangePasswordError400Body;
}

union ChangePasswordError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ChangePasswordError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ChangePasswordError403Body;
}

union ChangePasswordError422Body {
  IdMagic.Contract.PasswordReuseError,
}

model ChangePasswordError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: ChangePasswordError422Body;
}

@doc("認証済みユーザーが自身のパスワードを変更する。current_password を verify し、new_password は 12〜128 文字の長さ制約と直近 history_depth (5) 件のハッシュとの不一致 (PasswordHistoryNoReuse) を満たさなければならない。")
@TypeSpec.OpenAPI.operationId("ChangePassword")
@route("/api/auth/change_password")
@post
op ChangePassword(
  @body body: ChangePasswordHttpRequest
): ChangePasswordSuccess_200 | ChangePasswordError400 | ChangePasswordError403 | ChangePasswordError422;

model GetAccountSecurityHttpResponse {
  security: IdMagic.Contract.AccountSecurityResponse;
}

model GetAccountSecuritySuccess_200 {
  @statusCode statusCode: 200;
  @body body: GetAccountSecurityHttpResponse;
}

union GetAccountSecurityError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountSecurityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountSecurityError403Body;
}

@doc("認証済みユーザーが自身のセキュリティ概要を取得する (self-service)。対象は subject.id に\n固定。パスワード変更日時と MFA (TOTP) 登録状態を返す。")
@TypeSpec.OpenAPI.operationId("GetAccountSecurity")
@route("/api/account/v1/security")
@get
op GetAccountSecurity(

): GetAccountSecuritySuccess_200 | GetAccountSecurityError403;

model StartTotpEnrollmentHttpResponse {
  enrollment: IdMagic.Contract.TOTPEnrollmentStart;
}

model StartTotpEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartTotpEnrollmentHttpResponse;
}

union StartTotpEnrollmentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartTotpEnrollmentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartTotpEnrollmentError400Body;
}

union StartTotpEnrollmentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartTotpEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartTotpEnrollmentError403Body;
}

@doc("認証済みユーザーが TOTP 登録を開始する (self-service)。新しい secret (20 バイト) と\notpauth URI を返すだけで永続化はしない。既に TOTP factor がある場合は拒否する。")
@TypeSpec.OpenAPI.operationId("StartTotpEnrollment")
@route("/api/account/v1/mfa/totp/enroll/start")
@post
op StartTotpEnrollment(

): StartTotpEnrollmentSuccess_200 | StartTotpEnrollmentError400 | StartTotpEnrollmentError403;

model ConfirmTotpEnrollmentHttpRequest {
  request: IdMagic.Contract.TOTPEnrollmentConfirmRequest;
}

model ConfirmTotpEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
}

union ConfirmTotpEnrollmentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfirmTotpEnrollmentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfirmTotpEnrollmentError400Body;
}

union ConfirmTotpEnrollmentError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ConfirmTotpEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfirmTotpEnrollmentError403Body;
}

@doc("secret に対するコードの所持証明を検証し、TOTP factor を永続化する (self-service)。\nsubject.id に固定し、確定時に User.mfa_enrolled を true にする。")
@TypeSpec.OpenAPI.operationId("ConfirmTotpEnrollment")
@route("/api/account/v1/mfa/totp/enroll/confirm")
@post
op ConfirmTotpEnrollment(
  @body body: ConfirmTotpEnrollmentHttpRequest
): ConfirmTotpEnrollmentSuccess_200 | ConfirmTotpEnrollmentError400 | ConfirmTotpEnrollmentError403;

model RemoveTotpFactorHttpRequest {
  request: IdMagic.Contract.MfaFactorRemoveRequest;
}

model RemoveTotpFactorSuccess_200 {
  @statusCode statusCode: 200;
}

union RemoveTotpFactorError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RemoveTotpFactorError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RemoveTotpFactorError400Body;
}

union RemoveTotpFactorError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RemoveTotpFactorError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveTotpFactorError403Body;
}

@doc("有効な TOTP コード (所持証明) を検証してから TOTP factor を解除する (self-service)。\nsubject.id に固定し、解除時に User.mfa_enrolled を false に戻す。")
@TypeSpec.OpenAPI.operationId("RemoveTotpFactor")
@route("/api/account/v1/mfa/totp/remove")
@post
op RemoveTotpFactor(
  @body body: RemoveTotpFactorHttpRequest
): RemoveTotpFactorSuccess_200 | RemoveTotpFactorError400 | RemoveTotpFactorError403;

model StartWebAuthnRegistrationHttpResponse {
  options: IdMagic.Contract.WebAuthnRegistrationOptions;
}

model StartWebAuthnRegistrationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartWebAuthnRegistrationHttpResponse;
}

union StartWebAuthnRegistrationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartWebAuthnRegistrationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartWebAuthnRegistrationError400Body;
}

union StartWebAuthnRegistrationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartWebAuthnRegistrationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartWebAuthnRegistrationError403Body;
}

@doc("認証済みユーザーが WebAuthn / Passkey 登録を開始する (self-service)。\nPublicKeyCredentialCreationOptions を返し、challenge をサーバー側セッションに束縛する。\n永続化はまだ行わない。")
@TypeSpec.OpenAPI.operationId("StartWebAuthnRegistration")
@route("/api/account/v1/mfa/webauthn/register/start")
@post
op StartWebAuthnRegistration(

): StartWebAuthnRegistrationSuccess_200 | StartWebAuthnRegistrationError400 | StartWebAuthnRegistrationError403;

model FinishWebAuthnRegistrationHttpRequest {
  request: IdMagic.Contract.WebAuthnRegistrationRequest;
}

model FinishWebAuthnRegistrationSuccess_200 {
  @statusCode statusCode: 200;
}

union FinishWebAuthnRegistrationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model FinishWebAuthnRegistrationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: FinishWebAuthnRegistrationError400Body;
}

union FinishWebAuthnRegistrationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model FinishWebAuthnRegistrationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: FinishWebAuthnRegistrationError403Body;
}

@doc("attestation を検証して WebAuthn credential を永続化する (self-service)。subject.id に\n固定し、最初の 1 件で User.mfa_enrolled を true にする。RP ID / origin / challenge を検証する。")
@TypeSpec.OpenAPI.operationId("FinishWebAuthnRegistration")
@route("/api/account/v1/mfa/webauthn/register/finish")
@post
op FinishWebAuthnRegistration(
  @body body: FinishWebAuthnRegistrationHttpRequest
): FinishWebAuthnRegistrationSuccess_200 | FinishWebAuthnRegistrationError400 | FinishWebAuthnRegistrationError403;

model RemoveWebAuthnCredentialHttpRequest {
  request: IdMagic.Contract.WebAuthnCredentialRemoveRequest;
}

model RemoveWebAuthnCredentialSuccess_200 {
  @statusCode statusCode: 200;
}

union RemoveWebAuthnCredentialError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RemoveWebAuthnCredentialError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RemoveWebAuthnCredentialError400Body;
}

union RemoveWebAuthnCredentialError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RemoveWebAuthnCredentialError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveWebAuthnCredentialError403Body;
}

@doc("指定した WebAuthn credential を解除する (self-service, step-up 必須)。subject.id に固定し、\n残る第二要素が無くなれば User.mfa_enrolled を false に戻す。")
@TypeSpec.OpenAPI.operationId("RemoveWebAuthnCredential")
@route("/api/account/v1/mfa/webauthn/remove")
@post
op RemoveWebAuthnCredential(
  @body body: RemoveWebAuthnCredentialHttpRequest
): RemoveWebAuthnCredentialSuccess_200 | RemoveWebAuthnCredentialError400 | RemoveWebAuthnCredentialError403;

model StartBrowserWebAuthnHttpResponse {
  options: IdMagic.Contract.WebAuthnAssertionOptions;
}

model StartBrowserWebAuthnSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartBrowserWebAuthnHttpResponse;
}

union StartBrowserWebAuthnError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartBrowserWebAuthnError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartBrowserWebAuthnError400Body;
}

union StartBrowserWebAuthnError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartBrowserWebAuthnError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartBrowserWebAuthnError403Body;
}

@doc("SPA JSON API から WebAuthn assertion challenge を発行する。pending login session が\nsub を解決し、PublicKeyCredentialRequestOptions を返す。challenge はそのセッションに束縛する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartBrowserWebAuthn")
@route("/api/auth/webauthn/challenge")
@post
op StartBrowserWebAuthn(

): StartBrowserWebAuthnSuccess_200 | StartBrowserWebAuthnError400 | StartBrowserWebAuthnError403;

model SubmitBrowserWebAuthnHttpRequest {
  request: IdMagic.Contract.BrowserWebAuthnRequest;
}

model SubmitBrowserWebAuthnHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model SubmitBrowserWebAuthnSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SubmitBrowserWebAuthnHttpResponse;
}

union SubmitBrowserWebAuthnError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SubmitBrowserWebAuthnError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserWebAuthnError400Body;
}

union SubmitBrowserWebAuthnError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserWebAuthnError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserWebAuthnError403Body;
}

@doc("SPA JSON API から WebAuthn assertion を検証する。成功すれば LoginSession を認証完了\n状態に昇格させ、後続フロー (consent / authorization code 発行) に進む。sign_count 逆行は\nclone として拒否する (sign_count_regression=reject)。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserWebAuthn")
@route("/api/auth/webauthn")
@post
op SubmitBrowserWebAuthn(
  @body body: SubmitBrowserWebAuthnHttpRequest
): SubmitBrowserWebAuthnSuccess_200 | SubmitBrowserWebAuthnError400 | SubmitBrowserWebAuthnError403;

model SubmitBrowserRecoveryCodeHttpRequest {
  request: IdMagic.Contract.BrowserRecoveryCodeRequest;
}

model SubmitBrowserRecoveryCodeHttpResponse {
  response: IdMagic.Contract.BrowserFlowResponse;
}

model SubmitBrowserRecoveryCodeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: SubmitBrowserRecoveryCodeHttpResponse;
}

union SubmitBrowserRecoveryCodeError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model SubmitBrowserRecoveryCodeError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserRecoveryCodeError400Body;
}

union SubmitBrowserRecoveryCodeError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserRecoveryCodeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserRecoveryCodeError403Body;
}

@doc("SPA JSON API から backup recovery code を第二要素として検証する。成功すれば\nLoginSession を認証完了状態に昇格させ、消費した code は再利用不可にする。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserRecoveryCode")
@route("/api/auth/recovery-code")
@post
op SubmitBrowserRecoveryCode(
  @body body: SubmitBrowserRecoveryCodeHttpRequest
): SubmitBrowserRecoveryCodeSuccess_200 | SubmitBrowserRecoveryCodeError400 | SubmitBrowserRecoveryCodeError403;

model GenerateRecoveryCodesHttpResponse {
  codes: IdMagic.Contract.RecoveryCodesResponse;
}

model GenerateRecoveryCodesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: GenerateRecoveryCodesHttpResponse;
}

union GenerateRecoveryCodesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model GenerateRecoveryCodesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: GenerateRecoveryCodesError400Body;
}

union GenerateRecoveryCodesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GenerateRecoveryCodesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GenerateRecoveryCodesError403Body;
}

@doc("認証済みユーザーが backup recovery code を生成 / 再生成する (self-service, step-up 必須)。\n平文を一度だけ返し、DB には hash のみ保存する。既存 set は全置換する (count=10 件)。")
@TypeSpec.OpenAPI.operationId("GenerateRecoveryCodes")
@route("/api/account/v1/mfa/recovery-codes/generate")
@post
op GenerateRecoveryCodes(

): GenerateRecoveryCodesSuccess_200 | GenerateRecoveryCodesError400 | GenerateRecoveryCodesError403;

model RevokeRecoveryCodesSuccess_200 {
  @statusCode statusCode: 200;
}

union RevokeRecoveryCodesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeRecoveryCodesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeRecoveryCodesError400Body;
}

union RevokeRecoveryCodesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeRecoveryCodesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeRecoveryCodesError403Body;
}

@doc("認証済みユーザーが backup recovery code を明示的に失効する (self-service, step-up 必須)。\nsubject.id に固定し、有効な code をすべて削除する。")
@TypeSpec.OpenAPI.operationId("RevokeRecoveryCodes")
@route("/api/account/v1/mfa/recovery-codes/revoke")
@post
op RevokeRecoveryCodes(

): RevokeRecoveryCodesSuccess_200 | RevokeRecoveryCodesError400 | RevokeRecoveryCodesError403;

model ListMySignInActivityHttpResponse {
  activity: IdMagic.Contract.UserSignInActivityResponse;
}

model ListMySignInActivitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListMySignInActivityHttpResponse;
}

union ListMySignInActivityError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMySignInActivityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMySignInActivityError403Body;
}

union ListMySignInActivityError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListMySignInActivityError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListMySignInActivityError400Body;
}

@doc("認証済みユーザーが自身の直近サインイン履歴を取得する (self-service)。対象は\nsubject.id に固定。既存の監査イベントストア (UserAuthenticated) から発生時刻の降順で\n最大 limit 件を返す。limit 既定 10 / 上限 50。")
@TypeSpec.OpenAPI.operationId("ListMySignInActivity")
@route("/api/account/v1/signin_activity")
@get
op ListMySignInActivity(
  @query limit?: int64
): ListMySignInActivitySuccess_200 | ListMySignInActivityError403 | ListMySignInActivityError400;

model ListUserSignInActivityHttpResponse {
  activity: IdMagic.Contract.UserSignInActivityResponse;
}

model ListUserSignInActivitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListUserSignInActivityHttpResponse;
}

union ListUserSignInActivityError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListUserSignInActivityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserSignInActivityError403Body;
}

union ListUserSignInActivityError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListUserSignInActivityError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListUserSignInActivityError400Body;
}

@doc("admin が対象ユーザーのサインイン履歴を取得する。tenant 境界内に限定し、\nパス sub のユーザーの UserAuthenticated を発生時刻の降順で最大 limit 件返す。\nlimit 既定 10 / 上限 50。")
@TypeSpec.OpenAPI.operationId("ListUserSignInActivity")
@route("/api/admin/v1/users/{user_id}/signin_activity")
@get
op ListUserSignInActivity(
  @path user_id: string,
  @query limit?: int64
): ListUserSignInActivitySuccess_200 | ListUserSignInActivityError403 | ListUserSignInActivityError400;

model ListAuthenticationEventBucketsHttpResponse {
  buckets: IdMagic.Contract.AuthenticationEventBucketListResponse;
}

model ListAuthenticationEventBucketsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListAuthenticationEventBucketsHttpResponse;
}

union ListAuthenticationEventBucketsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListAuthenticationEventBucketsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListAuthenticationEventBucketsError403Body;
}

union ListAuthenticationEventBucketsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ListAuthenticationEventBucketsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ListAuthenticationEventBucketsError400Body;
}

@doc("admin が認証イベント集約 (bucket) を window_start の降順で一覧する。攻撃時に個別行へ\n落とさず 5 分窓へ集約したログイン失敗などを、所属テナント境界内で双方向 keyset pagination により\n取得する。cursor は応答の Link response header (rel=\"prev\" / rel=\"next\") から取得する。limit 既定\n50 / 上限 200。")
@TypeSpec.OpenAPI.operationId("ListAuthenticationEventBuckets")
@route("/api/admin/v1/authentication_event_buckets")
@get
op ListAuthenticationEventBuckets(
  @query cursor?: string,
  @query limit?: int64
): ListAuthenticationEventBucketsSuccess_200 | ListAuthenticationEventBucketsError403 | ListAuthenticationEventBucketsError400;

model ListMySessionsHttpResponse {
  sessions: IdMagic.Contract.AccountSessionListResponse;
}

model ListMySessionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListMySessionsHttpResponse;
}

union ListMySessionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMySessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMySessionsError403Body;
}

@doc("認証済みユーザーが自身の有効なセッションを取得する (self-service)。対象は\nsubject.id に固定。現在のセッションを current=true でマークし、開始時刻の降順で返す。")
@TypeSpec.OpenAPI.operationId("ListMySessions")
@route("/api/account/v1/sessions")
@get
op ListMySessions(

): ListMySessionsSuccess_200 | ListMySessionsError403;

model RevokeMySessionSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMySessionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeMySessionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeMySessionError400Body;
}

union RevokeMySessionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMySessionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMySessionError403Body;
}

@doc("認証済みユーザーが自身のセッション 1 件を失効する (self-service)。対象が本人の\nものでなければ拒否する。失効は LoginSession に revoked_at / revoke_reason=self_revoke を\n設定する tombstone であり、物理削除しない。既に失効済みの対象への再失効は idempotent に\n成功する (revoked_at は初回の値を保持する)。")
@TypeSpec.OpenAPI.operationId("RevokeMySession")
@route("/api/account/v1/sessions/{id}/revoke")
@post
op RevokeMySession(
  @path id: string
): RevokeMySessionSuccess_204 | RevokeMySessionError400 | RevokeMySessionError403;

model RevokeMyOtherSessionsSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMyOtherSessionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyOtherSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyOtherSessionsError403Body;
}

@doc("認証済みユーザーが現在のセッションを除く自身の全セッションを失効する (self-service)。\n\"他のセッションを全て終了\" 操作。各対象は revoked_at / revoke_reason=self_revoke を設定する\ntombstone であり、既に失効済みの対象は idempotent にスキップする。")
@TypeSpec.OpenAPI.operationId("RevokeMyOtherSessions")
@route("/api/account/v1/sessions/revoke_others")
@post
op RevokeMyOtherSessions(

): RevokeMyOtherSessionsSuccess_204 | RevokeMyOtherSessionsError403;

model ListSessionsHttpResponse {
  sessions: IdMagic.Contract.SessionRecordListResponse;
}

model ListSessionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListSessionsHttpResponse;
}

union ListSessionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSessionsError403Body;
}

@doc("admin が対象ユーザーの有効なセッションを一覧する。tenant 境界内に\n限定し、開始時刻の降順で返す。ListMySessions と異なり対象は任意の user_id で、\ncurrent マーカーは持たない。")
@TypeSpec.OpenAPI.operationId("ListSessions")
@route("/api/admin/v1/users/{user_id}/sessions")
@get
op ListSessions(
  @path user_id: string
): ListSessionsSuccess_200 | ListSessionsError403;

model RevokeSessionHttpRequest {
  user_id: string;
}

model RevokeSessionSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeSessionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeSessionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeSessionError400Body;
}

union RevokeSessionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeSessionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeSessionError403Body;
}

@doc("admin が対象ユーザーのセッション1件を失効する。RevokeMySession と同じ\ntombstone 契約 (revoked_at / revoke_reason=admin_revoke を設定し物理削除しない)。\n既に失効済みの対象への再失効は idempotent に成功する。")
@TypeSpec.OpenAPI.operationId("RevokeSession")
@route("/api/admin/v1/users/{user_id}/sessions/{id}/revoke")
@post
op RevokeSession(
  @path user_id: string,
  @path id: string,
  @body body: RevokeSessionHttpRequest
): RevokeSessionSuccess_204 | RevokeSessionError400 | RevokeSessionError403;

model RevokeUserSessionsHttpRequest {
  user_id: string;
}

model RevokeUserSessionsSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeUserSessionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeUserSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeUserSessionsError403Body;
}

@doc("admin が対象ユーザーの全セッションを失効する。\"全セッションを終了\"\n操作。RevokeMyOtherSessions と異なり操作者自身のセッションではないため除外対象は\n無く、対象ユーザーの有効な全セッションを tombstone にする。各対象は idempotent に\n失効する。")
@TypeSpec.OpenAPI.operationId("RevokeUserSessions")
@route("/api/admin/v1/users/{user_id}/sessions/revoke_all")
@post
op RevokeUserSessions(
  @path user_id: string,
  @body body: RevokeUserSessionsHttpRequest
): RevokeUserSessionsSuccess_204 | RevokeUserSessionsError403;

model StartStepUpAuthenticationHttpResponse {
  methods: IdMagic.Contract.StepUpStartResponse;
}

model StartStepUpAuthenticationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartStepUpAuthenticationHttpResponse;
}

union StartStepUpAuthenticationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartStepUpAuthenticationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartStepUpAuthenticationError403Body;
}

@doc("高 sensitivity 操作の前段として step-up 再認証を開始する (self-service)。subject 本人のセッションに固定し、利用可能な factor (password 常時 / totp は\nenrolled 時) を返す。本 interface 自体には step-up を要求しない (再認証の入口のため)。")
@TypeSpec.OpenAPI.operationId("StartStepUpAuthentication")
@route("/api/account/v1/step_up/start")
@post
op StartStepUpAuthentication(

): StartStepUpAuthenticationSuccess_200 | StartStepUpAuthenticationError403;

model CompleteStepUpAuthenticationHttpRequest {
  request: IdMagic.Contract.StepUpCompleteRequest;
}

model CompleteStepUpAuthenticationSuccess_200 {
  @statusCode statusCode: 200;
}

union CompleteStepUpAuthenticationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CompleteStepUpAuthenticationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CompleteStepUpAuthenticationError400Body;
}

union CompleteStepUpAuthenticationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CompleteStepUpAuthenticationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CompleteStepUpAuthenticationError403Body;
}

@doc("提示された factor (password または totp コード) を検証し、成立すればセッションに\nstep_up_at を刻む。以降 recency 窓 (5 分) 内は step_up: required の\n操作を許可する。検証失敗は AccessDeniedError。")
@TypeSpec.OpenAPI.operationId("CompleteStepUpAuthentication")
@route("/api/account/v1/step_up/complete")
@post
op CompleteStepUpAuthentication(
  @body body: CompleteStepUpAuthenticationHttpRequest
): CompleteStepUpAuthenticationSuccess_200 | CompleteStepUpAuthenticationError400 | CompleteStepUpAuthenticationError403;

model IssueMfaEnrollmentBypassHttpRequest {
  user_id: string;
  request: IdMagic.Contract.MfaEnrollmentBypassIssueRequest;
}

model IssueMfaEnrollmentBypassHttpResponse {
  response: IdMagic.Contract.MfaEnrollmentBypassResponse;
}

model IssueMfaEnrollmentBypassSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IssueMfaEnrollmentBypassHttpResponse;
}

union IssueMfaEnrollmentBypassError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model IssueMfaEnrollmentBypassError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: IssueMfaEnrollmentBypassError400Body;
}

union IssueMfaEnrollmentBypassError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model IssueMfaEnrollmentBypassError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: IssueMfaEnrollmentBypassError403Body;
}

union IssueMfaEnrollmentBypassError422Body {
  IdMagic.Contract.MfaEnrollmentNotAllowedError,
}

model IssueMfaEnrollmentBypassError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: IssueMfaEnrollmentBypassError422Body;
}

@doc("テナント管理者が同一テナントの MFA 未登録 active user に短期・単発の enrollment bypass を発行する。同一 user の既存未消費 bypass は取り消して置換する。ログイン免除ではなく登録専用 pending flow への承認である。")
@TypeSpec.OpenAPI.operationId("IssueMfaEnrollmentBypass")
@route("/api/admin/v1/users/{user_id}/mfa-enrollment-bypass")
@post
op IssueMfaEnrollmentBypass(
  @path user_id: string,
  @body body: IssueMfaEnrollmentBypassHttpRequest
): IssueMfaEnrollmentBypassSuccess_200 | IssueMfaEnrollmentBypassError400 | IssueMfaEnrollmentBypassError403 | IssueMfaEnrollmentBypassError422;

model RevokeMfaEnrollmentBypassSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMfaEnrollmentBypassError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMfaEnrollmentBypassError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMfaEnrollmentBypassError403Body;
}

@doc("テナント管理者が対象 user の未消費 enrollment bypass を取り消す。既に消費・期限切れの場合もログインを許可せず冪等に終了する。")
@TypeSpec.OpenAPI.operationId("RevokeMfaEnrollmentBypass")
@route("/api/admin/v1/users/{user_id}/mfa-enrollment-bypass")
@delete
op RevokeMfaEnrollmentBypass(
  @path user_id: string
): RevokeMfaEnrollmentBypassSuccess_204 | RevokeMfaEnrollmentBypassError403;

model ResetUserAuthenticatorsHttpRequest {
  user_id: string;
  request: IdMagic.Contract.AuthenticatorResetRequest;
}

model ResetUserAuthenticatorsHttpResponse {
  response: IdMagic.Contract.AuthenticatorResetResponse;
}

model ResetUserAuthenticatorsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ResetUserAuthenticatorsHttpResponse;
}

union ResetUserAuthenticatorsError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ResetUserAuthenticatorsError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ResetUserAuthenticatorsError400Body;
}

union ResetUserAuthenticatorsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ResetUserAuthenticatorsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResetUserAuthenticatorsError403Body;
}

union ResetUserAuthenticatorsError422Body {
  IdMagic.Contract.AuthenticatorResetNotAllowedError,
}

model ResetUserAuthenticatorsError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: ResetUserAuthenticatorsError422Body;
}

@doc("テナント管理者が対象 user の認証器 (TOTP factor / WebAuthn credential /\nrecovery code) から選択した種別を削除し、mfa_enrolled を再計算する (緊急\nbackstop)。管理者は新しい factor を代わりに登録できず、削除と再登録要求に限定する。\nTOTP と WebAuthn が両方無くなった場合は既存の管理者承認 enrollment bypass\n(IssueMfaEnrollmentBypass と同じ機構) を自動発行して置換し、次回ログインを\nfail-closed な enrollment-required flow へ誘導する。片方の要素のみ削除するなど mfa_enrolled\nが true のまま残る場合は bypass を発行せず、残存要素で通常ログインを継続できる。")
@TypeSpec.OpenAPI.operationId("ResetUserAuthenticators")
@route("/api/admin/v1/users/{user_id}/authenticator-reset")
@post
op ResetUserAuthenticators(
  @path user_id: string,
  @body body: ResetUserAuthenticatorsHttpRequest
): ResetUserAuthenticatorsSuccess_200 | ResetUserAuthenticatorsError400 | ResetUserAuthenticatorsError403 | ResetUserAuthenticatorsError422;

model RequestPasswordResetHttpRequest {
  request: IdMagic.Contract.BrowserForgotPasswordRequest;
}

model RequestPasswordResetSuccess_200 {
  @statusCode statusCode: 200;
}

union RequestPasswordResetError429Body {
  IdMagic.Contract.RateLimitedError,
}

model RequestPasswordResetError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/problem+json";
  @body body: RequestPasswordResetError429Body;
}

@doc("未認証ユーザーがパスワードリセットリンクの送信を要求する。\nanti-enumeration のため、email の存在有無に関わらず常に 204 を返し、\nverified email を持つ user が見つかった場合のみ token を発行して送信する。\n文面は Tenancy の通知テンプレートカタログ (template_key=PasswordReset) から\n解決する。locale は受信者 User の locale 属性 → テナントの default_locale →\nシステム既定 locale の順に決め、テナント上書きがあれば組込み既定より優先する。\nプレーンテキストと HTML の両方を必ず生成し、リセットリンクは\nリクエストの発行元 URL から組み立てる (テンプレート側に URL 結合をさせない)。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("RequestPasswordReset")
@route("/api/auth/forgot_password")
@post
op RequestPasswordReset(
  @body body: RequestPasswordResetHttpRequest
): RequestPasswordResetSuccess_200 | RequestPasswordResetError429;

model ResetPasswordWithTokenHttpRequest {
  request: IdMagic.Contract.BrowserResetPasswordRequest;
}

model ResetPasswordWithTokenSuccess_200 {
  @statusCode statusCode: 200;
}

union ResetPasswordWithTokenError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ResetPasswordWithTokenError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ResetPasswordWithTokenError400Body;
}

union ResetPasswordWithTokenError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ResetPasswordWithTokenError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ResetPasswordWithTokenError403Body;
}

union ResetPasswordWithTokenError422Body {
  IdMagic.Contract.PasswordReuseError,
}

model ResetPasswordWithTokenError422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: ResetPasswordWithTokenError422Body;
}

@doc("reset リンクから戻ってきたユーザーが新パスワードを設定する。\ntoken を単発消費し (期限 1800 秒、consumed_at 設定済みは再利用不可)、12〜128 文字の\n長さ制約と直近 history_depth (5) 件のハッシュとの不一致 (PasswordHistoryNoReuse) を\n通過した new_password で User.password_hash を更新する。成功時 PasswordChanged を emit。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ResetPasswordWithToken")
@route("/api/auth/reset_password")
@post
op ResetPasswordWithToken(
  @body body: ResetPasswordWithTokenHttpRequest
): ResetPasswordWithTokenSuccess_200 | ResetPasswordWithTokenError400 | ResetPasswordWithTokenError403 | ResetPasswordWithTokenError422;

model DiscoverIdentityProvidersHttpResponse {
  providers: IdMagic.Contract.ExternalIdentityProviderSummary[];
}

model DiscoverIdentityProvidersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DiscoverIdentityProvidersHttpResponse;
}

@doc("login UI が request tenant で Active な upstream provider の表示用情報だけを取得する。secret、endpoint、certificate、link policy は公開しない。optional email/domain hint は管理時に検証済み routing rule だけに適用する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DiscoverIdentityProviders")
@route("/api/auth/federation/providers")
@get
op DiscoverIdentityProviders(
  @query hint?: string
): DiscoverIdentityProvidersSuccess_200;

model StartFederatedLoginHttpResponse {
  response: IdMagic.Contract.FederatedLoginStartResponse;
}

model StartFederatedLoginSuccess_200 {
  @statusCode statusCode: 200;
  @body body: StartFederatedLoginHttpResponse;
}

union StartFederatedLoginError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartFederatedLoginError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartFederatedLoginError400Body;
}

union StartFederatedLoginError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartFederatedLoginError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartFederatedLoginError403Body;
}

@doc("Active connection を tenant 内で解決し、OIDC state / nonce / PKCE または SAML RelayState / AuthnRequest ID を単発 FederatedLoginAttempt に保存して、管理時検証済み upstream endpoint への redirect を返す。任意 URL は受け取らない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartFederatedLogin")
@route("/api/auth/federation/start")
@get
op StartFederatedLogin(
  @query provider_id: string,
  @query return_to?: string,
  @query link?: boolean
): StartFederatedLoginSuccess_200 | StartFederatedLoginError400 | StartFederatedLoginError403;

model CompleteFederatedLogin1HttpResponse {
  completion: IdMagic.Contract.FederatedLoginCompletion;
}

model CompleteFederatedLogin1Success_200 {
  @statusCode statusCode: 200;
  @body body: CompleteFederatedLogin1HttpResponse;
}

union CompleteFederatedLogin1Error400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CompleteFederatedLogin1Error400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CompleteFederatedLogin1Error400Body;
}

union CompleteFederatedLogin1Error403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CompleteFederatedLogin1Error403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CompleteFederatedLogin1Error403Body;
}

union CompleteFederatedLogin1Error422Body {
  IdMagic.Contract.QuotaExceededError,
}

model CompleteFederatedLogin1Error422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CompleteFederatedLogin1Error422Body;
}

@doc("callback を single-use login attempt と相関し、OIDC token または SAML response の trust、issuer、audience、nonce/InResponseTo、time、replay を検証する。既存 link、明示 policy 下の verified-email link、明示 JIT の順で local User を解決し、LoginSession 発行へ渡す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteFederatedLogin")
@route("/api/auth/federation/oidc/callback")
@get
op CompleteFederatedLogin1(
  @query provider_id: string,
  @query state: string,
  @query response: string
): CompleteFederatedLogin1Success_200 | CompleteFederatedLogin1Error400 | CompleteFederatedLogin1Error403 | CompleteFederatedLogin1Error422;

model CompleteFederatedLogin2HttpRequest {
  provider_id: string;
  state: string;
  response: string;
}

model CompleteFederatedLogin2HttpResponse {
  completion: IdMagic.Contract.FederatedLoginCompletion;
}

model CompleteFederatedLogin2Success_200 {
  @statusCode statusCode: 200;
  @body body: CompleteFederatedLogin2HttpResponse;
}

union CompleteFederatedLogin2Error400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CompleteFederatedLogin2Error400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CompleteFederatedLogin2Error400Body;
}

union CompleteFederatedLogin2Error403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CompleteFederatedLogin2Error403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CompleteFederatedLogin2Error403Body;
}

union CompleteFederatedLogin2Error422Body {
  IdMagic.Contract.QuotaExceededError,
}

model CompleteFederatedLogin2Error422 {
  @statusCode statusCode: 422;
  @header contentType: "application/problem+json";
  @body body: CompleteFederatedLogin2Error422Body;
}

@doc("callback を single-use login attempt と相関し、OIDC token または SAML response の trust、issuer、audience、nonce/InResponseTo、time、replay を検証する。既存 link、明示 policy 下の verified-email link、明示 JIT の順で local User を解決し、LoginSession 発行へ渡す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteFederatedLogin")
@route("/api/auth/federation/saml/callback")
@post
op CompleteFederatedLogin2(
  @body body: CompleteFederatedLogin2HttpRequest
): CompleteFederatedLogin2Success_200 | CompleteFederatedLogin2Error400 | CompleteFederatedLogin2Error403 | CompleteFederatedLogin2Error422;

model ListLinkedExternalIdentitiesHttpResponse {
  identities: IdMagic.Contract.FederatedIdentity[];
}

model ListLinkedExternalIdentitiesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListLinkedExternalIdentitiesHttpResponse;
}

union ListLinkedExternalIdentitiesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListLinkedExternalIdentitiesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListLinkedExternalIdentitiesError403Body;
}

@doc("ResourceOwner が自身に結び付く external identity の provider 表示名、protocol、linked_at、last_login_at を取得する。external subject と token は返さない。")
@TypeSpec.OpenAPI.operationId("ListLinkedExternalIdentities")
@route("/api/account/v1/linked-identities")
@get
op ListLinkedExternalIdentities(

): ListLinkedExternalIdentitiesSuccess_200 | ListLinkedExternalIdentitiesError403;

model LinkExternalIdentityHttpRequest {
  provider_id: string;
}

model LinkExternalIdentityHttpResponse {
  response: IdMagic.Contract.FederatedLoginStartResponse;
}

model LinkExternalIdentitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: LinkExternalIdentityHttpResponse;
}

union LinkExternalIdentityError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model LinkExternalIdentityError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: LinkExternalIdentityError400Body;
}

union LinkExternalIdentityError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model LinkExternalIdentityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: LinkExternalIdentityError403Body;
}

@doc("直近5分以内に step-up 済みの ResourceOwner が provider の外部認証を開始し、callback で得た subject を自身へ明示 link する。既に他 user に link 済みの subject は付け替えない。")
@TypeSpec.OpenAPI.operationId("LinkExternalIdentity")
@route("/api/account/v1/linked-identities/{provider_id}")
@post
op LinkExternalIdentity(
  @path provider_id: string,
  @body body: LinkExternalIdentityHttpRequest
): LinkExternalIdentitySuccess_200 | LinkExternalIdentityError400 | LinkExternalIdentityError403;

model UnlinkExternalIdentitySuccess_204 {
  @statusCode statusCode: 204;
}

union UnlinkExternalIdentityError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UnlinkExternalIdentityError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UnlinkExternalIdentityError400Body;
}

union UnlinkExternalIdentityError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UnlinkExternalIdentityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UnlinkExternalIdentityError403Body;
}

@doc("直近5分以内に step-up 済みの ResourceOwner が自身の provider link を解除する。利用可能な password credential も他の link も無い場合は account lockout 防止のため拒否する。")
@TypeSpec.OpenAPI.operationId("UnlinkExternalIdentity")
@route("/api/account/v1/linked-identities/{provider_id}")
@delete
op UnlinkExternalIdentity(
  @path provider_id: string
): UnlinkExternalIdentitySuccess_204 | UnlinkExternalIdentityError400 | UnlinkExternalIdentityError403;

model ListIdentityProviderConnectionsHttpResponse {
  connections: IdMagic.Contract.IdentityProviderConnection[];
}

model ListIdentityProviderConnectionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ListIdentityProviderConnectionsHttpResponse;
}

union ListIdentityProviderConnectionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListIdentityProviderConnectionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListIdentityProviderConnectionsError403Body;
}

@doc("tenant administrator が connection 一覧を取得する。secret_reference と secret は返さない。")
@TypeSpec.OpenAPI.operationId("ListIdentityProviderConnections")
@route("/api/admin/v1/identity-providers")
@get
op ListIdentityProviderConnections(

): ListIdentityProviderConnectionsSuccess_200 | ListIdentityProviderConnectionsError403;

model CreateIdentityProviderConnectionHttpRequest {
  connection: IdMagic.Contract.IdentityProviderConnection;
}

model CreateIdentityProviderConnectionHttpResponse {
  connection: IdMagic.Contract.IdentityProviderConnection;
}

model CreateIdentityProviderConnectionSuccess_201 {
  @statusCode statusCode: 201;
  @body body: CreateIdentityProviderConnectionHttpResponse;
}

union CreateIdentityProviderConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CreateIdentityProviderConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CreateIdentityProviderConnectionError400Body;
}

union CreateIdentityProviderConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model CreateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateIdentityProviderConnectionError403Body;
}

@doc("tenant administrator が connection を作る。作成直後の初期状態は Disabled。OIDC issuer または SAML metadata を管理時に検証し last-known-good trust material を保存する。client secret は secret_reference で受け取り、書き込み専用のエンベロープ暗号化保存にまわす。")
@TypeSpec.OpenAPI.operationId("CreateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers")
@post
op CreateIdentityProviderConnection(
  @body body: CreateIdentityProviderConnectionHttpRequest
): CreateIdentityProviderConnectionSuccess_201 | CreateIdentityProviderConnectionError400 | CreateIdentityProviderConnectionError403;

model UpdateIdentityProviderConnectionHttpRequest {
  provider_id: string;
  connection: IdMagic.Contract.IdentityProviderConnection;
}

model UpdateIdentityProviderConnectionHttpResponse {
  connection: IdMagic.Contract.IdentityProviderConnection;
}

model UpdateIdentityProviderConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: UpdateIdentityProviderConnectionHttpResponse;
}

union UpdateIdentityProviderConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateIdentityProviderConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateIdentityProviderConnectionError400Body;
}

union UpdateIdentityProviderConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model UpdateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateIdentityProviderConnectionError403Body;
}

@doc("tenant administrator が connection の display、mapping、policy、secret reference、trust source を更新する。issuer / endpoint / certificate / protocol などの trust source を変更したときだけ Active を Disabled に落とし、display_name や claim_mapping など非 trust フィールドの変更では状態を保持する。trust source 変更時に再検証が失敗した場合は last-known-good を維持する。")
@TypeSpec.OpenAPI.operationId("UpdateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}")
@put
op UpdateIdentityProviderConnection(
  @path provider_id: string,
  @body body: UpdateIdentityProviderConnectionHttpRequest
): UpdateIdentityProviderConnectionSuccess_200 | UpdateIdentityProviderConnectionError400 | UpdateIdentityProviderConnectionError403;

model DeleteIdentityProviderConnectionSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteIdentityProviderConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DeleteIdentityProviderConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DeleteIdentityProviderConnectionError400Body;
}

union DeleteIdentityProviderConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DeleteIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteIdentityProviderConnectionError403Body;
}

@doc("tenant administrator が connection を状態を問わずいつでも削除する。link が残る場合は削除を拒否し、先に明示 cleanup を要求する。")
@TypeSpec.OpenAPI.operationId("DeleteIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}")
@delete
op DeleteIdentityProviderConnection(
  @path provider_id: string
): DeleteIdentityProviderConnectionSuccess_204 | DeleteIdentityProviderConnectionError400 | DeleteIdentityProviderConnectionError403;

model ActivateIdentityProviderConnectionHttpRequest {
  provider_id: string;
}

model ActivateIdentityProviderConnectionSuccess_204 {
  @statusCode statusCode: 204;
}

union ActivateIdentityProviderConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ActivateIdentityProviderConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ActivateIdentityProviderConnectionError400Body;
}

union ActivateIdentityProviderConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ActivateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ActivateIdentityProviderConnectionError403Body;
}

@doc("trust validation と接続 test に成功した Disabled connection を Active にする。")
@TypeSpec.OpenAPI.operationId("ActivateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/activate")
@post
op ActivateIdentityProviderConnection(
  @path provider_id: string,
  @body body: ActivateIdentityProviderConnectionHttpRequest
): ActivateIdentityProviderConnectionSuccess_204 | ActivateIdentityProviderConnectionError400 | ActivateIdentityProviderConnectionError403;

model DisableIdentityProviderConnectionHttpRequest {
  provider_id: string;
}

model DisableIdentityProviderConnectionSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableIdentityProviderConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model DisableIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableIdentityProviderConnectionError403Body;
}

@doc("Active connection を Disabled にし、新規 routing と callback completion を直ちに拒否する。")
@TypeSpec.OpenAPI.operationId("DisableIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/disable")
@post
op DisableIdentityProviderConnection(
  @path provider_id: string,
  @body body: DisableIdentityProviderConnectionHttpRequest
): DisableIdentityProviderConnectionSuccess_204 | DisableIdentityProviderConnectionError403;

model RefreshIdentityProviderMetadataHttpRequest {
  provider_id: string;
}

model RefreshIdentityProviderMetadataHttpResponse {
  connection: IdMagic.Contract.IdentityProviderConnection;
}

model RefreshIdentityProviderMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: RefreshIdentityProviderMetadataHttpResponse;
}

union RefreshIdentityProviderMetadataError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RefreshIdentityProviderMetadataError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RefreshIdentityProviderMetadataError400Body;
}

union RefreshIdentityProviderMetadataError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RefreshIdentityProviderMetadataError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RefreshIdentityProviderMetadataError403Body;
}

@doc("tenant administrator が固定済み issuer / metadata source から discovery/JWKS または SAML metadata を再取得し、SSRF と trust validation 成功時だけ last-known-good を更新する。")
@TypeSpec.OpenAPI.operationId("RefreshIdentityProviderMetadata")
@route("/api/admin/v1/identity-providers/{provider_id}/refresh")
@post
op RefreshIdentityProviderMetadata(
  @path provider_id: string,
  @body body: RefreshIdentityProviderMetadataHttpRequest
): RefreshIdentityProviderMetadataSuccess_200 | RefreshIdentityProviderMetadataError400 | RefreshIdentityProviderMetadataError403;

model TestIdentityProviderConnectionHttpRequest {
  provider_id: string;
}

model TestIdentityProviderConnectionHttpResponse {
  result: IdMagic.Contract.IdentityProviderConnectionTestResult;
}

model TestIdentityProviderConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: TestIdentityProviderConnectionHttpResponse;
}

union TestIdentityProviderConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model TestIdentityProviderConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: TestIdentityProviderConnectionError400Body;
}

union TestIdentityProviderConnectionError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model TestIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: TestIdentityProviderConnectionError403Body;
}

@doc("保存済み last-known-good trust source に対して実際に到達性・解決可能性を検査する。OIDC は authorization_endpoint/token_endpoint/jwks_uri への到達性と secret_reference の解決可能性を、SAML は保存済み証明書の X.509 パース可否と有効期限を確認する。secret の実値、token、certificate 本文は返さない。")
@TypeSpec.OpenAPI.operationId("TestIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/test")
@post
op TestIdentityProviderConnection(
  @path provider_id: string,
  @body body: TestIdentityProviderConnectionHttpRequest
): TestIdentityProviderConnectionSuccess_200 | TestIdentityProviderConnectionError400 | TestIdentityProviderConnectionError403;

model PreviewIdentityProviderMappingHttpRequest {
  provider_id: string;
  claims: unknown;
}

model PreviewIdentityProviderMappingHttpResponse {
  preview: IdMagic.Contract.ExternalIdentityClaimsPreview;
}

model PreviewIdentityProviderMappingSuccess_200 {
  @statusCode statusCode: 200;
  @body body: PreviewIdentityProviderMappingHttpResponse;
}

union PreviewIdentityProviderMappingError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model PreviewIdentityProviderMappingError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: PreviewIdentityProviderMappingError400Body;
}

union PreviewIdentityProviderMappingError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model PreviewIdentityProviderMappingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewIdentityProviderMappingError403Body;
}

@doc("tenant administrator が保存済み test response の検証済み claim を mapping し、token/assertionを含まない正規化 preview を取得する。")
@TypeSpec.OpenAPI.operationId("PreviewIdentityProviderMapping")
@route("/api/admin/v1/identity-providers/{provider_id}/mapping-preview")
@post
op PreviewIdentityProviderMapping(
  @path provider_id: string,
  @body body: PreviewIdentityProviderMappingHttpRequest
): PreviewIdentityProviderMappingSuccess_200 | PreviewIdentityProviderMappingError400 | PreviewIdentityProviderMappingError403;
}

}
