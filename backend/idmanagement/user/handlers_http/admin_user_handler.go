package handlers_http

import (
	"errors"
	"net/http"
	"slices"
	"time"

	authusecases "github.com/ambi/idmagic/backend/authentication/password/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type adminUserCreateRequest struct {
	PreferredUsername string   `json:"preferred_username"`
	Password          string   `json:"password"`
	Name              *string  `json:"name"`
	Email             *string  `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Roles             []string `json:"roles"`
}

type adminUserUpdateRequest struct {
	PreferredUsername *string                               `json:"preferred_username"`
	Name              *string                               `json:"name"`
	GivenName         *string                               `json:"given_name"`
	FamilyName        *string                               `json:"family_name"`
	Email             *string                               `json:"email"`
	EmailVerified     *bool                                 `json:"email_verified"`
	Roles             *[]string                             `json:"roles"`
	Attributes        *map[string]userdomain.AttributeValue `json:"attributes"`
}

type adminUserDeleteRequest struct {
	Reason string `json:"reason"`
	// Force が true のとき soft-delete をスキップして即時完全削除 (purge) する。
	// クエリ ?purge=true と同義。
	Force bool `json:"force"`
}

type adminUserResponse struct {
	ID                string                               `json:"id"`
	PreferredUsername string                               `json:"preferred_username"`
	Name              *string                              `json:"name,omitempty"`
	GivenName         *string                              `json:"given_name,omitempty"`
	FamilyName        *string                              `json:"family_name,omitempty"`
	Email             *string                              `json:"email,omitempty"`
	EmailVerified     bool                                 `json:"email_verified"`
	MfaEnrolled       bool                                 `json:"mfa_enrolled"`
	Roles             []string                             `json:"roles"`
	Status            idmdomain.UserStatus                 `json:"status"`
	Attributes        map[string]userdomain.AttributeValue `json:"attributes,omitempty"`
	RequiredActions   []idmdomain.RequiredAction           `json:"required_actions,omitempty"`
	LastLoginAt       *time.Time                           `json:"last_login_at,omitempty"`
	PasswordChangedAt *time.Time                           `json:"password_changed_at,omitempty"`
	// DisabledAt は status から導出した後方互換フィールド (現行 UI 用)。
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	// PendingDeletionAt は status == PendingDeletion のとき soft-delete された時刻
	// (status_changed_at)。PurgeAfter は自動 purge される時刻 (soft-delete + 猶予期間)。
	// UI が猶予残日数を表示するために使う。
	PendingDeletionAt *time.Time `json:"pending_deletion_at,omitempty"`
	PurgeAfter        *time.Time `json:"purge_after,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ScimSource        *string    `json:"scim_source,omitempty"`
}

type adminRequiredActionRequest struct {
	Action string `json:"action"`
}

func HandleListAdminUsers(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	// lazy-on-access: 猶予期間を過ぎた削除予約 user を一覧取得のついでに Purge する。
	// 専用スケジューラは別 WI に切り出す。
	if err := userusecases.PurgeExpiredSoftDeleted(c.Request().Context(), adminUserDeps(d), time.Now().UTC()); err != nil {
		return err
	}
	users, err := d.UserRepo.FindAll(c.Request().Context(), support.RequestTenantID(c))
	if err != nil {
		return err
	}
	response := make([]adminUserResponse, len(users))
	for i, user := range users {
		response[i] = toAdminUserResponse(user)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"users": response})
}

func HandleGetAdminUser(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	user, err := d.UserRepo.FindBySub(c.Request().Context(), c.Param("sub"))
	if err != nil {
		return err
	}
	if user == nil {
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "The user does not exist.")
	}
	if user.TenantID != support.RequestTenantID(c) {
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "The user does not exist.")
	}
	res := toAdminUserResponse(user)
	if d.ScimRepo != nil {
		ref, _ := d.ScimRepo.FindUserRefByUserID(c.Request().Context(), user.TenantID, user.ID)
		if ref != nil {
			src := "SCIM"
			res.ScimSource = &src
		}
	}
	return support.NoStoreJSON(c, http.StatusOK, res)
}

func HandleCreateAdminUser(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input adminUserCreateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	user, err := userusecases.CreateUser(
		ctx,
		adminUserDeps(d),
		userusecases.CreateUserInput{
			ActorUserID: actor.ID, PreferredUsername: input.PreferredUsername,
			Password: input.Password, Name: input.Name, Email: input.Email,
			EmailVerified: input.EmailVerified, Roles: input.Roles, Now: time.Now().UTC(),
		},
	)
	if err != nil {
		return writeAdminUserError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toAdminUserResponse(user))
}

func HandleUpdateAdminUser(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input adminUserUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	user, err := userusecases.UpdateUser(
		ctx,
		adminUserDeps(d),
		userusecases.UpdateUserInput{
			ActorUserID: actor.ID, Sub: c.Param("sub"),
			PreferredUsername: input.PreferredUsername, Name: input.Name,
			GivenName: input.GivenName, FamilyName: input.FamilyName, Email: input.Email,
			EmailVerified: input.EmailVerified, Roles: input.Roles,
			Attributes: input.Attributes, Now: time.Now().UTC(),
		},
	)
	if err != nil {
		return writeAdminUserError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminUserResponse(user))
}

func HandleDisableAdminUser(d Deps, c *echo.Context) error {
	return handleSetAdminUserDisabled(d, c, true)
}

func HandleEnableAdminUser(d Deps, c *echo.Context) error {
	return handleSetAdminUserDisabled(d, c, false)
}

func HandleDeleteAdminUser(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input adminUserDeleteRequest
	if c.Request().ContentLength > 0 {
		if err := support.DecodeJSON(c.Request(), &input); err != nil {
			return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
		}
	}
	// 既定は soft-delete (削除予約)。?purge=true または body force=true で完全削除
	// (ADR-036 の anonymize cascade) に分岐する。
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	if c.QueryParam("purge") == "true" || input.Force {
		err = userusecases.DeleteUser(ctx, adminUserDeps(d), userusecases.DeleteUserInput{
			ActorUserID: actor.ID, Sub: c.Param("sub"), Reason: input.Reason, Now: time.Now().UTC(),
		})
	} else {
		err = userusecases.SoftDeleteUser(ctx, adminUserDeps(d), userusecases.SoftDeleteUserInput{
			ActorUserID: actor.ID, Sub: c.Param("sub"), Reason: input.Reason, Now: time.Now().UTC(),
		})
	}
	if err != nil {
		return writeAdminUserError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleRestoreAdminUser(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	user, err := userusecases.RestoreUser(
		ctx, adminUserDeps(d), actor.ID, c.Param("sub"), time.Now().UTC(),
	)
	if err != nil {
		return writeAdminUserError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminUserResponse(user))
}

func handleSetAdminUserDisabled(d Deps, c *echo.Context, disabled bool) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	_, err = userusecases.SetUserDisabled(
		ctx, adminUserDeps(d), actor.ID, c.Param("sub"), disabled, time.Now().UTC(),
	)
	if err != nil {
		return writeAdminUserError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func adminUserDeps(d Deps) userusecases.AdminUserDeps {
	deps := userusecases.AdminUserDeps{
		UserRepo: d.UserRepo, GroupRepo: d.GroupRepo, AttrSchemaRepo: d.AttrSchemaRepo,
		UserMutationCommitter: d.UserMutationCommitter,
		ProvisioningNotifier:  d.ProvisioningNotifier,
		ConsentRepo:           d.ConsentRepo, RefreshStore: d.RefreshStore,
		DeviceCodeStore: d.DeviceCodeStore, MfaFactorRepo: d.MfaFactorRepo,
		PasswordHasher: d.PasswordHasher, PasswordHistoryRepo: d.PasswordHistoryRepo,
		Emit: d.LegacyEmit(), QuotaRepo: d.QuotaRepo,
	}
	if d.SessionManager != nil {
		deps.SessionStore = d.SessionManager.Store
	}
	return deps
}

func writeAdminUserError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, idmusecases.ErrUserNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "user_not_found", "The user does not exist.")
	case errors.Is(err, userusecases.ErrUsernameConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "username_conflict", "The username is already in use.")
	case errors.Is(err, idmusecases.ErrInvalidRole):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_role", "The role is invalid.")
	case errors.Is(err, userusecases.ErrSelfDeleteForbidden):
		return support.WriteBrowserError(c, http.StatusBadRequest, "self_delete_forbidden", "Administrators cannot delete themselves.")
	case errors.Is(err, userusecases.ErrSelfDisableForbidden):
		return support.WriteBrowserError(c, http.StatusBadRequest, "self_disable_forbidden", "Administrators cannot disable themselves.")
	case errors.Is(err, userusecases.ErrUserNotPendingDeletion):
		return support.WriteBrowserError(c, http.StatusBadRequest, "not_pending_deletion", "The user is not scheduled for deletion.")
	case errors.Is(err, userusecases.ErrRestoreGracePeriodExpired):
		return support.WriteBrowserError(c, http.StatusBadRequest, "restore_grace_expired", "The restoration grace period has expired.")
	case errors.Is(err, userusecases.ErrInvalidAttribute):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_attribute", "The attribute does not conform to the schema.")
	case errors.Is(err, userusecases.ErrInvalidRequiredAction):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_required_action", "The required action is invalid.")
	default:
		var policyErr *authusecases.PasswordPolicyError
		if errors.As(err, &policyErr) {
			violations := make([]string, len(policyErr.Violations))
			for i, violation := range policyErr.Violations {
				violations[i] = string(violation)
			}
			return support.NoStoreJSON(c, http.StatusBadRequest, map[string]any{
				"error": "password_policy", "message": "The password does not meet the security requirements.",
				"violations": violations,
			})
		}
		return err
	}
}

func toAdminUserResponse(user *userdomain.User) adminUserResponse {
	var disabledAt *time.Time
	if user.Lifecycle.Status == idmdomain.UserStatusDisabled {
		disabledAt = user.Lifecycle.StatusChangedAt
	}
	var pendingDeletionAt, purgeAfter *time.Time
	if user.Lifecycle.EffectiveStatus() == idmdomain.UserStatusPendingDeletion {
		pendingDeletionAt = user.Lifecycle.StatusChangedAt
		if pendingDeletionAt != nil {
			deadline := pendingDeletionAt.Add(userusecases.UserSoftDeleteGracePeriodSeconds * time.Second)
			purgeAfter = &deadline
		}
	}
	return adminUserResponse{
		ID: user.ID, PreferredUsername: user.PreferredUsername, Name: user.Name,
		GivenName: user.GivenName, FamilyName: user.FamilyName,
		Email: user.Email, EmailVerified: user.EmailVerified, MfaEnrolled: user.MfaEnrolled,
		Roles: slices.Clone(user.Roles), Status: user.Lifecycle.EffectiveStatus(),
		Attributes:        user.Attributes,
		RequiredActions:   slices.Clone(user.Lifecycle.RequiredActions),
		LastLoginAt:       user.Lifecycle.LastLoginAt,
		PasswordChangedAt: user.Lifecycle.PasswordChangedAt,
		DisabledAt:        disabledAt,
		PendingDeletionAt: pendingDeletionAt,
		PurgeAfter:        purgeAfter,
		CreatedAt:         user.CreatedAt, UpdatedAt: user.UpdatedAt,
	}
}

func HandleSetUserRequiredAction(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input adminRequiredActionRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	user, err := userusecases.SetUserRequiredAction(
		ctx, adminUserDeps(d), actor.ID, c.Param("sub"),
		idmdomain.RequiredAction(input.Action), time.Now().UTC(),
	)
	if err != nil {
		return writeAdminUserError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminUserResponse(user))
}

func HandleClearUserRequiredAction(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	ctx, cancel := d.OperationContext(c.Request().Context())
	defer cancel()
	user, err := userusecases.ClearUserRequiredAction(
		ctx, adminUserDeps(d), actor.ID, c.Param("sub"),
		idmdomain.RequiredAction(c.Param("action")), time.Now().UTC(),
	)
	if err != nil {
		return writeAdminUserError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminUserResponse(user))
}
