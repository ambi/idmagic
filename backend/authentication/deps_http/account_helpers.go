package deps_http

import (
	"errors"
	"net/http"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfausecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	recoveryusecases "github.com/ambi/idmagic/backend/authentication/recovery/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	webauthnusecases "github.com/ambi/idmagic/backend/authentication/webauthn/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

// AccountProfileDeps は self-service プロフィール系ユースケースの依存を組み立てる。
// mfa/shared 双方の handler から呼ばれる横断ヘルパー。
func AccountProfileDeps(d Deps) userusecases.AccountProfileDeps {
	return userusecases.AccountProfileDeps{
		UserRepo: d.UserRepo, AttrSchemaRepo: d.AttrSchemaRepo, Emit: d.LegacyEmit(),
	}
}

// RequireAuthenticatedSub は認証済み (pending でない) セッションの sub を返す。
// self-service では actor == target なので sub をそのまま操作対象に使う。
func RequireAuthenticatedSub(d Deps, c *echo.Context) (string, error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", support.ErrAdminAuthenticationRequired
	}
	return authn.UserID, nil
}

// RequireStepUpSub は認証済みセッションを解決し、step-up gate を通過した sub を返す
// (高 sensitivity 操作用)。recency 窓を外れていれば ErrStepUpRequired。
func RequireStepUpSub(d Deps, c *echo.Context) (string, error) {
	sub, _, err := RequireStepUpSession(d, c)
	return sub, err
}

// RequireStepUpSession は RequireStepUpSub と同じゲートに加え、現在の sessionID を返す
// (revoke_others のように除外対象の session を要するハンドラ用)。
func RequireStepUpSession(d Deps, c *echo.Context) (sub, sessionID string, err error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", "", support.ErrAdminAuthenticationRequired
	}
	if !mfausecases.StepUpSatisfied(authn, time.Now().UTC()) {
		return "", "", mfausecases.ErrStepUpRequired
	}
	return authn.UserID, authn.SessionID, nil
}

// RequireAuthenticatedAuthn は認証済み (pending でない) セッションの AuthenticationContext を
// 返す。step-up start / complete は step-up gate 自体を掛けない (再認証の入口のため)。
func RequireAuthenticatedAuthn(d Deps, c *echo.Context) (*authdomain.AuthenticationContext, error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return nil, err
	}
	if authn == nil || authn.AuthenticationPending {
		return nil, support.ErrAdminAuthenticationRequired
	}
	return authn, nil
}

// WriteAccountError は self-service account 系ハンドラ共通のエラー → HTTP 応答マッピング。
func WriteAccountError(c *echo.Context, err error) error {
	if handled, result := WriteAccountMfaError(c, err); handled {
		return result
	}
	switch {
	case errors.Is(err, support.ErrAdminAuthenticationRequired):
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証済みセッションが必要です")
	case errors.Is(err, mfausecases.ErrStepUpRequired):
		return support.WriteBrowserError(c, http.StatusForbidden, "step_up_required", "この操作には再認証が必要です")
	case errors.Is(err, idmusecases.ErrUserNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "ユーザーが存在しません")
	case errors.Is(err, sessionusecases.ErrSessionNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "session_not_found", "セッションが存在しません")
	default:
		return err
	}
}

// WriteAccountMfaError は MFA (totp/webauthn/recovery) self-service 固有のエラーを HTTP に
// 写す。該当した場合は handled=true と書き込み結果を返し、それ以外は handled=false で
// WriteAccountError に委ねる。
func WriteAccountMfaError(c *echo.Context, err error) (handled bool, result error) {
	switch {
	case errors.Is(err, mfausecases.ErrMfaAlreadyEnrolled):
		return true, support.WriteBrowserError(c, http.StatusConflict, "mfa_already_enrolled", "認証アプリは既に登録されています")
	case errors.Is(err, mfausecases.ErrMfaNotEnrolled):
		return true, support.WriteBrowserError(c, http.StatusNotFound, "mfa_not_enrolled", "登録済みの認証アプリがありません")
	case errors.Is(err, mfausecases.ErrInvalidTOTPCode):
		return true, support.WriteBrowserError(c, http.StatusBadRequest, "invalid_totp", "認証コードを確認してください。")
	case errors.Is(err, mfausecases.ErrInvalidTOTPSecret):
		return true, support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "登録手続きをやり直してください。")
	case errors.Is(err, webauthnusecases.ErrWebAuthnNotConfigured):
		return true, support.WriteBrowserError(c, http.StatusServiceUnavailable, "webauthn_unavailable", "パスキー認証は利用できません")
	case errors.Is(err, webauthnusecases.ErrWebAuthnChallengeMissing):
		return true, support.WriteBrowserError(c, http.StatusBadRequest, "webauthn_challenge_expired", "パスキー登録をやり直してください。")
	case errors.Is(err, webauthnusecases.ErrWebAuthnCredentialCloned):
		return true, support.WriteBrowserError(c, http.StatusUnauthorized, "webauthn_cloned", "このパスキーは利用できません。")
	case errors.Is(err, webauthnusecases.ErrWebAuthnVerification):
		return true, support.WriteBrowserError(c, http.StatusBadRequest, "invalid_webauthn", "パスキーの検証に失敗しました。")
	case errors.Is(err, webauthnusecases.ErrWebAuthnCredentialNotFound):
		return true, support.WriteBrowserError(c, http.StatusNotFound, "webauthn_not_found", "対象のパスキーが見つかりません。")
	case errors.Is(err, recoveryusecases.ErrRecoveryCodeInvalid):
		return true, support.WriteBrowserError(c, http.StatusBadRequest, "invalid_recovery_code", "リカバリコードを確認してください。")
	default:
		return false, nil
	}
}
