// /api/account/profile — エンドユーザ自身のプロフィール参照・編集 (self-service)。
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	mfausecases "github.com/ambi/idmagic/backend/authentication/mfa/usecases"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type AccountProfileResponse struct {
	ID                string                               `json:"id"`
	PreferredUsername string                               `json:"preferred_username"`
	Name              *string                              `json:"name,omitempty"`
	GivenName         *string                              `json:"given_name,omitempty"`
	FamilyName        *string                              `json:"family_name,omitempty"`
	Email             *string                              `json:"email,omitempty"`
	EmailVerified     bool                                 `json:"email_verified"`
	MfaEnrolled       bool                                 `json:"mfa_enrolled"`
	Status            idmdomain.UserStatus                 `json:"status"`
	Attributes        map[string]userdomain.AttributeValue `json:"attributes"`
	// ReadableAttributes は self が参照できる属性定義。
	ReadableAttributes []userdomain.UserAttributeDef `json:"readable_attributes"`
	// EditableAttributes は self が編集できる属性定義 (editable_by_user=true)。
	// UI がフォームを描画するために型・multi_valued 等のメタを併せて返す。
	EditableAttributes []userdomain.UserAttributeDef `json:"editable_attributes"`
}

// accountSummaryResponse は portal home 用のアカウント概要 (self-service)。
// admin shell 用の AccountContext とは別契約で roles を含めない (wi-21 / ADR-042)。
type accountSummaryResponse struct {
	ID                string                     `json:"id"`
	PreferredUsername string                     `json:"preferred_username"`
	Name              *string                    `json:"name,omitempty"`
	Email             *string                    `json:"email,omitempty"`
	EmailVerified     bool                       `json:"email_verified"`
	MfaEnrolled       bool                       `json:"mfa_enrolled"`
	Status            idmdomain.UserStatus       `json:"status"`
	LastLoginAt       *time.Time                 `json:"last_login_at,omitempty"`
	PasswordChangedAt *time.Time                 `json:"password_changed_at,omitempty"`
	RequiredActions   []idmdomain.RequiredAction `json:"required_actions"`
}

type accountProfileUpdateRequest struct {
	Name       *string                               `json:"name"`
	GivenName  *string                               `json:"given_name"`
	FamilyName *string                               `json:"family_name"`
	Attributes *map[string]userdomain.AttributeValue `json:"attributes"`
}

func toAccountProfileResponse(user *userdomain.User, defs []userdomain.UserAttributeDef) AccountProfileResponse {
	return AccountProfileResponse{
		ID: user.ID, PreferredUsername: user.PreferredUsername,
		Name: user.Name, GivenName: user.GivenName, FamilyName: user.FamilyName,
		Email: user.Email, EmailVerified: user.EmailVerified, MfaEnrolled: user.MfaEnrolled,
		Status:             user.Lifecycle.EffectiveStatus(),
		Attributes:         userusecases.SelfReadableAttributes(user.Attributes, defs),
		ReadableAttributes: userusecases.SelfReadableAttributeDefs(defs),
		EditableAttributes: userusecases.EditableAttributeDefs(defs),
	}
}

func accountProfileDeps(d Deps) userusecases.AccountProfileDeps {
	return userusecases.AccountProfileDeps{
		UserRepo: d.UserRepo, AttrSchemaRepo: d.AttrSchemaRepo, Emit: d.LegacyEmit(),
	}
}

func toAccountSummaryResponse(user *userdomain.User) accountSummaryResponse {
	actions := user.Lifecycle.RequiredActions
	if actions == nil {
		actions = []idmdomain.RequiredAction{}
	}
	return accountSummaryResponse{
		ID: user.ID, PreferredUsername: user.PreferredUsername, Name: user.Name,
		Email: user.Email, EmailVerified: user.EmailVerified, MfaEnrolled: user.MfaEnrolled,
		Status:            user.Lifecycle.EffectiveStatus(),
		LastLoginAt:       user.Lifecycle.LastLoginAt,
		PasswordChangedAt: user.Lifecycle.PasswordChangedAt,
		RequiredActions:   actions,
	}
}

func HandleGetAccountSummary(d Deps, c *echo.Context) error {
	sub, err := requireAuthenticatedSub(d, c)
	if err != nil {
		return writeAccountError(c, err)
	}
	user, _, err := userusecases.GetUserProfile(c.Request().Context(), accountProfileDeps(d), sub)
	if err != nil {
		return writeAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAccountSummaryResponse(user))
}

func HandleGetAccountProfile(d Deps, c *echo.Context) error {
	sub, err := requireAuthenticatedSub(d, c)
	if err != nil {
		return writeAccountError(c, err)
	}
	user, defs, err := userusecases.GetUserProfile(c.Request().Context(), accountProfileDeps(d), sub)
	if err != nil {
		return writeAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAccountProfileResponse(user, defs))
}

func HandleUpdateAccountProfile(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	sub, err := requireAuthenticatedSub(d, c)
	if err != nil {
		return writeAccountError(c, err)
	}
	var input accountProfileUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	user, defs, err := userusecases.UpdateUserProfile(c.Request().Context(), accountProfileDeps(d),
		userusecases.UpdateUserProfileInput{
			Sub: sub, Name: input.Name, GivenName: input.GivenName, FamilyName: input.FamilyName,
			Attributes: input.Attributes, Now: time.Now().UTC(),
		})
	if err != nil {
		return writeAccountError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAccountProfileResponse(user, defs))
}

// requireAuthenticatedSub は認証済み (pending でない) セッションの sub を返す。
// self-service では actor == target なので sub をそのまま操作対象に使う。
func requireAuthenticatedSub(d Deps, c *echo.Context) (string, error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", support.ErrAdminAuthenticationRequired
	}
	return authn.UserID, nil
}

func writeAccountError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, support.ErrAdminAuthenticationRequired):
		return support.WriteBrowserError(c, http.StatusUnauthorized, "authentication_required", "認証済みセッションが必要です")
	case errors.Is(err, mfausecases.ErrStepUpRequired):
		return support.WriteBrowserError(c, http.StatusForbidden, "step_up_required", "この操作には再認証が必要です")
	case errors.Is(err, idmusecases.ErrUserNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "ユーザーが存在しません")
	case errors.Is(err, sessionusecases.ErrSessionNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "session_not_found", "セッションが存在しません")
	case errors.Is(err, userusecases.ErrAttributeNotEditable):
		return support.WriteBrowserError(c, http.StatusForbidden, "attribute_not_editable", "この属性は編集できません")
	case errors.Is(err, userusecases.ErrInvalidAttribute):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_attribute", "属性がスキーマに適合していません")
	default:
		return err
	}
}

// requireStepUpSub は認証済みセッションを解決し、step-up gate を通過した sub を返す
// (primary email 変更など高 sensitivity な identity 操作用)。
func requireStepUpSub(d Deps, c *echo.Context) (string, error) {
	authn, err := d.ResolveAuthentication(c)
	if err != nil {
		return "", err
	}
	if authn == nil || authn.AuthenticationPending {
		return "", support.ErrAdminAuthenticationRequired
	}
	if !mfausecases.StepUpSatisfied(authn, time.Now().UTC()) {
		return "", mfausecases.ErrStepUpRequired
	}
	return authn.UserID, nil
}
