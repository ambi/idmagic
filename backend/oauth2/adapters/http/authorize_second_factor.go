// 第二要素 (login step) の WebAuthn / recovery code サポート (wi-26 / ADR-087)。
// password 認証後の pending login session に対し、TOTP に加えて passkey / recovery code を
// 選択式で検証できるようにする。検証成功後の共通後処理は finishSecondFactor に集約する。
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type browserWebAuthnRequest struct {
	Assertion json.RawMessage `json:"assertion"`
	ReturnTo  string          `json:"return_to,omitempty"`
}

type recoveryCodeAPIRequest struct {
	Code     string `json:"code"`
	ReturnTo string `json:"return_to,omitempty"`
}

func (d Deps) webAuthnLoginDeps() authusecases.WebAuthnDeps {
	return authusecases.WebAuthnDeps{
		RP:             d.WebAuthnRP,
		UserRepo:       d.UserRepo,
		CredentialRepo: d.WebAuthnCredentialRepo,
		MfaFactorRepo:  d.MfaFactorRepo,
		SessionStore:   d.WebAuthnSessionStore,
		Emit:           d.Emit,
	}
}

func (d Deps) recoveryCodesDeps() authusecases.RecoveryCodesDeps {
	return authusecases.RecoveryCodesDeps{
		UserRepo: d.UserRepo, RecoveryCodeRepo: d.RecoveryCodeRepo, Emit: d.Emit,
	}
}

// secondFactorTransaction は第二要素待ち (kind=totp) の transaction 応答を組み立てる。
func (d Deps) secondFactorTransaction(c *echo.Context, csrf string, authn *authdomain.AuthenticationContext) transactionResponse {
	if authn.PendingPurpose == authdomain.LoginPendingEnrollment {
		return transactionResponse{Kind: "mfa_enrollment", CSRFToken: csrf}
	}
	return transactionResponse{
		Kind:                "totp",
		CSRFToken:           csrf,
		SecondFactorMethods: d.secondFactorMethods(c, authn.UserID),
	}
}

// secondFactorMethods は sub が利用できる第二要素 method を返す (totp / webauthn / recovery_code)。
func (d Deps) secondFactorMethods(c *echo.Context, sub string) []string {
	methods := []string{}
	if d.canUseTOTP(c, sub) {
		methods = append(methods, "totp")
	}
	if d.WebAuthnRP != nil && d.WebAuthnCredentialRepo != nil {
		if creds, err := d.WebAuthnCredentialRepo.ListBySub(c.Request().Context(), sub); err == nil && len(creds) > 0 {
			methods = append(methods, "webauthn")
		}
	}
	if d.RecoveryCodeRepo != nil {
		if codes, err := d.RecoveryCodeRepo.ListBySub(c.Request().Context(), sub); err == nil {
			for _, code := range codes {
				if code.ConsumedAt == nil {
					methods = append(methods, "recovery_code")
					break
				}
			}
		}
	}
	return methods
}

// handleWebAuthnChallengeAPI は login の WebAuthn assertion challenge を発行する。
func (d Deps) handleWebAuthnChallengeAPI(c *echo.Context) error {
	if d.WebAuthnRP == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "webauthn_unavailable", "パスキー認証は利用できません")
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || authn.SessionID == "" {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証セッションがありません")
	}
	if !authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "追加の認証は不要です")
	}
	assertion, err := authusecases.BeginWebAuthnAssertion(c.Request().Context(), d.webAuthnLoginDeps(), authn.SessionID, authn.UserID)
	if err != nil {
		if errors.Is(err, authusecases.ErrWebAuthnNoCredential) {
			return support.WriteBrowserError(c, http.StatusNotFound, "webauthn_not_enrolled", "登録済みのパスキーがありません")
		}
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, assertion)
}

// handleWebAuthnAPI は login の WebAuthn assertion を検証し、成功すれば認証を完了させる。
func (d Deps) handleWebAuthnAPI(c *echo.Context) error {
	if d.WebAuthnRP == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "webauthn_unavailable", "パスキー認証は利用できません")
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || authn.SessionID == "" {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証セッションがありません")
	}
	if !authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "追加の認証は不要です")
	}
	var input browserWebAuthnRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	req, transactionErr := d.transactionRequest(c)
	directAdminLogin := transactionErr != nil && input.ReturnTo != ""
	if directAdminLogin {
		if !validReturnTo(c, input.ReturnTo) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to が不正です")
		}
	} else if transactionErr != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", transactionErr.Error())
	}
	if _, err := authusecases.FinishWebAuthnAssertion(
		c.Request().Context(), d.webAuthnLoginDeps(), authn.SessionID, authn.UserID, []byte(input.Assertion), time.Now().UTC(),
	); err != nil {
		d.recordLoginOutcome("failure", "webauthn_invalid", "webauthn")
		d.emitAuthenticationFailure(c, authn.UserID, "webauthn_invalid")
		return support.WriteBrowserError(c, http.StatusUnauthorized, "invalid_webauthn", "パスキー認証に失敗しました。")
	}
	return d.finishSecondFactor(c, authn.SessionID, req, "webauthn", directAdminLogin, input.ReturnTo)
}

// handleRecoveryCodeAPI は login の第二要素として backup recovery code を消費する。
func (d Deps) handleRecoveryCodeAPI(c *echo.Context) error {
	if d.RecoveryCodeRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "recovery_unavailable", "リカバリコードは利用できません")
	}
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	authn, _ := d.ResolveAuthentication(c)
	if authn == nil || authn.SessionID == "" {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証セッションがありません")
	}
	if !authn.AuthenticationPending {
		return support.WriteBrowserError(c, http.StatusForbidden, "access_denied", "追加の認証は不要です")
	}
	var input recoveryCodeAPIRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	req, transactionErr := d.transactionRequest(c)
	directAdminLogin := transactionErr != nil && input.ReturnTo != ""
	if directAdminLogin {
		if !validReturnTo(c, input.ReturnTo) {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "return_to が不正です")
		}
	} else if transactionErr != nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "transaction_unavailable", transactionErr.Error())
	}
	if _, err := authusecases.ConsumeRecoveryCode(
		c.Request().Context(), d.recoveryCodesDeps(), authn.UserID, input.Code, time.Now().UTC(),
	); err != nil {
		if errors.Is(err, authusecases.ErrRecoveryCodeInvalid) {
			d.recordLoginOutcome("failure", "recovery_code_invalid", "recovery_code")
			d.emitAuthenticationFailure(c, authn.UserID, "recovery_code_invalid")
			return support.WriteBrowserError(c, http.StatusUnauthorized, "invalid_recovery_code", "リカバリコードを確認してください。")
		}
		return err
	}
	return d.finishSecondFactor(c, authn.SessionID, req, "rc", directAdminLogin, input.ReturnTo)
}

// finishSecondFactor は第二要素 (otp / webauthn / rc) 検証後の共通後処理。session を認証完了に
// 昇格させ、last_login gate / directAdmin / 認可継続を処理する。
func (d Deps) finishSecondFactor(
	c *echo.Context,
	sessionID string,
	req *domain.AuthorizationRequest,
	amr string,
	directAdminLogin bool,
	returnTo string,
) error {
	completed, err := d.SessionManager.CompleteFactor(c.Request().Context(), sessionID, []string{amr})
	if err != nil {
		return err
	}
	if completed == nil {
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "セッションが失効しました")
	}
	d.setSessionCookie(c, completed.SessionID)
	var user *idmdomain.User
	found, err := d.UserRepo.FindBySub(c.Request().Context(), completed.UserID)
	if err != nil {
		return err
	}
	user = found
	clientID := ""
	if req != nil {
		clientID = req.ClientID
	}
	d.emitAuthenticationSuccess(c, time.Now().UTC(), user, completed, clientID)
	// full authentication 完了 (pwd + 第二要素)。last_login_at 記録 + required action gate。
	if user != nil {
		gateNext, err := d.recordLoginAndRequiredAction(c, user, time.Now().UTC())
		if err != nil {
			return err
		}
		if gateNext != "" {
			return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{Next: gateNext})
		}
	}
	if directAdminLogin {
		return support.NoStoreJSON(c, http.StatusOK, browserFlowResponse{RedirectTo: returnTo})
	}
	if err := d.RequestStore.AttachAuthentication(
		c.Request().Context(), req.ID, completed.UserID, completed.AuthTime, completed.AMR, completed.ACR,
	); err != nil {
		return writeOAuthError(c, err)
	}
	req.UserID, req.AuthTime, req.AMR, req.ACR = &completed.UserID, &completed.AuthTime, completed.AMR, &completed.ACR
	client, err := d.ClientRepo.FindByID(c.Request().Context(), support.RequestTenantID(c), req.ClientID)
	if err != nil || client == nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_transaction", "クライアントが存在しません")
	}
	next, err := d.completeAfterAuthn(c, req, client, completed)
	if err != nil {
		return err
	}
	if next.RedirectTo != "" {
		d.clearTransactionCookie(c)
	}
	return writeAuthorizationNext(c, next)
}
