import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

@tag("Authentication")
namespace IdMagic.Authentication {

namespace Operations {
model GetAccountContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountContextResponse;
}

union GetAccountContextError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountContextError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountContextError403Body;
}

@doc("認証済みのブラウザーのセッションが共有する、ポータルの起動情報と CSRF トークンを読む。Bearer トークンは、管理ポータルには `idmagic.admin`、アカウントポータルには `idmagic.account`、セルフサービス API のクライアントには `account:read` のいずれかを含まなければならない。認証の途中のセッションと、ほかのスコープしか含まない Bearer トークンは、どちらも拒否する。")
@TypeSpec.OpenAPI.operationId("GetAccountContext")
@route("/api/auth/account")
@get
op GetAccountContext(

): GetAccountContextSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | GetAccountContextError403;

model GetPasswordResetContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserCsrfContext;
}

@doc("未認証のパスワードリセットの画面が、その POST を守る CSRF トークンを読む。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetPasswordResetContext")
@route("/api/auth/password-reset-context")
@get
op GetPasswordResetContext(

): GetPasswordResetContextSuccess_200;

model StartStepUpWebAuthnChallengeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WebAuthnAssertionOptions;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartStepUpWebAuthnChallengeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartStepUpWebAuthnChallengeError403Body;
}

union StartStepUpWebAuthnChallengeError503Body {
  IdMagic.Contract.WebAuthnUnavailableError,
}

model StartStepUpWebAuthnChallengeError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: StartStepUpWebAuthnChallengeError503Body;
}

@doc("認証済みのセッションに結び付く、ステップアップのための WebAuthn のアサーションのチャレンジを発行する。CSRF を検証し、WebAuthn が無効のときはフェイルクローズで失敗する。")
@TypeSpec.OpenAPI.operationId("StartStepUpWebAuthnChallenge")
@route("/api/account/v1/step-up/webauthn/challenge")
@post
op StartStepUpWebAuthnChallenge(

): StartStepUpWebAuthnChallengeSuccess_200 | StartStepUpWebAuthnChallengeError400 | StartStepUpWebAuthnChallengeError403 | StartStepUpWebAuthnChallengeError503;

model SubmitBrowserLoginHttpRequest {
  username: string;
  password: string;
  @doc("サインイン後の移動先。同じレルムの `/admin` の下に限る。")
  return_to?: string;
}

model SubmitBrowserLoginSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
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
  @header contentType: "application/json";
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: SubmitBrowserLoginError429Body;
}

@doc("ブラウザーのパスワードのステップを SPA の JSON API で処理する。MFA が必要で要素を登録していないとき、強制の開始前は、パスワードのセッションでのサインインを許し、前もって登録するようユーザーを促す。開始後は、有効な一回限りの登録の回避と猶予の期限の両方を満たすときだけ、同じ LoginSession を Enrollment の保留へ移す。それ以外はフェイルクローズで失敗し、サインインは完了しない。クラスター全体のカウンターで、アカウントごと（900 秒のウィンドウで 10 回、900 秒のロックアウト）と IP ごと（900 秒のウィンドウで 30 回、900 秒のロックアウト）に抑制を適用し、共有のカウンターのストアに到達できないときもフェイルクローズのままにする。この失敗回数による抑制とは別に、IP とテナントをキーとする EndpointRateLimitPolicy が、成功したかどうかにかかわらずすべてのリクエストを数える。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserLogin")
@route("/api/auth/login")
@post
op SubmitBrowserLogin(
  @body body: SubmitBrowserLoginHttpRequest
): SubmitBrowserLoginSuccess_200 | SubmitBrowserLoginError400 | IdMagic.Contract.AuthenticationRequiredResponse | SubmitBrowserLoginError403 | SubmitBrowserLoginError422 | SubmitBrowserLoginError429;

model StartBrowserMfaEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TOTPEnrollmentStart;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
  IdMagic.Contract.MfaEnrollmentNotAllowedError,
}

model StartBrowserMfaEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartBrowserMfaEnrollmentError403Body;
}

union StartBrowserMfaEnrollmentError409Body {
  IdMagic.Contract.MfaAlreadyEnrolledError,
}

model StartBrowserMfaEnrollmentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: StartBrowserMfaEnrollmentError409Body;
}

@doc("TOTP の登録を開始できるのは、`pending_purpose=Enrollment` で期限切れでない LoginSession だけである。通常の未認証のセッション、Challenge の保留中のセッション、期限切れのセッションは 403 で拒否する。登録済みのユーザーは代わりに 409 で拒否する。これは管理とセルフサービスのエンドポイントが同じ条件に使うステータスであり、サインインの画面が「することがない」と「管理者に連絡する」を区別できる。シークレットは登録が確定するまで永続化しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartBrowserMfaEnrollment")
@route("/api/auth/mfa/enrollment/totp/start")
@post
op StartBrowserMfaEnrollment(

): StartBrowserMfaEnrollmentSuccess_200 | StartBrowserMfaEnrollmentError400 | StartBrowserMfaEnrollmentError403 | StartBrowserMfaEnrollmentError409;

model ConfirmBrowserMfaEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
  IdMagic.Contract.MfaEnrollmentNotAllowedError,
}

model ConfirmBrowserMfaEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfirmBrowserMfaEnrollmentError403Body;
}

union ConfirmBrowserMfaEnrollmentError409Body {
  IdMagic.Contract.MfaAlreadyEnrolledError,
}

model ConfirmBrowserMfaEnrollmentError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: ConfirmBrowserMfaEnrollmentError409Body;
}

@doc("`pending_purpose` が Enrollment で期限切れでない LoginSession で、TOTP の所持の証明を検証し、要素を保存し、同じ LoginSession に `otp` を加えて MFA を満たした状態へ引き上げ、元の認可のトランザクションを再開する。登録済みのユーザーは、開始のエンドポイントと同じく 409 で拒否する。通常のセルフサービスの登録のエンドポイントとは権限を共有しない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ConfirmBrowserMfaEnrollment")
@route("/api/auth/mfa/enrollment/totp/confirm")
@post
op ConfirmBrowserMfaEnrollment(
  @body body: IdMagic.Contract.BrowserMfaEnrollmentConfirmRequest
): ConfirmBrowserMfaEnrollmentSuccess_200 | ConfirmBrowserMfaEnrollmentError400 | IdMagic.Contract.AuthenticationRequiredResponse | ConfirmBrowserMfaEnrollmentError403 | ConfirmBrowserMfaEnrollmentError409;

model SubmitBrowserTotpSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserTotpError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserTotpError403Body;
}

union SubmitBrowserTotpError503Body {
  IdMagic.Contract.MfaUnavailableError,
}

model SubmitBrowserTotpError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserTotpError503Body;
}

@doc("SPA の JSON API で TOTP のコードを検証する。成功すると LoginSession を完全に認証した状態へ引き上げ、フローは同意と認可コードの発行へ進む。ほかの要素にはそれぞれのエンドポイントがある。対象のユーザーは未認証の保留中のログインセッションの Cookie で解決するので、認証は不要である。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserTotp")
@route("/api/auth/totp")
@post
op SubmitBrowserTotp(
  @body body: IdMagic.Contract.BrowserTotpRequest
): SubmitBrowserTotpSuccess_200 | SubmitBrowserTotpError400 | IdMagic.Contract.AuthenticationRequiredResponse | SubmitBrowserTotpError403 | SubmitBrowserTotpError503;

union ChangePasswordError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ChangePasswordError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ChangePasswordError400Body;
}

union ChangePasswordError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.StepUpRequiredError,
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

model ChangePasswordSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("認証済みのユーザーが自身のパスワードを変更する。`current_password` を検証し、`new_password` は 12 から 128 文字の長さの制約を満たし、直近の `history_depth`（5）個のハッシュと異なっていなければならない（PasswordHistoryNoReuse）。")
@TypeSpec.OpenAPI.operationId("ChangePassword")
@route("/api/auth/change-password")
@post
op ChangePassword(
  @body body: IdMagic.Contract.BrowserChangePasswordRequest
): ChangePasswordSuccess_204 | ChangePasswordError400 | IdMagic.Contract.AuthenticationRequiredResponse | ChangePasswordError403 | ChangePasswordError422;

model GetAccountSecuritySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountSecurityResponse;
}

union GetAccountSecurityError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountSecurityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountSecurityError403Body;
}

@doc("認証済みのユーザーが、自身のセキュリティの概要を読む（セルフサービス）。対象は `subject.id` に固定される。パスワードの変更時刻と、MFA（TOTP）を登録しているかどうかを返す。")
@TypeSpec.OpenAPI.operationId("GetAccountSecurity")
@route("/api/account/v1/security")
@get
op GetAccountSecurity(

): GetAccountSecuritySuccess_200 | GetAccountSecurityError403;

model GetMyNotificationPreferencesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountNotificationPreferences;
}

union GetMyNotificationPreferencesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model GetMyNotificationPreferencesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetMyNotificationPreferencesError403Body;
}

@doc("認証済みのユーザーが、自身のセキュリティ通知の設定を読む（セルフサービス）。対象は `actor.sub` に固定される。すべての分類を仕様が定める安定した順序で返し、ユーザーが無効にできない分類は `mandatory=true` を含む。")
@TypeSpec.OpenAPI.operationId("GetMyNotificationPreferences")
@route("/api/account/v1/notification-preferences")
@get
op GetMyNotificationPreferences(

): GetMyNotificationPreferencesSuccess_200 | GetMyNotificationPreferencesError403;

model UpdateMyNotificationPreferencesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountNotificationPreferences;
}

union UpdateMyNotificationPreferencesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model UpdateMyNotificationPreferencesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: UpdateMyNotificationPreferencesError400Body;
}

union UpdateMyNotificationPreferencesError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateMyNotificationPreferencesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateMyNotificationPreferencesError403Body;
}

@doc("認証済みのユーザーが、自身のセキュリティ通知の設定を更新する（セルフサービス）。対象は `actor.sub` に固定される。通知を止めることは、アカウントを乗っ取った攻撃者が最初に行うことなので、最近のステップアップの再認証を求める。`disabled_categories` に必須の分類を載せたリクエストは 400 で拒否し、一部だけを適用することはない。")
@TypeSpec.OpenAPI.operationId("UpdateMyNotificationPreferences")
@route("/api/account/v1/notification-preferences")
@put
op UpdateMyNotificationPreferences(
  @body body: IdMagic.Contract.UpdateNotificationPreferencesRequest
): UpdateMyNotificationPreferencesSuccess_200 | UpdateMyNotificationPreferencesError400 | UpdateMyNotificationPreferencesError403;

model StartTotpEnrollmentSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.TOTPEnrollmentStart;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartTotpEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartTotpEnrollmentError403Body;
}

@doc("認証済みのユーザーが TOTP の登録を開始する（セルフサービス）。新しい 20 バイトのシークレットと `otpauth` の URI を返し、何も永続化しない。すでに TOTP の要素を登録しているユーザーは拒否する。")
@TypeSpec.OpenAPI.operationId("StartTotpEnrollment")
@route("/api/account/v1/mfa/totp/enroll/start")
@post
op StartTotpEnrollment(

): StartTotpEnrollmentSuccess_200 | StartTotpEnrollmentError400 | StartTotpEnrollmentError403;

union ConfirmTotpEnrollmentError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model ConfirmTotpEnrollmentError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: ConfirmTotpEnrollmentError400Body;
}

union ConfirmTotpEnrollmentError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ConfirmTotpEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ConfirmTotpEnrollmentError403Body;
}

model ConfirmTotpEnrollmentSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("シークレットに対するコードで所持の証明を検証し、TOTP の要素を永続化する（セルフサービス）。対象は `subject.id` に固定され、確定すると `User.mfa_enrolled` を true にする。")
@TypeSpec.OpenAPI.operationId("ConfirmTotpEnrollment")
@route("/api/account/v1/mfa/totp/enroll/confirm")
@post
op ConfirmTotpEnrollment(
  @body body: IdMagic.Contract.TOTPEnrollmentConfirmRequest
): ConfirmTotpEnrollmentSuccess_204 | ConfirmTotpEnrollmentError400 | ConfirmTotpEnrollmentError403;

union RemoveTotpFactorError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RemoveTotpFactorError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RemoveTotpFactorError400Body;
}

union RemoveTotpFactorError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RemoveTotpFactorError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveTotpFactorError403Body;
}

model RemoveTotpFactorSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("有効な TOTP のコードを所持の証明として検証してから、TOTP の要素を削除する（セルフサービス）。対象は `subject.id` に固定され、削除すると `User.mfa_enrolled` を false に戻す。")
@TypeSpec.OpenAPI.operationId("RemoveTotpFactor")
@route("/api/account/v1/mfa/totp/remove")
@post
op RemoveTotpFactor(
  @body body: IdMagic.Contract.MfaFactorRemoveRequest
): RemoveTotpFactorSuccess_204 | RemoveTotpFactorError400 | RemoveTotpFactorError403;

model StartWebAuthnRegistrationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WebAuthnRegistrationOptions;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartWebAuthnRegistrationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartWebAuthnRegistrationError403Body;
}

@doc("認証済みのユーザーが、WebAuthn またはパスキーの登録を開始する（セルフサービス）。PublicKeyCredentialCreationOptions を返し、チャレンジをサーバー側のセッションに結び付ける。まだ何も永続化しない。")
@TypeSpec.OpenAPI.operationId("StartWebAuthnRegistration")
@route("/api/account/v1/mfa/webauthn/register/start")
@post
op StartWebAuthnRegistration(

): StartWebAuthnRegistrationSuccess_200 | StartWebAuthnRegistrationError400 | StartWebAuthnRegistrationError403;

union FinishWebAuthnRegistrationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model FinishWebAuthnRegistrationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: FinishWebAuthnRegistrationError400Body;
}

union FinishWebAuthnRegistrationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model FinishWebAuthnRegistrationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: FinishWebAuthnRegistrationError403Body;
}

model FinishWebAuthnRegistrationSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("アテステーションを検証し、WebAuthn の資格情報を永続化する（セルフサービス）。対象は `subject.id` に固定され、最初の資格情報で `User.mfa_enrolled` を true にする。RP ID、オリジン、チャレンジをすべて検証する。")
@TypeSpec.OpenAPI.operationId("FinishWebAuthnRegistration")
@route("/api/account/v1/mfa/webauthn/register/finish")
@post
op FinishWebAuthnRegistration(
  @body body: IdMagic.Contract.WebAuthnRegistrationRequest
): FinishWebAuthnRegistrationSuccess_204 | FinishWebAuthnRegistrationError400 | FinishWebAuthnRegistrationError403;

union RemoveWebAuthnCredentialError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RemoveWebAuthnCredentialError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RemoveWebAuthnCredentialError400Body;
}

union RemoveWebAuthnCredentialError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RemoveWebAuthnCredentialError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RemoveWebAuthnCredentialError403Body;
}

model RemoveWebAuthnCredentialSuccess_204 {
  @statusCode statusCode: 204;
}

union RemoveWebAuthnCredentialError503Body {
  IdMagic.Contract.WebAuthnUnavailableError,
}

model RemoveWebAuthnCredentialError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: RemoveWebAuthnCredentialError503Body;
}

@doc("指定した WebAuthn の資格情報を削除する（セルフサービス。ステップアップが必要）。対象は `subject.id` に固定され、第二要素が残らないときは `User.mfa_enrolled` を false に戻す。")
@TypeSpec.OpenAPI.operationId("RemoveWebAuthnCredential")
@route("/api/account/v1/mfa/webauthn/remove")
@post
op RemoveWebAuthnCredential(
  @body body: IdMagic.Contract.WebAuthnCredentialRemoveRequest
): RemoveWebAuthnCredentialSuccess_204 | RemoveWebAuthnCredentialError400 | RemoveWebAuthnCredentialError403 | RemoveWebAuthnCredentialError503;

model StartBrowserWebAuthnSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WebAuthnAssertionOptions;
}

union StartBrowserWebAuthnError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartBrowserWebAuthnError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartBrowserWebAuthnError403Body;
}

union StartBrowserWebAuthnError404Body {
  IdMagic.Contract.WebAuthnNotEnrolledError,
}

model StartBrowserWebAuthnError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: StartBrowserWebAuthnError404Body;
}

union StartBrowserWebAuthnError503Body {
  IdMagic.Contract.WebAuthnUnavailableError,
}

model StartBrowserWebAuthnError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: StartBrowserWebAuthnError503Body;
}

@doc("SPA の JSON API で WebAuthn のアサーションのチャレンジを発行する。保留中のログインセッションが `sub` を解決し、PublicKeyCredentialRequestOptions を返す。チャレンジはそのセッションに結び付く。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartBrowserWebAuthn")
@route("/api/auth/webauthn/challenge")
@post
op StartBrowserWebAuthn(

): StartBrowserWebAuthnSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | StartBrowserWebAuthnError403 | StartBrowserWebAuthnError404 | StartBrowserWebAuthnError503;

model SubmitBrowserWebAuthnSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserWebAuthnError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserWebAuthnError403Body;
}

union SubmitBrowserWebAuthnError503Body {
  IdMagic.Contract.WebAuthnUnavailableError,
}

model SubmitBrowserWebAuthnError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserWebAuthnError503Body;
}

@doc("SPA の JSON API で WebAuthn のアサーションを検証する。成功すると LoginSession を完全に認証した状態へ引き上げ、フローは同意と認可コードの発行へ進む。後退する `sign_count` は複製として拒否する（`sign_count_regression=reject`）。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserWebAuthn")
@route("/api/auth/webauthn")
@post
op SubmitBrowserWebAuthn(
  @body body: IdMagic.Contract.BrowserWebAuthnRequest
): SubmitBrowserWebAuthnSuccess_200 | SubmitBrowserWebAuthnError400 | IdMagic.Contract.AuthenticationRequiredResponse | SubmitBrowserWebAuthnError403 | SubmitBrowserWebAuthnError503;

model SubmitBrowserRecoveryCodeSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserFlowResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserRecoveryCodeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserRecoveryCodeError403Body;
}

union SubmitBrowserRecoveryCodeError503Body {
  IdMagic.Contract.RecoveryUnavailableError,
}

model SubmitBrowserRecoveryCodeError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserRecoveryCodeError503Body;
}

@doc("SPA の JSON API で、バックアップのリカバリーコードを第二要素として検証する。成功すると LoginSession を完全に認証した状態へ引き上げ、使用したコードは二度と使えない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserRecoveryCode")
@route("/api/auth/recovery-code")
@post
op SubmitBrowserRecoveryCode(
  @body body: IdMagic.Contract.BrowserRecoveryCodeRequest
): SubmitBrowserRecoveryCodeSuccess_200 | SubmitBrowserRecoveryCodeError400 | IdMagic.Contract.AuthenticationRequiredResponse | SubmitBrowserRecoveryCodeError403 | SubmitBrowserRecoveryCodeError503;

model GenerateRecoveryCodesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.RecoveryCodesResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model GenerateRecoveryCodesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GenerateRecoveryCodesError403Body;
}

union GenerateRecoveryCodesError503Body {
  IdMagic.Contract.RecoveryUnavailableError,
}

model GenerateRecoveryCodesError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: GenerateRecoveryCodesError503Body;
}

@doc("認証済みのユーザーが、バックアップのリカバリーコードを生成または再生成する（セルフサービス。ステップアップが必要）。平文は一度だけ返し、ハッシュだけを保存する。既存の組は丸ごと置き換える（10 個のコード）。")
@TypeSpec.OpenAPI.operationId("GenerateRecoveryCodes")
@route("/api/account/v1/mfa/recovery-codes/generate")
@post
op GenerateRecoveryCodes(

): GenerateRecoveryCodesSuccess_200 | GenerateRecoveryCodesError400 | GenerateRecoveryCodesError403 | GenerateRecoveryCodesError503;

union RevokeRecoveryCodesError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeRecoveryCodesError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeRecoveryCodesError400Body;
}

union RevokeRecoveryCodesError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeRecoveryCodesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeRecoveryCodesError403Body;
}

model RevokeRecoveryCodesSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeRecoveryCodesError503Body {
  IdMagic.Contract.RecoveryUnavailableError,
}

model RevokeRecoveryCodesError503 {
  @statusCode statusCode: 503;
  @header contentType: "application/problem+json";
  @body body: RevokeRecoveryCodesError503Body;
}

@doc("認証済みのユーザーが、自身のバックアップのリカバリーコードを明示的に無効にする（セルフサービス。ステップアップが必要）。対象は `subject.id` に固定され、有効なコードをすべて削除する。")
@TypeSpec.OpenAPI.operationId("RevokeRecoveryCodes")
@route("/api/account/v1/mfa/recovery-codes/revoke")
@post
op RevokeRecoveryCodes(

): RevokeRecoveryCodesSuccess_204 | RevokeRecoveryCodesError400 | RevokeRecoveryCodesError403 | RevokeRecoveryCodesError503;

model ListMySignInActivitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserSignInActivityResponse;
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

@doc("認証済みのユーザーが、自身の最近のサインインの履歴を読む（セルフサービス）。対象は `subject.id` に固定される。既存の監査イベントのストアから、UserAuthenticated のイベントを新しい順に最大 `limit` 件返す。`limit` のデフォルトは 10、上限は 50 である。")
@TypeSpec.OpenAPI.operationId("ListMySignInActivity")
@route("/api/account/v1/signin-activity")
@get
op ListMySignInActivity(
  @query limit?: int64
): ListMySignInActivitySuccess_200 | ListMySignInActivityError403 | ListMySignInActivityError400;

model ListUserSignInActivitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserSignInActivityResponse;
}

union ListUserSignInActivityError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListUserSignInActivityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListUserSignInActivityError403Body;
}

@doc("管理者が、対象のユーザーのサインインの履歴を読む。テナントの境界の中にとどまり、パスで指定したユーザーの UserAuthenticated のイベントを新しい順に最大 `limit` 件返す。`limit` のデフォルトは 10、上限は 50 である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:read"])
@TypeSpec.OpenAPI.operationId("ListUserSignInActivity")
@route("/api/admin/v1/users/{user_id}/signin-activity")
@get
op ListUserSignInActivity(
  @path user_id: string,
  @query limit?: int64
): ListUserSignInActivitySuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListUserSignInActivityError403;

model ListAuthenticationEventBucketsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuthenticationEventBucketListResponse;
}

union ListAuthenticationEventBucketsError403Body {
  IdMagic.Contract.InsufficientScopeError,
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

@doc("管理者が、認証イベントの集約（バケット）を `window_start` の降順に一覧する。攻撃の間に個々の行として残さず 5 分のウィンドウへ集約したサインインの失敗などを、テナントの境界の中で、双方向のキーセットページングで読む。カーソルはレスポンスの Link レスポンスヘッダー（`rel=\"prev\"` と `rel=\"next\"`）から得る。`limit` のデフォルトは 50、上限は 200 である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ListAuthenticationEventBuckets")
@route("/api/admin/v1/authentication-event-buckets")
@get
op ListAuthenticationEventBuckets(
  @query cursor?: string,
  @query limit?: int64
): ListAuthenticationEventBucketsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListAuthenticationEventBucketsError403 | ListAuthenticationEventBucketsError400;

model ListMySessionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountSessionListResponse;
}

union ListMySessionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMySessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMySessionsError403Body;
}

@doc("認証済みのユーザーが、自身の有効なセッションを読む（セルフサービス）。対象は `subject.id` に固定される。現在のセッションには `current=true` の印を付け、新しい順に返す。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMySessionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMySessionError403Body;
}

@doc("認証済みのユーザーが、自身のセッションの一つを失効させる（セルフサービス）。自身のものでないセッションは拒否する。失効は LoginSession に `revoked_at` と `revoke_reason=self_revoke` を設定する墓標であり、行は削除しない。失効済みのセッションの失効は冪等に成功し、`revoked_at` は最初の値を保つ。")
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyOtherSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyOtherSessionsError403Body;
}

@doc("認証済みのユーザーが、現在のセッションを除く自身のすべてのセッションを失効させる（セルフサービス）。「ほかのすべての場所からサインアウトする」操作である。各対象は `revoked_at` と `revoke_reason=self_revoke` を設定する墓標で、失効済みのものは冪等に飛ばす。")
@TypeSpec.OpenAPI.operationId("RevokeMyOtherSessions")
@route("/api/account/v1/sessions/revoke-others")
@post
op RevokeMyOtherSessions(

): RevokeMyOtherSessionsSuccess_204 | RevokeMyOtherSessionsError403;

model ListMyTrustedDevicesSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AccountTrustedDeviceListResponse;
}

union ListMyTrustedDevicesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListMyTrustedDevicesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListMyTrustedDevicesError403Body;
}

@doc("認証済みのユーザーが、自身の有効な信頼済みデバイスを読む（セルフサービス）。対象は `subject.id` に固定され、最後に使った新しい順に返す。現在のリクエストが提示した Cookie に一致するデバイスには `current=true` の印を付ける。セレクターも `verifier_hash` も返さない。")
@TypeSpec.OpenAPI.operationId("ListMyTrustedDevices")
@route("/api/account/v1/trusted-devices")
@get
op ListMyTrustedDevices(

): ListMyTrustedDevicesSuccess_200 | ListMyTrustedDevicesError403;

model RevokeMyTrustedDeviceSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMyTrustedDeviceError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RevokeMyTrustedDeviceError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RevokeMyTrustedDeviceError400Body;
}

union RevokeMyTrustedDeviceError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyTrustedDeviceError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyTrustedDeviceError403Body;
}

@doc("認証済みのユーザーが、自身の信頼済みデバイスの一つを失効させる（セルフサービス）。条件付きで第二要素を省ける能力を取り去る機微な操作なので、最近のステップアップの再認証を求める。自身のものでないデバイスは拒否する。失効は `revoked_at` と `revoke_reason=self_revoke` を設定する墓標で、失効済みのデバイスの失効は冪等に成功する。")
@TypeSpec.OpenAPI.operationId("RevokeMyTrustedDevice")
@route("/api/account/v1/trusted-devices/{id}/revoke")
@post
op RevokeMyTrustedDevice(
  @path id: string
): RevokeMyTrustedDeviceSuccess_204 | RevokeMyTrustedDeviceError400 | RevokeMyTrustedDeviceError403;

model RevokeMyTrustedDevicesSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMyTrustedDevicesError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyTrustedDevicesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyTrustedDevicesError403Body;
}

@doc("認証済みのユーザーが、自身のすべての信頼済みデバイスを失効させる（セルフサービス）。「すべてのデバイスを忘れる」操作で現在のデバイスも含むので、最近のステップアップの再認証を求める。各対象は墓標として失効させ、失効済みのものは冪等に飛ばす。")
@TypeSpec.OpenAPI.operationId("RevokeMyTrustedDevices")
@route("/api/account/v1/trusted-devices/revoke-all")
@post
op RevokeMyTrustedDevices(

): RevokeMyTrustedDevicesSuccess_204 | RevokeMyTrustedDevicesError403;

model ListSessionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SessionRecordListResponse;
}

union ListSessionsError403Body {
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSessionsError403Body;
}

@doc("管理者が、対象のユーザーの有効なセッションを一覧する。テナントの境界の中にとどまり、新しい順に返す。ListMySessions と違い対象は任意の `user_id` で、`current` の印はない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:read"])
@TypeSpec.OpenAPI.operationId("ListSessions")
@route("/api/admin/v1/users/{user_id}/sessions")
@get
op ListSessions(
  @path user_id: string
): ListSessionsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListSessionsError403;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeSessionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeSessionError403Body;
}

@doc("管理者が、対象のユーザーのセッションを一つ失効させる。墓標の契約は RevokeMySession と同じで、`revoked_at` と `revoke_reason=admin_revoke` を設定し、行は削除しない。失効済みのセッションの失効は冪等に成功する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:write"])
@TypeSpec.OpenAPI.operationId("RevokeSession")
@route("/api/admin/v1/users/{user_id}/sessions/{id}/revoke")
@post
op RevokeSession(
  @path user_id: string,
  @path id: string,): RevokeSessionSuccess_204 | RevokeSessionError400 | IdMagic.Contract.AuthenticationRequiredResponse | RevokeSessionError403;

model RevokeUserSessionsSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeUserSessionsError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeUserSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeUserSessionsError403Body;
}

@doc("管理者が、対象のユーザーのすべてのセッションを失効させる。「すべての場所からサインアウトする」操作である。RevokeMyOtherSessions と違い、セッションは呼び出し元自身のものではないので何も除外せず、対象のユーザーの有効なセッションはすべて墓標になる。それぞれを冪等に失効させる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:write"])
@TypeSpec.OpenAPI.operationId("RevokeUserSessions")
@route("/api/admin/v1/users/{user_id}/sessions/revoke-all")
@post
op RevokeUserSessions(
  @path user_id: string,): RevokeUserSessionsSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | RevokeUserSessionsError403;

model StartStepUpAuthenticationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.StepUpStartResponse;
}

union StartStepUpAuthenticationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartStepUpAuthenticationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartStepUpAuthenticationError403Body;
}

@doc("機微性の高い操作の前に、ステップアップの再認証を開始する（セルフサービス）。サブジェクト自身のセッションに固定され、使える要素を返す。パスワードは常に、TOTP は登録済みのときに使える。再認証への入口なので、このインターフェース自体はステップアップを求めない。")
@TypeSpec.OpenAPI.operationId("StartStepUpAuthentication")
@route("/api/account/v1/step-up/start")
@post
op StartStepUpAuthentication(

): StartStepUpAuthenticationSuccess_200 | StartStepUpAuthenticationError403;

union CompleteStepUpAuthenticationError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model CompleteStepUpAuthenticationError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: CompleteStepUpAuthenticationError400Body;
}

union CompleteStepUpAuthenticationError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CompleteStepUpAuthenticationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CompleteStepUpAuthenticationError403Body;
}

model CompleteStepUpAuthenticationSuccess_204 {
  @statusCode statusCode: 204;
}

@doc("提示した要素（パスワードまたは TOTP のコード）を検証し、成り立つときはセッションに `step_up_at` を記録する。以降 5 分の有効期間の間は、`step_up: required` の印を付けた操作を許可する。検証に失敗すると AccessDeniedError になる。")
@TypeSpec.OpenAPI.operationId("CompleteStepUpAuthentication")
@route("/api/account/v1/step-up/complete")
@post
op CompleteStepUpAuthentication(
  @body body: IdMagic.Contract.StepUpCompleteRequest
): CompleteStepUpAuthenticationSuccess_204 | CompleteStepUpAuthenticationError400 | CompleteStepUpAuthenticationError403;

union IssueMfaEnrollmentBypassError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model IssueMfaEnrollmentBypassError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: IssueMfaEnrollmentBypassError400Body;
}

union IssueMfaEnrollmentBypassError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

model IssueMfaEnrollmentBypassSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.MfaEnrollmentBypassResponse;
}

@doc("テナントの管理者が、MFA を登録していない同じテナントの有効なユーザーに、短命で一回限りの登録の回避を発行する。同じユーザーの未使用の既存の回避は失効させて置き換える。サインインの免除ではなく、登録のためだけに存在する保留中のフローに対する承認である。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("IssueMfaEnrollmentBypass")
@route("/api/admin/v1/users/{user_id}/mfa-enrollment-bypass")
@post
op IssueMfaEnrollmentBypass(
  @path user_id: string,
  @body body: IdMagic.Contract.MfaEnrollmentBypassIssueRequest
): IssueMfaEnrollmentBypassSuccess_201 | IssueMfaEnrollmentBypassError400 | IdMagic.Contract.AuthenticationRequiredResponse | IssueMfaEnrollmentBypassError403 | IssueMfaEnrollmentBypassError422;

model RevokeMfaEnrollmentBypassSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMfaEnrollmentBypassError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMfaEnrollmentBypassError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMfaEnrollmentBypassError403Body;
}

@doc("テナントの管理者が、対象のユーザーの未使用の登録の回避を失効させる。すでに使用済みまたは期限切れのときも、呼び出しはサインインを許さず、冪等に終わる。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("RevokeMfaEnrollmentBypass")
@route("/api/admin/v1/users/{user_id}/mfa-enrollment-bypass")
@delete
op RevokeMfaEnrollmentBypass(
  @path user_id: string
): RevokeMfaEnrollmentBypassSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | RevokeMfaEnrollmentBypassError403;

model ResetUserAuthenticatorsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuthenticatorResetResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
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

@doc("テナントの管理者が、対象のユーザーから、選んだ種類の認証器（TOTP の要素、WebAuthn の資格情報、リカバリーコード）を削除し、`mfa_enrolled` を計算し直す。緊急時の最後の手段である。管理者はユーザーの代わりに新しい要素を登録できず、削除して再登録を求めることだけができる。TOTP と WebAuthn がどちらもなくなったときは、IssueMfaEnrollmentBypass と同じ仕組みで管理者の承認した登録の回避を自動で発行し、既存のものを置き換えるので、次のサインインはフェイルクローズの登録必須のフローへ向かう。一種類だけを削除して `mfa_enrolled` が true のままのときは回避を発行せず、残った要素で通常どおりサインインする。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ResetUserAuthenticators")
@route("/api/admin/v1/users/{user_id}/authenticator-reset")
@post
op ResetUserAuthenticators(
  @path user_id: string,
  @body body: IdMagic.Contract.AuthenticatorResetRequest
): ResetUserAuthenticatorsSuccess_200 | ResetUserAuthenticatorsError400 | IdMagic.Contract.AuthenticationRequiredResponse | ResetUserAuthenticatorsError403 | ResetUserAuthenticatorsError422;

union RequestPasswordResetError429Body {
  IdMagic.Contract.RateLimitedError,
}

model RequestPasswordResetError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("クライアントが再試行の前に待たなければならない秒数。ボディの `retry_after_seconds` と同じ値である。")
  @header("Retry-After") retryAfter: int32;
  @body body: RequestPasswordResetError429Body;
}

model RequestPasswordResetSuccess_204 {
  @statusCode statusCode: 204;
}

union RequestPasswordResetError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model RequestPasswordResetError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: RequestPasswordResetError400Body;
}

union RequestPasswordResetError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RequestPasswordResetError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RequestPasswordResetError403Body;
}

@doc("未認証のユーザーが、パスワードリセットのリンクを求める。列挙を防ぐため、メールアドレスが存在するかにかかわらず常に 204 を返し、確認済みのメールアドレスのユーザーが見つかったときだけトークンを発行して送る。文面は Tenancy の通知のテンプレートのカタログ（`template_key=PasswordReset`）から解決する。ロケールは、受信者の User のロケールの属性、テナントの `default_locale`、システムのデフォルトのロケールの順に決め、テナントの上書きは組み込みのデフォルトより優先する。プレーンテキストと HTML の本文を常に両方生成し、リセットのリンクはリクエストの発行者の URL から組み立てるので、テンプレートが自ら URL を連結する必要はない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("RequestPasswordReset")
@route("/api/auth/forgot-password")
@post
op RequestPasswordReset(
  @body body: IdMagic.Contract.BrowserForgotPasswordRequest
): RequestPasswordResetSuccess_204 | RequestPasswordResetError400 | RequestPasswordResetError403 | RequestPasswordResetError429;

model ResetPasswordWithTokenHttpResponse {
  @doc("常に `ok` である。")
  status: string;
}

model ResetPasswordWithTokenSuccess_200 {
  @statusCode statusCode: 200;
  @body body: ResetPasswordWithTokenHttpResponse;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
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

union ResetPasswordWithTokenError410Body {
  IdMagic.Contract.InvalidResetTokenError,
}

model ResetPasswordWithTokenError410 {
  @statusCode statusCode: 410;
  @header contentType: "application/problem+json";
  @body body: ResetPasswordWithTokenError410Body;
}

@doc("リセットのリンクから戻ったユーザーが新しいパスワードを設定する。トークンは一度だけ使う。有効期間は 1800 秒で、`consumed_at` を設定した後は再利用できない。12 から 128 文字の長さの制約を満たし、直近の `history_depth`（5）個のハッシュと異なる `new_password` で `User.password_hash` を更新する（PasswordHistoryNoReuse）。成功すると PasswordChanged を発行する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ResetPasswordWithToken")
@route("/api/auth/reset-password")
@post
op ResetPasswordWithToken(
  @body body: IdMagic.Contract.BrowserResetPasswordRequest
): ResetPasswordWithTokenSuccess_200 | ResetPasswordWithTokenError400 | ResetPasswordWithTokenError403 | ResetPasswordWithTokenError410 | ResetPasswordWithTokenError422;

model DiscoverIdentityProvidersHttpResponse {
  providers: IdMagic.Contract.ExternalIdentityProviderSummary[];
}

model DiscoverIdentityProvidersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DiscoverIdentityProvidersHttpResponse;
}

@doc("サインインの UI が、リクエストのテナントで Active の上流のプロバイダーの表示情報だけを読む。シークレット、エンドポイント、証明書、連携のポリシーは公開しない。任意のメールアドレスまたはドメインのヒントは、設定時に検証済みの振り分けの規則に対してだけ適用する。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DiscoverIdentityProviders")
@route("/api/auth/federation/providers")
@get
op DiscoverIdentityProviders(
  @query hint?: string
): DiscoverIdentityProvidersSuccess_200;

union StartFederatedLoginError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model StartFederatedLoginError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: StartFederatedLoginError400Body;
}

union StartFederatedLoginError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model StartFederatedLoginError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartFederatedLoginError403Body;
}

model StartFederatedLoginSuccess_303 {
  @statusCode statusCode: 303;
}

@doc("テナントの中の Active な接続を解決し、OIDC の state、nonce、PKCE、または SAML の RelayState と AuthnRequest の ID を一回限りの FederatedLoginAttempt に保存して、設定時に検証済みの上流のエンドポイントへのリダイレクトを返す。任意の URL は受け付けない。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartFederatedLogin")
@route("/api/auth/federation/start")
@get
op StartFederatedLogin(
  @query provider_id: string,
  @query return_to?: string,
  @query link?: boolean
): StartFederatedLoginSuccess_303 | StartFederatedLoginError400 | StartFederatedLoginError403;

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

model CompleteFederatedLogin1Success_303 {
  @statusCode statusCode: 303;
}

@doc("コールバックを一回限りのログインの試行と照合し、OIDC のトークンまたは SAML のレスポンスの信頼、発行者、オーディエンス、nonce または InResponseTo、時刻、リプレイを検証する。ローカルの User は、既存の連携、明示的なポリシーの下での確認済みのメールアドレスによる連携、明示的な JIT のプロビジョニングの順に解決し、LoginSession の発行へ渡す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteFederatedLogin")
@route("/api/auth/federation/oidc/callback")
@get
op CompleteFederatedLogin1(
  @doc("このコールバックが属する一回限りの FederatedLoginAttempt。プロバイダーはクエリではなく試行から解決する。")
  @query state: string,
  @doc("上流の認可コード（RFC 6749 の 4.1.2 節）。")
  @query code: string
): CompleteFederatedLogin1Success_303 | CompleteFederatedLogin1Error400 | IdMagic.Contract.AuthenticationRequiredResponse | CompleteFederatedLogin1Error403 | CompleteFederatedLogin1Error422;

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

model CompleteFederatedLogin2Success_303 {
  @statusCode statusCode: 303;
}

@doc("コールバックを一回限りのログインの試行と照合し、OIDC のトークンまたは SAML のレスポンスの信頼、発行者、オーディエンス、nonce または InResponseTo、時刻、リプレイを検証する。ローカルの User は、既存の連携、明示的なポリシーの下での確認済みのメールアドレスによる連携、明示的な JIT のプロビジョニングの順に解決し、LoginSession の発行へ渡す。")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteSamlFederatedLogin")
@route("/api/auth/federation/saml/callback")
@post
op CompleteFederatedLogin2(
  @header contentType: "application/x-www-form-urlencoded",
  @body body: {
    @doc("SAML 2.0 の HTTP POST バインディングが送る、base64 で符号化した SAML の Response。")
    SAMLResponse: string;
    @doc("AuthnRequest が運んだ RelayState。このコールバックを一回限りの FederatedLoginAttempt と照合する。")
    RelayState?: string;
  }
): CompleteFederatedLogin2Success_303 | CompleteFederatedLogin2Error400 | IdMagic.Contract.AuthenticationRequiredResponse | CompleteFederatedLogin2Error403 | CompleteFederatedLogin2Error422;

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

@doc("ResourceOwner が、自身に結び付く外部の ID の、プロバイダーの表示名、プロトコル、`linked_at`、`last_login_at` を読む。外部のサブジェクトもトークンも返さない。")
@TypeSpec.OpenAPI.operationId("ListLinkedExternalIdentities")
@route("/api/account/v1/linked-identities")
@get
op ListLinkedExternalIdentities(

): ListLinkedExternalIdentitiesSuccess_200 | ListLinkedExternalIdentitiesError403;

union LinkExternalIdentityError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model LinkExternalIdentityError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: LinkExternalIdentityError400Body;
}

union LinkExternalIdentityError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model LinkExternalIdentityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: LinkExternalIdentityError403Body;
}

model LinkExternalIdentitySuccess_303 {
  @statusCode statusCode: 303;
}

@doc("直近 5 分以内にステップアップした ResourceOwner が、プロバイダーで外部の認証を開始し、コールバックが返すサブジェクトを明示的に自身へ連携する。すでに別のユーザーに連携しているサブジェクトは移さない。")
@TypeSpec.OpenAPI.operationId("LinkExternalIdentity")
@route("/api/account/v1/linked-identities/{provider_id}")
@post
op LinkExternalIdentity(
  @path provider_id: string,): LinkExternalIdentitySuccess_303 | LinkExternalIdentityError400 | LinkExternalIdentityError403;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.UnlinkDeniedError,
  IdMagic.Contract.AccessDeniedError,
}

model UnlinkExternalIdentityError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UnlinkExternalIdentityError403Body;
}

@doc("直近 5 分以内にステップアップした ResourceOwner が、自身のプロバイダーの一つの連携を外す。使えるパスワードの資格情報もほかの連携も残らなくなるときは、アカウントが自らを締め出さないように拒否する。")
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
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.AccessDeniedError,
}

model ListIdentityProviderConnectionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListIdentityProviderConnectionsError403Body;
}

@doc("テナントの管理者が接続の一覧を読む。`secret_reference` もシークレットも返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListIdentityProviderConnections")
@route("/api/admin/v1/identity-providers")
@get
op ListIdentityProviderConnections(

): ListIdentityProviderConnectionsSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | ListIdentityProviderConnectionsError403;

model CreateIdentityProviderConnectionSuccess_201 {
  @statusCode statusCode: 201;
  @body body: IdMagic.Contract.IdentityProviderConnection;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model CreateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateIdentityProviderConnectionError403Body;
}

@doc("テナントの管理者が接続を作成する。接続は Disabled で始まる。OIDC の発行者または SAML のメタデータを設定時に検証し、最後に正常だった信頼の情報を保存する。クライアントシークレットは `secret_reference` として届き、書き込み専用のエンベロープ暗号化したストレージに入る。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("CreateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers")
@post
op CreateIdentityProviderConnection(
  @body body: IdMagic.Contract.IdentityProviderConnectionRequest
): CreateIdentityProviderConnectionSuccess_201 | CreateIdentityProviderConnectionError400 | IdMagic.Contract.AuthenticationRequiredResponse | CreateIdentityProviderConnectionError403;

model UpdateIdentityProviderConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.IdentityProviderConnection;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model UpdateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateIdentityProviderConnectionError403Body;
}

union UpdateIdentityProviderConnectionError404Body {
  IdMagic.Contract.IdentityProviderConnectionNotFoundError,
}

model UpdateIdentityProviderConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: UpdateIdentityProviderConnectionError404Body;
}

@doc("テナントの管理者が、接続の表示、対応付け、ポリシー、シークレットの参照、または信頼の取得元を更新する。Active を Disabled へ落とすのは、信頼の取得元（発行者、エンドポイント、証明書、プロトコル）の変更だけで、`display_name` や `claim_mapping` のような信頼に関わらないフィールドの変更では状態を保つ。信頼の取得元の変更中に再検証が失敗したときは、最後に正常だった情報を保つ。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("UpdateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}")
@put
op UpdateIdentityProviderConnection(
  @path provider_id: string,
  @body body: IdMagic.Contract.IdentityProviderConnectionRequest
): UpdateIdentityProviderConnectionSuccess_200 | UpdateIdentityProviderConnectionError400 | IdMagic.Contract.AuthenticationRequiredResponse | UpdateIdentityProviderConnectionError403 | UpdateIdentityProviderConnectionError404;

model DeleteIdentityProviderConnectionSuccess_204 {
  @statusCode statusCode: 204;
}

union DeleteIdentityProviderConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DeleteIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DeleteIdentityProviderConnectionError403Body;
}

union DeleteIdentityProviderConnectionError409Body {
  IdMagic.Contract.IdentityProviderConnectionInUseError,
}

model DeleteIdentityProviderConnectionError409 {
  @statusCode statusCode: 409;
  @header contentType: "application/problem+json";
  @body body: DeleteIdentityProviderConnectionError409Body;
}

@doc("テナントの管理者が、状態にかかわらずいつでも接続を削除する。連携した ID がまだ結び付いているときは IdentityProviderConnectionInUseError で削除を拒否し、先に連携を外す必要がある。存在しない接続の削除も成功する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DeleteIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}")
@delete
op DeleteIdentityProviderConnection(
  @path provider_id: string
): DeleteIdentityProviderConnectionSuccess_204 | IdMagic.Contract.AuthenticationRequiredResponse | DeleteIdentityProviderConnectionError403 | DeleteIdentityProviderConnectionError409;

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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model ActivateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ActivateIdentityProviderConnectionError403Body;
}

union ActivateIdentityProviderConnectionError404Body {
  IdMagic.Contract.IdentityProviderConnectionNotFoundError,
}

model ActivateIdentityProviderConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: ActivateIdentityProviderConnectionError404Body;
}

@doc("信頼の検証と接続の試験がどちらも成功したら、Disabled の接続を Active へ移す。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ActivateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/activate")
@post
op ActivateIdentityProviderConnection(
  @path provider_id: string,): ActivateIdentityProviderConnectionSuccess_204 | ActivateIdentityProviderConnectionError400 | IdMagic.Contract.AuthenticationRequiredResponse | ActivateIdentityProviderConnectionError403 | ActivateIdentityProviderConnectionError404;

model DisableIdentityProviderConnectionSuccess_204 {
  @statusCode statusCode: 204;
}

union DisableIdentityProviderConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model DisableIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: DisableIdentityProviderConnectionError403Body;
}

union DisableIdentityProviderConnectionError400Body {
  IdMagic.Contract.InvalidRequestError,
}

model DisableIdentityProviderConnectionError400 {
  @statusCode statusCode: 400;
  @header contentType: "application/problem+json";
  @body body: DisableIdentityProviderConnectionError400Body;
}

union DisableIdentityProviderConnectionError404Body {
  IdMagic.Contract.IdentityProviderConnectionNotFoundError,
}

model DisableIdentityProviderConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: DisableIdentityProviderConnectionError404Body;
}

@doc("Active の接続を Disabled へ移し、新しい振り分けとコールバックの完了を即座に拒否する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DisableIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/disable")
@post
op DisableIdentityProviderConnection(
  @path provider_id: string,): DisableIdentityProviderConnectionSuccess_204 | DisableIdentityProviderConnectionError400 | IdMagic.Contract.AuthenticationRequiredResponse | DisableIdentityProviderConnectionError403 | DisableIdentityProviderConnectionError404;

model RefreshIdentityProviderMetadataSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.IdentityProviderConnection;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model RefreshIdentityProviderMetadataError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RefreshIdentityProviderMetadataError403Body;
}

union RefreshIdentityProviderMetadataError404Body {
  IdMagic.Contract.IdentityProviderConnectionNotFoundError,
}

model RefreshIdentityProviderMetadataError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: RefreshIdentityProviderMetadataError404Body;
}

@doc("テナントの管理者が、固定した発行者またはメタデータの取得元から、ディスカバリーの文書と JWKS、または SAML のメタデータを取得し直し、SSRF のガードと信頼の検証の両方を通ったときだけ、最後に正常だった情報を更新する。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("RefreshIdentityProviderMetadata")
@route("/api/admin/v1/identity-providers/{provider_id}/refresh")
@post
op RefreshIdentityProviderMetadata(
  @path provider_id: string,): RefreshIdentityProviderMetadataSuccess_200 | RefreshIdentityProviderMetadataError400 | IdMagic.Contract.AuthenticationRequiredResponse | RefreshIdentityProviderMetadataError403 | RefreshIdentityProviderMetadataError404;

model TestIdentityProviderConnectionHttpResponse {
  result: IdMagic.Contract.IdentityProviderConnectionTestResult;
}

model TestIdentityProviderConnectionSuccess_200 {
  @statusCode statusCode: 200;
  @body body: TestIdentityProviderConnectionHttpResponse;
}

union TestIdentityProviderConnectionError403Body {
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model TestIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: TestIdentityProviderConnectionError403Body;
}

union TestIdentityProviderConnectionError404Body {
  IdMagic.Contract.IdentityProviderConnectionNotFoundError,
}

model TestIdentityProviderConnectionError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: TestIdentityProviderConnectionError404Body;
}

@doc("保存した最後に正常だった信頼の取得元に、到達でき解決できるかを実際に試す。OIDC では authorization_endpoint、token_endpoint、jwks_uri への到達性と、`secret_reference` が解決するかを確かめる。SAML では、保存した証明書が X.509 として解析でき、まだ有効であることを確かめる。シークレットの値、トークン、証明書の本体は返さない。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("TestIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/test")
@post
op TestIdentityProviderConnection(
  @path provider_id: string,): TestIdentityProviderConnectionSuccess_200 | IdMagic.Contract.AuthenticationRequiredResponse | TestIdentityProviderConnectionError403 | TestIdentityProviderConnectionError404;

model PreviewIdentityProviderMappingHttpRequest {
  claims: Record<unknown>;
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
  IdMagic.Contract.CsrfFailedError,
  IdMagic.Contract.InsufficientScopeError,
  IdMagic.Contract.InvalidOriginError,
  IdMagic.Contract.AccessDeniedError,
}

model PreviewIdentityProviderMappingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewIdentityProviderMappingError403Body;
}

union PreviewIdentityProviderMappingError404Body {
  IdMagic.Contract.IdentityProviderConnectionNotFoundError,
}

model PreviewIdentityProviderMappingError404 {
  @statusCode statusCode: 404;
  @header contentType: "application/problem+json";
  @body body: PreviewIdentityProviderMappingError404Body;
}

@doc("テナントの管理者が、保存した試験のレスポンスの検証済みのクレームを対応付け、トークンもアサーションも含まない正規化したプレビューを読む。")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("PreviewIdentityProviderMapping")
@route("/api/admin/v1/identity-providers/{provider_id}/mapping-preview")
@post
op PreviewIdentityProviderMapping(
  @path provider_id: string,
  @body body: PreviewIdentityProviderMappingHttpRequest
): PreviewIdentityProviderMappingSuccess_200 | PreviewIdentityProviderMappingError400 | IdMagic.Contract.AuthenticationRequiredResponse | PreviewIdentityProviderMappingError403 | PreviewIdentityProviderMappingError404;
}

}
