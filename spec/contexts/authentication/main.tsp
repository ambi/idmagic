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
  IdMagic.Contract.AccessDeniedError,
}

model GetAccountContextError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GetAccountContextError403Body;
}

@doc("Reads the portal bootstrap information and the CSRF token shared by an authenticated browser session. A Bearer token must carry one of `idmagic.admin` for the administration portal, `idmagic.account` for the account portal, or `account:read` for a self-service API client. A session still partway through authentication, and a Bearer token carrying only some other scope, are both refused.")
@TypeSpec.OpenAPI.operationId("GetAccountContext")
@route("/api/auth/account")
@get
op GetAccountContext(

): GetAccountContextSuccess_200 | GetAccountContextError403;

model GetPasswordResetContextSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.BrowserCsrfContext;
}

@doc("The unauthenticated password reset screen reads the CSRF token that protects its POST.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("GetPasswordResetContext")
@route("/api/auth/password_reset_context")
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
  IdMagic.Contract.AccessDeniedError,
}

model StartStepUpWebAuthnChallengeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartStepUpWebAuthnChallengeError403Body;
}

@doc("Issues a WebAuthn assertion challenge for step-up, bound to the authenticated session. It verifies CSRF, and fails closed when WebAuthn is disabled.")
@TypeSpec.OpenAPI.operationId("StartStepUpWebAuthnChallenge")
@route("/api/account/v1/step_up/webauthn/challenge")
@post
op StartStepUpWebAuthnChallenge(

): StartStepUpWebAuthnChallengeSuccess_200 | StartStepUpWebAuthnChallengeError400 | StartStepUpWebAuthnChallengeError403;

model SubmitBrowserLoginHttpRequest {
  username: string;
  password: string;
  @doc("Where to go after signing in, confined to the `/admin` subtree of the same realm.")
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
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: SubmitBrowserLoginError429Body;
}

@doc("Handles the browser password step through the SPA JSON API. When MFA is required and no factor is enrolled: before the enforcement start the password session is allowed to sign in, which nudges the user to enroll ahead of time; after it, the same LoginSession moves to Enrollment pending only when both a valid single-use enrollment bypass and the grace deadline are satisfied. Anything else fails closed and the sign-in is not completed. Throttling applies per account (10 attempts in a 900-second window, 900-second lockout) and per IP (30 attempts in a 900-second window, 900-second lockout) through a cluster-wide counter, and stays fail-closed when the shared counter store is unreachable. Separately from this failure-count throttle, an EndpointRateLimitPolicy keyed by IP and tenant consumes every request, successful or not.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserLogin")
@route("/api/auth/login")
@post
op SubmitBrowserLogin(
  @body body: SubmitBrowserLoginHttpRequest
): SubmitBrowserLoginSuccess_200 | SubmitBrowserLoginError400 | SubmitBrowserLoginError403 | SubmitBrowserLoginError422 | SubmitBrowserLoginError429;

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

@doc("Only a LoginSession with `pending_purpose=Enrollment` that has not expired may start a TOTP enrollment. An ordinary unauthenticated session, one pending a Challenge, and an expired one are refused with 403. An already-enrolled user is refused with 409 instead, the same status the administration and self-service endpoints use for that condition, so the sign-in screen can tell \"nothing to do\" apart from \"contact an administrator\". The secret is not persisted until the enrollment is settled.")
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

@doc("With a LoginSession whose `pending_purpose` is Enrollment and which has not expired, verifies proof of possession of the TOTP, saves the factor, adds `otp` to the same LoginSession to promote it to MFA-satisfied, and resumes the original authorization transaction. A user who is already enrolled is refused with 409, as at the start endpoint. It does not share permissions with the ordinary self-service enrollment endpoint.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ConfirmBrowserMfaEnrollment")
@route("/api/auth/mfa/enrollment/totp/confirm")
@post
op ConfirmBrowserMfaEnrollment(
  @body body: IdMagic.Contract.BrowserMfaEnrollmentConfirmRequest
): ConfirmBrowserMfaEnrollmentSuccess_200 | ConfirmBrowserMfaEnrollmentError400 | ConfirmBrowserMfaEnrollmentError403 | ConfirmBrowserMfaEnrollmentError409;

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
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserTotpError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserTotpError403Body;
}

@doc("Verifies a TOTP code through the SPA JSON API. On success the LoginSession is promoted to fully authenticated and the flow continues into consent and authorization code issuance. Other factors have their own endpoints. No authentication is required, because the unauthenticated pending login session cookie is what resolves the target user.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserTotp")
@route("/api/auth/totp")
@post
op SubmitBrowserTotp(
  @body body: IdMagic.Contract.BrowserTotpRequest
): SubmitBrowserTotpSuccess_200 | SubmitBrowserTotpError400 | SubmitBrowserTotpError403;

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

@doc("An authenticated user changes their own password. `current_password` is verified, and `new_password` has to satisfy the length constraint of 12 to 128 characters and differ from the most recent `history_depth` (5) hashes (PasswordHistoryNoReuse).")
@TypeSpec.OpenAPI.operationId("ChangePassword")
@route("/api/auth/change_password")
@post
op ChangePassword(
  @body body: IdMagic.Contract.BrowserChangePasswordRequest
): ChangePasswordSuccess_200 | ChangePasswordError400 | ChangePasswordError403 | ChangePasswordError422;

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

@doc("An authenticated user reads their own security summary (self-service). The target is fixed to\n`subject.id`. It returns the password change time and whether MFA (TOTP) is enrolled.")
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

@doc("An authenticated user reads their own security notification preferences (self-service). The target is fixed to\n`actor.sub`. Every category is returned in the stable order the specification fixes, and a category the user may not\nturn off carries `mandatory=true`.")
@TypeSpec.OpenAPI.operationId("GetMyNotificationPreferences")
@route("/api/account/v1/notification_preferences")
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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateMyNotificationPreferencesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateMyNotificationPreferencesError403Body;
}

@doc("An authenticated user updates their own security notification preferences (self-service). The target is fixed to\n`actor.sub`. Silencing notifications is the first thing an attacker does after taking over an account, so a recent\nstep-up reauthentication is required. A request listing a mandatory category in `disabled_categories` is refused with\n400, and nothing is applied partially.")
@TypeSpec.OpenAPI.operationId("UpdateMyNotificationPreferences")
@route("/api/account/v1/notification_preferences")
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
  IdMagic.Contract.AccessDeniedError,
}

model StartTotpEnrollmentError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartTotpEnrollmentError403Body;
}

@doc("An authenticated user starts a TOTP enrollment (self-service). It returns a new 20-byte secret and an\n`otpauth` URI and persists nothing. A user who already has a TOTP factor is refused.")
@TypeSpec.OpenAPI.operationId("StartTotpEnrollment")
@route("/api/account/v1/mfa/totp/enroll/start")
@post
op StartTotpEnrollment(

): StartTotpEnrollmentSuccess_200 | StartTotpEnrollmentError400 | StartTotpEnrollmentError403;

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

@doc("Verifies proof of possession of a code against the secret and persists the TOTP factor (self-service).\nThe target is fixed to `subject.id`, and settling it sets `User.mfa_enrolled` to true.")
@TypeSpec.OpenAPI.operationId("ConfirmTotpEnrollment")
@route("/api/account/v1/mfa/totp/enroll/confirm")
@post
op ConfirmTotpEnrollment(
  @body body: IdMagic.Contract.TOTPEnrollmentConfirmRequest
): ConfirmTotpEnrollmentSuccess_200 | ConfirmTotpEnrollmentError400 | ConfirmTotpEnrollmentError403;

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

@doc("Verifies a valid TOTP code as proof of possession and then removes the TOTP factor (self-service).\nThe target is fixed to `subject.id`, and removing it returns `User.mfa_enrolled` to false.")
@TypeSpec.OpenAPI.operationId("RemoveTotpFactor")
@route("/api/account/v1/mfa/totp/remove")
@post
op RemoveTotpFactor(
  @body body: IdMagic.Contract.MfaFactorRemoveRequest
): RemoveTotpFactorSuccess_200 | RemoveTotpFactorError400 | RemoveTotpFactorError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model StartWebAuthnRegistrationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartWebAuthnRegistrationError403Body;
}

@doc("An authenticated user starts a WebAuthn or passkey registration (self-service).\nIt returns the PublicKeyCredentialCreationOptions and binds the challenge to the server-side session.\nNothing is persisted yet.")
@TypeSpec.OpenAPI.operationId("StartWebAuthnRegistration")
@route("/api/account/v1/mfa/webauthn/register/start")
@post
op StartWebAuthnRegistration(

): StartWebAuthnRegistrationSuccess_200 | StartWebAuthnRegistrationError400 | StartWebAuthnRegistrationError403;

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

@doc("Verifies the attestation and persists the WebAuthn credential (self-service). The target is fixed to\n`subject.id`, and the first credential sets `User.mfa_enrolled` to true. The RP ID, the origin, and the challenge are all verified.")
@TypeSpec.OpenAPI.operationId("FinishWebAuthnRegistration")
@route("/api/account/v1/mfa/webauthn/register/finish")
@post
op FinishWebAuthnRegistration(
  @body body: IdMagic.Contract.WebAuthnRegistrationRequest
): FinishWebAuthnRegistrationSuccess_200 | FinishWebAuthnRegistrationError400 | FinishWebAuthnRegistrationError403;

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

@doc("Removes a named WebAuthn credential (self-service; step-up required). The target is fixed to `subject.id`, and\nwhen no second factor is left `User.mfa_enrolled` returns to false.")
@TypeSpec.OpenAPI.operationId("RemoveWebAuthnCredential")
@route("/api/account/v1/mfa/webauthn/remove")
@post
op RemoveWebAuthnCredential(
  @body body: IdMagic.Contract.WebAuthnCredentialRemoveRequest
): RemoveWebAuthnCredentialSuccess_200 | RemoveWebAuthnCredentialError400 | RemoveWebAuthnCredentialError403;

model StartBrowserWebAuthnSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.WebAuthnAssertionOptions;
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

@doc("Issues a WebAuthn assertion challenge through the SPA JSON API. The pending login session resolves the\n`sub`, and the PublicKeyCredentialRequestOptions are returned. The challenge is bound to that session.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartBrowserWebAuthn")
@route("/api/auth/webauthn/challenge")
@post
op StartBrowserWebAuthn(

): StartBrowserWebAuthnSuccess_200 | StartBrowserWebAuthnError400 | StartBrowserWebAuthnError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserWebAuthnError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserWebAuthnError403Body;
}

@doc("Verifies a WebAuthn assertion through the SPA JSON API. On success the LoginSession is promoted to fully\nauthenticated and the flow continues into consent and authorization code issuance. A `sign_count` that goes backwards is\nrefused as a clone (`sign_count_regression=reject`).")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserWebAuthn")
@route("/api/auth/webauthn")
@post
op SubmitBrowserWebAuthn(
  @body body: IdMagic.Contract.BrowserWebAuthnRequest
): SubmitBrowserWebAuthnSuccess_200 | SubmitBrowserWebAuthnError400 | SubmitBrowserWebAuthnError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model SubmitBrowserRecoveryCodeError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: SubmitBrowserRecoveryCodeError403Body;
}

@doc("Verifies a backup recovery code as a second factor through the SPA JSON API. On success the\nLoginSession is promoted to fully authenticated and the consumed code cannot be used again.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("SubmitBrowserRecoveryCode")
@route("/api/auth/recovery-code")
@post
op SubmitBrowserRecoveryCode(
  @body body: IdMagic.Contract.BrowserRecoveryCodeRequest
): SubmitBrowserRecoveryCodeSuccess_200 | SubmitBrowserRecoveryCodeError400 | SubmitBrowserRecoveryCodeError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model GenerateRecoveryCodesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: GenerateRecoveryCodesError403Body;
}

@doc("An authenticated user generates or regenerates backup recovery codes (self-service; step-up required).\nThe plaintext is returned once and only the hashes are stored. An existing set is replaced outright (10 codes).")
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

@doc("An authenticated user explicitly invalidates their backup recovery codes (self-service; step-up required).\nThe target is fixed to `subject.id`, and every valid code is deleted.")
@TypeSpec.OpenAPI.operationId("RevokeRecoveryCodes")
@route("/api/account/v1/mfa/recovery-codes/revoke")
@post
op RevokeRecoveryCodes(

): RevokeRecoveryCodesSuccess_200 | RevokeRecoveryCodesError400 | RevokeRecoveryCodesError403;

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

@doc("An authenticated user reads their own recent sign-in history (self-service). The target is fixed to\n`subject.id`. It returns at most `limit` UserAuthenticated events from the existing audit event store, newest first.\n`limit` defaults to 10 and tops out at 50.")
@TypeSpec.OpenAPI.operationId("ListMySignInActivity")
@route("/api/account/v1/signin_activity")
@get
op ListMySignInActivity(
  @query limit?: int64
): ListMySignInActivitySuccess_200 | ListMySignInActivityError403 | ListMySignInActivityError400;

model ListUserSignInActivitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.UserSignInActivityResponse;
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

@doc("An administrator reads the sign-in history of a target user. It stays inside the tenant boundary and returns\nat most `limit` UserAuthenticated events for the user named in the path, newest first.\n`limit` defaults to 10 and tops out at 50.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:read"])
@TypeSpec.OpenAPI.operationId("ListUserSignInActivity")
@route("/api/admin/v1/users/{user_id}/signin_activity")
@get
op ListUserSignInActivity(
  @path user_id: string,
  @query limit?: int64
): ListUserSignInActivitySuccess_200 | ListUserSignInActivityError403 | ListUserSignInActivityError400;

model ListAuthenticationEventBucketsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.AuthenticationEventBucketListResponse;
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

@doc("An administrator lists the authentication event aggregates (buckets) by `window_start` descending. Sign-in failures and\nthe like, aggregated into 5-minute windows rather than kept as individual rows during an attack, are read inside the tenant\nboundary through bidirectional keyset pagination. The cursor comes from the Link response header of the response\n(`rel=\"prev\"` and `rel=\"next\"`). `limit` defaults to 50 and tops out at 200.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["audit:read"])
@TypeSpec.OpenAPI.operationId("ListAuthenticationEventBuckets")
@route("/api/admin/v1/authentication_event_buckets")
@get
op ListAuthenticationEventBuckets(
  @query cursor?: string,
  @query limit?: int64
): ListAuthenticationEventBucketsSuccess_200 | ListAuthenticationEventBucketsError403 | ListAuthenticationEventBucketsError400;

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

@doc("An authenticated user reads their own live sessions (self-service). The target is fixed to\n`subject.id`. The current session is marked `current=true` and they are returned newest first.")
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

@doc("An authenticated user revokes one of their own sessions (self-service). A session that is not theirs\nis refused. Revocation is a tombstone that sets `revoked_at` and `revoke_reason=self_revoke` on the LoginSession; the row\nis not deleted. Revoking an already-revoked session succeeds idempotently, and `revoked_at` keeps its first value.")
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

@doc("An authenticated user revokes all of their own sessions except the current one (self-service).\nThis is the \"sign out everywhere else\" operation. Each target is a tombstone setting `revoked_at` and\n`revoke_reason=self_revoke`, and one already revoked is skipped idempotently.")
@TypeSpec.OpenAPI.operationId("RevokeMyOtherSessions")
@route("/api/account/v1/sessions/revoke_others")
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

@doc("An authenticated user reads their own live trusted devices (self-service). The target is fixed to\n`subject.id`, and they are returned by last use, newest first. The device matching the cookie the current request\npresented is marked `current=true`. Neither the selector nor `verifier_hash` is returned.")
@TypeSpec.OpenAPI.operationId("ListMyTrustedDevices")
@route("/api/account/v1/trusted_devices")
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
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyTrustedDeviceError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyTrustedDeviceError403Body;
}

@doc("An authenticated user revokes one of their own trusted devices (self-service). Taking away the ability to\nskip the second factor conditionally is a sensitive operation, so a recent step-up reauthentication is required.\nA device that is not theirs is refused. Revocation is a tombstone setting `revoked_at` and `revoke_reason=self_revoke`,\nand revoking an already-revoked device succeeds idempotently.")
@TypeSpec.OpenAPI.operationId("RevokeMyTrustedDevice")
@route("/api/account/v1/trusted_devices/{id}/revoke")
@post
op RevokeMyTrustedDevice(
  @path id: string
): RevokeMyTrustedDeviceSuccess_204 | RevokeMyTrustedDeviceError400 | RevokeMyTrustedDeviceError403;

model RevokeMyTrustedDevicesSuccess_204 {
  @statusCode statusCode: 204;
}

union RevokeMyTrustedDevicesError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model RevokeMyTrustedDevicesError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RevokeMyTrustedDevicesError403Body;
}

@doc("An authenticated user revokes all of their own trusted devices (self-service). This is the \"forget every\ndevice\" operation and includes the current one, so a recent step-up reauthentication is required.\nEach target is revoked as a tombstone, and one already revoked is skipped idempotently.")
@TypeSpec.OpenAPI.operationId("RevokeMyTrustedDevices")
@route("/api/account/v1/trusted_devices/revoke_all")
@post
op RevokeMyTrustedDevices(

): RevokeMyTrustedDevicesSuccess_204 | RevokeMyTrustedDevicesError403;

model ListSessionsSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.SessionRecordListResponse;
}

union ListSessionsError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model ListSessionsError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: ListSessionsError403Body;
}

@doc("An administrator lists the live sessions of a target user. It stays inside the tenant boundary and\nreturns them newest first. Unlike ListMySessions the target is any `user_id`, and there is no\n`current` marker.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:read"])
@TypeSpec.OpenAPI.operationId("ListSessions")
@route("/api/admin/v1/users/{user_id}/sessions")
@get
op ListSessions(
  @path user_id: string
): ListSessionsSuccess_200 | ListSessionsError403;

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

@doc("An administrator revokes one session of a target user. The tombstone contract is the same as\nRevokeMySession: `revoked_at` and `revoke_reason=admin_revoke` are set and the row is not deleted.\nRevoking an already-revoked session succeeds idempotently.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:write"])
@TypeSpec.OpenAPI.operationId("RevokeSession")
@route("/api/admin/v1/users/{user_id}/sessions/{id}/revoke")
@post
op RevokeSession(
  @path user_id: string,
  @path id: string,): RevokeSessionSuccess_204 | RevokeSessionError400 | RevokeSessionError403;

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

@doc("An administrator revokes every session of a target user -- the \"sign out everywhere\"\noperation. Unlike RevokeMyOtherSessions the sessions are not the caller's own, so nothing is excluded and every\nlive session of the target user becomes a tombstone. Each one is revoked idempotently.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["sessions:write"])
@TypeSpec.OpenAPI.operationId("RevokeUserSessions")
@route("/api/admin/v1/users/{user_id}/sessions/revoke_all")
@post
op RevokeUserSessions(
  @path user_id: string,): RevokeUserSessionsSuccess_204 | RevokeUserSessionsError403;

model StartStepUpAuthenticationSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.StepUpStartResponse;
}

union StartStepUpAuthenticationError403Body {
  IdMagic.Contract.AccessDeniedError,
}

model StartStepUpAuthenticationError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: StartStepUpAuthenticationError403Body;
}

@doc("Starts a step-up reauthentication ahead of a high-sensitivity operation (self-service). It is fixed to the subject's own session and returns the factors available: password always, TOTP once enrolled. The interface itself does not require step-up, being the way into a reauthentication.")
@TypeSpec.OpenAPI.operationId("StartStepUpAuthentication")
@route("/api/account/v1/step_up/start")
@post
op StartStepUpAuthentication(

): StartStepUpAuthenticationSuccess_200 | StartStepUpAuthenticationError403;

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

@doc("Verifies the factor presented -- a password or a TOTP code -- and, when it holds, stamps `step_up_at` on the\nsession. Inside the recency window of five minutes, operations marked `step_up: required` are then permitted. A failed\nverification gives AccessDeniedError.")
@TypeSpec.OpenAPI.operationId("CompleteStepUpAuthentication")
@route("/api/account/v1/step_up/complete")
@post
op CompleteStepUpAuthentication(
  @body body: IdMagic.Contract.StepUpCompleteRequest
): CompleteStepUpAuthenticationSuccess_200 | CompleteStepUpAuthenticationError400 | CompleteStepUpAuthenticationError403;

model IssueMfaEnrollmentBypassSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.MfaEnrollmentBypassResponse;
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

@doc("A tenant administrator issues a short-lived, single-use enrollment bypass to an active user of the same tenant who has no MFA enrolled. An existing unconsumed bypass for the same user is revoked and replaced. It is not an exemption from signing in; it is approval for a pending flow that exists only to enroll.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("IssueMfaEnrollmentBypass")
@route("/api/admin/v1/users/{user_id}/mfa-enrollment-bypass")
@post
op IssueMfaEnrollmentBypass(
  @path user_id: string,
  @body body: IdMagic.Contract.MfaEnrollmentBypassIssueRequest
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

@doc("A tenant administrator revokes the unconsumed enrollment bypass of a target user. When it has already been consumed or has expired, the call still does not permit a sign-in and finishes idempotently.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("RevokeMfaEnrollmentBypass")
@route("/api/admin/v1/users/{user_id}/mfa-enrollment-bypass")
@delete
op RevokeMfaEnrollmentBypass(
  @path user_id: string
): RevokeMfaEnrollmentBypassSuccess_204 | RevokeMfaEnrollmentBypassError403;

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

@doc("A tenant administrator deletes the chosen kinds of authenticator from a target user -- the TOTP factor,\nthe WebAuthn credentials, the recovery codes -- and recomputes `mfa_enrolled`. It is an emergency backstop.\nAn administrator cannot enroll a new factor in the user's place; they may only delete and require re-enrollment.\nWhen both TOTP and WebAuthn are gone, an administrator-approved enrollment bypass is issued automatically -- the same\nmechanism as IssueMfaEnrollmentBypass -- replacing any existing one, so the next sign-in is directed into the\nfail-closed enrollment-required flow. When `mfa_enrolled` stays true because only one kind was deleted, no bypass is\nissued and the remaining factor carries the ordinary sign-in.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["users:write"])
@TypeSpec.OpenAPI.operationId("ResetUserAuthenticators")
@route("/api/admin/v1/users/{user_id}/authenticator-reset")
@post
op ResetUserAuthenticators(
  @path user_id: string,
  @body body: IdMagic.Contract.AuthenticatorResetRequest
): ResetUserAuthenticatorsSuccess_200 | ResetUserAuthenticatorsError400 | ResetUserAuthenticatorsError403 | ResetUserAuthenticatorsError422;

model RequestPasswordResetSuccess_200 {
  @statusCode statusCode: 200;
}

union RequestPasswordResetError429Body {
  IdMagic.Contract.RateLimitedError,
}

model RequestPasswordResetError429 {
  @statusCode statusCode: 429;
  @header contentType: "application/json";
  @doc("Seconds the client must wait before retrying, the same number the body carries in `retry_after_seconds`.")
  @header("Retry-After") retryAfter: int32;
  @body body: RequestPasswordResetError429Body;
}

@doc("An unauthenticated user asks for a password reset link.\nTo prevent enumeration it always returns 204, whether or not the email exists, and a token is issued and sent only\nwhen a user with a verified email is found.\nThe wording is resolved from the Tenancy notification template catalogue (`template_key=PasswordReset`).\nThe locale is decided in the order: the recipient User's locale attribute, the tenant's `default_locale`, the system\ndefault locale; a tenant override takes precedence over the built-in default.\nBoth a plain text and an HTML body are always produced, and the reset link is built from the issuer URL of the\nrequest, so the template never has to join URLs itself.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("RequestPasswordReset")
@route("/api/auth/forgot_password")
@post
op RequestPasswordReset(
  @body body: IdMagic.Contract.BrowserForgotPasswordRequest
): RequestPasswordResetSuccess_200 | RequestPasswordResetError429;

model ResetPasswordWithTokenHttpResponse {
  @doc("Always `ok`.")
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

@doc("A user returning from the reset link sets a new password.\nThe token is consumed once -- it lives 1800 seconds and cannot be reused once `consumed_at` is set -- and\n`User.password_hash` is updated with a `new_password` that passed the length constraint of 12 to 128 characters and\ndiffers from the most recent `history_depth` (5) hashes (PasswordHistoryNoReuse). On success PasswordChanged is emitted.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("ResetPasswordWithToken")
@route("/api/auth/reset_password")
@post
op ResetPasswordWithToken(
  @body body: IdMagic.Contract.BrowserResetPasswordRequest
): ResetPasswordWithTokenSuccess_200 | ResetPasswordWithTokenError400 | ResetPasswordWithTokenError403 | ResetPasswordWithTokenError422;

model DiscoverIdentityProvidersHttpResponse {
  providers: IdMagic.Contract.ExternalIdentityProviderSummary[];
}

model DiscoverIdentityProvidersSuccess_200 {
  @statusCode statusCode: 200;
  @body body: DiscoverIdentityProvidersHttpResponse;
}

@doc("The sign-in UI reads nothing but the display information of the upstream providers that are Active in the request tenant. Secrets, endpoints, certificates, and the link policy are not exposed. An optional email or domain hint is applied only against routing rules already validated at configuration time.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("DiscoverIdentityProviders")
@route("/api/auth/federation/providers")
@get
op DiscoverIdentityProviders(
  @query hint?: string
): DiscoverIdentityProvidersSuccess_200;

model StartFederatedLoginSuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.FederatedLoginStartResponse;
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

@doc("Resolves an Active connection inside the tenant, stores the OIDC state, nonce, and PKCE, or the SAML RelayState and AuthnRequest ID, in a single-use FederatedLoginAttempt, and returns a redirect to the upstream endpoint validated at configuration time. It never accepts an arbitrary URL.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("StartFederatedLogin")
@route("/api/auth/federation/start")
@get
op StartFederatedLogin(
  @query provider_id: string,
  @query return_to?: string,
  @query link?: boolean
): StartFederatedLoginSuccess_200 | StartFederatedLoginError400 | StartFederatedLoginError403;

model CompleteFederatedLogin1Success_200 {
  @statusCode statusCode: 200;
  @doc("Where the browser continues: the caller's `return_to`, the account security page after a link, or the resume path of a pending authorization.")
  @header("Location") location: url;
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

@doc("Correlates the callback with the single-use login attempt and verifies the trust, issuer, audience, nonce or InResponseTo, time, and replay of the OIDC token or the SAML response. The local User is resolved in the order: an existing link, a verified-email link under an explicit policy, then explicit JIT provisioning, and handed on to issue a LoginSession.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteFederatedLogin")
@route("/api/auth/federation/oidc/callback")
@get
op CompleteFederatedLogin1(
  @doc("The single-use FederatedLoginAttempt this callback belongs to. The provider is resolved from the attempt, not from the query.")
  @query state: string,
  @doc("The upstream authorization code (RFC 6749 section 4.1.2).")
  @query code: string
): CompleteFederatedLogin1Success_200 | CompleteFederatedLogin1Error400 | CompleteFederatedLogin1Error403 | CompleteFederatedLogin1Error422;

model CompleteFederatedLogin2Success_200 {
  @statusCode statusCode: 200;
  @doc("Where the browser continues: the caller's `return_to`, the account security page after a link, or the resume path of a pending authorization.")
  @header("Location") location: url;
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

@doc("Correlates the callback with the single-use login attempt and verifies the trust, issuer, audience, nonce or InResponseTo, time, and replay of the OIDC token or the SAML response. The local User is resolved in the order: an existing link, a verified-email link under an explicit policy, then explicit JIT provisioning, and handed on to issue a LoginSession.")
@useAuth(NoAuth)
@TypeSpec.OpenAPI.operationId("CompleteFederatedLogin")
@route("/api/auth/federation/saml/callback")
@post
op CompleteFederatedLogin2(
  @header contentType: "application/x-www-form-urlencoded",
  @body body: {
    @doc("The base64-encoded SAML Response, as the HTTP POST binding of SAML 2.0 sends it.")
    SAMLResponse: string;
    @doc("The RelayState the AuthnRequest carried, correlating this callback with its single-use FederatedLoginAttempt.")
    RelayState?: string;
  }
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

@doc("A ResourceOwner reads the provider display name, protocol, `linked_at`, and `last_login_at` of the external identities tied to them. Neither the external subject nor any token is returned.")
@TypeSpec.OpenAPI.operationId("ListLinkedExternalIdentities")
@route("/api/account/v1/linked-identities")
@get
op ListLinkedExternalIdentities(

): ListLinkedExternalIdentitiesSuccess_200 | ListLinkedExternalIdentitiesError403;

model LinkExternalIdentitySuccess_200 {
  @statusCode statusCode: 200;
  @body body: IdMagic.Contract.FederatedLoginStartResponse;
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

@doc("A ResourceOwner who stepped up within the last five minutes starts external authentication at a provider and explicitly links the subject the callback returns to themselves. A subject already linked to another user is not moved.")
@TypeSpec.OpenAPI.operationId("LinkExternalIdentity")
@route("/api/account/v1/linked-identities/{provider_id}")
@post
op LinkExternalIdentity(
  @path provider_id: string,): LinkExternalIdentitySuccess_200 | LinkExternalIdentityError400 | LinkExternalIdentityError403;

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

@doc("A ResourceOwner who stepped up within the last five minutes unlinks one of their own providers. When they would be left with neither a usable password credential nor another link, it is refused, so the account cannot lock itself out.")
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

@doc("A tenant administrator reads the connection list. Neither `secret_reference` nor the secret is returned.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ListIdentityProviderConnections")
@route("/api/admin/v1/identity-providers")
@get
op ListIdentityProviderConnections(

): ListIdentityProviderConnectionsSuccess_200 | ListIdentityProviderConnectionsError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model CreateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: CreateIdentityProviderConnectionError403Body;
}

@doc("A tenant administrator creates a connection. It starts Disabled. The OIDC issuer or the SAML metadata is validated at configuration time and the last-known-good trust material is stored. The client secret arrives as a `secret_reference` and goes into write-only envelope-encrypted storage.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("CreateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers")
@post
op CreateIdentityProviderConnection(
  @body body: IdMagic.Contract.IdentityProviderConnectionRequest
): CreateIdentityProviderConnectionSuccess_201 | CreateIdentityProviderConnectionError400 | CreateIdentityProviderConnectionError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model UpdateIdentityProviderConnectionError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: UpdateIdentityProviderConnectionError403Body;
}

@doc("A tenant administrator updates the display, mapping, policy, secret reference, or trust source of a connection. Only a change to a trust source -- the issuer, an endpoint, a certificate, the protocol -- drops Active to Disabled; a change to a non-trust field such as `display_name` or `claim_mapping` keeps the state. When revalidation fails during a trust source change, the last-known-good material is kept.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("UpdateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}")
@put
op UpdateIdentityProviderConnection(
  @path provider_id: string,
  @body body: IdMagic.Contract.IdentityProviderConnectionRequest
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

@doc("A tenant administrator deletes a connection at any time, whatever its state. When links to it remain, the deletion is refused and an explicit cleanup is required first.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DeleteIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}")
@delete
op DeleteIdentityProviderConnection(
  @path provider_id: string
): DeleteIdentityProviderConnectionSuccess_204 | DeleteIdentityProviderConnectionError400 | DeleteIdentityProviderConnectionError403;

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

@doc("Moves a Disabled connection to Active once trust validation and a connection test have both succeeded.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("ActivateIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/activate")
@post
op ActivateIdentityProviderConnection(
  @path provider_id: string,): ActivateIdentityProviderConnectionSuccess_204 | ActivateIdentityProviderConnectionError400 | ActivateIdentityProviderConnectionError403;

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

@doc("Moves an Active connection to Disabled, refusing new routing and callback completion immediately.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("DisableIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/disable")
@post
op DisableIdentityProviderConnection(
  @path provider_id: string,): DisableIdentityProviderConnectionSuccess_204 | DisableIdentityProviderConnectionError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model RefreshIdentityProviderMetadataError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: RefreshIdentityProviderMetadataError403Body;
}

@doc("A tenant administrator re-fetches the discovery document and JWKS, or the SAML metadata, from the pinned issuer or metadata source, and updates the last-known-good material only when the SSRF guard and trust validation both pass.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("RefreshIdentityProviderMetadata")
@route("/api/admin/v1/identity-providers/{provider_id}/refresh")
@post
op RefreshIdentityProviderMetadata(
  @path provider_id: string,): RefreshIdentityProviderMetadataSuccess_200 | RefreshIdentityProviderMetadataError400 | RefreshIdentityProviderMetadataError403;

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

@doc("Actually tests whether the stored last-known-good trust source is reachable and resolvable. For OIDC it checks reachability of the authorization_endpoint, token_endpoint, and jwks_uri and whether the `secret_reference` resolves; for SAML it checks that the stored certificate parses as X.509 and is still valid. Neither the secret value, nor a token, nor the certificate body is returned.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("TestIdentityProviderConnection")
@route("/api/admin/v1/identity-providers/{provider_id}/test")
@post
op TestIdentityProviderConnection(
  @path provider_id: string,): TestIdentityProviderConnectionSuccess_200 | TestIdentityProviderConnectionError400 | TestIdentityProviderConnectionError403;

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
  IdMagic.Contract.AccessDeniedError,
}

model PreviewIdentityProviderMappingError403 {
  @statusCode statusCode: 403;
  @header contentType: "application/problem+json";
  @body body: PreviewIdentityProviderMappingError403Body;
}

@doc("A tenant administrator maps the verified claims of a stored test response and reads a normalized preview that carries neither the token nor the assertion.")
@TypeSpec.OpenAPI.extension("x-api-token-scopes", #["interactive_session"])
@TypeSpec.OpenAPI.operationId("PreviewIdentityProviderMapping")
@route("/api/admin/v1/identity-providers/{provider_id}/mapping-preview")
@post
op PreviewIdentityProviderMapping(
  @path provider_id: string,
  @body body: PreviewIdentityProviderMappingHttpRequest
): PreviewIdentityProviderMappingSuccess_200 | PreviewIdentityProviderMappingError400 | PreviewIdentityProviderMappingError403;
}

}
